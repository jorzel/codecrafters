package commands

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
)

// SetHandler is responsible for handling SET commands.
// replication means that the command is being propagated to replicas from the master.
// it can be ping, replconf, or any write command that is propagated to replicas.
type SetHandler struct {
	storage                storage.Storage
	transactionsController TransactionsController
}

func NewSetHandler(
	storage storage.Storage, transactionsController TransactionsController,
) *SetHandler {
	return &SetHandler{
		storage:                storage,
		transactionsController: transactionsController,
	}
}

func (h *SetHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	connID := conn.RemoteAddr().String()
	var msg []byte
	var commandErr error
	if h.transactionsController.IsTransactionActive(connID) {
		err := h.transactionsController.QueueCommand(connID, command)
		if err != nil {
			errMsg := "Failed to queue command in transaction: " + err.Error()
			return HandleResult{
				CommandError: fmt.Errorf(errMsg),
			}, fmt.Errorf(errMsg)
		}
		msg = protocol.SimpleString("QUEUED")
	} else {
		msg, commandErr = applySet(h.storage, command)
	}
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

// ExecSetHandler is responsible for handling SET commands that are executed in a transaction.
type ExecSetHandler struct {
	storage storage.Storage
}

func NewExecSetHandler(storage storage.Storage) *ExecSetHandler {
	return &ExecSetHandler{
		storage: storage,
	}
}

func (h *ExecSetHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := applySet(h.storage, command)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, nil
}

// ReplicatedSetHandler
type ReplicatedSetHandler struct {
	storage storage.Storage
}

func NewReplicatedSetHandler(storage storage.Storage) *ReplicatedSetHandler {
	return &ReplicatedSetHandler{
		storage: storage,
	}
}

func (h *ReplicatedSetHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	_, commandErr := applySet(h.storage, command)
	if commandErr != nil {
		return HandleResult{
			CommandError: commandErr,
		}, nil
	}
	return HandleResult{
		CommandError: nil,
	}, nil
}

func applySet(store storage.Storage, command protocol.Command) ([]byte, error) {
	if len(command.Args) < 2 {
		errMsg := "SET command requires at least 2 arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	record := storage.StringRecord{
		Value: command.Args[1],
	}
	if len(command.Args) > 2 {
		// If an expiration time is provided, parse it
		if strings.ToLower(command.Args[2]) == "px" {
			if len(command.Args) < 4 {
				errMsg := "SET command with PX requires 4 arguments"
				return protocol.Error(errMsg), fmt.Errorf(errMsg)
			}
			expiration, err := time.ParseDuration(command.Args[3] + "ms")
			if err != nil {
				errMsg := "Invalid expiration time: " + err.Error()
				return protocol.Error(errMsg), fmt.Errorf(errMsg)
			}
			expireAt := time.Now().Add(expiration)
			record.ExpireAt = &expireAt
		} else {
			errMsg := "Unsupported expiration format, only PX is supported"
			return protocol.Error(errMsg), fmt.Errorf(errMsg)
		}
	}

	store.SetString(command.Args[0], &record)
	return protocol.SimpleString("OK"), nil
}

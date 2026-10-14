package commands

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
)

type ExecIncrHandler struct {
	storage storage.Storage
}

func NewExecIncrHandler(storage storage.Storage) *ExecIncrHandler {
	return &ExecIncrHandler{
		storage: storage,
	}
}

// ExecIncrHandler is responsible for handling INCR commands that are executed in a transaction.
func (h *ExecIncrHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := applyIncr(h.storage, command)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, nil
}

type IncrHandler struct {
	storage                storage.Storage
	transactionsController TransactionsController
}

func NewIncrHandler(
	storage storage.Storage, transactionsController TransactionsController,
) *IncrHandler {
	return &IncrHandler{
		storage:                storage,
		transactionsController: transactionsController,
	}
}

func (h *IncrHandler) Handle(
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
		msg, commandErr = applyIncr(h.storage, command)
	}
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func applyIncr(store storage.Storage, command protocol.Command) ([]byte, error) {
	if len(command.Args) != 1 {
		errMsg := "INCR command requires 1 argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	key := command.Args[0]
	record, err := store.GetString(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	var valueInt int
	if record == nil {
		valueInt = 1
		record = &storage.StringRecord{
			Value: strconv.Itoa(valueInt),
		}
	} else {
		valueInt, err = strconv.Atoi(record.Value)
		if err != nil {
			errMsg := "value is not an integer or out of range"
			return protocol.Error(errMsg), fmt.Errorf(errMsg)
		}
		valueInt++
		record.Value = strconv.Itoa(valueInt)
	}
	if err := store.SetString(key, record); err != nil {
		errMsg := "Failed to incr record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	return protocol.SimpleInteger(valueInt), nil
}

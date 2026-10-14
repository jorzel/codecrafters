package commands

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
)

type GetHandler struct {
	storage                storage.Storage
	transactionsController TransactionsController
}

func NewGetHandler(storage storage.Storage, transactionsController TransactionsController) *GetHandler {
	return &GetHandler{
		storage:                storage,
		transactionsController: transactionsController,
	}
}

func (h *GetHandler) Handle(
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
		msg, commandErr = applyGet(h.storage, command)
	}
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

type ExecGetHandler struct {
	storage storage.Storage
}

func NewExecGetHandler(storage storage.Storage) *ExecGetHandler {
	return &ExecGetHandler{
		storage: storage,
	}
}

func (h *ExecGetHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := applyGet(h.storage, command)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, nil
}

func applyGet(store storage.Storage, command protocol.Command) ([]byte, error) {
	if len(command.Args) != 1 {
		errMsg := "GET command requires exactly 1 argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	deserializedRecord, err := store.GetString(command.Args[0])
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if deserializedRecord == nil {
		return protocol.Nil(), nil
	}
	if deserializedRecord.ExpireAt != nil && deserializedRecord.ExpireAt.Before(time.Now()) {
		// If the record has expired, return nil
		return protocol.Nil(), nil
	}
	return protocol.BulkString(deserializedRecord.Value), nil
}

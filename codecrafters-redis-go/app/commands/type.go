package commands

import (
	"context"
	"fmt"
	"net"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
)

const (
	Stream = "stream"
	String = "string"
	List   = "list"
	ZSet   = "zset"
	None   = "none"
)

type TypeHandler struct {
	storage storage.Storage
}

func NewTypeHandler(storage storage.Storage) *TypeHandler {
	return &TypeHandler{
		storage: storage,
	}
}

func (h *TypeHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, err
}

func (h *TypeHandler) apply(command protocol.Command) ([]byte, error) {
	if len(command.Args) != 1 {
		errMsg := "TYPE command requires exactly 1 argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	record, err := h.storage.Get(command.Args[0])
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if record == nil {
		return protocol.SimpleString(None), nil
	}
	_, ok := record.(*storage.StringRecord)
	if ok {
		return protocol.SimpleString(String), nil
	}
	_, ok = record.(*storage.ListRecord)
	if ok {
		return protocol.SimpleString(List), nil
	}
	_, ok = record.(*storage.SortedSetRecord)
	if ok {
		return protocol.SimpleString(ZSet), nil
	}
	_, ok = record.(*storage.StreamRecord)
	if ok {
		return protocol.SimpleString(Stream), nil
	}
	return protocol.SimpleString(None), nil
}

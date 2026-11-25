package commands

import (
	"context"
	"fmt"
	"net"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
)

type XAddHandler struct {
	storage storage.Storage
}

func NewXAddHandler(storage storage.Storage) *XAddHandler {
	return &XAddHandler{
		storage: storage,
	}
}

func (h *XAddHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, err
}

func (h *XAddHandler) apply(command protocol.Command) ([]byte, error) {
	if len(command.Args) < 4 {
		errMsg := "XADD command requires at least 4 arguments (key, id, field, value, ...)"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	key := command.Args[0]
	record, err := h.storage.GetStream(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if record == nil {
		record = storage.NewStreamRecord()
	}
	idString := command.Args[1]
	if idString == "0-0" {
		errMsg := "The ID specified in XADD must be greater than 0-0"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	fields := []storage.Field{}
	for i := 2; i < len(command.Args); i += 2 {
		fields = append(fields, storage.Field{Name: command.Args[i], Value: command.Args[i+1]})
	}
	streamID, err := record.AddItem(idString, fields)
	if err != nil {
		errMsg := "The ID specified in XADD is equal or smaller than the target stream top item"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	err = h.storage.SetStream(key, record)
	if err != nil {
		errMsg := "Failed to set record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	return protocol.BulkString(streamID.String()), nil
}

type XRangeHandler struct {
	storage storage.Storage
}

func NewXRangeHandler(storage storage.Storage) *XRangeHandler {
	return &XRangeHandler{
		storage: storage,
	}
}

func (h *XRangeHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, err
}

func (h *XRangeHandler) apply(command protocol.Command) ([]byte, error) {
	if len(command.Args) != 3 {
		errMsg := "XRange command requires 3 arguments (key, start, end)"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	key := command.Args[0]
	record, err := h.storage.GetStream(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if record == nil {
		return protocol.Array([]string{}), nil
	}
	start, err := storage.ParseRangeStart(command.Args[1])
	if err != nil {
		errMsg := "Failed to parse start: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	end, err := storage.ParseRangeEnd(command.Args[2])
	if err != nil {
		errMsg := "Failed to parse end: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	items := record.Range(start, end)
	replies := []string{}
	for _, item := range items {
		fieldsArray := []string{}
		for _, field := range item.Fields() {
			fieldsArray = append(fieldsArray, field.Name)
			fieldsArray = append(fieldsArray, field.Value)
		}

		pair := protocol.Array([]string{
			string(protocol.BulkString(item.ID.String())),
			string(protocol.BulkArray(fieldsArray)),
		})
		replies = append(replies, string(pair))

	}
	return protocol.Array(replies), nil
}

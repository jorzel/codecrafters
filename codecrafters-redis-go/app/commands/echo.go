package commands

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
)

type EchoHandler struct{}

func NewEchoHandler() *EchoHandler {
	return &EchoHandler{}
}

func (h *EchoHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *EchoHandler) apply(_ context.Context, command protocol.Command) ([]byte, error) {
	if len(command.Args) != 1 {
		errorMsg := "ECHO command requires exactly 1 argument"
		return protocol.Error(errorMsg), fmt.Errorf(errorMsg)
	}
	return protocol.BulkString(strings.Join(command.Args, protocol.CRLF)), nil
}

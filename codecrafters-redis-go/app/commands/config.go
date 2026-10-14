package commands

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/protocol"
)

func sendMsg(_ context.Context, conn net.Conn, msg []byte) error {
	if _, err := conn.Write(msg); err != nil {
		return fmt.Errorf("Failed to write response: " + err.Error())
	}
	return nil
}

type ConfigHandler struct {
	config *config.Config
}

func NewConfigHandler(config *config.Config) *ConfigHandler {
	return &ConfigHandler{
		config: config,
	}
}

func (h *ConfigHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, command)
	err := sendMsg(ctx, conn, msg)

	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *ConfigHandler) apply(_ context.Context, command protocol.Command) ([]byte, error) {
	if len(command.Args) != 2 {
		errMsg := "CONFIG command requires exactly 2 arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if strings.ToLower(command.Args[0]) != "get" {
		errMsg := "CONFIG command only supports 'get' argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if command.Args[1] == "dbfilename" {
		dbFileName := h.config.DbFileName
		return protocol.BulkArray([]string{"dbfilename", dbFileName}), nil
	}
	if command.Args[1] == "dir" {
		dir := h.config.Dir
		return protocol.BulkArray([]string{"dir", dir}), nil
	}
	errMsg := "CONFIG command only supports 'get dbfilename' and 'get dir' arguments, got: " + command.Args[1]
	return protocol.Error(errMsg), fmt.Errorf(errMsg)
}

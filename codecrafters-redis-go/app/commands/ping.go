package commands

import (
	"context"
	"net"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
)

type PingHandler struct {
	replication      bool
	pubsubController PubSubController
}

func NewPingHandler(replication bool, pubsubController PubSubController) *PingHandler {
	return &PingHandler{
		replication:      replication,
		pubsubController: pubsubController,
	}
}

func (h *PingHandler) Handle(
	ctx context.Context, conn net.Conn, _ protocol.Command,
) (HandleResult, error) {
	if h.pubsubController != nil {
		clientID := conn.RemoteAddr().String()
		if h.pubsubController.IsSubscribedMode(clientID) {
			msg := protocol.BulkArray([]string{"pong", ""})
			sendMsg(ctx, conn, msg)
			return HandleResult{}, nil
		}
	}

	msg := protocol.SimpleString("PONG")
	var err error
	if !h.replication {
		err = sendMsg(ctx, conn, msg)
	}
	return HandleResult{}, err
}

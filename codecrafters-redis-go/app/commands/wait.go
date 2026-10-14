package commands

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/replication"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
	"github.com/rs/zerolog"
)

type WaitHandler struct {
	storage      storage.Storage
	replicasInfo *replication.ReplicasInfo
	replicator   Replicator
	waits        *replication.AckWaitStore
}

func NewWaitHandler(
	replicasInfo *replication.ReplicasInfo, replicator Replicator, waits *replication.AckWaitStore,
) *WaitHandler {
	return &WaitHandler{
		storage:      storage.NewStorage(),
		replicasInfo: replicasInfo,
		replicator:   replicator,
		waits:        waits,
	}
}

func (h *WaitHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, conn, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *WaitHandler) apply(ctx context.Context, conn net.Conn, command protocol.Command) ([]byte, error) {
	logger := zerolog.Ctx(ctx)
	if len(command.Args) != 2 {
		errMsg := "WAIT command requires exactly 2 arguments: num_replicas, timeout"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	expectedAcks, err := strconv.Atoi(command.Args[0])
	if err != nil {
		errMsg := "Invalid number of replicas: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	timeoutMilliseconds, err := strconv.Atoi(command.Args[1])
	if err != nil {
		errMsg := "Invalid timeout value: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	timeout := time.Duration(timeoutMilliseconds) * time.Millisecond

	replicasCount := h.replicasInfo.Replicas.GetCount()
	masterOffset := h.replicasInfo.GetMasterOffset()
	if masterOffset == 0 {
		logger.Debug().
			Int("replicas_count", replicasCount).
			Msg("No replicated data, returning count")
		return protocol.SimpleInteger(replicasCount), nil
	}

	expectedOffset, exists := h.replicasInfo.GetExpectedMasterOffset(conn)
	if !exists {
		logger.Debug().
			Str("remote_addr", conn.RemoteAddr().String()).
			Int("replicas_count", replicasCount).
			Interface("expected_offsets", h.replicasInfo.ExpectedOffset).
			Msg("Cannot find any pending replication to wait for client")
		return protocol.SimpleInteger(h.replicasInfo.Replicas.GetCount()), nil
	}

	acks := h.ackWaitLoop(ctx, expectedAcks, expectedOffset, timeout)
	return protocol.SimpleInteger(acks), nil
}

func (h *WaitHandler) ackWaitLoop(
	ctx context.Context, expectedAcks int, expectedOffset int, timeout time.Duration,
) int {
	logger := zerolog.Ctx(ctx)

	ackCheckInterval := 100 * time.Millisecond
	deadline := time.Now().Add(timeout)

	conns := h.replicasInfo.Replicas.GetAll()

	h.waits.Clear() // Clear previous waits
	acks := 0

	go h.getAckLoop(ctx, deadline)
	for time.Now().Before(deadline) {
		for _, rConn := range conns {
			connID := rConn.RemoteAddr().String()
			logger.Info().
				Str("remote_addr", connID).
				Int("expected_count", expectedAcks).
				Int("success_count", acks).
				Msg("Waiting for ACK from replica")
			ackWait, ok := h.waits.Get(connID)
			if !ok {
				h.waits.Set(connID, replication.AckWait{
					IsSuccess:      false,
					ExpectedOffset: expectedOffset,
				})
			}
			if ackWait.IsSuccess {
				continue
			}
		}
		acks = h.waits.CountSuccessful()

		if acks >= expectedAcks {
			break
		}
		time.Sleep(ackCheckInterval)
	}

	return h.waits.CountSuccessful()
}

func (h *WaitHandler) getAckLoop(ctx context.Context, deadline time.Time) {
	getAckInterval := 100 * time.Millisecond

	for time.Now().Before(deadline) {
		time.Sleep(getAckInterval)
		h.replicator.ReplConfGetAck(ctx)
	}
}

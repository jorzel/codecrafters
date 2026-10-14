package commands

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/replication"
	"github.com/rs/zerolog"
)

type PsyncHandler struct {
	replicasInfo *replication.ReplicasInfo
}

func NewPsyncHandler(replicasInfo *replication.ReplicasInfo) *PsyncHandler {
	return &PsyncHandler{
		replicasInfo: replicasInfo,
	}
}

func (h *PsyncHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	logger := zerolog.Ctx(ctx)

	msg, commandErr := h.apply(ctx, conn, command)
	err := sendMsg(ctx, conn, msg)
	if err != nil || commandErr != nil {
		return HandleResult{
			CommandError: commandErr,
		}, err
	}
	//time.Sleep(100 * time.Millisecond) // Simulate some delay
	msg, commandErr = h.getDBFile(ctx)

	if commandErr == nil {
		if conn == nil {
			logger.Error().Msg("Connection is nil, cannot add to replicas connections")
			return HandleResult{
				CommandError: nil,
			}, fmt.Errorf("connection is nil")
		}
		logger.Info().Str("replica_addr", conn.RemoteAddr().String()).Msg("Adding replica connection")
		err = h.replicasInfo.Replicas.Add(conn)
		if err != nil {
			return HandleResult{
				CommandError: commandErr,
			}, fmt.Errorf("failed to add replica connection: %w", err)
		}
	}

	err = sendMsg(ctx, conn, msg)
	logger.Info().Msg("Sending DB file to replica")
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *PsyncHandler) apply(
	ctx context.Context, conn net.Conn, command protocol.Command,
) ([]byte, error) {
	if len(command.Args) != 2 {
		errMsg := "PSYNC command requires exactly 2 arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if command.Args[0] != "?" || command.Args[1] != "-1" {
		errMsg := "PSYNC command only supports '?' and '-1' arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	return protocol.SimpleString("FULLRESYNC " + h.replicasInfo.ReplID + " 0"), nil
}

func (h *PsyncHandler) getDBFile(_ context.Context) ([]byte, error) {
	emptyDBFile := "524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2"
	DBContent, _ := hex.DecodeString(emptyDBFile)
	return protocol.FileContent(DBContent), nil
}

type ReplicaReplConfHandler struct {
	masterInfo *replication.MasterInfo
}

func NewReplicaReplConfHandler(masterInfo *replication.MasterInfo) *ReplicaReplConfHandler {
	return &ReplicaReplConfHandler{
		masterInfo: masterInfo,
	}
}

func (h *ReplicaReplConfHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	var err error
	msg, commandErr := h.executeReplicaReplConf(ctx, conn, command)
	if msg != nil {
		err = sendMsg(ctx, conn, msg)
	}
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *ReplicaReplConfHandler) executeReplicaReplConf(
	_ context.Context, _ net.Conn, command protocol.Command,
) ([]byte, error) {
	if len(command.Args) != 2 {
		errMsg := "REPLCONF command requires exactly 2 arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	if strings.ToLower(command.Args[0]) == "getack" {
		offset := fmt.Sprintf("%d", h.masterInfo.GetReplicaOffset())
		return protocol.BulkArray([]string{"REPLCONF", "ACK", offset}), nil
	}
	errMsg := fmt.Sprintf(
		"REPLCONF command for replica only accept ack arguments, got: %s",
		strings.Join(command.Args, ", "),
	)
	return protocol.Error(errMsg), fmt.Errorf(errMsg)
}

type MasterReplConfHandler struct {
	waits       *replication.AckWaitStore
	replication bool
}

func NewMasterReplConfHandler(
	waits *replication.AckWaitStore,
) *MasterReplConfHandler {
	return &MasterReplConfHandler{
		waits: waits,
	}
}

func (h *MasterReplConfHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	var err error
	msg, commandErr := h.executeMasterReplConf(ctx, conn, command)
	if msg != nil {
		err = sendMsg(ctx, conn, msg)
	}
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *MasterReplConfHandler) executeMasterReplConf(
	ctx context.Context, conn net.Conn, command protocol.Command,
) ([]byte, error) {
	if len(command.Args) != 2 {
		errMsg := "REPLCONF command requires exactly 2 arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	if strings.ToLower(command.Args[0]) == "listening-port" {
		return protocol.SimpleString("OK"), nil
	} else if strings.ToLower(command.Args[0]) == "capa" && strings.ToLower(command.Args[1]) == "psync2" {
		return protocol.SimpleString("OK"), nil
	} else if strings.ToLower(command.Args[0]) == "ack" {
		offset, err := strconv.Atoi(command.Args[1])
		if err != nil {
			errMsg := "Invalid offset value for REPLCONF ACK: " + err.Error()
			return protocol.Error(errMsg), fmt.Errorf(errMsg)
		}
		h.executeReplConfAck(ctx, conn, offset)
		return nil, nil
	}
	errMsg := fmt.Sprintf(
		"REPLCONF command only supports 'listening-port <PORT>', ack, getack, and 'capa psync2' arguments, got: %s",
		strings.Join(command.Args, ", "),
	)
	return protocol.Error(errMsg), fmt.Errorf(errMsg)
}

func (h *MasterReplConfHandler) executeReplConfAck(ctx context.Context, conn net.Conn, ackOffset int) {
	logger := *zerolog.Ctx(ctx)
	connID := conn.RemoteAddr().String()
	logger = logger.With().Str("remote_addr", connID).Logger()

	logger.Debug().
		Int("ack_offset", ackOffset).
		Msg("Received REPLCONF ACK from replica")
	ackWait, ok := h.waits.Get(connID)
	if !ok {
		logger.Warn().
			Msg("No ACK wait found for this connection, creating a new one")
	}
	if ackOffset >= ackWait.ExpectedOffset {
		logger.Debug().Msg("ACK received for expected offset, marking success")
		ackWait.IsSuccess = true
		h.waits.Set(connID, ackWait)
	} else {
		logger.Warn().
			Int("expected_offset", ackWait.ExpectedOffset).
			Msg("ACK received for an unexpected offset, ignoring")
	}
}

type MasterInfoHandler struct {
	replicasInfo *replication.ReplicasInfo
}

func NewMasterInfoHandler(replicasInfo *replication.ReplicasInfo) *MasterInfoHandler {
	return &MasterInfoHandler{
		replicasInfo: replicasInfo,
	}
}

func (h *MasterInfoHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *MasterInfoHandler) apply(_ context.Context, command protocol.Command) ([]byte, error) {
	if len(command.Args) > 1 {
		errMsg := "INFO command too many arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	if len(command.Args) == 1 && strings.ToLower(command.Args[0]) != "replication" {
		errMsg := "INFO command only supports 'replication' argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	return protocol.BulkString(h.masterInfoReplication()), nil
}

func (h *MasterInfoHandler) masterInfoReplication() string {
	params := map[string]string{}
	params["role"] = "master"
	params["master_repl_offset"] = strconv.Itoa(h.replicasInfo.GetMasterOffset())
	params["master_replid"] = h.replicasInfo.ReplID

	infoString := "#Replication"
	for key, value := range params {
		infoString += protocol.CRLF + key + ":" + value
	}
	return infoString
}

type ReplicaInfoHandler struct{}

func NewReplicaInfoHandler() *ReplicaInfoHandler {
	return &ReplicaInfoHandler{}
}

func (h *ReplicaInfoHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *ReplicaInfoHandler) apply(_ context.Context, command protocol.Command) ([]byte, error) {
	if len(command.Args) > 1 {
		errMsg := "INFO command too many arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	if len(command.Args) == 1 && strings.ToLower(command.Args[0]) != "replication" {
		errMsg := "INFO command only supports 'replication' argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	return protocol.BulkString(h.replicaInfoReplication()), nil
}

func (h *ReplicaInfoHandler) replicaInfoReplication() string {
	params := map[string]string{}
	params["role"] = "slave"

	infoString := "#Replication"
	for key, value := range params {
		infoString += protocol.CRLF + key + ":" + value
	}
	return infoString
}

type FullResyncHandler struct {
	masterInfo *replication.MasterInfo
}

func NewFullResyncHandler(masterInfo *replication.MasterInfo) *FullResyncHandler {
	return &FullResyncHandler{
		masterInfo: masterInfo,
	}
}

func (h *FullResyncHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("command", command.Name).
		Str("args", strings.Join(command.Args, ", ")).
		Msg("Handling FULLRESYNC command")
	h.masterInfo.ReplID = command.Args[0]
	offset, err := strconv.Atoi(command.Args[1])
	h.masterInfo.SetReplicaOffset(offset)
	return HandleResult{}, err
}

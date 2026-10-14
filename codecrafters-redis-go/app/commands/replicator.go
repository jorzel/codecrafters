package commands

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/replication"
	"github.com/rs/zerolog"
)

type Replicator interface {
	Propagate(ctx context.Context, conn net.Conn, command protocol.Command) error
	ReplConfGetAck(ctx context.Context)
}

type DefaultReplicator struct {
	replicasInfo *replication.ReplicasInfo
}

func NewReplicator(replicasInfo *replication.ReplicasInfo) Replicator {
	return &DefaultReplicator{
		replicasInfo: replicasInfo,
	}
}

// Propagate sends a write command handled by the master to all replicas
func (r *DefaultReplicator) Propagate(ctx context.Context, clientConn net.Conn, command protocol.Command) error {
	for _, replicaConn := range r.replicasInfo.Replicas.GetAll() {
		if err := r.propagate(ctx, replicaConn, command); err != nil {
			return err
		}
	}
	masterOffset := r.replicasInfo.AddMasterOffset(command)
	r.replicasInfo.SetExpectedOffset(clientConn, masterOffset)
	return nil
}

func (r *DefaultReplicator) propagate(ctx context.Context, conn net.Conn, command protocol.Command) error {
	logger := zerolog.Ctx(ctx)
	content := []string{command.Name}
	for _, value := range command.Args {
		content = append(content, value)
	}
	msg := protocol.BulkArray(content)
	logger.Debug().Str("message", string(msg)).Str("command", command.Name).Msg("Sending command for replication")
	_, err := conn.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to send SET command: %w", err)
	}
	return nil
}

func (s *DefaultReplicator) ReplConfGetAck(ctx context.Context) {
	command := protocol.Command{
		Name:      protocol.REPLCONF,
		Args:      []string{"GETACK", "*"},
		BytesSize: len(protocol.BulkArray([]string{protocol.REPLCONF, "GETACK", "*"})),
	}
	for _, rConn := range s.replicasInfo.Replicas.GetAll() {
		go func(ctx context.Context, command protocol.Command) {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
			defer cancel()
			s.replConf(ctx, rConn, command)
		}(ctx, command)
		time.Sleep(3 * time.Millisecond) // Simulate some delay
	}
	s.replicasInfo.AddMasterOffset(command)
}

func (s *DefaultReplicator) replConf(ctx context.Context, conn net.Conn, command protocol.Command) error {
	logger := zerolog.Ctx(ctx)
	content := []string{command.Name}
	if len(command.Args) > 0 {
		content = append(content, command.Args...)
	}
	msg := protocol.BulkArray(content)
	logger.Debug().Str("message", string(msg)).Msg("Sending REPLCONF command")
	_, err := conn.Write(protocol.BulkArray(content))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to send REPLCONF command")
		return fmt.Errorf("failed to send REPLCONF command: %w", err)
	}
	return nil
}

// temporary function to simulate checking if enough acks have been received
// should part of protocol parsing
func extractAckValue(resp string) (int, error) {
	const prefix = "*3\r\n$8\r\nREPLCONF\r\n$3\r\nACK\r\n$3\r\n"
	if !strings.HasPrefix(resp, prefix) {
		return 0, fmt.Errorf("unexpected format")
	}

	// Trim the prefix and extract the number before \r\n
	rest := strings.TrimPrefix(resp, prefix)
	numStr := strings.TrimSuffix(rest, "\r\n")
	return strconv.Atoi(numStr)
}

func readLine(conn net.Conn) (string, error) {
	buf := make([]byte, 0, 512)
	tmp := make([]byte, 1)
	for {
		_, err := conn.Read(tmp)
		if err != nil {
			return "", err
		}
		if tmp[0] == '\n' && len(buf) > 0 && buf[len(buf)-1] == '\r' {
			return string(buf[:len(buf)-1]), nil // strip \r
		}
		buf = append(buf, tmp[0])
	}
}

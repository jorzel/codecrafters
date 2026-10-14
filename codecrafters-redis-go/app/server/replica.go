package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"slices"
	"strconv"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/replication"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
	"github.com/rs/zerolog"
)

var _ Server = (*ReplicaServer)(nil)

type ReplicaServer struct {
	listener                 net.Listener
	commandParser            protocol.CommandParser
	commandHandler           commands.CommandDispatcher
	replicatedCommandHandler commands.CommandDispatcher
	config                   *config.Config
	masterInfo               *replication.MasterInfo
	role                     string
	storage                  storage.Storage
}

func NewReplicaServer(cfg *config.Config) (*ReplicaServer, error) {
	addr := fmt.Sprintf("0.0.0.0:%d", cfg.ServerPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	storage := storage.NewStorage()
	return &ReplicaServer{
		listener:                 ln,
		commandParser:            protocol.NewCommandParser(),
		config:                   cfg,
		commandHandler:           commands.NewReplicaCommandDispatcher(storage, cfg),
		replicatedCommandHandler: nil,
		role:                     config.ReplicaRole,
		storage:                  storage,
	}, nil
}

func (rs *ReplicaServer) Start(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	if err := rs.setup(ctx); err != nil {
		return fmt.Errorf("failed to setup replica server: %w", err)
	}
	logger.Info().Str("address", rs.listener.Addr().String()).Msg("Replica listening on...")
	for {
		conn, err := rs.listener.Accept()
		if err != nil {
			continue
		}
		go rs.handleConnection(ctx, conn)
	}
}

func (rs *ReplicaServer) isMasterConnected(ctx context.Context) bool {
	logger := zerolog.Ctx(ctx)
	if rs.masterInfo == nil {
		logger.Warn().Msg("Master info is not set, cannot check connection status")
		return false
	}
	if rs.replicatedCommandHandler == nil {
		logger.Warn().Msg("Replicated command handler is not set")
		return false
	}
	return rs.masterInfo.IsConnected()
}

func (rs *ReplicaServer) setup(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)

	if rs.config.ReplicaOf == nil {
		return fmt.Errorf("replica server is not configured to replicate from a master server")
	}

	masterAddress := fmt.Sprintf("%s:%d", rs.config.ReplicaOf.Host, rs.config.ReplicaOf.Port)
	logger.Info().
		Str("replica_of", masterAddress).
		Msg("Server configured as replica of")
	err := rs.handshake(ctx, masterAddress)

	// Start a goroutine to handle the connection to the master server
	// to be able to handle replication writes
	return err
}

func (rs *ReplicaServer) handshake(ctx context.Context, address string) error {
	logger := zerolog.Ctx(ctx).With().Timestamp().Logger()
	logger.Info().Msg("Performing handshake with the master server...")

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to server at %s: %w", address, err)
	}
	reader := bufio.NewReader(conn)
	logger.Info().Str("remote_addr", conn.RemoteAddr().String()).Msg("Connected to master server")

	conn.Write(protocol.BulkArray([]string{"PING"}))
	if err := expectSimpleResponse(reader, "PONG"); err != nil {
		return fmt.Errorf("failed to perform ping with the master server: %w", err)
	}

	logger.Info().Msg("Ping successful, proceeding with REPLCONF handshake")
	conn.Write(protocol.BulkArray([]string{"REPLCONF", "listening-port", strconv.Itoa(rs.config.ServerPort)}))
	if err := expectSimpleResponse(reader, "OK"); err != nil {
		return fmt.Errorf("failed to perform REPLCONF listening-port with the master server: %w", err)
	}

	logger.Info().Msg("REPLCONF listening-port successful, proceeding with REPLCONF capa psync2")
	conn.Write(protocol.BulkArray([]string{"REPLCONF", "capa", "psync2"}))
	if err := expectSimpleResponse(reader, "OK"); err != nil {
		return fmt.Errorf("failed to perform REPLCONF capa psync2 with the master server: %w", err)
	}

	defferedContext := context.WithoutCancel(ctx)
	go rs.handleReplicationConnection(defferedContext, conn)
	rs.masterInfo = replication.NewMasterInfo(conn, "", 0)
	rs.replicatedCommandHandler = commands.NewReplicaReplicatedCommandDispatcher(
		rs.masterInfo,
		rs.storage,
	)

	logger.Info().Msg("REPLCONF capa psync2 successful, proceeding with PSYNC handshake")
	conn.Write(protocol.BulkArray([]string{"PSYNC", "?", "-1"}))

	go ackLoop(ctx, conn, rs.masterInfo)
	logger.Info().Msg("Handshake successful")
	return nil
}

func ackLoop(ctx context.Context, conn net.Conn, masterInfo *replication.MasterInfo) {
	ticker := time.NewTicker(60 * time.Second) // Adjust the interval as needed
	defer ticker.Stop()

	logger := zerolog.Ctx(ctx).With().
		Str("role", config.ReplicaRole).
		Str("op", "ack_loop").Logger()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			offset := masterInfo.GetReplicaOffset()
			msg := protocol.BulkArray([]string{"REPLCONF", "ACK", strconv.Itoa(offset)})

			_, err := conn.Write(msg)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to send REPLCONF ACK to master. Exiting ack loop.")
				// Optional: Cancel context or trigger reconnect
				return
			}

			logger.Debug().
				Int("replica_offset", offset).
				Msg("Sent REPLCONF ACK to master")
		}
	}
}

func expectSimpleResponse(reader *bufio.Reader, expected string) error {
	resp, err := protocol.ParseResponse(reader)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	if resp.Type != protocol.SimpleStringType {
		return fmt.Errorf("unexpected response type: %s, expected: %s", resp.Type, protocol.SimpleStringType)
	}
	if resp.Value != expected {
		return fmt.Errorf("unexpected response: %s", resp.Value)
	}
	return nil
}

func (rs *ReplicaServer) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	logger := zerolog.Ctx(ctx).With().
		Str("role", rs.role).
		Str("op", "handle_conn").
		Str("remote_addr", conn.RemoteAddr().String()).
		Logger()
	logger.Info().Msg("Handling new connection")

	buffer := make([]byte, 1024*4)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				logger.Info().Msg("Connection closed by client")
				return
			}
			logger.Err(err).Msg("Error reading from connection")
			return
		}

		logger.Debug().Str("data", string(buffer[:n])).Msg("Received data")
		result, err := rs.commandParser.Parse(buffer[:n])
		if err != nil {
			logger.Err(err).Msg("Failed to parse received data")
			continue
		}

		for i, command := range result.Commands {
			logger := logger.With().
				Str("command", command.Name).
				Interface("args", command.Args).Logger()
			logger.Info().Int("index", i).Msg("Parsed command")
			result, err := rs.commandHandler.Dispatch(ctx, conn, command)
			if err != nil {
				logger.Err(err).Msg("Failed to handle command")
				continue
			}
			if result.CommandError != nil {
				logger.Err(result.CommandError).Msg("Command error occurred, sending error response")
			}
		}
	}
}

func (rs *ReplicaServer) handleReplicationConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	logger := zerolog.Ctx(ctx).With().
		Str("role", rs.role).
		Str("local_addr", conn.LocalAddr().String()).
		Str("op", "handle_replication_conn").Logger()
	logger.Info().Msg("Handling new connection for replication")

	buffer := make([]byte, 1024*4)

	for {

		n, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				logger.Err(err).Msg("Connection closed by client")
				return
			}
			logger.Err(err).Msg("Error reading from connection")
			return
		}

		logger.Debug().Str("data", string(buffer[:n])).Msg("Received data for replication")
		result, err := rs.commandParser.Parse(buffer[:n])
		if err != nil {
			logger.Err(err).Msg("Failed to parse received data")
			continue
		}

		for i, command := range result.Commands {
			logger := logger.With().
				Str("command", command.Name).
				Interface("args", command.Args).Logger()
			if !rs.isMasterConnected(ctx) {
				logger.Warn().Msg("Master is not correctly connected or setup, skipping command handling")
				continue
			}
			logger.Info().Int("index", i).Msg("Parsed command")
			_, err := rs.replicatedCommandHandler.Dispatch(ctx, conn, command)
			if err != nil {
				logger.Err(err).Msg("Failed to handle replicated command")
				continue
			}
			if slices.Contains([]string{protocol.FULLRESYNC}, command.Name) {
				continue
			}
			rs.masterInfo.AddReplicaOffset(command.BytesSize)
			logger.Info().
				Int("size", command.BytesSize).
				Int("replica_offset", rs.masterInfo.GetReplicaOffset()).
				Msg("Replicated command handled successfully")
		}
	}
}

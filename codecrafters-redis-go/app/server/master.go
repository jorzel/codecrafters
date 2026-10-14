package server

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/replication"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
	"github.com/rs/zerolog"
)

var _ Server = (*MasterServer)(nil)

type MasterServer struct {
	listener       net.Listener
	commandParser  protocol.CommandParser
	commandHandler commands.CommandDispatcher
	config         *config.Config
	role           string
	replicator     commands.Replicator
	replicasInfo   *replication.ReplicasInfo
}

func NewMasterServer(cfg *config.Config) (*MasterServer, error) {
	addr := fmt.Sprintf("0.0.0.0:%d", cfg.ServerPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	storage := storage.NewStorage()
	replicasInfo, err := replication.NewReplicasInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to create replicas info: %w", err)
	}
	replicator := commands.NewReplicator(replicasInfo)
	return &MasterServer{
		listener:       ln,
		commandParser:  protocol.NewCommandParser(),
		commandHandler: commands.NewMasterCommandDispatcher(replicasInfo, replicator, storage, cfg),
		config:         cfg,
		role:           config.MasterRole,
		replicator:     replicator,
		replicasInfo:   replicasInfo,
	}, nil
}

func (ms *MasterServer) Start(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().
		Str("address", ms.listener.Addr().String()).
		Interface("cfg", ms.config).
		Msg("Master listening on...")
	for {
		conn, err := ms.listener.Accept()
		if err != nil {
			continue
		}
		go ms.handleConnection(ctx, conn)
	}
}

func (ms *MasterServer) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	logger := zerolog.Ctx(ctx).With().
		Str("role", ms.role).
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
		result, err := ms.commandParser.Parse(buffer[:n])
		if err != nil {
			logger.Err(err).Msg("Failed to parse received data")
			continue
		}

		for i, command := range result.Commands {
			logger := logger.With().
				Str("command", command.Name).
				Interface("args", command.Args).Logger()
			logger.Info().Int("index", i).Msg("Parsed command")

			result, err := ms.commandHandler.Dispatch(ctx, conn, command)
			if err != nil {
				logger.Err(err).Msg("Failed to handle command")
				continue
			}
			if result.CommandError != nil {
				logger.Err(result.CommandError).Msg("Command error occurred, sending error response")
			}
			if !ms.shouldPropagateCommand(command) {
				continue
			}
			ms.replicator.Propagate(ctx, conn, command)
			logger.Debug().
				Str("command", command.Name).
				Int("master_offset", ms.replicasInfo.GetMasterOffset()).
				Msg("Command propagated to replicas")
		}

	}
}

func (ms *MasterServer) shouldPropagateCommand(command protocol.Command) bool {
	return command.IsWrite()
}

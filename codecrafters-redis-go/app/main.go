package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/server"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	ctx := logger.WithContext(context.Background())

	initSpec, err := getInitSpecsFromArgs()
	if err != nil {
		logger.Err(err).Msg("Failed to parse initialization specifications from arguments")
		os.Exit(1)
	}

	config, err := config.NewConfig(initSpec)
	if err != nil {
		logger.Err(err).Msg("Failed to create configuration")
		os.Exit(1)
	}
	logger.Info().Interface("config", config).Msg("Starting configuration")

	var srv server.Server
	if initSpec.ReplicaOf != nil {
		srv, err = server.NewReplicaServer(config)
	} else {
		srv, err = server.NewMasterServer(config)
	}

	if err != nil {
		logger.Err(err).Msg("Failed to initialize server")
		os.Exit(1)
	}

	err = srv.Start(ctx)
	if err != nil {
		logger.Err(err).Msg("Failed to start server")
		os.Exit(1)
	}
}

func getInitSpecsFromArgs() (config.InitSpec, error) {
	port := flag.Int("port", 6379, "Port to listen on")
	replicaOf := flag.String("replicaof", "", "Address of the master server")
	dir := flag.String("dir", "", "Directory to store data files")
	dbfilename := flag.String("dbfilename", "", "Name of the database file")
	flag.Parse()

	if port == nil {
		return config.InitSpec{}, fmt.Errorf("port flag is required")
	}
	if *port < 1 || *port > 65535 {
		return config.InitSpec{}, fmt.Errorf("port must be between 1 and 65535, got %d", *port)
	}

	var err error
	var deserializedReplicaOf *config.Node
	if replicaOf == nil || *replicaOf == "" {
		deserializedReplicaOf = nil
	} else {
		deserializedReplicaOf, err = validateReplicaOf(*replicaOf)
		if err != nil {
			return config.InitSpec{}, fmt.Errorf("invalid replicaof address: %w", err)
		}
	}

	return config.InitSpec{
		ReplicaOf:  deserializedReplicaOf,
		ServerPort: *port,
		Dir:        *dir,
		DbFileName: *dbfilename,
	}, nil
}

func validateReplicaOf(replicaOf string) (*config.Node, error) {
	if replicaOf == "" {
		return nil, nil
	}

	parts := strings.Split(replicaOf, " ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("replicaof must be in the format <host>:<port>, got %s", replicaOf)
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port number in replicaof: %s", parts[1])
	}

	return &config.Node{
		Host: parts[0],
		Port: port,
	}, nil
}

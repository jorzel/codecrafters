package main

import (
	"context"
	"net"
	"os"

	"github.com/codecrafters-io/kafka-starter-go/app/transport/tcp"
	"github.com/rs/zerolog"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

func main() {
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	ctx := context.Background()
	ctx = logger.WithContext(ctx)

	port := 9092
	srv := tcp.NewBroker()
	if err := srv.Start(ctx, port); err != nil {
		logger.Fatal().Err(err).Msg("Error starting server")
	}
}

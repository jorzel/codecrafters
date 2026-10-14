package tcp

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/codecrafters-io/kafka-starter-go/app/api"
	"github.com/rs/zerolog"
)

type Broker struct {
}

func NewBroker() *Broker {
	return &Broker{}
}

func (s *Broker) Start(ctx context.Context, port int) error {
	logger := zerolog.Ctx(ctx)

	logger.Info().Int("port", port).Msg("Starting TCP server, accepting new connections...")
	l, err := net.Listen("tcp", "0.0.0.0:"+fmt.Sprint(port))
	if err != nil {
		return fmt.Errorf("error starting TCP server: %w", err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			return fmt.Errorf("error accepting connection: %w", err)
		}
		go func(ctx context.Context) {
			err = handleConnection(ctx, conn)
			if err != nil {
				logger.Error().Err(err).Msg("Error handling connection")
			}
		}(ctx)
	}
}

func handleConnection(ctx context.Context, conn net.Conn) error {
	defer conn.Close()
	logger := zerolog.Ctx(ctx)

	for {
		bufferSize := 10 * 1024
		buffer := make([]byte, bufferSize)
		n, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				logger.Info().Msg("Connection closed by client")
				return nil
			}
			return fmt.Errorf("error reading from connection: %w", err)
		}
		logger.Debug().
			Str("data", fmt.Sprintf("%x", buffer[:n])).
			Msg("Received data from client")

		request, err := api.ParseRequest(ctx, buffer)
		if err != nil {
			return fmt.Errorf("error parsing KafkaRequest: %w", err)
		}

		logger.Info().
			Interface("correlation_id", request.CorrelationID()).
			Interface("api_key", request.APIKey()).
			Interface("api_version", request.APIVersion()).
			Interface("error_code", request.ErrorCode()).
			Msg("Parsed KafkaRequest")

		response, err := api.HandleRequest(request)
		if err != nil {
			return fmt.Errorf("error creating KafkaResponse: %w", err)
		}
		responseBytes, err := response.Encode(ctx)
		if err != nil {
			return fmt.Errorf("error encoding KafkaResponse: %w", err)
		}
		conn.Write(responseBytes)
	}
}

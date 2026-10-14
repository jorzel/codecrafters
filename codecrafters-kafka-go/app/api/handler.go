package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/rs/zerolog"
)

type KafkaRequest interface {
	CorrelationID() CorrelationID
	APIVersion() APIVersion
	APIKey() APIKey
	ErrorCode() ErrorCode
}

type KafkaResponse interface {
	Encode(context.Context) ([]byte, error)
}

type ErrorCode int16
type MessageSize int32
type CorrelationID int32
type APIKey int16
type APIVersion int16
type TagBuffer uint8
type ThrottleTime int32

type requestHeader struct {
	correlationID CorrelationID
	apiKey        APIKey
	apiVersion    APIVersion
	clientID      compactString
	tagBuffer     TagBuffer
}

type responseHeaderV0 struct {
	correlationID CorrelationID
}

type responseHeaderV1 struct {
	correlationID CorrelationID
	tagBuffer     TagBuffer
}

type compactString struct {
	length  int16
	content []byte
}

func ParseRequest(ctx context.Context, payload []byte) (KafkaRequest, error) {
	logger := zerolog.Ctx(ctx)
	buf := bytes.NewReader(payload)

	var messageSize MessageSize
	if err := binary.Read(buf, binary.BigEndian, &messageSize); err != nil {
		return nil, fmt.Errorf("failed to read message size: %w", err)
	}
	logger.Debug().Interface("message_size", messageSize).Msg("Parsed message size")

	header, err := parseRequestHeader(ctx, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse request header: %w", err)
	}

	supportedAPIVersions := []int16{0, 1, 2, 3, 4}
	isSupported := false
	for _, v := range supportedAPIVersions {
		if int16(header.apiVersion) == v {
			isSupported = true
			break
		}
	}
	logger.Debug().
		Bool("is_supported", isSupported).
		Interface("api_version", header.apiVersion).
		Msg("Parsed API version")

	errorCode := ErrorCode(0)
	if !isSupported {
		errorCode = ErrorCode(35) // UnsupportedVersion
	}

	var request KafkaRequest
	switch header.apiKey {
	case 18: // ApiVersions
		body, err := parseApiVersionsRequestBody(ctx, buf)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ApiVersionRequest body: %w", err)
		}
		request = &apiVersionsRequest{
			messageSize: messageSize,
			header:      header,
			errorCode:   errorCode,
			body:        body,
		}

	case 75: // DescribeTopicPartitions (Kafka 3.7+)
		body, err := parseDescribeTopicPartitionsRequestBody(ctx, buf)
		if err != nil {
			return nil, err
		}
		return &describeTopicPartitionsRequest{
			messageSize: messageSize,
			header:      header,
			errorCode:   errorCode,
			body:        body,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported apiKey: %d", header.apiKey)
	}

	return request, nil
}

func parseRequestHeader(ctx context.Context, buf *bytes.Reader) (*requestHeader, error) {
	logger := zerolog.Ctx(ctx)
	header := &requestHeader{}
	if err := binary.Read(buf, binary.BigEndian, &header.apiKey); err != nil {
		return nil, fmt.Errorf("failed to read API key: %w", err)
	}
	logger.Debug().Interface("api_key", header.apiKey).Msg("Parsed API key")

	if err := binary.Read(buf, binary.BigEndian, &header.apiVersion); err != nil {
		return nil, fmt.Errorf("failed to read API version: %w", err)
	}
	logger.Debug().Interface("api_version", header.apiVersion).Msg("Parsed API version")

	if err := binary.Read(buf, binary.BigEndian, &header.correlationID); err != nil {
		return nil, fmt.Errorf("failed to read correlation ID: %w", err)
	}
	logger.Debug().Interface("correlation_id", header.correlationID).Msg("Parsed correlation ID")

	if err := binary.Read(buf, binary.BigEndian, &header.clientID.length); err != nil {
		return nil, fmt.Errorf("failed to read client ID length: %w", err)
	}
	logger.Debug().Int16("client_id_length", header.clientID.length).Msg("Parsed client ID length")

	header.clientID.content = make([]byte, header.clientID.length)
	if err := binary.Read(buf, binary.BigEndian, &header.clientID.content); err != nil {
		return nil, fmt.Errorf("failed to read client ID content: %w", err)
	}
	logger.Debug().Str("client_id", string(header.clientID.content)).Msg("Parsed client ID content")

	if err := binary.Read(buf, binary.BigEndian, &header.tagBuffer); err != nil {
		return nil, fmt.Errorf("failed to read tag buffer: %w", err)
	}
	return header, nil
}

func HandleRequest(request KafkaRequest) (KafkaResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	switch request.(type) {
	case *apiVersionsRequest:
		resp := &apiVersionsResponse{
			header: responseHeaderV0{
				correlationID: request.CorrelationID(),
			},
			body: apiVersionsResponseBody{
				errorCode: request.ErrorCode(),
				apiKeys:   newApiKeyVersions(),
			},
		}
		return resp, nil

	case *describeTopicPartitionsRequest:
		r := request.(*describeTopicPartitionsRequest)
		t := r.body.topics[0]
		resp := &describeTopicPartitionsResponse{
			header: responseHeaderV1{
				correlationID: request.CorrelationID(),
			},
			body: &describeTopicPartitionsResponseBody{
				topics: []responseTopic{
					{
						topicName: string(t.topicName),
						errorCode: 3,
						topicID:   [16]byte{},
					},
				},
				nextCursor: -1,
			},
		}
		return resp, nil

	default:
		return nil, fmt.Errorf("unknown request type")
	}

}

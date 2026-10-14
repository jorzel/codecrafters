package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/rs/zerolog"
)

var _ KafkaRequest = (*apiVersionsRequest)(nil)

type describeTopicPartitionsRequest struct {
	messageSize MessageSize
	header      *requestHeader
	errorCode   ErrorCode
	body        *describeTopicPartitionsRequestBody
}

func (r *describeTopicPartitionsRequest) CorrelationID() CorrelationID {
	return r.header.correlationID
}

func (r *describeTopicPartitionsRequest) APIVersion() APIVersion {
	return r.header.apiVersion
}

func (r *describeTopicPartitionsRequest) APIKey() APIKey {
	return r.header.apiKey
}

func (r *describeTopicPartitionsRequest) ErrorCode() ErrorCode {
	return r.errorCode
}

type requestTopic struct {
	topicNameLength int8
	topicName       []byte
	tagBuffer       int8 // always 0 for now
}

type describeTopicPartitionsRequestBody struct {
	topics                 []requestTopic
	responsePartitionLimit int32
	cursor                 int8
	tagBuffer              TagBuffer
}

func parseDescribeTopicPartitionsRequestBody(ctx context.Context, buf *bytes.Reader) (
	*describeTopicPartitionsRequestBody, error,
) {
	logger := zerolog.Ctx(ctx)
	body := &describeTopicPartitionsRequestBody{}

	var topicsCount uint8
	if err := binary.Read(buf, binary.BigEndian, &topicsCount); err != nil {
		return nil, fmt.Errorf("failed to read body topics count: %w", err)
	}

	topicsCount-- // Kafka encodes the length of arrays as an unsigned varint + 1

	logger.Debug().Interface("body_topics_count", topicsCount).Msg("Parsed body topics count")
	body.topics = make([]requestTopic, topicsCount)
	for i := uint8(0); i < topicsCount; i++ {
		var t requestTopic
		if err := binary.Read(buf, binary.BigEndian, &t.topicNameLength); err != nil {
			return nil, fmt.Errorf("failed to read body topic name length: %w", err)
		}
		t.topicNameLength-- // Kafka encodes the length of strings as an unsigned varint + 1
		logger.Debug().Interface("body_topic_name_length", t.topicNameLength).Msg("Parsed body topic name length")

		t.topicName = make([]byte, t.topicNameLength)
		if err := binary.Read(buf, binary.BigEndian, &t.topicName); err != nil {
			return nil, fmt.Errorf("failed to read body topic name content: %w", err)
		}
		logger.Debug().Str("body_topic_name", string(t.topicName)).Msg("Parsed body topic name content")

		if err := binary.Read(buf, binary.BigEndian, &t.tagBuffer); err != nil {
			return nil, fmt.Errorf("failed to read body topic tag buffer: %w", err)
		}
		logger.Debug().Int8("body_topic_tag_buffer", t.tagBuffer).Msg("Parsed body topic tag buffer")

		body.topics[i] = t
	}
	if err := binary.Read(buf, binary.BigEndian, &body.responsePartitionLimit); err != nil {
		return nil, fmt.Errorf("failed to read body response partition limit: %w", err)
	}
	logger.Debug().
		Int32("body_response_partition_limit", body.responsePartitionLimit).
		Msg("Parsed body response partition limit")

	if err := binary.Read(buf, binary.BigEndian, &body.cursor); err != nil {
		return nil, fmt.Errorf("failed to read body cursor: %w", err)
	}
	logger.Debug().
		Interface("body_cursor", body.cursor).
		Msg("Parsed body cursor")

	if err := binary.Read(buf, binary.BigEndian, &body.tagBuffer); err != nil {
		return nil, fmt.Errorf("failed to read body tag buffer: %w", err)
	}
	return body, nil
}

var _ KafkaResponse = (*apiVersionsResponse)(nil)

type describeTopicPartitionsResponse struct {
	header      responseHeaderV1
	body        *describeTopicPartitionsResponseBody
	messageSize MessageSize
}

func (r *describeTopicPartitionsResponse) Encode(ctx context.Context) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write header first
	if err := binary.Write(buf, binary.BigEndian, r.header.correlationID); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, r.header.tagBuffer); err != nil {
		return nil, err
	}

	bodyBytes, err := r.body.Encode(ctx)
	if err != nil {
		return nil, err
	}
	// Write body next
	if _, err := buf.Write(bodyBytes); err != nil {
		return nil, err
	}

	// Compute message size (everything after the size field)
	r.messageSize = MessageSize((buf.Len()))

	// Now prepend messageSize
	finalBuf := new(bytes.Buffer)
	if err := binary.Write(finalBuf, binary.BigEndian, r.messageSize); err != nil {
		return nil, err
	}

	if _, err := finalBuf.Write(buf.Bytes()); err != nil {
		return nil, err
	}

	return finalBuf.Bytes(), nil
}

type describeTopicPartitionsResponseBody struct {
	throttleTime ThrottleTime
	topics       []responseTopic
	nextCursor   int8
	tagBuffer    TagBuffer
}

func (r *describeTopicPartitionsResponseBody) Encode(ctx context.Context) ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, r.throttleTime); err != nil {
		return nil, err
	}

	// The length of the API Versions array + 1, encoded as an unsigned varint
	if err := buf.WriteByte(byte(len(r.topics) + 1)); err != nil {
		return nil, err
	}

	for _, key := range r.topics {

		keyBytes, err := key.Encode(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode response topic: %w", err)
		}
		if _, err := buf.Write(keyBytes); err != nil {
			return nil, fmt.Errorf("failed to write response topic bytes: %w", err)
		}
	}

	if err := binary.Write(buf, binary.BigEndian, r.nextCursor); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(byte(r.tagBuffer)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type responseTopic struct {
	errorCode                 ErrorCode
	topicName                 string
	topicID                   [16]byte
	isInternal                int8
	partitions                uint8
	topicAuthorizedOperations int32
	tagBuffer                 TagBuffer // always 0 for now
}

func (v *responseTopic) Encode(ctx context.Context) ([]byte, error) {
	logger := zerolog.Ctx(ctx)
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, v.errorCode); err != nil {
		return nil, fmt.Errorf("failed to write error code: %w", err)
	}
	logger.Debug().Interface("error_code", v.errorCode).Msg("Wrote error code")

	topicNameLength := int8(len(v.topicName) + 1)
	if err := binary.Write(buf, binary.BigEndian, topicNameLength); err != nil {
		return nil, fmt.Errorf("failed to write topic name length: %w", err)
	}

	if err := binary.Write(buf, binary.BigEndian, []byte(v.topicName)); err != nil {
		return nil, fmt.Errorf("failed to write topic name: %w", err)
	}

	if err := binary.Write(buf, binary.BigEndian, v.topicID); err != nil {
		return nil, fmt.Errorf("failed to write topic ID: %w", err)
	}

	if err := binary.Write(buf, binary.BigEndian, v.isInternal); err != nil {
		return nil, fmt.Errorf("failed to write is internal: %w", err)
	}

	if err := binary.Write(buf, binary.BigEndian, v.partitions+1); err != nil {
		return nil, fmt.Errorf("failed to write partitions: %w", err)
	}

	if err := binary.Write(buf, binary.BigEndian, v.topicAuthorizedOperations); err != nil {
		return nil, fmt.Errorf("failed to write topic authorized operations: %w", err)
	}

	if err := binary.Write(buf, binary.BigEndian, v.tagBuffer); err != nil {
		return nil, fmt.Errorf("failed to write tag buffer: %w", err)
	}

	return buf.Bytes(), nil
}

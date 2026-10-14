package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/rs/zerolog"
)

var _ KafkaRequest = (*apiVersionsRequest)(nil)

type apiVersionsRequest struct {
	messageSize MessageSize
	header      *requestHeader
	errorCode   ErrorCode
	body        *apiVersionsRequestBody
}

func (r *apiVersionsRequest) CorrelationID() CorrelationID {
	return r.header.correlationID
}

func (r *apiVersionsRequest) APIVersion() APIVersion {
	return r.header.apiVersion
}

func (r *apiVersionsRequest) APIKey() APIKey {
	return r.header.apiKey
}

func (r *apiVersionsRequest) ErrorCode() ErrorCode {
	return r.errorCode
}

type apiVersionsRequestBody struct {
	clientID              apiVersionsRequestBodyClient
	clientSoftwareVersion apiVersionsRequestBodyClient
	tagBuffer             int8
}

type apiVersionsRequestBodyClient struct {
	length  int8
	content []byte
}

func parseApiVersionsRequestBody(ctx context.Context, buf *bytes.Reader) (*apiVersionsRequestBody, error) {
	logger := zerolog.Ctx(ctx)

	body := &apiVersionsRequestBody{}
	if err := binary.Read(buf, binary.BigEndian, &body.clientID.length); err != nil {
		return nil, fmt.Errorf("failed to read body client ID length: %w", err)
	}
	logger.Debug().Int8("body_client_id_length", body.clientID.length).Msg("Parsed body client ID length")

	body.clientID.content = make([]byte, body.clientID.length)
	if err := binary.Read(buf, binary.BigEndian, &body.clientID.content); err != nil {
		return nil, fmt.Errorf("failed to read body client ID content: %w", err)
	}
	logger.Debug().Str("body_client_id", string(body.clientID.content)).Msg("Parsed body client ID content")

	if err := binary.Read(buf, binary.BigEndian, &body.clientSoftwareVersion.length); err != nil {
		return nil, fmt.Errorf("failed to read client software version length: %w", err)
	}
	logger.Debug().Int8("client_software_version_length", body.clientSoftwareVersion.length).Msg("Parsed client software version length")

	body.clientSoftwareVersion.content = make([]byte, body.clientSoftwareVersion.length)
	if err := binary.Read(buf, binary.BigEndian, &body.clientSoftwareVersion.content); err != nil {
		return nil, fmt.Errorf("failed to read client software version content: %w", err)
	}
	logger.Debug().Str("client_software_version", string(body.clientSoftwareVersion.content)).Msg("Parsed client software version content")

	if err := binary.Read(buf, binary.BigEndian, &body.tagBuffer); err != nil {
		return nil, fmt.Errorf("failed to read body tag buffer: %w", err)
	}
	logger.Debug().Int8("body_tag_buffer", body.tagBuffer).Msg("Parsed body tag buffer")
	return body, nil
}

var _ KafkaResponse = (*apiVersionsResponse)(nil)

type apiVersionsResponse struct {
	header      responseHeaderV0
	body        apiVersionsResponseBody
	messageSize MessageSize
}

func (r *apiVersionsResponse) Encode(ctx context.Context) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write header first
	if err := binary.Write(buf, binary.BigEndian, r.header.correlationID); err != nil {
		return nil, err
	}

	bodyBytes, err := r.body.Encode()
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

type apiVersionsResponseBody struct {
	errorCode    ErrorCode
	throttleTime ThrottleTime
	apiKeys      []apiKeyVersion
	tagBuffer    TagBuffer
}

func (b *apiVersionsResponseBody) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, b.errorCode); err != nil {
		return nil, err
	}

	// The length of the API Versions array + 1, encoded as an unsigned varint
	if err := buf.WriteByte(byte(len(b.apiKeys) + 1)); err != nil {
		return nil, err
	}

	for _, key := range b.apiKeys {

		keyBytes, err := key.Encode()
		if err != nil {
			return nil, err
		}
		if _, err := buf.Write(keyBytes); err != nil {
			return nil, err
		}
	}

	if err := binary.Write(buf, binary.BigEndian, b.throttleTime); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(byte(b.tagBuffer)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type apiKeyVersion struct {
	apiKey     int16
	minVersion int16
	maxVersion int16
	tagBuffer  int8 // always 0 for now
}

func newApiKeyVersion(apiKey, minVersion, maxVersion int16) apiKeyVersion {
	return apiKeyVersion{
		apiKey:     apiKey,
		minVersion: minVersion,
		maxVersion: maxVersion,
		tagBuffer:  0,
	}
}

func (v *apiKeyVersion) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, v.apiKey); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, v.minVersion); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, v.maxVersion); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(byte(v.tagBuffer)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func newApiKeyVersions() []apiKeyVersion {
	return []apiKeyVersion{
		newApiKeyVersion(18, 0, 4), // ApiVersions
		newApiKeyVersion(75, 0, 0), // DescribeTopicPartitions
	}

}

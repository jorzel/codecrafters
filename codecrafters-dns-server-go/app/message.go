package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type DNSMessage struct {
	Header    DNSHeader
	Questions []DNSQuestion
	Answers   []DNSResourceRecord
}

func (d *DNSMessage) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Encode Header
	headerBytes, err := d.Header.Encode()
	if err != nil {
		return nil, err
	}
	buf.Write(headerBytes)

	// Encode Questions
	for _, question := range d.Questions {
		questionBytes, err := question.Encode()
		if err != nil {
			return nil, err
		}
		buf.Write(questionBytes)
	}

	// Encode Answers
	for _, answer := range d.Answers {
		answerBytes, err := answer.Encode()
		if err != nil {
			return nil, err
		}
		buf.Write(answerBytes)
	}

	return buf.Bytes(), nil
}

// DNSHeader represents the 12-byte DNS message header.
type DNSHeader struct {
	PacketIdentifier       uint16 // 16 bits
	QueryResponseIndicator uint8  // 1 bit
	OperationCode          uint8  // 4 bits
	AuthoritativeAnswer    uint8  // 1 bit
	Truncation             uint8  // 1 bit
	RecursionDesired       uint8  // 1 bit
	RecursionAvailable     uint8  // 1 bit
	Reserved               uint8  // 3 bits
	ResponseCode           uint8  // 4 bits
	QuestionCount          uint16 // 16 bits
	AnswerRecordCount      uint16 // 16 bits
	AuthorityRecordCount   uint16 // 16 bits
	AdditionalRecordCount  uint16 // 16 bits
}

// packFlags combines all bit fields into one 16-bit value
func (h *DNSHeader) packFlags() uint16 {
	var flags uint16

	flags |= uint16(h.QueryResponseIndicator&0x01) << 15
	flags |= uint16(h.OperationCode&0x0F) << 11
	flags |= uint16(h.AuthoritativeAnswer&0x01) << 10
	flags |= uint16(h.Truncation&0x01) << 9
	flags |= uint16(h.RecursionDesired&0x01) << 8
	flags |= uint16(h.RecursionAvailable&0x01) << 7
	flags |= uint16(h.Reserved&0x07) << 4
	flags |= uint16(h.ResponseCode & 0x0F)

	return flags
}

// WriteBinary serializes the DNSHeader into bytes (12 bytes total)
func (h *DNSHeader) Encode() ([]byte, error) {
	// Write ID

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, h.PacketIdentifier); err != nil {
		return nil, err
	}

	// Pack and write flags
	flags := h.packFlags()
	if err := binary.Write(buf, binary.BigEndian, flags); err != nil {
		return nil, err
	}

	// Write the remaining 16-bit fields
	fields := []uint16{
		h.QuestionCount,
		h.AnswerRecordCount,
		h.AuthorityRecordCount,
		h.AdditionalRecordCount,
	}
	for _, f := range fields {
		if err := binary.Write(buf, binary.BigEndian, f); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

type DNSQuestion struct {
	Name  string
	Type  uint16
	Class uint16
}

func (q *DNSQuestion) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Encode the domain name in DNS format
	labels := bytes.Split([]byte(q.Name), []byte("."))
	for _, label := range labels {
		if len(label) > 63 {
			return nil, fmt.Errorf("label too long: %s", label)
		}
		buf.WriteByte(byte(len(label)))
		buf.Write(label)
	}
	buf.WriteByte(0) // End of the domain name

	// Write Type and Class
	if err := binary.Write(buf, binary.BigEndian, q.Type); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, q.Class); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type DNSResourceRecord struct {
	Name     string
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
	RData    []byte
}

func (rr *DNSResourceRecord) Encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Encode the domain name in DNS format
	labels := bytes.Split([]byte(rr.Name), []byte("."))
	for _, label := range labels {
		if len(label) > 63 {
			return nil, fmt.Errorf("label too long: %s", label)
		}
		buf.WriteByte(byte(len(label)))
		buf.Write(label)
	}
	buf.WriteByte(0) // End of the domain name

	// Write Type, Class, TTL, RDLength
	if err := binary.Write(buf, binary.BigEndian, rr.Type); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, rr.Class); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, rr.TTL); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, uint16(rr.RDLength)); err != nil {
		return nil, err
	}

	if _, err := buf.Write(rr.RData); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

package storage

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type IDType int

const (
	IDConcrete IDType = iota // concrete numeric id
	IDAuto                   // "*"
	IDMin                    // "-"
	IDMax                    // "+"
	IDMsAuto                 // "ms-*"
)

type StreamID struct {
	Type IDType
	Ms   uint64
	Seq  uint64
}

func (id StreamID) String() string {
	switch id.Type {
	case IDAuto:
		return "*"
	case IDMin:
		return "-"
	case IDMax:
		return "+"
	case IDMsAuto:
		return fmt.Sprintf("%d-*", id.Ms)
	default:
		return fmt.Sprintf("%d-%d", id.Ms, id.Seq)
	}
}

func parseStreamID(s string) (StreamID, error) {
	switch s {
	case "*":
		return StreamID{Type: IDAuto}, nil
	case "-":
		return StreamID{Type: IDMin}, nil
	case "+":
		return StreamID{Type: IDMax}, nil
	}

	parts := strings.Split(s, "-")
	if len(parts) == 1 {
		// Only milliseconds part
		ms, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return StreamID{}, fmt.Errorf("invalid ID")
		}
		return StreamID{Type: IDConcrete, Ms: ms, Seq: 0}, nil
	}

	if len(parts) == 2 {
		ms, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return StreamID{}, fmt.Errorf("invalid ms part")
		}

		if parts[1] == "*" {
			return StreamID{Type: IDMsAuto, Ms: ms}, nil
		}

		seq, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return StreamID{}, fmt.Errorf("invalid seq part")
		}
		return StreamID{Type: IDConcrete, Ms: ms, Seq: seq}, nil
	}

	return StreamID{}, fmt.Errorf("invalid stream ID format")
}

func CompareIDs(a, b StreamID) int {
	// handle special tokens
	if a.Type == IDMin || b.Type == IDMax {
		return -1
	}
	if a.Type == IDMax || b.Type == IDMin {
		return 1
	}
	if a.Type != IDConcrete || b.Type != IDConcrete {
		panic("cannot compare symbolic IDs directly")
	}

	if a.Ms < b.Ms {
		return -1
	}
	if a.Ms > b.Ms {
		return 1
	}
	if a.Seq < b.Seq {
		return -1
	}
	if a.Seq > b.Seq {
		return 1
	}
	return 0
}

// helper: encode ms into 8-byte big-endian string key for radix ordering
func keyFromMs(ms uint64) string {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, ms)
	return string(b)
}

// convenience: now() as ms
func nowMs() uint64 { return uint64(time.Now().UnixNano() / 1e6) }

// StreamItem represents a single entry in a stream
type StreamItem struct {
	ID     StreamID
	fields map[string]string
}

func NewStreamItem(id StreamID, fields map[string]string) StreamItem {
	return StreamItem{
		ID:     id,
		fields: fields,
	}
}

type StreamRecord struct {
	items     []StreamItem
	mu        sync.Mutex
	generator *idGenerator
}

func NewStreamRecord() *StreamRecord {
	return &StreamRecord{
		items:     []StreamItem{},
		generator: newIDGenerator(),
	}
}

func (sr *StreamRecord) getLastID() StreamID {
	if len(sr.items) == 0 {
		return StreamID{}
	}
	return sr.items[len(sr.items)-1].ID
}

func (sr *StreamRecord) AddItem(idString string, fields map[string]string) (StreamID, error) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	id, err := parseStreamID(idString)
	if err != nil {
		return StreamID{}, err
	}

	lastID := sr.getLastID()

	switch id.Type {
	case IDAuto:
		id = sr.generator.Next()
	case IDMsAuto:
		if id.Ms < lastID.Ms {
			return StreamID{}, fmt.Errorf("ID smaller than last entry")
		}
		if id.Ms == lastID.Ms {
			id.Seq = lastID.Seq + 1
		} else {
			id.Seq = 0
		}
		id.Type = IDConcrete
	case IDConcrete:
		// Validate > last
		if CompareIDs(id, lastID) <= 0 {
			return StreamID{}, fmt.Errorf("ID smaller than last entry")
		}
	}
	sr.items = append(sr.items, NewStreamItem(id, fields))
	return id, nil
}

type idGenerator struct {
	mu       sync.Mutex
	lastMs   uint64
	sequence uint64
}

func newIDGenerator() *idGenerator { return &idGenerator{} }

func (g *idGenerator) Next() StreamID {
	g.mu.Lock()
	defer g.mu.Unlock()

	ms := nowMs()
	if ms < g.lastMs {
		ms = g.lastMs
	}
	if ms == g.lastMs {
		g.sequence++
	} else {
		g.lastMs = ms
		g.sequence = 0
	}
	return StreamID{Type: IDConcrete, Ms: ms, Seq: g.sequence}
}

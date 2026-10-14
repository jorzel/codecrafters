package replication

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
)


type ReplicasInfo struct {
	ReplID         string         // Unique identifier for the master
	Replicas       ConnectionPool // Pool of replica connections
	offset         int
	ExpectedOffset map[string]int // Expected offsets for commands
	mu             sync.Mutex     // Mutex to protect concurrent access
}

func NewReplicasInfo() (*ReplicasInfo, error) {
	replID, err := generateReplicaID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate replica ID: %w", err)
	}

	return &ReplicasInfo{
		ReplID:         replID,
		Replicas:       NewConnectionPool(),
		ExpectedOffset: make(map[string]int),
	}, nil
}

func (info *ReplicasInfo) SetExpectedOffset(conn net.Conn, offset int) {
	info.mu.Lock()
	defer info.mu.Unlock()

	if info.ExpectedOffset == nil {
		info.ExpectedOffset = make(map[string]int)
	}
	info.ExpectedOffset[getClientID(conn)] = offset
}

func (info *ReplicasInfo) GetExpectedMasterOffset(conn net.Conn) (int, bool) {
	info.mu.Lock()
	defer info.mu.Unlock()

	offset, exists := info.ExpectedOffset[getClientID(conn)]
	return offset, exists
}

func (info *ReplicasInfo) AddMasterOffset(command protocol.Command) int {
	info.mu.Lock()
	defer info.mu.Unlock()

	info.offset += command.BytesSize
	return info.offset
}

func (info *ReplicasInfo) GetMasterOffset() int {
	info.mu.Lock()
	defer info.mu.Unlock()

	return info.offset
}

const replIDLength = 40
const alphanum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateReplicaID() (string, error) {
	result := make([]byte, replIDLength)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphanum))))
		if err != nil {
			return "", err
		}
		result[i] = alphanum[n.Int64()]
	}
	return string(result), nil
}

func getClientID(conn net.Conn) string {
	if conn == nil {
		return "unknown"
	}
	return conn.RemoteAddr().String()
}

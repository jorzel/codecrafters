package replication

import (
	"net"
	"sync"
)

// MasterInfo holds information about the master node
// It is used on replica nodes
type MasterInfo struct {
	Conn   net.Conn // Connection to the master node
	ReplID string   // Unique identifier for the master (replID)
	offset int      // Current replication offset seen by the replica
	mu     sync.Mutex
}

func NewMasterInfo(conn net.Conn, replID string, offset int) *MasterInfo {
	return &MasterInfo{
		Conn:   conn,
		ReplID: replID,
		offset: offset,
	}
}

func (info *MasterInfo) SetReplicaOffset(offset int) {
	info.mu.Lock()
	defer info.mu.Unlock()
	info.offset = offset
}

func (info *MasterInfo) AddReplicaOffset(delta int) {
	info.mu.Lock()
	defer info.mu.Unlock()
	info.offset += delta
}

func (info *MasterInfo) GetReplicaOffset() int {
	info.mu.Lock()
	defer info.mu.Unlock()
	return info.offset
}

func (info *MasterInfo) IsConnected() bool {
	if info.Conn == nil {
		return false
	}
	// Check if the connection is still open
	if _, err := info.Conn.Write([]byte{}); err != nil {
		return false
	}
	return true
}

package storage

import "time"

type StringRecord struct {
	Value    string
	ExpireAt *time.Time
}

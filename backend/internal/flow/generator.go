package flow

import (
	"time"

	"github.com/google/uuid"
)

// GenerateID creates a new unique ID (UUIDv4)
func GenerateID() string {
	return uuid.New().String()
}

// GenerateExecutionID creates a Blueprint execution ID (UUIDv4)
func GenerateExecutionID() string {
	return uuid.New().String()
}

// Now returns the current UTC time
func Now() time.Time {
	return time.Now().UTC()
}

// Timestamp returns current Unix time in milliseconds
func Timestamp() int64 {
	return time.Now().UnixMilli()
}

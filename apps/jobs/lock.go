package jobs

import (
	"fmt"
	"os"
	"time"

	"github.com/getevo/evo/v2/lib/log"
	"github.com/nats-io/nats.go"
)

// LockManager handles distributed locks using NATS KV
type LockManager struct {
	kv         nats.KeyValue
	instanceID string
}

// NewLockManager creates a new lock manager with NATS KV backend
func NewLockManager(js nats.JetStreamContext) (*LockManager, error) {
	if js == nil {
		return nil, fmt.Errorf("JetStream context is nil")
	}

	// Generate instance ID from hostname + pid
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%d", hostname, os.Getpid())

	// Create or bind to KV bucket for locks
	kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:      "job_locks",
		Description: "Distributed locks for background jobs",
		TTL:         30 * time.Minute, // Auto-expire locks after 30 minutes
	})
	if err != nil {
		// Bucket might already exist, try to bind
		kv, err = js.KeyValue("job_locks")
		if err != nil {
			return nil, fmt.Errorf("failed to create/bind job_locks KV bucket: %w", err)
		}
	}

	log.Info("Lock manager initialized with instance ID: %s", instanceID)

	return &LockManager{
		kv:         kv,
		instanceID: instanceID,
	}, nil
}

// TryLock attempts to acquire a lock for the given job name
// Returns true if lock was acquired, false otherwise
func (lm *LockManager) TryLock(jobName string) bool {
	// Try to create the key - this is atomic and fails if key exists
	_, err := lm.kv.Create(jobName, []byte(lm.instanceID))
	if err != nil {
		// Key exists - check if we already own it
		entry, getErr := lm.kv.Get(jobName)
		if getErr == nil && string(entry.Value()) == lm.instanceID {
			// We own the lock, update it to extend TTL
			_, updateErr := lm.kv.Put(jobName, []byte(lm.instanceID))
			if updateErr == nil {
				return true
			}
		}
		return false
	}

	log.Debug("Lock acquired for job: %s by instance: %s", jobName, lm.instanceID)
	return true
}

// Unlock releases the lock for the given job name
// Only releases if this instance owns the lock
func (lm *LockManager) Unlock(jobName string) {
	entry, err := lm.kv.Get(jobName)
	if err != nil {
		// Key doesn't exist, nothing to unlock
		return
	}

	// Only delete if we own the lock
	if string(entry.Value()) == lm.instanceID {
		if err := lm.kv.Delete(jobName); err != nil {
			log.Warning("Failed to release lock for job %s: %v", jobName, err)
		} else {
			log.Debug("Lock released for job: %s", jobName)
		}
	}
}

// IsLocked checks if a job is currently locked
func (lm *LockManager) IsLocked(jobName string) bool {
	_, err := lm.kv.Get(jobName)
	return err == nil
}

// GetLockOwner returns the instance ID that owns the lock, or empty string if not locked
func (lm *LockManager) GetLockOwner(jobName string) string {
	entry, err := lm.kv.Get(jobName)
	if err != nil {
		return ""
	}
	return string(entry.Value())
}

// GetInstanceID returns this instance's ID
func (lm *LockManager) GetInstanceID() string {
	return lm.instanceID
}

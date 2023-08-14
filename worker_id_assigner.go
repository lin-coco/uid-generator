package uidgenerator

// WorkerIdAssigner Represents a worker id assigner for DefaultUidGenerator
type WorkerIdAssigner interface {
	// Assign worker id for DefaultUidGenerator
	assignWorkerId() (int64, error)
}

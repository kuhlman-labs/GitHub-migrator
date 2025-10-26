package batch

// Batch status constants
const (
	StatusPending             = "pending"
	StatusReady               = "ready"
	StatusInProgress          = "in_progress"
	StatusCompleted           = "completed"
	StatusCompletedWithErrors = "completed_with_errors"
	StatusFailed              = "failed"
	StatusCancelled           = "cancelled"
)

// Batch type constants
const (
	TypePilot = "pilot"
	// Wave types are dynamically created as "wave_1", "wave_2", etc.
)

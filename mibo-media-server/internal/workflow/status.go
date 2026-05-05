package workflow

const (
	RunStatusQueued     = "queued"
	RunStatusRunning    = "running"
	RunStatusCompleted  = "completed"
	RunStatusFailed     = "failed"
	RunStatusCancelled  = "cancelled"
	RunStatusSuperseded = "superseded"
)

const (
	TaskStatusBlocked    = "blocked"
	TaskStatusQueued     = "queued"
	TaskStatusRunning    = "running"
	TaskStatusCompleted  = "completed"
	TaskStatusFailed     = "failed"
	TaskStatusRetrying   = "retrying"
	TaskStatusSkipped    = "skipped"
	TaskStatusCancelled  = "cancelled"
	TaskStatusSuperseded = "superseded"
)

const (
	TransitionStart     = "start"
	TransitionComplete  = "complete"
	TransitionFail      = "fail"
	TransitionRetry     = "retry"
	TransitionSkip      = "skip"
	TransitionCancel    = "cancel"
	TransitionSupersede = "supersede"
)

func IsActiveRunStatus(status string) bool {
	switch status {
	case RunStatusQueued, RunStatusRunning:
		return true
	default:
		return false
	}
}

func IsTerminalRunStatus(status string) bool {
	switch status {
	case RunStatusCompleted, RunStatusFailed, RunStatusCancelled, RunStatusSuperseded:
		return true
	default:
		return false
	}
}

func IsActiveTaskStatus(status string) bool {
	switch status {
	case TaskStatusBlocked, TaskStatusQueued, TaskStatusRunning, TaskStatusRetrying:
		return true
	default:
		return false
	}
}

func IsTerminalTaskStatus(status string) bool {
	switch status {
	case TaskStatusCompleted, TaskStatusFailed, TaskStatusSkipped, TaskStatusCancelled, TaskStatusSuperseded:
		return true
	default:
		return false
	}
}

func CanTransitionTask(from string, transition string) bool {
	switch transition {
	case TransitionStart:
		return from == TaskStatusQueued || from == TaskStatusRetrying
	case TransitionComplete:
		return from == TaskStatusRunning
	case TransitionFail:
		return from == TaskStatusRunning || from == TaskStatusQueued || from == TaskStatusRetrying
	case TransitionRetry:
		return from == TaskStatusFailed || from == TaskStatusRunning
	case TransitionSkip:
		return from == TaskStatusBlocked || from == TaskStatusQueued
	case TransitionCancel:
		return IsActiveTaskStatus(from)
	case TransitionSupersede:
		return IsActiveTaskStatus(from)
	default:
		return false
	}
}

func CanTransitionRun(from string, transition string) bool {
	switch transition {
	case TransitionStart:
		return from == RunStatusQueued
	case TransitionComplete:
		return from == RunStatusRunning || from == RunStatusQueued
	case TransitionFail:
		return from == RunStatusRunning || from == RunStatusQueued
	case TransitionCancel:
		return IsActiveRunStatus(from)
	case TransitionSupersede:
		return IsActiveRunStatus(from)
	default:
		return false
	}
}

package contracts

const EventName = "harbor:event"

const (
	StateProvisioning = "provisioning"
	StateRunning      = "running"
	StateStopping     = "stopping"
	StateStopped      = "stopped"
	StateFailed       = "failed"
)

func (e *AppError) Error() string { return e.Message }

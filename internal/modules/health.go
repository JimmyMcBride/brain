package modules

type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
)

type Health struct {
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
}

func (h Health) valid() bool {
	switch h.Status {
	case HealthHealthy, HealthDegraded, HealthUnhealthy:
		return true
	default:
		return false
	}
}

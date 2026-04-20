package ports

type CircuitBreakerStats struct {
	Name             string
	State            string // "closed", "open", "half-open"
	FailureCount     int
	SuccessCount     int
	TotalRequests    int64
	RejectedRequests int64
}

type CircuitBreaker interface {
	Name() string
	IsOpen() bool
	Stats() CircuitBreakerStats
}

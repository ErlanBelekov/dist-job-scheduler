package handler

const (
	errInternalServer = "Internal server error"
	errJobNotFound    = "Job not found"
	errDuplicateJob   = "Job with this idempotency key already exists"
	errTokenInvalid   = "Token is invalid or expired"
)

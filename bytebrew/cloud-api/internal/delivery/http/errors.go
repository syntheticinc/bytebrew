package http

import "github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"

// Sentinel errors reused across handlers.
var (
	invalidBodyError        = errors.InvalidInput("invalid request body")
	invalidLicenseParamError = errors.InvalidInput("license query parameter required")
	malformedJWTError       = errors.InvalidInput("malformed JWT")
)

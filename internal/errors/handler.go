package errors

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/route53domains/types"
	"github.com/aws/smithy-go"
)

// ExitCode represents different exit codes for the CLI
type ExitCode int

const (
	ExitSuccess        ExitCode = 0 // Success (domain checked successfully)
	ExitValidation     ExitCode = 1 // Validation error (invalid domain format)
	ExitAuthentication ExitCode = 2 // Authentication error (AWS credentials issue)
	ExitAuthorization  ExitCode = 3 // Authorization error (insufficient permissions)
	ExitAPIError       ExitCode = 4 // API error (AWS service error)
	ExitSystemError    ExitCode = 5 // System error (unexpected error)
)

// GetExitCode returns the appropriate exit code for an error
func GetExitCode(err error) ExitCode {
	if err == nil {
		return ExitSuccess
	}

	// Check for context errors first
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return ExitSystemError
	}

	// Check for custom error types
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return ExitValidation
	}

	var authenticationErr *AuthenticationError
	if errors.As(err, &authenticationErr) {
		return ExitAuthentication
	}

	var authorizationErr *AuthorizationError
	if errors.As(err, &authorizationErr) {
		return ExitAuthorization
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return ExitAPIError
	}

	var systemErr *SystemError
	if errors.As(err, &systemErr) {
		return ExitSystemError
	}

	// Check for AWS SDK errors
	if exitCode := classifyAWSError(err); exitCode != ExitSystemError {
		return exitCode
	}

	// Fallback to string-based classification for backward compatibility
	return classifyErrorByMessage(err)
}

// classifyAWSError classifies AWS SDK errors into appropriate categories
func classifyAWSError(err error) ExitCode {
	// Check for AWS API errors
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoCredentialsErr", "CredentialsNotFound", "InvalidAccessKeyId", "SignatureDoesNotMatch":
			return ExitAuthentication
		case "UnauthorizedOperation", "AccessDenied", "Forbidden":
			return ExitAuthorization
		case "InvalidDomainName", "DomainLimitExceeded", "UnsupportedTLD":
			return ExitValidation
		case "TooManyRequests", "Throttling", "RequestLimitExceeded":
			return ExitAPIError
		case "ServiceUnavailable", "InternalFailure", "RequestTimeout":
			return ExitAPIError
		default:
			return ExitAPIError
		}
	}

	// Check for Route 53 Domains specific errors
	var invalidInput *types.InvalidInput
	if errors.As(err, &invalidInput) {
		return ExitValidation
	}

	var operationLimitExceeded *types.OperationLimitExceeded
	if errors.As(err, &operationLimitExceeded) {
		return ExitAPIError
	}

	var unsupportedTLD *types.UnsupportedTLD
	if errors.As(err, &unsupportedTLD) {
		return ExitValidation
	}

	return ExitSystemError
}

// classifyErrorByMessage provides fallback error classification based on error messages
func classifyErrorByMessage(err error) ExitCode {
	errorMsg := strings.ToLower(err.Error())

	// Authentication errors
	authPatterns := []string{
		"no credentials",
		"credentials not found",
		"invalid access key",
		"signature does not match",
		"failed to initialize aws configuration",
		"no valid providers in chain",
	}
	for _, pattern := range authPatterns {
		if strings.Contains(errorMsg, pattern) {
			return ExitAuthentication
		}
	}

	// Authorization errors
	authzPatterns := []string{
		"access denied",
		"unauthorized",
		"forbidden",
		"insufficient permissions",
	}
	for _, pattern := range authzPatterns {
		if strings.Contains(errorMsg, pattern) {
			return ExitAuthorization
		}
	}

	// Validation errors
	validationPatterns := []string{
		"domain validation failed",
		"invalid domain format",
		"unsupported tld",
		"domain cannot be empty",
		"domain name too long",
		"domain name too short",
		"invalid domain name",
	}
	for _, pattern := range validationPatterns {
		if strings.Contains(errorMsg, pattern) {
			return ExitValidation
		}
	}

	// API errors
	apiPatterns := []string{
		"aws api call failed",
		"too many requests",
		"throttling",
		"request limit exceeded",
		"service unavailable",
		"internal failure",
		"request timeout",
	}
	for _, pattern := range apiPatterns {
		if strings.Contains(errorMsg, pattern) {
			return ExitAPIError
		}
	}

	return ExitSystemError
}

// WrapAWSError wraps AWS SDK errors with appropriate custom error types
func WrapAWSError(err error, service, operation string) error {
	if err == nil {
		return nil
	}

	// Check for AWS API errors first
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoCredentialsErr", "CredentialsNotFound", "InvalidAccessKeyId", "SignatureDoesNotMatch":
			return NewAuthenticationError("aws-sdk",
				"AWS credentials not found or invalid. Please configure your AWS credentials using environment variables, shared credentials file, or IAM roles.",
				err)
		case "UnauthorizedOperation", "AccessDenied", "Forbidden":
			return NewAuthorizationError(operation, service,
				"Insufficient permissions to perform this operation. Please ensure your AWS credentials have the 'route53domains:CheckDomainAvailability' permission.",
				err)
		case "InvalidDomainName", "UnsupportedTLD":
			return NewValidationError("", "domain",
				"The domain name is invalid or uses an unsupported TLD",
				err)
		case "TooManyRequests", "Throttling", "RequestLimitExceeded":
			return NewAPIError(service, operation,
				"Request rate limit exceeded. Please wait before retrying.",
				err).WithStatusCode(429)
		case "ServiceUnavailable", "InternalFailure":
			return NewAPIError(service, operation,
				"AWS service is temporarily unavailable. Please try again later.",
				err).WithStatusCode(503)
		case "RequestTimeout":
			return NewAPIError(service, operation,
				"Request timed out. Please check your network connection and try again.",
				err).WithStatusCode(408)
		default:
			return NewAPIError(service, operation,
				"AWS API call failed with an unexpected error",
				err)
		}
	}

	// Check for Route 53 Domains specific errors
	var invalidInput *types.InvalidInput
	if errors.As(err, &invalidInput) {
		return NewValidationError("", "domain",
			"The domain name format is invalid",
			err)
	}

	var operationLimitExceeded *types.OperationLimitExceeded
	if errors.As(err, &operationLimitExceeded) {
		return NewAPIError(service, operation,
			"Operation limit exceeded for Route 53 Domains API. Please wait before retrying.",
			err).WithStatusCode(429)
	}

	var unsupportedTLD *types.UnsupportedTLD
	if errors.As(err, &unsupportedTLD) {
		return NewValidationError("", "tld",
			"The top-level domain (TLD) is not supported by Route 53 Domains",
			err)
	}

	// For other AWS errors, wrap as API error
	return NewAPIError(service, operation,
		"AWS API call failed",
		err)
}

// WrapValidationError wraps domain validation errors
func WrapValidationError(domain string, err error) error {
	if err == nil {
		return nil
	}

	return NewValidationError(domain, "format", err.Error(), err)
}

// WrapSystemError wraps system-level errors
func WrapSystemError(component string, err error) error {
	if err == nil {
		return nil
	}

	return NewSystemError(component, err.Error(), err)
}

// IsRetryable determines if an error is retryable
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for retryable custom error types
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		// Rate limiting and temporary service errors are retryable
		return apiErr.StatusCode == 429 || apiErr.StatusCode == 503 || apiErr.StatusCode == 408
	}

	// Check for AWS SDK retryable errors
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "TooManyRequests", "Throttling", "RequestLimitExceeded", "ServiceUnavailable", "InternalFailure", "RequestTimeout":
			return true
		}
	}

	// Context timeout is not retryable (user-defined timeout)
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return false
}

package errors

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/route53domains/types"
	"github.com/aws/smithy-go"
)

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ExitCode
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ExitSuccess,
		},
		{
			name:     "validation error",
			err:      NewValidationError("test.com", "format", "invalid format", nil),
			expected: ExitValidation,
		},
		{
			name:     "authentication error",
			err:      NewAuthenticationError("aws-sdk", "credentials not found", nil),
			expected: ExitAuthentication,
		},
		{
			name:     "authorization error",
			err:      NewAuthorizationError("CheckDomainAvailability", "route53domains", "access denied", nil),
			expected: ExitAuthorization,
		},
		{
			name:     "API error",
			err:      NewAPIError("route53domains", "CheckDomainAvailability", "service error", nil),
			expected: ExitAPIError,
		},
		{
			name:     "system error",
			err:      NewSystemError("context", "operation cancelled", nil),
			expected: ExitSystemError,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: ExitSystemError,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: ExitSystemError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetExitCode(tt.err)
			if result != tt.expected {
				t.Errorf("GetExitCode(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestClassifyAWSError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ExitCode
	}{
		{
			name:     "credentials not found",
			err:      &smithy.GenericAPIError{Code: "NoCredentialsErr", Message: "no credentials"},
			expected: ExitAuthentication,
		},
		{
			name:     "invalid access key",
			err:      &smithy.GenericAPIError{Code: "InvalidAccessKeyId", Message: "invalid key"},
			expected: ExitAuthentication,
		},
		{
			name:     "access denied",
			err:      &smithy.GenericAPIError{Code: "AccessDenied", Message: "access denied"},
			expected: ExitAuthorization,
		},
		{
			name:     "unauthorized operation",
			err:      &smithy.GenericAPIError{Code: "UnauthorizedOperation", Message: "unauthorized"},
			expected: ExitAuthorization,
		},
		{
			name:     "invalid domain name",
			err:      &smithy.GenericAPIError{Code: "InvalidDomainName", Message: "invalid domain"},
			expected: ExitValidation,
		},
		{
			name:     "too many requests",
			err:      &smithy.GenericAPIError{Code: "TooManyRequests", Message: "rate limited"},
			expected: ExitAPIError,
		},
		{
			name:     "service unavailable",
			err:      &smithy.GenericAPIError{Code: "ServiceUnavailable", Message: "service down"},
			expected: ExitAPIError,
		},
		{
			name:     "unknown AWS error",
			err:      &smithy.GenericAPIError{Code: "UnknownError", Message: "unknown"},
			expected: ExitAPIError,
		},
		{
			name:     "Route53 invalid input",
			err:      &types.InvalidInput{Message: stringPtr("invalid input")},
			expected: ExitValidation,
		},
		{
			name:     "Route53 operation limit exceeded",
			err:      &types.OperationLimitExceeded{Message: stringPtr("operation limit exceeded")},
			expected: ExitAPIError,
		},
		{
			name:     "Route53 unsupported TLD",
			err:      &types.UnsupportedTLD{Message: stringPtr("unsupported TLD")},
			expected: ExitValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyAWSError(tt.err)
			if result != tt.expected {
				t.Errorf("classifyAWSError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestClassifyErrorByMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ExitCode
	}{
		{
			name:     "authentication - no credentials",
			err:      errors.New("no credentials found"),
			expected: ExitAuthentication,
		},
		{
			name:     "authentication - invalid access key",
			err:      errors.New("invalid access key"),
			expected: ExitAuthentication,
		},
		{
			name:     "authorization - access denied",
			err:      errors.New("access denied"),
			expected: ExitAuthorization,
		},
		{
			name:     "authorization - unauthorized",
			err:      errors.New("unauthorized operation"),
			expected: ExitAuthorization,
		},
		{
			name:     "validation - domain validation failed",
			err:      errors.New("domain validation failed"),
			expected: ExitValidation,
		},
		{
			name:     "validation - invalid domain format",
			err:      errors.New("invalid domain format"),
			expected: ExitValidation,
		},
		{
			name:     "validation - unsupported TLD",
			err:      errors.New("unsupported tld"),
			expected: ExitValidation,
		},
		{
			name:     "API - AWS API call failed",
			err:      errors.New("aws api call failed"),
			expected: ExitAPIError,
		},
		{
			name:     "API - too many requests",
			err:      errors.New("too many requests"),
			expected: ExitAPIError,
		},
		{
			name:     "system - unknown error",
			err:      errors.New("some unknown error"),
			expected: ExitSystemError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyErrorByMessage(tt.err)
			if result != tt.expected {
				t.Errorf("classifyErrorByMessage(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestWrapAWSError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		service   string
		operation string
		expected  ErrorCategory
	}{
		{
			name:      "nil error",
			err:       nil,
			service:   "route53domains",
			operation: "CheckDomainAvailability",
			expected:  "",
		},
		{
			name:      "credentials not found",
			err:       &smithy.GenericAPIError{Code: "NoCredentialsErr", Message: "no credentials"},
			service:   "route53domains",
			operation: "CheckDomainAvailability",
			expected:  CategoryAuthentication,
		},
		{
			name:      "access denied",
			err:       &smithy.GenericAPIError{Code: "AccessDenied", Message: "access denied"},
			service:   "route53domains",
			operation: "CheckDomainAvailability",
			expected:  CategoryAuthorization,
		},
		{
			name:      "invalid domain name",
			err:       &smithy.GenericAPIError{Code: "InvalidDomainName", Message: "invalid domain"},
			service:   "route53domains",
			operation: "CheckDomainAvailability",
			expected:  CategoryValidation,
		},
		{
			name:      "too many requests",
			err:       &smithy.GenericAPIError{Code: "TooManyRequests", Message: "rate limited"},
			service:   "route53domains",
			operation: "CheckDomainAvailability",
			expected:  CategoryAPI,
		},
		{
			name:      "generic AWS error",
			err:       &smithy.GenericAPIError{Code: "UnknownError", Message: "unknown"},
			service:   "route53domains",
			operation: "CheckDomainAvailability",
			expected:  CategoryAPI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapAWSError(tt.err, tt.service, tt.operation)

			if tt.err == nil {
				if result != nil {
					t.Errorf("WrapAWSError(nil) should return nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("WrapAWSError(%v) should not return nil", tt.err)
				return
			}

			var customErr interface {
				GetCategory() ErrorCategory
			}
			if !errors.As(result, &customErr) {
				t.Errorf("WrapAWSError(%v) should return a custom error type", tt.err)
				return
			}

			if customErr.GetCategory() != tt.expected {
				t.Errorf("WrapAWSError(%v) category = %v, want %v", tt.err, customErr.GetCategory(), tt.expected)
			}
		})
	}
}

func TestWrapValidationError(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			domain:   "test.com",
			err:      nil,
			expected: false,
		},
		{
			name:     "validation error",
			domain:   "test.com",
			err:      errors.New("invalid format"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapValidationError(tt.domain, tt.err)

			if tt.expected {
				if result == nil {
					t.Errorf("WrapValidationError(%v) should not return nil", tt.err)
					return
				}

				var validationErr *ValidationError
				if !errors.As(result, &validationErr) {
					t.Errorf("WrapValidationError(%v) should return ValidationError", tt.err)
					return
				}

				if validationErr.Domain != tt.domain {
					t.Errorf("ValidationError.Domain = %v, want %v", validationErr.Domain, tt.domain)
				}
			} else {
				if result != nil {
					t.Errorf("WrapValidationError(nil) should return nil, got %v", result)
				}
			}
		})
	}
}

func TestWrapSystemError(t *testing.T) {
	tests := []struct {
		name      string
		component string
		err       error
		expected  bool
	}{
		{
			name:      "nil error",
			component: "context",
			err:       nil,
			expected:  false,
		},
		{
			name:      "system error",
			component: "context",
			err:       errors.New("operation cancelled"),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapSystemError(tt.component, tt.err)

			if tt.expected {
				if result == nil {
					t.Errorf("WrapSystemError(%v) should not return nil", tt.err)
					return
				}

				var systemErr *SystemError
				if !errors.As(result, &systemErr) {
					t.Errorf("WrapSystemError(%v) should return SystemError", tt.err)
					return
				}

				if systemErr.Component != tt.component {
					t.Errorf("SystemError.Component = %v, want %v", systemErr.Component, tt.component)
				}
			} else {
				if result != nil {
					t.Errorf("WrapSystemError(nil) should return nil, got %v", result)
				}
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "API error with 429 status",
			err:      NewAPIError("route53domains", "CheckDomainAvailability", "rate limited", nil).WithStatusCode(429),
			expected: true,
		},
		{
			name:     "API error with 503 status",
			err:      NewAPIError("route53domains", "CheckDomainAvailability", "service unavailable", nil).WithStatusCode(503),
			expected: true,
		},
		{
			name:     "API error with 408 status",
			err:      NewAPIError("route53domains", "CheckDomainAvailability", "timeout", nil).WithStatusCode(408),
			expected: true,
		},
		{
			name:     "API error with 400 status",
			err:      NewAPIError("route53domains", "CheckDomainAvailability", "bad request", nil).WithStatusCode(400),
			expected: false,
		},
		{
			name:     "AWS TooManyRequests error",
			err:      &smithy.GenericAPIError{Code: "TooManyRequests", Message: "rate limited"},
			expected: true,
		},
		{
			name:     "AWS ServiceUnavailable error",
			err:      &smithy.GenericAPIError{Code: "ServiceUnavailable", Message: "service down"},
			expected: true,
		},
		{
			name:     "AWS AccessDenied error",
			err:      &smithy.GenericAPIError{Code: "AccessDenied", Message: "access denied"},
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "validation error",
			err:      NewValidationError("test.com", "format", "invalid", nil),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryable(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}

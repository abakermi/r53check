package errors

import (
	"errors"
	"testing"
)

func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		field    string
		message  string
		cause    error
		expected string
	}{
		{
			name:     "validation error with domain",
			domain:   "example.com",
			field:    "format",
			message:  "invalid format",
			cause:    nil,
			expected: "domain validation failed for 'example.com': invalid format",
		},
		{
			name:     "validation error without domain",
			domain:   "",
			field:    "format",
			message:  "invalid format",
			cause:    nil,
			expected: "validation error: invalid format",
		},
		{
			name:     "validation error with cause",
			domain:   "test.com",
			field:    "tld",
			message:  "unsupported TLD",
			cause:    errors.New("underlying error"),
			expected: "domain validation failed for 'test.com': unsupported TLD: underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewValidationError(tt.domain, tt.field, tt.message, tt.cause)

			if err.Error() != tt.expected {
				t.Errorf("ValidationError.Error() = %v, want %v", err.Error(), tt.expected)
			}

			if err.GetCategory() != CategoryValidation {
				t.Errorf("ValidationError.GetCategory() = %v, want %v", err.GetCategory(), CategoryValidation)
			}

			if err.Domain != tt.domain {
				t.Errorf("ValidationError.Domain = %v, want %v", err.Domain, tt.domain)
			}

			if err.Field != tt.field {
				t.Errorf("ValidationError.Field = %v, want %v", err.Field, tt.field)
			}

			if tt.cause != nil && !errors.Is(err, tt.cause) {
				t.Errorf("ValidationError should wrap the cause error")
			}
		})
	}
}

func TestAuthenticationError(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		message  string
		cause    error
		expected string
	}{
		{
			name:     "authentication error with provider",
			provider: "aws-sdk",
			message:  "credentials not found",
			cause:    nil,
			expected: "authentication failed with provider 'aws-sdk': credentials not found",
		},
		{
			name:     "authentication error without provider",
			provider: "",
			message:  "credentials not found",
			cause:    nil,
			expected: "authentication error: credentials not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAuthenticationError(tt.provider, tt.message, tt.cause)

			if err.Error() != tt.expected {
				t.Errorf("AuthenticationError.Error() = %v, want %v", err.Error(), tt.expected)
			}

			if err.GetCategory() != CategoryAuthentication {
				t.Errorf("AuthenticationError.GetCategory() = %v, want %v", err.GetCategory(), CategoryAuthentication)
			}

			if err.Provider != tt.provider {
				t.Errorf("AuthenticationError.Provider = %v, want %v", err.Provider, tt.provider)
			}
		})
	}
}

func TestAuthorizationError(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		resource  string
		message   string
		cause     error
		expected  string
	}{
		{
			name:      "authorization error with operation and resource",
			operation: "CheckDomainAvailability",
			resource:  "route53domains",
			message:   "access denied",
			cause:     nil,
			expected:  "authorization failed for operation 'CheckDomainAvailability' on resource 'route53domains': access denied",
		},
		{
			name:      "authorization error without operation and resource",
			operation: "",
			resource:  "",
			message:   "access denied",
			cause:     nil,
			expected:  "authorization error: access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAuthorizationError(tt.operation, tt.resource, tt.message, tt.cause)

			if err.Error() != tt.expected {
				t.Errorf("AuthorizationError.Error() = %v, want %v", err.Error(), tt.expected)
			}

			if err.GetCategory() != CategoryAuthorization {
				t.Errorf("AuthorizationError.GetCategory() = %v, want %v", err.GetCategory(), CategoryAuthorization)
			}

			if err.Operation != tt.operation {
				t.Errorf("AuthorizationError.Operation = %v, want %v", err.Operation, tt.operation)
			}

			if err.Resource != tt.resource {
				t.Errorf("AuthorizationError.Resource = %v, want %v", err.Resource, tt.resource)
			}
		})
	}
}

func TestAPIError(t *testing.T) {
	tests := []struct {
		name      string
		service   string
		operation string
		message   string
		cause     error
		expected  string
	}{
		{
			name:      "API error basic",
			service:   "route53domains",
			operation: "CheckDomainAvailability",
			message:   "service unavailable",
			cause:     nil,
			expected:  "AWS route53domains API error in CheckDomainAvailability: service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewAPIError(tt.service, tt.operation, tt.message, tt.cause)

			if err.Error() != tt.expected {
				t.Errorf("APIError.Error() = %v, want %v", err.Error(), tt.expected)
			}

			if err.GetCategory() != CategoryAPI {
				t.Errorf("APIError.GetCategory() = %v, want %v", err.GetCategory(), CategoryAPI)
			}

			if err.Service != tt.service {
				t.Errorf("APIError.Service = %v, want %v", err.Service, tt.service)
			}

			if err.Operation != tt.operation {
				t.Errorf("APIError.Operation = %v, want %v", err.Operation, tt.operation)
			}
		})
	}
}

func TestAPIErrorWithRequestID(t *testing.T) {
	err := NewAPIError("route53domains", "CheckDomainAvailability", "service error", nil)
	err = err.WithRequestID("req-123")

	expected := "AWS route53domains API error in CheckDomainAvailability (RequestID: req-123): service error"
	if err.Error() != expected {
		t.Errorf("APIError.Error() with RequestID = %v, want %v", err.Error(), expected)
	}

	if err.RequestID != "req-123" {
		t.Errorf("APIError.RequestID = %v, want %v", err.RequestID, "req-123")
	}

	if err.Context["requestId"] != "req-123" {
		t.Errorf("APIError.Context[requestId] = %v, want %v", err.Context["requestId"], "req-123")
	}
}

func TestAPIErrorWithStatusCode(t *testing.T) {
	err := NewAPIError("route53domains", "CheckDomainAvailability", "service error", nil)
	err = err.WithStatusCode(429)

	if err.StatusCode != 429 {
		t.Errorf("APIError.StatusCode = %v, want %v", err.StatusCode, 429)
	}

	if err.Context["statusCode"] != 429 {
		t.Errorf("APIError.Context[statusCode] = %v, want %v", err.Context["statusCode"], 429)
	}
}

func TestSystemError(t *testing.T) {
	tests := []struct {
		name      string
		component string
		message   string
		cause     error
		expected  string
	}{
		{
			name:      "system error with component",
			component: "context",
			message:   "operation cancelled",
			cause:     nil,
			expected:  "system error in context: operation cancelled",
		},
		{
			name:      "system error without component",
			component: "",
			message:   "unexpected error",
			cause:     nil,
			expected:  "system error: unexpected error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewSystemError(tt.component, tt.message, tt.cause)

			if err.Error() != tt.expected {
				t.Errorf("SystemError.Error() = %v, want %v", err.Error(), tt.expected)
			}

			if err.GetCategory() != CategorySystem {
				t.Errorf("SystemError.GetCategory() = %v, want %v", err.GetCategory(), CategorySystem)
			}

			if err.Component != tt.component {
				t.Errorf("SystemError.Component = %v, want %v", err.Component, tt.component)
			}
		})
	}
}

func TestBaseErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewValidationError("test.com", "format", "invalid", cause)

	if !errors.Is(err, cause) {
		t.Errorf("BaseError should unwrap to the cause error")
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped == nil {
		t.Errorf("BaseError.Unwrap() should return the cause error")
	}
}

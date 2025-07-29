package errors

import (
	"fmt"
)

// ErrorCategory represents different categories of errors
type ErrorCategory string

const (
	CategoryValidation     ErrorCategory = "VALIDATION"
	CategoryAuthentication ErrorCategory = "AUTHENTICATION"
	CategoryAuthorization  ErrorCategory = "AUTHORIZATION"
	CategoryAPI            ErrorCategory = "API"
	CategorySystem         ErrorCategory = "SYSTEM"
)

// BaseError provides common functionality for all custom errors
type BaseError struct {
	Category ErrorCategory
	Message  string
	Cause    error
	Context  map[string]interface{}
}

func (e *BaseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Cause.Error())
	}
	return e.Message
}

func (e *BaseError) Unwrap() error {
	return e.Cause
}

func (e *BaseError) GetCategory() ErrorCategory {
	return e.Category
}

func (e *BaseError) GetContext() map[string]interface{} {
	return e.Context
}

// ValidationError represents domain validation failures
type ValidationError struct {
	*BaseError
	Domain string
	Field  string
}

func NewValidationError(domain, field, message string, cause error) *ValidationError {
	return &ValidationError{
		BaseError: &BaseError{
			Category: CategoryValidation,
			Message:  message,
			Cause:    cause,
			Context: map[string]interface{}{
				"domain": domain,
				"field":  field,
			},
		},
		Domain: domain,
		Field:  field,
	}
}

func (e *ValidationError) Error() string {
	var baseMsg string
	if e.Domain != "" {
		baseMsg = fmt.Sprintf("domain validation failed for '%s': %s", e.Domain, e.Message)
	} else {
		baseMsg = fmt.Sprintf("validation error: %s", e.Message)
	}

	if e.Cause != nil {
		return fmt.Sprintf("%s: %s", baseMsg, e.Cause.Error())
	}
	return baseMsg
}

// AuthenticationError represents AWS credential issues
type AuthenticationError struct {
	*BaseError
	Provider string
}

func NewAuthenticationError(provider, message string, cause error) *AuthenticationError {
	return &AuthenticationError{
		BaseError: &BaseError{
			Category: CategoryAuthentication,
			Message:  message,
			Cause:    cause,
			Context: map[string]interface{}{
				"provider": provider,
			},
		},
		Provider: provider,
	}
}

func (e *AuthenticationError) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("authentication failed with provider '%s': %s", e.Provider, e.Message)
	}
	return fmt.Sprintf("authentication error: %s", e.Message)
}

// AuthorizationError represents insufficient permissions
type AuthorizationError struct {
	*BaseError
	Operation string
	Resource  string
}

func NewAuthorizationError(operation, resource, message string, cause error) *AuthorizationError {
	return &AuthorizationError{
		BaseError: &BaseError{
			Category: CategoryAuthorization,
			Message:  message,
			Cause:    cause,
			Context: map[string]interface{}{
				"operation": operation,
				"resource":  resource,
			},
		},
		Operation: operation,
		Resource:  resource,
	}
}

func (e *AuthorizationError) Error() string {
	if e.Operation != "" && e.Resource != "" {
		return fmt.Sprintf("authorization failed for operation '%s' on resource '%s': %s", e.Operation, e.Resource, e.Message)
	}
	return fmt.Sprintf("authorization error: %s", e.Message)
}

// APIError represents AWS API call failures
type APIError struct {
	*BaseError
	Service    string
	Operation  string
	RequestID  string
	StatusCode int
}

func NewAPIError(service, operation, message string, cause error) *APIError {
	return &APIError{
		BaseError: &BaseError{
			Category: CategoryAPI,
			Message:  message,
			Cause:    cause,
			Context: map[string]interface{}{
				"service":   service,
				"operation": operation,
			},
		},
		Service:   service,
		Operation: operation,
	}
}

func (e *APIError) WithRequestID(requestID string) *APIError {
	e.RequestID = requestID
	e.Context["requestId"] = requestID
	return e
}

func (e *APIError) WithStatusCode(statusCode int) *APIError {
	e.StatusCode = statusCode
	e.Context["statusCode"] = statusCode
	return e
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("AWS %s API error in %s (RequestID: %s): %s", e.Service, e.Operation, e.RequestID, e.Message)
	}
	return fmt.Sprintf("AWS %s API error in %s: %s", e.Service, e.Operation, e.Message)
}

// SystemError represents unexpected system errors
type SystemError struct {
	*BaseError
	Component string
}

func NewSystemError(component, message string, cause error) *SystemError {
	return &SystemError{
		BaseError: &BaseError{
			Category: CategorySystem,
			Message:  message,
			Cause:    cause,
			Context: map[string]interface{}{
				"component": component,
			},
		},
		Component: component,
	}
}

func (e *SystemError) Error() string {
	if e.Component != "" {
		return fmt.Sprintf("system error in %s: %s", e.Component, e.Message)
	}
	return fmt.Sprintf("system error: %s", e.Message)
}

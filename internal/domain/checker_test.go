package domain

import (
	"context"
	"errors"
	"testing"
	"time"

	customErrors "github.com/abakermi/r53check/internal/errors"

	"github.com/aws/aws-sdk-go-v2/service/route53domains"
	"github.com/aws/aws-sdk-go-v2/service/route53domains/types"
)

// MockValidator implements the Validator interface for testing
type MockValidator struct {
	shouldFail bool
	failError  error
}

func (m *MockValidator) ValidateDomain(domain string) error {
	if m.shouldFail {
		return m.failError
	}
	return nil
}

// MockRoute53Client implements the Route53Client interface for testing
type MockRoute53Client struct {
	response *route53domains.CheckDomainAvailabilityOutput
	err      error
	callLog  []string
}

func (m *MockRoute53Client) CheckDomainAvailability(ctx context.Context, domain string) (*route53domains.CheckDomainAvailabilityOutput, error) {
	m.callLog = append(m.callLog, domain)

	// Check if context was cancelled (for timeout tests)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return m.response, m.err
}

func TestNewDomainChecker(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{}

	checker := NewDomainChecker(validator, client)

	if checker.validator != validator {
		t.Error("Expected validator to be set")
	}
	if checker.awsClient != client {
		t.Error("Expected AWS client to be set")
	}
	if checker.timeout != 10*time.Second {
		t.Errorf("Expected default timeout of 10s, got %v", checker.timeout)
	}
}

func TestNewDomainCheckerWithTimeout(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{}
	customTimeout := 5 * time.Second

	checker := NewDomainCheckerWithTimeout(validator, client, customTimeout)

	if checker.timeout != customTimeout {
		t.Errorf("Expected timeout of %v, got %v", customTimeout, checker.timeout)
	}
}

func TestCheckAvailability_ValidationFailure(t *testing.T) {
	validator := &MockValidator{
		shouldFail: true,
		failError:  errors.New("invalid domain format"),
	}
	client := &MockRoute53Client{}
	checker := NewDomainChecker(validator, client)

	ctx := context.Background()
	result, err := checker.CheckAvailability(ctx, "invalid-domain")

	if err == nil {
		t.Error("Expected error for validation failure")
	}
	if result.Status != StatusUnknown {
		t.Errorf("Expected status %s, got %s", StatusUnknown, result.Status)
	}
	if result.Available {
		t.Error("Expected Available to be false for validation failure")
	}
	if result.Domain != "invalid-domain" {
		t.Errorf("Expected domain to be 'invalid-domain', got %s", result.Domain)
	}
}

func TestCheckAvailability_AWSError(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{
		err: errors.New("AWS API error"),
	}
	checker := NewDomainChecker(validator, client)

	ctx := context.Background()
	result, err := checker.CheckAvailability(ctx, "example.com")

	if err == nil {
		t.Error("Expected error for AWS API failure")
	}
	if result.Status != StatusUnknown {
		t.Errorf("Expected status %s, got %s", StatusUnknown, result.Status)
	}
	if result.Available {
		t.Error("Expected Available to be false for AWS error")
	}
}

func TestCheckAvailability_DomainAvailable(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{
		response: &route53domains.CheckDomainAvailabilityOutput{
			Availability: types.DomainAvailabilityAvailable,
		},
	}
	checker := NewDomainChecker(validator, client)

	ctx := context.Background()
	result, err := checker.CheckAvailability(ctx, "example.com")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Status != StatusAvailable {
		t.Errorf("Expected status %s, got %s", StatusAvailable, result.Status)
	}
	if !result.Available {
		t.Error("Expected Available to be true for available domain")
	}
	if result.Message == "" {
		t.Error("Expected non-empty message")
	}
	if result.CheckedAt.IsZero() {
		t.Error("Expected CheckedAt to be set")
	}
}

func TestCheckAvailability_DomainUnavailable(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{
		response: &route53domains.CheckDomainAvailabilityOutput{
			Availability: types.DomainAvailabilityUnavailable,
		},
	}
	checker := NewDomainChecker(validator, client)

	ctx := context.Background()
	result, err := checker.CheckAvailability(ctx, "example.com")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Status != StatusUnavailable {
		t.Errorf("Expected status %s, got %s", StatusUnavailable, result.Status)
	}
	if result.Available {
		t.Error("Expected Available to be false for unavailable domain")
	}
	if result.Message == "" {
		t.Error("Expected non-empty message")
	}
}

func TestCheckAvailability_DomainReserved(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{
		response: &route53domains.CheckDomainAvailabilityOutput{
			Availability: types.DomainAvailabilityReserved,
		},
	}
	checker := NewDomainChecker(validator, client)

	ctx := context.Background()
	result, err := checker.CheckAvailability(ctx, "example.com")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Status != StatusReserved {
		t.Errorf("Expected status %s, got %s", StatusReserved, result.Status)
	}
	if result.Available {
		t.Error("Expected Available to be false for reserved domain")
	}
	if result.Message == "" {
		t.Error("Expected non-empty message")
	}
}

func TestCheckAvailability_DomainUnknown(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{
		response: &route53domains.CheckDomainAvailabilityOutput{
			Availability: types.DomainAvailabilityDontKnow,
		},
	}
	checker := NewDomainChecker(validator, client)

	ctx := context.Background()
	result, err := checker.CheckAvailability(ctx, "example.com")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Status != StatusUnknown {
		t.Errorf("Expected status %s, got %s", StatusUnknown, result.Status)
	}
	if result.Available {
		t.Error("Expected Available to be false for unknown domain")
	}
	if result.Message == "" {
		t.Error("Expected non-empty message")
	}
}

func TestCheckAvailability_NilResponse(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{
		response: nil, // Simulate nil response
	}
	checker := NewDomainChecker(validator, client)

	ctx := context.Background()
	result, err := checker.CheckAvailability(ctx, "example.com")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Status != StatusUnknown {
		t.Errorf("Expected status %s, got %s", StatusUnknown, result.Status)
	}
	if result.Available {
		t.Error("Expected Available to be false for nil response")
	}
	if result.Message != "No response from AWS API" {
		t.Errorf("Expected specific message for nil response, got: %s", result.Message)
	}
}

func TestCheckAvailability_ContextTimeout(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{
		response: &route53domains.CheckDomainAvailabilityOutput{
			Availability: types.DomainAvailabilityAvailable,
		},
	}

	// Set a very short timeout
	checker := NewDomainCheckerWithTimeout(validator, client, 1*time.Nanosecond)

	ctx := context.Background()
	result, err := checker.CheckAvailability(ctx, "example.com")

	if err == nil {
		t.Error("Expected timeout error")
	}
	if result.Status != StatusUnknown {
		t.Errorf("Expected status %s, got %s", StatusUnknown, result.Status)
	}
	if result.Available {
		t.Error("Expected Available to be false for timeout")
	}
}

func TestCheckAvailability_ContextCancellation(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{
		response: &route53domains.CheckDomainAvailabilityOutput{
			Availability: types.DomainAvailabilityAvailable,
		},
	}
	checker := NewDomainChecker(validator, client)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := checker.CheckAvailability(ctx, "example.com")

	if err == nil {
		t.Error("Expected cancellation error")
	}
	if result.Status != StatusUnknown {
		t.Errorf("Expected status %s, got %s", StatusUnknown, result.Status)
	}
}

func TestSetTimeout(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{}
	checker := NewDomainChecker(validator, client)

	newTimeout := 15 * time.Second
	checker.SetTimeout(newTimeout)

	if checker.GetTimeout() != newTimeout {
		t.Errorf("Expected timeout %v, got %v", newTimeout, checker.GetTimeout())
	}
}

func TestMapAWSResponse_AllStatuses(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{}
	checker := NewDomainChecker(validator, client)

	testCases := []struct {
		name           string
		awsStatus      types.DomainAvailability
		expectedStatus AvailabilityStatus
		expectedAvail  bool
	}{
		{
			name:           "Available",
			awsStatus:      types.DomainAvailabilityAvailable,
			expectedStatus: StatusAvailable,
			expectedAvail:  true,
		},
		{
			name:           "Unavailable",
			awsStatus:      types.DomainAvailabilityUnavailable,
			expectedStatus: StatusUnavailable,
			expectedAvail:  false,
		},
		{
			name:           "Reserved",
			awsStatus:      types.DomainAvailabilityReserved,
			expectedStatus: StatusReserved,
			expectedAvail:  false,
		},
		{
			name:           "Don't Know",
			awsStatus:      types.DomainAvailabilityDontKnow,
			expectedStatus: StatusUnknown,
			expectedAvail:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			awsResult := &route53domains.CheckDomainAvailabilityOutput{
				Availability: tc.awsStatus,
			}

			result := &AvailabilityResult{
				Domain: "example.com",
			}

			checker.mapAWSResponse(awsResult, result)

			if result.Status != tc.expectedStatus {
				t.Errorf("Expected status %s, got %s", tc.expectedStatus, result.Status)
			}
			if result.Available != tc.expectedAvail {
				t.Errorf("Expected Available %v, got %v", tc.expectedAvail, result.Available)
			}
			if result.Message == "" {
				t.Error("Expected non-empty message")
			}
		})
	}
}

func TestCheckAvailability_Integration(t *testing.T) {
	// Test with real validator
	validator := NewDomainValidator()
	client := &MockRoute53Client{
		response: &route53domains.CheckDomainAvailabilityOutput{
			Availability: types.DomainAvailabilityAvailable,
		},
	}
	checker := NewDomainChecker(validator, client)

	ctx := context.Background()
	result, err := checker.CheckAvailability(ctx, "example.com")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Status != StatusAvailable {
		t.Errorf("Expected status %s, got %s", StatusAvailable, result.Status)
	}
	if !result.Available {
		t.Error("Expected Available to be true")
	}
	if len(client.callLog) != 1 || client.callLog[0] != "example.com" {
		t.Errorf("Expected AWS client to be called with 'example.com', got %v", client.callLog)
	}
}
func TestCheckAvailability_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name             string
		validatorErr     error
		awsErr           error
		expectedType     interface{}
		expectedCategory customErrors.ErrorCategory
	}{
		{
			name:             "validation error",
			validatorErr:     customErrors.NewValidationError("test.com", "format", "invalid format", nil),
			awsErr:           nil,
			expectedType:     &customErrors.ValidationError{},
			expectedCategory: customErrors.CategoryValidation,
		},
		{
			name:             "AWS authentication error",
			validatorErr:     nil,
			awsErr:           customErrors.NewAuthenticationError("aws-sdk", "credentials not found", nil),
			expectedType:     &customErrors.AuthenticationError{},
			expectedCategory: customErrors.CategoryAuthentication,
		},
		{
			name:             "AWS authorization error",
			validatorErr:     nil,
			awsErr:           customErrors.NewAuthorizationError("CheckDomainAvailability", "route53domains", "access denied", nil),
			expectedType:     &customErrors.AuthorizationError{},
			expectedCategory: customErrors.CategoryAuthorization,
		},
		{
			name:             "AWS API error",
			validatorErr:     nil,
			awsErr:           customErrors.NewAPIError("route53domains", "CheckDomainAvailability", "service error", nil),
			expectedType:     &customErrors.APIError{},
			expectedCategory: customErrors.CategoryAPI,
		},
		{
			name:             "generic AWS error gets wrapped",
			validatorErr:     nil,
			awsErr:           errors.New("generic AWS error"),
			expectedType:     &customErrors.APIError{},
			expectedCategory: customErrors.CategoryAPI,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := &MockValidator{
				shouldFail: tc.validatorErr != nil,
				failError:  tc.validatorErr,
			}
			client := &MockRoute53Client{
				err: tc.awsErr,
			}
			checker := NewDomainChecker(validator, client)

			ctx := context.Background()
			result, err := checker.CheckAvailability(ctx, "example.com")

			if err == nil {
				t.Errorf("Expected error, got nil")
				return
			}

			// Check if it's the right type of error
			if !errors.As(err, tc.expectedType) {
				t.Errorf("expected error type %T, got %T", tc.expectedType, err)
				return
			}

			// Check error category
			var customErr interface {
				GetCategory() customErrors.ErrorCategory
			}
			if errors.As(err, &customErr) {
				if customErr.GetCategory() != tc.expectedCategory {
					t.Errorf("expected error category %v, got %v", tc.expectedCategory, customErr.GetCategory())
				}
			} else {
				t.Errorf("error should implement GetCategory() method")
			}

			// Check that result has error set
			if result.Error == nil {
				t.Errorf("expected result.Error to be set")
			}

			// Check that result status is unknown for errors
			if result.Status != StatusUnknown {
				t.Errorf("expected status %s for error, got %s", StatusUnknown, result.Status)
			}

			// Check that result is not available for errors
			if result.Available {
				t.Errorf("expected Available to be false for error")
			}
		})
	}
}

func TestCheckAvailability_ContextErrors(t *testing.T) {
	validator := &MockValidator{}
	client := &MockRoute53Client{
		response: &route53domains.CheckDomainAvailabilityOutput{
			Availability: types.DomainAvailabilityAvailable,
		},
	}
	checker := NewDomainChecker(validator, client)

	testCases := []struct {
		name        string
		setupCtx    func() context.Context
		expectedErr error
	}{
		{
			name: "context canceled",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			expectedErr: context.Canceled,
		},
		{
			name: "context deadline exceeded",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(1 * time.Millisecond) // Ensure timeout
				return ctx
			},
			expectedErr: context.DeadlineExceeded,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.setupCtx()
			result, err := checker.CheckAvailability(ctx, "example.com")

			if err == nil {
				t.Errorf("Expected error, got nil")
				return
			}

			// Check that the context error is preserved
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("expected error to wrap %v, got %v", tc.expectedErr, err)
			}

			// Check that result has error set
			if result.Error == nil {
				t.Errorf("expected result.Error to be set")
			}

			// Check that result status is unknown for errors
			if result.Status != StatusUnknown {
				t.Errorf("expected status %s for error, got %s", StatusUnknown, result.Status)
			}
		})
	}
}

func TestCheckAvailability_ErrorPropagation(t *testing.T) {
	// Test that validation errors are returned as-is (not double-wrapped)
	validationErr := customErrors.NewValidationError("test.com", "format", "invalid format", nil)
	validator := &MockValidator{
		shouldFail: true,
		failError:  validationErr,
	}
	client := &MockRoute53Client{}
	checker := NewDomainChecker(validator, client)

	ctx := context.Background()
	result, err := checker.CheckAvailability(ctx, "test.com")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// The error should be the same instance (not wrapped again)
	if err != validationErr {
		t.Errorf("expected same error instance, got different error: %v", err)
	}

	// Check that result.Error is also set to the same error
	if result.Error != validationErr {
		t.Errorf("expected result.Error to be same instance, got: %v", result.Error)
	}
}

func TestCheckAvailability_AWSErrorWrapping(t *testing.T) {
	// Test that non-custom AWS errors get wrapped
	validator := &MockValidator{}
	genericErr := errors.New("generic AWS error")
	client := &MockRoute53Client{
		err: genericErr,
	}
	checker := NewDomainChecker(validator, client)

	ctx := context.Background()
	result, err := checker.CheckAvailability(ctx, "example.com")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Check that result has error set
	if result.Error == nil {
		t.Errorf("expected result.Error to be set")
	}

	// Should be wrapped as APIError
	var apiErr *customErrors.APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("expected APIError, got %T", err)
		return
	}

	// Should have the right service and operation
	if apiErr.Service != "route53domains" {
		t.Errorf("expected service 'route53domains', got %s", apiErr.Service)
	}
	if apiErr.Operation != "CheckDomainAvailability" {
		t.Errorf("expected operation 'CheckDomainAvailability', got %s", apiErr.Operation)
	}

	// Should wrap the original error
	if !errors.Is(err, genericErr) {
		t.Errorf("expected error to wrap original error")
	}
}

func TestCheckAvailability_CustomErrorPreservation(t *testing.T) {
	// Test that custom AWS errors are preserved (not double-wrapped)
	validator := &MockValidator{}
	customErr := customErrors.NewAPIError("route53domains", "CheckDomainAvailability", "rate limited", nil)
	client := &MockRoute53Client{
		err: customErr,
	}
	checker := NewDomainChecker(validator, client)

	ctx := context.Background()
	result, err := checker.CheckAvailability(ctx, "example.com")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// The error should be the same instance (not wrapped again)
	if err != customErr {
		t.Errorf("expected same error instance, got different error: %v", err)
	}

	// Check that result.Error is also set to the same error
	if result.Error != customErr {
		t.Errorf("expected result.Error to be same instance, got: %v", result.Error)
	}
}

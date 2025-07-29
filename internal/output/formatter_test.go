package output

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/abakermi/r53check/internal/domain"
)

func TestNewConsoleFormatter(t *testing.T) {
	formatter := NewConsoleFormatter()

	if formatter == nil {
		t.Fatal("NewConsoleFormatter returned nil")
	}

	if formatter.ShowTimestamp {
		t.Error("Expected ShowTimestamp to be false by default")
	}

	if formatter.Verbose {
		t.Error("Expected Verbose to be false by default")
	}
}

func TestNewVerboseConsoleFormatter(t *testing.T) {
	formatter := NewVerboseConsoleFormatter()

	if formatter == nil {
		t.Fatal("NewVerboseConsoleFormatter returned nil")
	}

	if !formatter.ShowTimestamp {
		t.Error("Expected ShowTimestamp to be true for verbose formatter")
	}

	if !formatter.Verbose {
		t.Error("Expected Verbose to be true for verbose formatter")
	}
}

func TestConsoleFormatter_FormatResult(t *testing.T) {
	formatter := NewConsoleFormatter()
	testTime := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		result   *domain.AvailabilityResult
		expected string
	}{
		{
			name: "Available domain",
			result: &domain.AvailabilityResult{
				Domain:    "example.com",
				Available: true,
				Status:    domain.StatusAvailable,
				Message:   "Domain example.com is available for registration",
				CheckedAt: testTime,
			},
			expected: "✓ example.com is AVAILABLE for registration",
		},
		{
			name: "Unavailable domain",
			result: &domain.AvailabilityResult{
				Domain:    "google.com",
				Available: false,
				Status:    domain.StatusUnavailable,
				Message:   "Domain google.com is already registered",
				CheckedAt: testTime,
			},
			expected: "✗ google.com is UNAVAILABLE (already registered)",
		},
		{
			name: "Reserved domain",
			result: &domain.AvailabilityResult{
				Domain:    "reserved.com",
				Available: false,
				Status:    domain.StatusReserved,
				Message:   "Domain reserved.com is reserved and cannot be registered",
				CheckedAt: testTime,
			},
			expected: "⚠ reserved.com is RESERVED and cannot be registered",
		},
		{
			name: "Unknown status domain",
			result: &domain.AvailabilityResult{
				Domain:    "unknown.com",
				Available: false,
				Status:    domain.StatusUnknown,
				Message:   "Unable to determine availability for domain unknown.com",
				CheckedAt: testTime,
			},
			expected: "? unknown.com availability is UNKNOWN",
		},
		{
			name: "Result with error",
			result: &domain.AvailabilityResult{
				Domain:    "error.com",
				Available: false,
				Status:    domain.StatusUnknown,
				CheckedAt: testTime,
				Error:     errors.New("test error"),
			},
			expected: "Error: test error",
		},
		{
			name:     "Nil result",
			result:   nil,
			expected: "Error: No result to format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatResult(tt.result)
			if result != tt.expected {
				t.Errorf("FormatResult() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestConsoleFormatter_FormatResult_Verbose(t *testing.T) {
	formatter := NewVerboseConsoleFormatter()
	testTime := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)

	result := &domain.AvailabilityResult{
		Domain:    "example.com",
		Available: true,
		Status:    domain.StatusAvailable,
		Message:   "Domain example.com is available for registration",
		CheckedAt: testTime,
	}

	output := formatter.FormatResult(result)

	// Check that verbose output contains expected elements
	if !strings.Contains(output, "✓ example.com is AVAILABLE for registration") {
		t.Error("Verbose output should contain main status line")
	}

	if !strings.Contains(output, "Status: AVAILABLE") {
		t.Error("Verbose output should contain status details")
	}

	if !strings.Contains(output, "Message: Domain example.com is available for registration") {
		t.Error("Verbose output should contain message details")
	}

	if !strings.Contains(output, "Checked at: 2023-12-25 10:30:00 UTC") {
		t.Error("Verbose output should contain timestamp")
	}
}

func TestConsoleFormatter_FormatError(t *testing.T) {
	formatter := NewConsoleFormatter()

	tests := []struct {
		name     string
		err      error
		contains []string
	}{
		{
			name:     "Nil error",
			err:      nil,
			contains: []string{},
		},
		{
			name: "Authentication error - NoCredentialsErr",
			err:  errors.New("NoCredentialsErr: no valid providers in chain"),
			contains: []string{
				"Authentication Error",
				"AWS credentials not found",
				"export AWS_ACCESS_KEY_ID",
				"aws configure",
			},
		},
		{
			name: "Authentication error - no valid providers",
			err:  errors.New("no valid providers in chain"),
			contains: []string{
				"Authentication Error",
				"AWS credentials not found",
				"export AWS_ACCESS_KEY_ID",
			},
		},
		{
			name: "Authorization error - UnauthorizedOperation",
			err:  errors.New("UnauthorizedOperation: insufficient permissions"),
			contains: []string{
				"Authorization Error",
				"Insufficient permissions",
				"route53domains:CheckDomainAvailability",
			},
		},
		{
			name: "Authorization error - AccessDenied",
			err:  errors.New("AccessDenied: access denied"),
			contains: []string{
				"Authorization Error",
				"Insufficient permissions",
				"route53domains:CheckDomainAvailability",
			},
		},
		{
			name: "Domain validation error",
			err:  errors.New("domain validation failed: invalid domain format"),
			contains: []string{
				"Domain Validation Error",
				"invalid domain format",
				"Must be a valid domain name",
				"supported TLD",
			},
		},
		{
			name: "Rate limit error - TooManyRequests",
			err:  errors.New("TooManyRequests: rate limit exceeded"),
			contains: []string{
				"Rate Limit Error",
				"Too many requests",
				"Wait a few seconds",
			},
		},
		{
			name: "Rate limit error - Throttling",
			err:  errors.New("Throttling: request throttled"),
			contains: []string{
				"Rate Limit Error",
				"Too many requests",
				"Wait a few seconds",
			},
		},
		{
			name: "Timeout error",
			err:  errors.New("context deadline exceeded"),
			contains: []string{
				"Timeout Error",
				"Request took too long",
				"Slow network connection",
			},
		},
		{
			name: "Network error",
			err:  errors.New("network connection failed"),
			contains: []string{
				"Network Error",
				"Unable to connect",
				"Check your internet connection",
			},
		},
		{
			name: "Generic error",
			err:  errors.New("some unexpected error"),
			contains: []string{
				"Error: some unexpected error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatError(tt.err)

			if tt.err == nil {
				if result != "" {
					t.Errorf("FormatError(nil) = %q, expected empty string", result)
				}
				return
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("FormatError() result should contain %q, got: %s", expected, result)
				}
			}
		})
	}
}

func TestConsoleFormatter_SettersAndGetters(t *testing.T) {
	formatter := NewConsoleFormatter()

	// Test initial state
	if formatter.IsVerbose() {
		t.Error("Expected IsVerbose() to return false initially")
	}

	if formatter.ShowsTimestamp() {
		t.Error("Expected ShowsTimestamp() to return false initially")
	}

	// Test SetVerbose
	formatter.SetVerbose(true)
	if !formatter.IsVerbose() {
		t.Error("Expected IsVerbose() to return true after SetVerbose(true)")
	}

	formatter.SetVerbose(false)
	if formatter.IsVerbose() {
		t.Error("Expected IsVerbose() to return false after SetVerbose(false)")
	}

	// Test SetShowTimestamp
	formatter.SetShowTimestamp(true)
	if !formatter.ShowsTimestamp() {
		t.Error("Expected ShowsTimestamp() to return true after SetShowTimestamp(true)")
	}

	formatter.SetShowTimestamp(false)
	if formatter.ShowsTimestamp() {
		t.Error("Expected ShowsTimestamp() to return false after SetShowTimestamp(false)")
	}
}

func TestConsoleFormatter_FormatResult_EdgeCases(t *testing.T) {
	formatter := NewConsoleFormatter()
	testTime := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		result   *domain.AvailabilityResult
		expected string
	}{
		{
			name: "Unknown status type",
			result: &domain.AvailabilityResult{
				Domain:    "test.com",
				Available: false,
				Status:    domain.AvailabilityStatus("CUSTOM_STATUS"),
				CheckedAt: testTime,
			},
			expected: "? test.com has unknown status: CUSTOM_STATUS",
		},
		{
			name: "Empty domain name",
			result: &domain.AvailabilityResult{
				Domain:    "",
				Available: true,
				Status:    domain.StatusAvailable,
				CheckedAt: testTime,
			},
			expected: "✓  is AVAILABLE for registration",
		},
		{
			name: "Result with empty message in verbose mode",
			result: &domain.AvailabilityResult{
				Domain:    "test.com",
				Available: true,
				Status:    domain.StatusAvailable,
				Message:   "",
				CheckedAt: testTime,
			},
			expected: "✓ test.com is AVAILABLE for registration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatResult(tt.result)
			if !strings.Contains(result, tt.expected) {
				t.Errorf("FormatResult() = %q, should contain %q", result, tt.expected)
			}
		})
	}
}

func TestConsoleFormatter_VerboseOutput_WithoutTimestamp(t *testing.T) {
	formatter := NewConsoleFormatter()
	formatter.SetVerbose(true)
	formatter.SetShowTimestamp(false)

	testTime := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)

	result := &domain.AvailabilityResult{
		Domain:    "example.com",
		Available: true,
		Status:    domain.StatusAvailable,
		Message:   "Domain example.com is available for registration",
		CheckedAt: testTime,
	}

	output := formatter.FormatResult(result)

	// Should contain verbose info but not timestamp
	if !strings.Contains(output, "Status: AVAILABLE") {
		t.Error("Verbose output should contain status details")
	}

	if strings.Contains(output, "Checked at:") {
		t.Error("Output should not contain timestamp when ShowTimestamp is false")
	}
}

// Benchmark tests for performance
func BenchmarkConsoleFormatter_FormatResult(b *testing.B) {
	formatter := NewConsoleFormatter()
	result := &domain.AvailabilityResult{
		Domain:    "example.com",
		Available: true,
		Status:    domain.StatusAvailable,
		Message:   "Domain example.com is available for registration",
		CheckedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.FormatResult(result)
	}
}

func BenchmarkConsoleFormatter_FormatError(b *testing.B) {
	formatter := NewConsoleFormatter()
	err := errors.New("NoCredentialsErr: no valid providers in chain")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatter.FormatError(err)
	}
}

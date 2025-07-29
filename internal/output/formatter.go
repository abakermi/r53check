package output

import (
	"fmt"
	"strings"

	"github.com/abakermi/r53check/internal/domain"
)

// Formatter interface defines methods for formatting output
type Formatter interface {
	FormatResult(result *domain.AvailabilityResult) string
	FormatError(err error) string
	FormatBulkResults(results []*domain.AvailabilityResult) string
}

// ConsoleFormatter implements human-readable console output
type ConsoleFormatter struct {
	// ShowTimestamp controls whether to include timestamp in output
	ShowTimestamp bool
	// Verbose controls the level of detail in output
	Verbose bool
}

// NewConsoleFormatter creates a new console formatter with default settings
func NewConsoleFormatter() *ConsoleFormatter {
	return &ConsoleFormatter{
		ShowTimestamp: false,
		Verbose:       false,
	}
}

// NewVerboseConsoleFormatter creates a console formatter with verbose output
func NewVerboseConsoleFormatter() *ConsoleFormatter {
	return &ConsoleFormatter{
		ShowTimestamp: true,
		Verbose:       true,
	}
}

// FormatResult formats a domain availability result for console output
func (f *ConsoleFormatter) FormatResult(result *domain.AvailabilityResult) string {
	if result == nil {
		return "Error: No result to format"
	}

	var output strings.Builder

	// Handle error cases
	if result.Error != nil {
		return f.FormatError(result.Error)
	}

	// Format the main result based on availability
	switch result.Status {
	case domain.StatusAvailable:
		output.WriteString(fmt.Sprintf("✓ %s is AVAILABLE for registration", result.Domain))
	case domain.StatusUnavailable:
		output.WriteString(fmt.Sprintf("✗ %s is UNAVAILABLE (already registered)", result.Domain))
	case domain.StatusReserved:
		output.WriteString(fmt.Sprintf("⚠ %s is RESERVED and cannot be registered", result.Domain))
	case domain.StatusUnknown:
		output.WriteString(fmt.Sprintf("? %s availability is UNKNOWN", result.Domain))
	default:
		output.WriteString(fmt.Sprintf("? %s has unknown status: %s", result.Domain, result.Status))
	}

	// Add verbose information if requested
	if f.Verbose {
		output.WriteString(fmt.Sprintf("\nStatus: %s", result.Status))
		if result.Message != "" {
			output.WriteString(fmt.Sprintf("\nMessage: %s", result.Message))
		}
		if f.ShowTimestamp {
			output.WriteString(fmt.Sprintf("\nChecked at: %s", result.CheckedAt.Format("2006-01-02 15:04:05 MST")))
		}
	}

	return output.String()
}

// FormatError formats various error types with clear, actionable messages
func (f *ConsoleFormatter) FormatError(err error) string {
	if err == nil {
		return ""
	}

	errorMsg := err.Error()

	// Handle specific AWS error types with helpful messages
	if strings.Contains(errorMsg, "NoCredentialsErr") || strings.Contains(errorMsg, "no valid providers in chain") {
		return f.formatAuthenticationError()
	}

	if strings.Contains(errorMsg, "UnauthorizedOperation") || strings.Contains(errorMsg, "AccessDenied") {
		return f.formatAuthorizationError()
	}

	if strings.Contains(errorMsg, "InvalidDomainName") {
		return f.formatDomainValidationError(errorMsg)
	}

	if strings.Contains(errorMsg, "TooManyRequests") || strings.Contains(errorMsg, "Throttling") {
		return f.formatRateLimitError()
	}

	if strings.Contains(errorMsg, "domain validation failed") {
		return f.formatDomainValidationError(errorMsg)
	}

	if strings.Contains(errorMsg, "context deadline exceeded") || strings.Contains(errorMsg, "timeout") {
		return f.formatTimeoutError()
	}

	if strings.Contains(errorMsg, "network") || strings.Contains(errorMsg, "connection") {
		return f.formatNetworkError(errorMsg)
	}

	// Generic error formatting
	return fmt.Sprintf("Error: %s", errorMsg)
}

// formatAuthenticationError provides helpful guidance for credential issues
func (f *ConsoleFormatter) formatAuthenticationError() string {
	var output strings.Builder
	output.WriteString("✗ Authentication Error: AWS credentials not found\n")
	output.WriteString("\nTo fix this issue, try one of the following:\n")
	output.WriteString("  1. Set environment variables:\n")
	output.WriteString("     export AWS_ACCESS_KEY_ID=your-access-key\n")
	output.WriteString("     export AWS_SECRET_ACCESS_KEY=your-secret-key\n")
	output.WriteString("  2. Configure AWS CLI: aws configure\n")
	output.WriteString("  3. Use IAM roles if running on EC2/ECS/Lambda\n")
	output.WriteString("  4. Set AWS_PROFILE environment variable to use a specific profile")
	return output.String()
}

// formatAuthorizationError provides guidance for permission issues
func (f *ConsoleFormatter) formatAuthorizationError() string {
	var output strings.Builder
	output.WriteString("✗ Authorization Error: Insufficient permissions\n")
	output.WriteString("\nTo fix this issue:\n")
	output.WriteString("  1. Ensure your AWS user/role has the 'route53domains:CheckDomainAvailability' permission\n")
	output.WriteString("  2. Check if your account has access to Route 53 Domains service\n")
	output.WriteString("  3. Verify you're using the correct AWS region (Route 53 Domains is global)")
	return output.String()
}

// formatDomainValidationError provides specific guidance for domain format issues
func (f *ConsoleFormatter) formatDomainValidationError(errorMsg string) string {
	var output strings.Builder
	output.WriteString("✗ Domain Validation Error\n")
	output.WriteString(fmt.Sprintf("Details: %s\n", errorMsg))
	output.WriteString("\nDomain format requirements:\n")
	output.WriteString("  • Must be a valid domain name (e.g., example.com)\n")
	output.WriteString("  • Must include a supported TLD (.com, .net, .org, .io, etc.)\n")
	output.WriteString("  • Cannot contain spaces or special characters\n")
	output.WriteString("  • Must be between 1-63 characters per label")
	return output.String()
}

// formatRateLimitError provides guidance for API rate limiting
func (f *ConsoleFormatter) formatRateLimitError() string {
	var output strings.Builder
	output.WriteString("✗ Rate Limit Error: Too many requests to AWS API\n")
	output.WriteString("\nTo fix this issue:\n")
	output.WriteString("  • Wait a few seconds and try again\n")
	output.WriteString("  • AWS Route 53 Domains has rate limits to prevent abuse\n")
	output.WriteString("  • Consider implementing delays between requests if checking multiple domains")
	return output.String()
}

// formatTimeoutError provides guidance for timeout issues
func (f *ConsoleFormatter) formatTimeoutError() string {
	var output strings.Builder
	output.WriteString("✗ Timeout Error: Request took too long to complete\n")
	output.WriteString("\nPossible causes:\n")
	output.WriteString("  • Slow network connection\n")
	output.WriteString("  • AWS service temporarily unavailable\n")
	output.WriteString("  • Request timeout set too low\n")
	output.WriteString("\nTry running the command again or check your network connection.")
	return output.String()
}

// formatNetworkError provides guidance for network-related issues
func (f *ConsoleFormatter) formatNetworkError(errorMsg string) string {
	var output strings.Builder
	output.WriteString("✗ Network Error: Unable to connect to AWS services\n")
	output.WriteString(fmt.Sprintf("Details: %s\n", errorMsg))
	output.WriteString("\nPossible solutions:\n")
	output.WriteString("  • Check your internet connection\n")
	output.WriteString("  • Verify firewall settings allow HTTPS traffic\n")
	output.WriteString("  • Check if you're behind a corporate proxy\n")
	output.WriteString("  • Try again in a few minutes")
	return output.String()
}

// SetVerbose enables or disables verbose output
func (f *ConsoleFormatter) SetVerbose(verbose bool) {
	f.Verbose = verbose
}

// SetShowTimestamp enables or disables timestamp display
func (f *ConsoleFormatter) SetShowTimestamp(show bool) {
	f.ShowTimestamp = show
}

// IsVerbose returns whether verbose mode is enabled
func (f *ConsoleFormatter) IsVerbose() bool {
	return f.Verbose
}

// ShowsTimestamp returns whether timestamp display is enabled
func (f *ConsoleFormatter) ShowsTimestamp() bool {
	return f.ShowTimestamp
}

// FormatBulkResults formats multiple domain availability results
func (f *ConsoleFormatter) FormatBulkResults(results []*domain.AvailabilityResult) string {
	if len(results) == 0 {
		return "No domains to check"
	}

	var output strings.Builder

	// Summary header
	availableCount := 0
	unavailableCount := 0
	errorCount := 0

	for _, result := range results {
		if result == nil {
			errorCount++
			continue
		}
		if result.Error != nil {
			errorCount++
		} else if result.Available {
			availableCount++
		} else {
			unavailableCount++
		}
	}

	output.WriteString(fmt.Sprintf("Bulk Domain Check Results (%d domains)\n", len(results)))
	output.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Individual results
	for _, result := range results {
		if result == nil {
			output.WriteString("? UNKNOWN: Invalid result\n")
			continue
		}

		if result.Error != nil {
			output.WriteString(fmt.Sprintf("✗ %s: ERROR - %s\n", result.Domain, result.Error.Error()))
			continue
		}

		switch result.Status {
		case domain.StatusAvailable:
			output.WriteString(fmt.Sprintf("✓ %s: AVAILABLE\n", result.Domain))
		case domain.StatusUnavailable:
			output.WriteString(fmt.Sprintf("✗ %s: UNAVAILABLE (already registered)\n", result.Domain))
		case domain.StatusReserved:
			output.WriteString(fmt.Sprintf("⚠ %s: RESERVED (cannot be registered)\n", result.Domain))
		case domain.StatusUnknown:
			output.WriteString(fmt.Sprintf("? %s: UNKNOWN (unable to determine)\n", result.Domain))
		default:
			output.WriteString(fmt.Sprintf("? %s: UNKNOWN STATUS\n", result.Domain))
		}

		// Add verbose details if enabled
		if f.Verbose && result.Error == nil {
			output.WriteString(fmt.Sprintf("  Message: %s\n", result.Message))
			if f.ShowTimestamp {
				output.WriteString(fmt.Sprintf("  Checked: %s\n", result.CheckedAt.Format("2006-01-02 15:04:05 MST")))
			}
		}
	}

	// Summary footer
	output.WriteString("\n" + strings.Repeat("=", 50) + "\n")
	output.WriteString("Summary:\n")
	output.WriteString(fmt.Sprintf("  ✓ Available: %d\n", availableCount))
	output.WriteString(fmt.Sprintf("  ✗ Unavailable: %d\n", unavailableCount))
	if errorCount > 0 {
		output.WriteString(fmt.Sprintf("  ⚠ Errors: %d\n", errorCount))
	}

	return output.String()
}

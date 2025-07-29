package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	customErrors "github.com/abakermi/r53check/internal/errors"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		contains string
	}{
		{
			name:     "help flag",
			args:     []string{"--help"},
			wantErr:  false,
			contains: "Route 53 Domain Availability Checker",
		},
		{
			name:     "version info in help",
			args:     []string{"--help"},
			wantErr:  false,
			contains: "Usage:",
		},
		{
			name:     "check subcommand exists",
			args:     []string{"--help"},
			wantErr:  false,
			contains: "check",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new root command for each test to avoid state pollution
			cmd := &cobra.Command{
				Use:   "route53-checker",
				Short: "Check domain availability in AWS Route 53",
				Long: `Route 53 Domain Availability Checker is a CLI tool for checking
domain availability within Amazon Route 53. It provides a fast, reliable
way to query Route 53 for domain registration status without navigating
the AWS console.

This tool is designed for developers, AWS administrators, and website
planners who need to verify domain availability for their projects.`,
			}

			// Add flags
			cmd.PersistentFlags().Duration("timeout", 10*time.Second, "Timeout for API requests")
			cmd.PersistentFlags().String("region", "", "AWS region (defaults to AWS SDK default)")
			cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

			// Add check command
			checkCmd := &cobra.Command{
				Use:   "check [domain]",
				Short: "Check if a domain is available for registration",
				Args:  cobra.ExactArgs(1),
			}
			cmd.AddCommand(checkCmd)

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("rootCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.contains != "" && !strings.Contains(output, tt.contains) {
				t.Errorf("Expected output to contain %q, got:\n%s", tt.contains, output)
			}
		})
	}
}

func TestCheckCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		contains string
	}{
		{
			name:     "help flag",
			args:     []string{"check", "--help"},
			wantErr:  false,
			contains: "Check if a domain is available for registration",
		},
		{
			name:     "no arguments",
			args:     []string{"check"},
			wantErr:  true,
			contains: "accepts 1 arg(s), received 0",
		},
		{
			name:     "too many arguments",
			args:     []string{"check", "example.com", "extra.com"},
			wantErr:  true,
			contains: "accepts 1 arg(s), received 2",
		},
		{
			name:     "examples in help",
			args:     []string{"check", "--help"},
			wantErr:  false,
			contains: "route53-checker check example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new root command for each test
			cmd := &cobra.Command{
				Use: "route53-checker",
			}

			// Add flags
			cmd.PersistentFlags().Duration("timeout", 10*time.Second, "Timeout for API requests")
			cmd.PersistentFlags().String("region", "", "AWS region (defaults to AWS SDK default)")
			cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

			// Add check command (without the actual implementation to avoid AWS dependencies in tests)
			checkCmd := &cobra.Command{
				Use:   "check [domain]",
				Short: "Check if a domain is available for registration",
				Long: `Check if a domain is available for registration in AWS Route 53.
	
The command validates the domain format and queries the Route 53 Domains API
to determine availability status. It returns clear messages indicating whether
the domain is available, registered, or if an error occurred.`,
				Example: `  # Check a single domain
  route53-checker check example.com

  # Check with .io TLD
  route53-checker check myapp.io

  # Check with custom timeout
  route53-checker --timeout 30s check example.com`,
				Args: cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Mock implementation for testing
					return nil
				},
			}
			cmd.AddCommand(checkCmd)

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			output := buf.String()

			if (err != nil) != tt.wantErr {
				t.Errorf("checkCmd.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.contains != "" && !strings.Contains(output, tt.contains) {
				t.Errorf("Expected output to contain %q, got:\n%s", tt.contains, output)
			}
		})
	}
}

func TestGlobalFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "timeout flag with valid duration",
			args:        []string{"--timeout", "30s", "check", "--help"},
			expectError: false,
		},
		{
			name:        "timeout flag with invalid duration",
			args:        []string{"--timeout", "invalid", "check", "--help"},
			expectError: true,
		},
		{
			name:        "region flag",
			args:        []string{"--region", "us-west-2", "check", "--help"},
			expectError: false,
		},
		{
			name:        "verbose flag short form",
			args:        []string{"-v", "check", "--help"},
			expectError: false,
		},
		{
			name:        "verbose flag long form",
			args:        []string{"--verbose", "check", "--help"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new root command for each test
			cmd := &cobra.Command{
				Use: "route53-checker",
			}

			// Add flags
			cmd.PersistentFlags().Duration("timeout", 10*time.Second, "Timeout for API requests")
			cmd.PersistentFlags().String("region", "", "AWS region (defaults to AWS SDK default)")
			cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

			// Add check command
			checkCmd := &cobra.Command{
				Use:  "check [domain]",
				Args: cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}
			cmd.AddCommand(checkCmd)

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestCommandStructure(t *testing.T) {
	// Test that the command structure is set up correctly
	cmd := &cobra.Command{
		Use: "route53-checker",
	}

	// Add flags
	cmd.PersistentFlags().Duration("timeout", 10*time.Second, "Timeout for API requests")
	cmd.PersistentFlags().String("region", "", "AWS region (defaults to AWS SDK default)")
	cmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	// Add check command
	checkCmd := &cobra.Command{
		Use:  "check [domain]",
		Args: cobra.ExactArgs(1),
	}
	cmd.AddCommand(checkCmd)

	// Test that flags are properly registered
	timeoutFlag := cmd.PersistentFlags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Error("timeout flag not found")
	}

	regionFlag := cmd.PersistentFlags().Lookup("region")
	if regionFlag == nil {
		t.Error("region flag not found")
	}

	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Error("verbose flag not found")
	}

	// Test that check command is properly added
	checkCommand := cmd.Commands()[0]
	if checkCommand.Use != "check [domain]" {
		t.Errorf("Expected check command Use to be 'check [domain]', got %q", checkCommand.Use)
	}
}

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected int
	}{
		{
			name:     "nil error",
			errMsg:   "",
			expected: int(customErrors.ExitSuccess),
		},
		{
			name:     "domain validation error",
			errMsg:   "domain validation failed: invalid format",
			expected: int(customErrors.ExitValidation),
		},
		{
			name:     "unsupported TLD error",
			errMsg:   "unsupported TLD: .xyz",
			expected: int(customErrors.ExitValidation),
		},
		{
			name:     "AWS credentials error",
			errMsg:   "NoCredentialsErr: no valid providers in chain",
			expected: int(customErrors.ExitAuthentication),
		},
		{
			name:     "AWS config error",
			errMsg:   "failed to initialize AWS configuration",
			expected: int(customErrors.ExitAuthentication),
		},
		{
			name:     "authorization error",
			errMsg:   "UnauthorizedOperation: insufficient permissions",
			expected: int(customErrors.ExitAuthorization),
		},
		{
			name:     "access denied error",
			errMsg:   "AccessDenied: user not authorized",
			expected: int(customErrors.ExitAuthorization),
		},
		{
			name:     "AWS API error",
			errMsg:   "AWS API call failed: service unavailable",
			expected: int(customErrors.ExitAPIError),
		},
		{
			name:     "throttling error",
			errMsg:   "TooManyRequests: rate limit exceeded",
			expected: int(customErrors.ExitAPIError),
		},
		{
			name:     "unknown error",
			errMsg:   "some unexpected error",
			expected: int(customErrors.ExitSystemError),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.errMsg != "" {
				err = fmt.Errorf(tt.errMsg)
			}

			result := int(customErrors.GetExitCode(err))
			if result != tt.expected {
				t.Errorf("customErrors.GetExitCode() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCreateFormatter(t *testing.T) {
	// Test with verbose = false
	verbose = false
	formatter := createFormatter()

	if formatter == nil {
		t.Error("createFormatter() returned nil")
	}

	// Test with verbose = true
	verbose = true
	verboseFormatter := createFormatter()

	if verboseFormatter == nil {
		t.Error("createFormatter() returned nil for verbose mode")
	}

	// Reset verbose to false for other tests
	verbose = false
}

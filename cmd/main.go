package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/abakermi/r53check/internal/aws"
	"github.com/abakermi/r53check/internal/domain"
	customErrors "github.com/abakermi/r53check/internal/errors"
	"github.com/abakermi/r53check/internal/output"

	awsSDK "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	timeout time.Duration
	region  string
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "route53-checker",
	Short: "Check domain availability in AWS Route 53",
	Long: `Route 53 Domain Availability Checker is a CLI tool for checking
domain availability within Amazon Route 53. It provides a fast, reliable
way to query Route 53 for domain registration status without navigating
the AWS console.

This tool is designed for developers, AWS administrators, and website
planners who need to verify domain availability for their projects.`,
	Example: `  # Check if example.com is available
  route53-checker check example.com

  # Check with custom timeout
  route53-checker --timeout 30s check example.com

  # Check with verbose output
  route53-checker --verbose check example.com`,
}

// checkCmd represents the check command
var checkCmd = &cobra.Command{
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
	RunE: runCheckCommand,
}

// bulkCmd represents the bulk command
var bulkCmd = &cobra.Command{
	Use:   "bulk [domains...]",
	Short: "Check availability for multiple domains",
	Long: `Check if multiple domains are available for registration in AWS Route 53.
	
You can provide domains as arguments or read from a file. The command will check
all domains concurrently and provide a summary of results.`,
	Example: `  # Check multiple domains
  r53check bulk example.com test.org myapp.io

  # Check domains from a file (one domain per line)
  r53check bulk --file domains.txt

  # Check with verbose output
  r53check --verbose bulk example.com test.org`,
	RunE: runBulkCommand,
}

var (
	// Bulk command flags
	domainsFile string
)

func init() {
	// Global flags
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 10*time.Second, "Timeout for API requests")
	rootCmd.PersistentFlags().StringVar(&region, "region", "", "AWS region (defaults to AWS SDK default)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add bulk command flags
	bulkCmd.Flags().StringVarP(&domainsFile, "file", "f", "", "Read domains from file (one domain per line)")

	// Add commands to root
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(bulkCmd)
}

func runCheckCommand(cmd *cobra.Command, args []string) error {
	domainName := args[0]

	// Set up signal handling for graceful cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		if verbose {
			fmt.Fprintln(os.Stderr, "\nReceived interrupt signal, cancelling request...")
		}
		cancel()
	}()

	// Create context with timeout
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeout)
	defer timeoutCancel()

	// Initialize components and run the check workflow
	exitCode, err := runDomainCheck(timeoutCtx, domainName)

	if err != nil {
		// Error has already been formatted and printed to stderr
		os.Exit(exitCode)
	}

	// Success case
	os.Exit(int(customErrors.ExitSuccess))
	return nil // This line should never be reached due to os.Exit above
}

// runDomainCheck encapsulates the complete domain checking workflow
func runDomainCheck(ctx context.Context, domainName string) (int, error) {
	// Initialize AWS configuration
	if verbose {
		fmt.Fprintf(os.Stderr, "Initializing AWS configuration...\n")
	}

	var awsConfig *awsSDK.Config
	var err error

	if region != "" {
		awsConfig, err = aws.NewConfigWithRegion(ctx, region)
		if verbose && err == nil {
			fmt.Fprintf(os.Stderr, "Using AWS region: %s\n", region)
		}
	} else {
		awsConfig, err = aws.NewConfig(ctx)
		if verbose && err == nil {
			fmt.Fprintf(os.Stderr, "Using default AWS region from configuration\n")
		}
	}

	if err != nil {
		exitCode := int(customErrors.GetExitCode(err))
		formatter := createFormatter()
		fmt.Fprintln(os.Stderr, formatter.FormatError(err))
		return exitCode, err
	}

	// Create AWS client
	if verbose {
		fmt.Fprintf(os.Stderr, "Creating AWS Route 53 Domains client...\n")
	}
	awsClient := aws.NewClient(awsConfig)

	// Create domain validator
	if verbose {
		fmt.Fprintf(os.Stderr, "Initializing domain validator...\n")
	}
	validator := domain.NewDomainValidator()

	// Create domain checker with timeout
	if verbose {
		fmt.Fprintf(os.Stderr, "Creating domain checker with %v timeout...\n", timeout)
	}
	checker := domain.NewDomainCheckerWithTimeout(validator, awsClient, timeout)

	// Create output formatter
	formatter := createFormatter()

	// Validate domain before making API call
	if verbose {
		fmt.Fprintf(os.Stderr, "Validating domain format: %s\n", domainName)
	}

	if err := validator.ValidateDomain(domainName); err != nil {
		exitCode := int(customErrors.GetExitCode(err))
		fmt.Fprintln(os.Stderr, formatter.FormatError(err))
		return exitCode, err
	}

	// Check domain availability
	if verbose {
		fmt.Fprintf(os.Stderr, "Checking domain availability with AWS Route 53...\n")
	}

	result, err := checker.CheckAvailability(ctx, domainName)
	if err != nil {
		exitCode := int(customErrors.GetExitCode(err))

		// Handle context cancellation gracefully
		if errors.Is(err, context.Canceled) {
			cancelErr := customErrors.NewSystemError("context", "Domain check was cancelled", err)
			fmt.Fprintln(os.Stderr, formatter.FormatError(cancelErr))
			return int(customErrors.ExitSystemError), cancelErr
		}

		// Handle timeout specifically
		if errors.Is(err, context.DeadlineExceeded) {
			timeoutErr := customErrors.NewAPIError("route53domains", "CheckDomainAvailability",
				fmt.Sprintf("domain check timed out after %v", timeout), err)
			fmt.Fprintln(os.Stderr, formatter.FormatError(timeoutErr))
			return int(customErrors.ExitAPIError), timeoutErr
		}

		fmt.Fprintln(os.Stderr, formatter.FormatError(err))
		return exitCode, err
	}

	// Display result to stdout
	fmt.Println(formatter.FormatResult(result))

	if verbose {
		fmt.Fprintf(os.Stderr, "Domain check completed successfully\n")
	}

	return int(customErrors.ExitSuccess), nil
}

// createFormatter creates an output formatter based on global flags
func createFormatter() output.Formatter {
	formatter := output.NewConsoleFormatter()
	formatter.SetVerbose(verbose)
	formatter.SetShowTimestamp(verbose)
	return formatter
}

func main() {
	// Execute the root command
	// Exit codes are handled within the command functions
	if err := rootCmd.Execute(); err != nil {
		// This should not normally be reached since we handle exits in runCheckCommand
		// But if it is reached, exit with a generic error code
		os.Exit(int(customErrors.ExitSystemError))
	}
}
func runBulkCommand(cmd *cobra.Command, args []string) error {
	var domains []string

	// Get domains from file or arguments
	if domainsFile != "" {
		fileDomains, err := readDomainsFromFile(domainsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading domains file: %v\n", err)
			os.Exit(int(customErrors.ExitValidation))
		}
		domains = fileDomains
	} else if len(args) > 0 {
		domains = args
	} else {
		fmt.Fprintf(os.Stderr, "Error: No domains provided. Use arguments or --file flag\n")
		os.Exit(int(customErrors.ExitValidation))
	}

	if len(domains) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No valid domains found\n")
		os.Exit(int(customErrors.ExitValidation))
	}

	// Set up signal handling for graceful cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		if verbose {
			fmt.Fprintf(os.Stderr, "\nReceived interrupt signal, cancelling bulk check...\n")
		}
		cancel()
	}()

	// Create context with timeout
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeout)
	defer timeoutCancel()

	// Run bulk domain check
	exitCode, err := runBulkDomainCheck(timeoutCtx, domains)

	if err != nil {
		// Error has already been formatted and printed to stderr
		os.Exit(exitCode)
	}

	// Success case
	os.Exit(int(customErrors.ExitSuccess))
	return nil // This line should never be reached due to os.Exit above
}

func runBulkDomainCheck(ctx context.Context, domains []string) (int, error) {
	// Initialize AWS configuration
	if verbose {
		fmt.Fprintf(os.Stderr, "Initializing AWS configuration...\n")
	}

	var awsConfig *awsSDK.Config
	var err error

	if region != "" {
		awsConfig, err = aws.NewConfigWithRegion(ctx, region)
		if verbose && err == nil {
			fmt.Fprintf(os.Stderr, "Using AWS region: %s\n", region)
		}
	} else {
		awsConfig, err = aws.NewConfig(ctx)
		if verbose && err == nil {
			fmt.Fprintf(os.Stderr, "Using default AWS region (us-east-1)\n")
		}
	}

	if err != nil {
		exitCode := int(customErrors.GetExitCode(err))
		formatter := createFormatter()
		fmt.Fprintln(os.Stderr, formatter.FormatError(err))
		return exitCode, err
	}

	// Create AWS client
	if verbose {
		fmt.Fprintf(os.Stderr, "Creating AWS Route 53 Domains client...\n")
	}
	awsClient := aws.NewClient(awsConfig)

	// Create domain validator
	if verbose {
		fmt.Fprintf(os.Stderr, "Initializing domain validator...\n")
	}
	validator := domain.NewDomainValidator()

	// Create domain checker with timeout
	if verbose {
		fmt.Fprintf(os.Stderr, "Creating domain checker with %v timeout...\n", timeout)
		fmt.Fprintf(os.Stderr, "Checking %d domains...\n", len(domains))
	}
	checker := domain.NewDomainCheckerWithTimeout(validator, awsClient, timeout)

	// Create output formatter
	formatter := createFormatter()

	// Check domain availability in bulk
	results, err := checker.CheckAvailabilityBulk(ctx, domains)
	if err != nil {
		exitCode := int(customErrors.GetExitCode(err))

		// Handle context cancellation gracefully
		if errors.Is(err, context.Canceled) {
			cancelErr := customErrors.NewSystemError("context", "Bulk domain check was cancelled", err)
			fmt.Fprintln(os.Stderr, formatter.FormatError(cancelErr))
			return int(customErrors.ExitSystemError), cancelErr
		}

		// Handle timeout specifically
		if errors.Is(err, context.DeadlineExceeded) {
			timeoutErr := customErrors.NewAPIError("route53domains", "CheckDomainAvailability",
				fmt.Sprintf("bulk domain check timed out after %v", timeout), err)
			fmt.Fprintln(os.Stderr, formatter.FormatError(timeoutErr))
			return int(customErrors.ExitAPIError), timeoutErr
		}

		fmt.Fprintln(os.Stderr, formatter.FormatError(err))
		return exitCode, err
	}

	// Display results to stdout
	fmt.Println(formatter.FormatBulkResults(results))

	if verbose {
		fmt.Fprintf(os.Stderr, "Bulk domain check completed successfully\n")
	}

	return int(customErrors.ExitSuccess), nil
}

func readDomainsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var domains []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			domains = append(domains, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return domains, nil
}

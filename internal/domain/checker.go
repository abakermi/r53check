package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	customErrors "github.com/abakermi/r53check/internal/errors"

	"github.com/aws/aws-sdk-go-v2/service/route53domains"
	"github.com/aws/aws-sdk-go-v2/service/route53domains/types"
)

// AvailabilityStatus represents the possible states of domain availability
type AvailabilityStatus string

const (
	StatusAvailable   AvailabilityStatus = "AVAILABLE"
	StatusUnavailable AvailabilityStatus = "UNAVAILABLE"
	StatusReserved    AvailabilityStatus = "RESERVED"
	StatusUnknown     AvailabilityStatus = "UNKNOWN"
)

// PricingInfo contains domain pricing information
type PricingInfo struct {
	RegistrationPrice *float64
	RenewalPrice      *float64
	TransferPrice     *float64
	Currency          string
}

// AvailabilityResult contains the result of a domain availability check
type AvailabilityResult struct {
	Domain    string
	Available bool
	Status    AvailabilityStatus
	Message   string
	CheckedAt time.Time
	Error     error
	Pricing   *PricingInfo // Optional pricing information
}

// Route53Client interface defines the methods needed for domain availability checking
type Route53Client interface {
	CheckDomainAvailability(ctx context.Context, domain string) (*route53domains.CheckDomainAvailabilityOutput, error)
	ListPrices(ctx context.Context, tld string) (*route53domains.ListPricesOutput, error)
}

// Checker interface defines the domain availability checking functionality
type Checker interface {
	CheckAvailability(ctx context.Context, domain string) (*AvailabilityResult, error)
	CheckAvailabilityWithPricing(ctx context.Context, domain string) (*AvailabilityResult, error)
}

// DomainChecker implements the Checker interface
type DomainChecker struct {
	validator Validator
	awsClient Route53Client
	timeout   time.Duration
}

// NewDomainChecker creates a new domain checker with the provided dependencies
func NewDomainChecker(validator Validator, awsClient Route53Client) *DomainChecker {
	return &DomainChecker{
		validator: validator,
		awsClient: awsClient,
		timeout:   10 * time.Second, // Default 10-second timeout
	}
}

// NewDomainCheckerWithTimeout creates a new domain checker with a custom timeout
func NewDomainCheckerWithTimeout(validator Validator, awsClient Route53Client, timeout time.Duration) *DomainChecker {
	return &DomainChecker{
		validator: validator,
		awsClient: awsClient,
		timeout:   timeout,
	}
}

// CheckAvailability checks if a domain is available for registration
func (c *DomainChecker) CheckAvailability(ctx context.Context, domain string) (*AvailabilityResult, error) {
	result := &AvailabilityResult{
		Domain:    domain,
		CheckedAt: time.Now(),
	}

	// Validate domain format first
	if err := c.validator.ValidateDomain(domain); err != nil {
		result.Error = err
		result.Status = StatusUnknown
		return result, err
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Call AWS API to check domain availability
	awsResult, err := c.awsClient.CheckDomainAvailability(timeoutCtx, domain)
	if err != nil {
		// Wrap the error if it's not already a custom error
		var customErr interface {
			GetCategory() customErrors.ErrorCategory
		}
		if !errors.As(err, &customErr) {
			err = customErrors.WrapAWSError(err, "route53domains", "CheckDomainAvailability")
		}

		result.Error = err
		result.Status = StatusUnknown
		return result, err
	}

	// Interpret AWS API response and map to business domain
	c.mapAWSResponse(awsResult, result)

	return result, nil
}

// CheckAvailabilityWithPricing checks domain availability and includes pricing information
func (c *DomainChecker) CheckAvailabilityWithPricing(ctx context.Context, domain string) (*AvailabilityResult, error) {
	// First check availability
	result, err := c.CheckAvailability(ctx, domain)
	if err != nil {
		return result, err
	}

	// If domain is available, get pricing information
	if result.Available {
		if err := c.addPricingInfo(ctx, domain, result); err != nil {
			// Don't fail the entire request if pricing fails, just log it
			// The availability check was successful
			if c.timeout > 0 {
				// Add a note about pricing failure in verbose mode
				result.Message += " (pricing information unavailable)"
			}
		}
	}

	return result, nil
}

// addPricingInfo fetches and adds pricing information to the result
func (c *DomainChecker) addPricingInfo(ctx context.Context, domain string, result *AvailabilityResult) error {
	// Extract TLD from domain
	tld := c.extractTLD(domain)
	if tld == "" {
		return fmt.Errorf("unable to extract TLD from domain: %s", domain)
	}

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Get pricing information for the TLD
	priceResult, err := c.awsClient.ListPrices(timeoutCtx, tld)
	if err != nil {
		return err
	}

	if priceResult != nil && len(priceResult.Prices) > 0 {
		pricing := &PricingInfo{
			Currency: "USD", // Route 53 pricing is in USD
		}

		// Extract pricing information from the first price entry
		price := priceResult.Prices[0]
		if price.RegistrationPrice != nil {
			regPrice := price.RegistrationPrice.Price
			pricing.RegistrationPrice = &regPrice
		}
		if price.RenewalPrice != nil {
			renewPrice := price.RenewalPrice.Price
			pricing.RenewalPrice = &renewPrice
		}
		if price.TransferPrice != nil {
			transferPrice := price.TransferPrice.Price
			pricing.TransferPrice = &transferPrice
		}

		result.Pricing = pricing
	}

	return nil
}

// extractTLD extracts the top-level domain from a full domain name
func (c *DomainChecker) extractTLD(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

// mapAWSResponse maps AWS API response to our business domain model
func (c *DomainChecker) mapAWSResponse(awsResult *route53domains.CheckDomainAvailabilityOutput, result *AvailabilityResult) {
	if awsResult == nil {
		result.Status = StatusUnknown
		result.Available = false
		result.Message = "No response from AWS API"
		return
	}

	// Map AWS availability status to our domain model
	switch awsResult.Availability {
	case types.DomainAvailabilityAvailable:
		result.Status = StatusAvailable
		result.Available = true
		result.Message = fmt.Sprintf("Domain %s is available for registration", result.Domain)

	case types.DomainAvailabilityUnavailable:
		result.Status = StatusUnavailable
		result.Available = false
		result.Message = fmt.Sprintf("Domain %s is already registered", result.Domain)

	case types.DomainAvailabilityReserved:
		result.Status = StatusReserved
		result.Available = false
		result.Message = fmt.Sprintf("Domain %s is reserved and cannot be registered", result.Domain)

	case types.DomainAvailabilityDontKnow:
		result.Status = StatusUnknown
		result.Available = false
		result.Message = fmt.Sprintf("Unable to determine availability for domain %s", result.Domain)

	default:
		result.Status = StatusUnknown
		result.Available = false
		result.Message = fmt.Sprintf("Unknown availability status for domain %s", result.Domain)
	}
}

// SetTimeout allows changing the timeout for API calls
func (c *DomainChecker) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// GetTimeout returns the current timeout setting
func (c *DomainChecker) GetTimeout() time.Duration {
	return c.timeout
}

// CheckAvailabilityBulk checks availability for multiple domains concurrently
func (c *DomainChecker) CheckAvailabilityBulk(ctx context.Context, domains []string) ([]*AvailabilityResult, error) {
	if len(domains) == 0 {
		return nil, customErrors.NewValidationError("", "domains", "no domains provided for bulk check", nil)
	}

	// Create a channel to collect results
	results := make([]*AvailabilityResult, len(domains))
	errors := make([]error, len(domains))

	// Use a semaphore to limit concurrent requests (AWS rate limiting)
	semaphore := make(chan struct{}, 5) // Limit to 5 concurrent requests

	// Use a wait group to wait for all goroutines
	var wg sync.WaitGroup

	for i, domain := range domains {
		wg.Add(1)
		go func(index int, domainName string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, err := c.CheckAvailability(ctx, domainName)
			results[index] = result
			errors[index] = err
		}(i, domain)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check if context was cancelled
	if ctx.Err() != nil {
		return results, customErrors.WrapSystemError("bulk-check", ctx.Err())
	}

	// Count successful results
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}

	// If no results were successful, return the first error
	if successCount == 0 && len(errors) > 0 {
		return results, errors[0]
	}

	return results, nil
}

// CheckAvailabilityBulkWithPricing checks availability for multiple domains concurrently with pricing
func (c *DomainChecker) CheckAvailabilityBulkWithPricing(ctx context.Context, domains []string) ([]*AvailabilityResult, error) {
	if len(domains) == 0 {
		return nil, customErrors.NewValidationError("", "domains", "no domains provided for bulk check", nil)
	}

	// Create a channel to collect results
	results := make([]*AvailabilityResult, len(domains))
	errors := make([]error, len(domains))

	// Use a semaphore to limit concurrent requests (AWS rate limiting)
	semaphore := make(chan struct{}, 5) // Limit to 5 concurrent requests

	// Use a wait group to wait for all goroutines
	var wg sync.WaitGroup

	for i, domain := range domains {
		wg.Add(1)
		go func(index int, domainName string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, err := c.CheckAvailabilityWithPricing(ctx, domainName)
			results[index] = result
			errors[index] = err
		}(i, domain)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check if context was cancelled
	if ctx.Err() != nil {
		return results, customErrors.WrapSystemError("bulk-check", ctx.Err())
	}

	// Count successful results
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}

	// If no results were successful, return the first error
	if successCount == 0 && len(errors) > 0 {
		return results, errors[0]
	}

	return results, nil
}

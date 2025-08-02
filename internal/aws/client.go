package aws

import (
	"context"

	"github.com/abakermi/r53check/internal/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53domains"
	"github.com/aws/aws-sdk-go-v2/service/route53domains/types"
)

// Route53Client interface defines the methods needed for domain availability checking
type Route53Client interface {
	CheckDomainAvailability(ctx context.Context, domain string) (*route53domains.CheckDomainAvailabilityOutput, error)
	ListPrices(ctx context.Context, tld string) (*route53domains.ListPricesOutput, error)
}

// Client wraps the AWS Route 53 Domains client
type Client struct {
	route53Client *route53domains.Client
}

// NewClient creates a new Route 53 client wrapper
func NewClient(cfg *aws.Config) *Client {
	return &Client{
		route53Client: route53domains.NewFromConfig(*cfg),
	}
}

// CheckDomainAvailability checks if a domain is available for registration
func (c *Client) CheckDomainAvailability(ctx context.Context, domain string) (*route53domains.CheckDomainAvailabilityOutput, error) {
	if domain == "" {
		return nil, errors.NewValidationError(domain, "domain", "domain cannot be empty", nil)
	}

	input := &route53domains.CheckDomainAvailabilityInput{
		DomainName: aws.String(domain),
	}

	result, err := c.route53Client.CheckDomainAvailability(ctx, input)
	if err != nil {
		return nil, errors.WrapAWSError(err, "route53domains", "CheckDomainAvailability")
	}

	return result, nil
}

// ListPrices gets pricing information for a specific TLD
func (c *Client) ListPrices(ctx context.Context, tld string) (*route53domains.ListPricesOutput, error) {
	if tld == "" {
		return nil, errors.NewValidationError(tld, "tld", "TLD cannot be empty", nil)
	}

	input := &route53domains.ListPricesInput{
		Tld: aws.String(tld),
	}

	result, err := c.route53Client.ListPrices(ctx, input)
	if err != nil {
		return nil, errors.WrapAWSError(err, "route53domains", "ListPrices")
	}

	return result, nil
}

// IsAvailable is a convenience method that returns true if the domain is available
func (c *Client) IsAvailable(ctx context.Context, domain string) (bool, error) {
	result, err := c.CheckDomainAvailability(ctx, domain)
	if err != nil {
		return false, err
	}

	return result.Availability == types.DomainAvailabilityAvailable, nil
}

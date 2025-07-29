package aws

import (
	"context"

	"github.com/abakermi/r53check/internal/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// Config holds AWS configuration settings
type Config struct {
	Region string
}

// NewConfig creates a new AWS configuration using the default credential chain
// This supports environment variables, shared credentials file, and IAM roles
// Defaults to us-east-1 region as Route 53 Domains API is only available there
func NewConfig(ctx context.Context) (*aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return nil, errors.WrapAWSError(err, "config", "LoadDefaultConfig")
	}

	return &cfg, nil
}

// NewConfigWithRegion creates a new AWS configuration with a specific region
func NewConfigWithRegion(ctx context.Context, region string) (*aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, errors.WrapAWSError(err, "config", "LoadDefaultConfig")
	}

	return &cfg, nil
}

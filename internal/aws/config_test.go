package aws

import (
	"context"
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	// Save original environment
	originalRegion := os.Getenv("AWS_REGION")
	originalAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	originalSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	// Clean up after test
	defer func() {
		os.Setenv("AWS_REGION", originalRegion)
		os.Setenv("AWS_ACCESS_KEY_ID", originalAccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", originalSecretKey)
	}()

	tests := []struct {
		name           string
		setupEnv       func()
		expectError    bool
		expectedRegion string
	}{
		{
			name: "default config without credentials",
			setupEnv: func() {
				// Clear AWS credentials to test default behavior
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
				os.Unsetenv("AWS_REGION")
			},
			expectError: false, // Should not error even without credentials
		},
		{
			name: "config with region from environment",
			setupEnv: func() {
				os.Setenv("AWS_REGION", "us-west-2")
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			},
			expectError:    false,
			expectedRegion: "us-west-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			cfg, err := NewConfig(context.Background())

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cfg == nil {
				t.Errorf("expected config to be created, got nil")
				return
			}

			if tt.expectedRegion != "" && cfg.Region != tt.expectedRegion {
				t.Errorf("expected region %s, got %s", tt.expectedRegion, cfg.Region)
			}
		})
	}
}

func TestNewConfigWithRegion(t *testing.T) {
	// Save original environment
	originalAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	originalSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	// Clean up after test
	defer func() {
		os.Setenv("AWS_ACCESS_KEY_ID", originalAccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", originalSecretKey)
	}()

	tests := []struct {
		name           string
		region         string
		setupEnv       func()
		expectError    bool
		expectedRegion string
	}{
		{
			name:   "config with specific region",
			region: "eu-west-1",
			setupEnv: func() {
				// Clear credentials to test default behavior
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			},
			expectError:    false,
			expectedRegion: "eu-west-1",
		},
		{
			name:   "config with us-east-1 region",
			region: "us-east-1",
			setupEnv: func() {
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			},
			expectError:    false,
			expectedRegion: "us-east-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			cfg, err := NewConfigWithRegion(context.Background(), tt.region)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if cfg == nil {
				t.Errorf("expected config to be created, got nil")
				return
			}

			if cfg.Region != tt.expectedRegion {
				t.Errorf("expected region %s, got %s", tt.expectedRegion, cfg.Region)
			}
		})
	}
}

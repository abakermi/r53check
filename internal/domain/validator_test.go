package domain

import (
	"errors"
	"strings"
	"testing"

	customErrors "github.com/abakermi/r53check/internal/errors"
)

func TestNewDomainValidator(t *testing.T) {
	validator := NewDomainValidator()

	if validator == nil {
		t.Fatal("NewDomainValidator returned nil")
	}

	if validator.supportedTLDs == nil {
		t.Fatal("supportedTLDs map is nil")
	}

	if validator.domainRegex == nil {
		t.Fatal("domainRegex is nil")
	}

	// Check that common TLDs are supported
	expectedTLDs := []string{"com", "net", "org", "io"}
	for _, tld := range expectedTLDs {
		if !validator.supportedTLDs[tld] {
			t.Errorf("Expected TLD %s to be supported", tld)
		}
	}
}

func TestValidateDomain_ValidDomains(t *testing.T) {
	validator := NewDomainValidator()

	validDomains := []string{
		"example.com",
		"test.net",
		"my-site.org",
		"api.io",
		"sub.domain.com",
		"a.co",
		"test123.info",
		"my-awesome-site.biz",
		"user.name",
		"site.me",
		"channel.tv",
		"short.cc",
		"mobile.mobi",
		"contact.tel",
		"business.asia",
		"american.us",
		"british.uk",
		"canadian.ca",
		"aussie.au",
		"german.de",
		"french.fr",
		"italian.it",
		"spanish.es",
		"dutch.nl",
		"belgian.be",
		"swiss.ch",
		"austrian.at",
		"swedish.se",
		"norwegian.no",
		"danish.dk",
		"finnish.fi",
		"polish.pl",
		"czech.cz",
		"russian.ru",
		"japanese.jp",
		"chinese.cn",
		"indian.in",
		"brazilian.br",
		"mexican.mx",
		"very-long-subdomain-name-that-is-still-valid.example.com",
		"123abc.com",
		"abc123.net",
	}

	for _, domain := range validDomains {
		t.Run(domain, func(t *testing.T) {
			err := validator.ValidateDomain(domain)
			if err != nil {
				t.Errorf("Expected domain %s to be valid, got error: %v", domain, err)
			}
		})
	}
}

func TestValidateDomain_InvalidDomains(t *testing.T) {
	validator := NewDomainValidator()

	testCases := []struct {
		domain      string
		expectedErr string
	}{
		{"", "domain cannot be empty"},
		{"   ", "domain cannot be empty"},
		{"ab", "domain name too short"},
		{"example", "invalid domain format"},
		{".com", "empty label in domain"},
		{"example.", "empty label in domain"},
		{"-example.com", "label cannot start or end with hyphen"},
		{"example-.com", "label cannot start or end with hyphen"},
		{"ex--ample.com", "consecutive hyphens not allowed"},
		{"example.invalidtld", "unsupported TLD"},
		{"example.xyz", "unsupported TLD"},
		{"123.com", "domain labels cannot be all numeric"},
		{"456.789.com", "domain labels cannot be all numeric"},
		{"example..com", "empty label in domain"},
		{"example.com.", "empty label in domain"},
		{strings.Repeat("a", 64) + ".com", "label too long"},
		{strings.Repeat("a", 250) + ".com", "domain name too long"},
	}

	for _, tc := range testCases {
		t.Run(tc.domain, func(t *testing.T) {
			err := validator.ValidateDomain(tc.domain)
			if err == nil {
				t.Errorf("Expected domain %s to be invalid", tc.domain)
				return
			}

			if !strings.Contains(err.Error(), tc.expectedErr) {
				t.Errorf("Expected error containing '%s', got: %v", tc.expectedErr, err)
			}
		})
	}
}

func TestValidateDomain_EdgeCases(t *testing.T) {
	validator := NewDomainValidator()

	testCases := []struct {
		name        string
		domain      string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Domain with uppercase letters",
			domain:      "EXAMPLE.COM",
			shouldError: false,
		},
		{
			name:        "Domain with mixed case",
			domain:      "ExAmPlE.CoM",
			shouldError: false,
		},
		{
			name:        "Domain with leading/trailing spaces",
			domain:      "  example.com  ",
			shouldError: false,
		},
		{
			name:        "Maximum length valid domain",
			domain:      strings.Repeat("a", 59) + "." + strings.Repeat("b", 59) + "." + strings.Repeat("c", 59) + "." + strings.Repeat("d", 59) + ".com",
			shouldError: false,
		},
		{
			name:        "Domain with maximum label length",
			domain:      strings.Repeat("a", 63) + ".com",
			shouldError: false,
		},
		{
			name:        "Domain with hyphen in middle",
			domain:      "my-awesome-domain.com",
			shouldError: false,
		},
		{
			name:        "Single character subdomain",
			domain:      "a.example.com",
			shouldError: false,
		},
		{
			name:        "Multiple subdomains",
			domain:      "api.v1.staging.example.com",
			shouldError: false,
		},
		{
			name:        "Domain starting with number",
			domain:      "1example.com",
			shouldError: false,
		},
		{
			name:        "Domain ending with number",
			domain:      "example1.com",
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateDomain(tc.domain)

			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error for domain %s", tc.domain)
					return
				}
				if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected domain %s to be valid, got error: %v", tc.domain, err)
				}
			}
		})
	}
}

func TestExtractTLD(t *testing.T) {
	validator := NewDomainValidator()

	testCases := []struct {
		domain      string
		expectedTLD string
	}{
		{"example.com", "com"},
		{"test.net", "net"},
		{"sub.domain.org", "org"},
		{"api.v1.example.io", "io"},
		{"single", ""},
		{"", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.domain, func(t *testing.T) {
			tld := validator.extractTLD(tc.domain)
			if tld != tc.expectedTLD {
				t.Errorf("Expected TLD %s for domain %s, got %s", tc.expectedTLD, tc.domain, tld)
			}
		})
	}
}

func TestValidateLabels(t *testing.T) {
	validator := NewDomainValidator()

	testCases := []struct {
		name        string
		domain      string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "Valid labels",
			domain:      "example.com",
			shouldError: false,
		},
		{
			name:        "Empty label",
			domain:      "example..com",
			shouldError: true,
			errorMsg:    "empty label",
		},
		{
			name:        "Label too long",
			domain:      strings.Repeat("a", 64) + ".com",
			shouldError: true,
			errorMsg:    "label too long",
		},
		{
			name:        "Label starts with hyphen",
			domain:      "-example.com",
			shouldError: true,
			errorMsg:    "cannot start or end with hyphen",
		},
		{
			name:        "Label ends with hyphen",
			domain:      "example-.com",
			shouldError: true,
			errorMsg:    "cannot start or end with hyphen",
		},
		{
			name:        "All numeric domain label",
			domain:      "123.com",
			shouldError: true,
			errorMsg:    "cannot be all numeric",
		},
		{
			name:        "Valid numeric TLD",
			domain:      "example.123",
			shouldError: false, // TLD can be numeric, but will fail TLD validation
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.validateLabels(tc.domain)

			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error for domain %s", tc.domain)
					return
				}
				if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected domain %s to have valid labels, got error: %v", tc.domain, err)
				}
			}
		})
	}
}

func TestIsAllNumeric(t *testing.T) {
	validator := NewDomainValidator()

	testCases := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"0", true},
		{"999", true},
		{"abc", false},
		{"123abc", false},
		{"abc123", false},
		{"12a34", false},
		{"", true}, // Empty string is considered all numeric
		{"12-34", false},
		{"12.34", false},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := validator.isAllNumeric(tc.input)
			if result != tc.expected {
				t.Errorf("Expected isAllNumeric(%s) to be %v, got %v", tc.input, tc.expected, result)
			}
		})
	}
}

func TestGetSupportedTLDs(t *testing.T) {
	validator := NewDomainValidator()

	tlds := validator.GetSupportedTLDs()

	if len(tlds) == 0 {
		t.Fatal("GetSupportedTLDs returned empty slice")
	}

	// Check that common TLDs are included
	expectedTLDs := []string{"com", "net", "org", "io"}
	tldMap := make(map[string]bool)
	for _, tld := range tlds {
		tldMap[tld] = true
	}

	for _, expectedTLD := range expectedTLDs {
		if !tldMap[expectedTLD] {
			t.Errorf("Expected TLD %s to be in supported TLDs list", expectedTLD)
		}
	}
}

func BenchmarkValidateDomain(b *testing.B) {
	validator := NewDomainValidator()
	domain := "example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateDomain(domain)
	}
}

func BenchmarkValidateDomainComplex(b *testing.B) {
	validator := NewDomainValidator()
	domain := "very-long-subdomain-name-with-multiple-parts.api.v1.staging.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateDomain(domain)
	}
}
func TestValidateDomain_ErrorTypes(t *testing.T) {
	validator := NewDomainValidator()

	testCases := []struct {
		name          string
		domain        string
		expectedType  interface{}
		expectedField string
	}{
		{
			name:          "empty domain",
			domain:        "",
			expectedType:  &customErrors.ValidationError{},
			expectedField: "domain",
		},
		{
			name:          "domain too long",
			domain:        strings.Repeat("a", 254) + ".com",
			expectedType:  &customErrors.ValidationError{},
			expectedField: "length",
		},
		{
			name:          "domain too short",
			domain:        "ab",
			expectedType:  &customErrors.ValidationError{},
			expectedField: "length",
		},
		{
			name:          "consecutive hyphens",
			domain:        "ex--ample.com",
			expectedType:  &customErrors.ValidationError{},
			expectedField: "format",
		},
		{
			name:          "invalid domain format",
			domain:        "example",
			expectedType:  &customErrors.ValidationError{},
			expectedField: "format",
		},
		{
			name:          "unsupported TLD",
			domain:        "example.invalidtld",
			expectedType:  &customErrors.ValidationError{},
			expectedField: "tld",
		},
		{
			name:          "unable to extract TLD",
			domain:        "example",
			expectedType:  &customErrors.ValidationError{},
			expectedField: "format", // This will fail format validation first
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateDomain(tc.domain)

			if err == nil {
				t.Errorf("Expected error for domain %s", tc.domain)
				return
			}

			// Check if it's the right type of error
			var validationErr *customErrors.ValidationError
			if !errors.As(err, &validationErr) {
				t.Errorf("expected ValidationError, got %T", err)
				return
			}

			// Check if it's a ValidationError with the right field
			if errors.As(err, &validationErr) {
				if validationErr.Field != tc.expectedField {
					t.Errorf("expected field %s, got %s", tc.expectedField, validationErr.Field)
				}
				if validationErr.Domain != tc.domain {
					t.Errorf("expected domain %s, got %s", tc.domain, validationErr.Domain)
				}
				if validationErr.GetCategory() != customErrors.CategoryValidation {
					t.Errorf("expected category %v, got %v", customErrors.CategoryValidation, validationErr.GetCategory())
				}
			}
		})
	}
}

func TestValidateDomain_ErrorWrapping(t *testing.T) {
	validator := NewDomainValidator()

	// Test that label validation errors are properly wrapped
	domain := strings.Repeat("a", 64) + ".com" // Label too long
	err := validator.ValidateDomain(domain)

	if err == nil {
		t.Fatal("Expected error for domain with long label")
	}

	// Should be wrapped as ValidationError
	var validationErr *customErrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Errorf("expected ValidationError, got %T", err)
		return
	}

	// Should have the domain set
	if validationErr.Domain != domain {
		t.Errorf("expected domain %s, got %s", domain, validationErr.Domain)
	}

	// Should have format field (since it's wrapped from validateLabels)
	if validationErr.Field != "format" {
		t.Errorf("expected field 'format', got %s", validationErr.Field)
	}
}

func TestValidateDomain_ErrorUnwrapping(t *testing.T) {
	validator := NewDomainValidator()

	domain := ""
	err := validator.ValidateDomain(domain)

	if err == nil {
		t.Fatal("Expected error for empty domain")
	}

	// Test that we can unwrap to get the base error
	var baseErr *customErrors.BaseError
	if !errors.As(err, &baseErr) {
		t.Errorf("expected to unwrap to BaseError, got %T", err)
		return
	}

	if baseErr.Category != customErrors.CategoryValidation {
		t.Errorf("expected category %v, got %v", customErrors.CategoryValidation, baseErr.Category)
	}
}

package domain

import (
	"fmt"
	"regexp"
	"github.com/abakermi/r53check/internal/errors"
	"strings"
)

// Validator defines the interface for domain validation
type Validator interface {
	ValidateDomain(domain string) error
}

// DomainValidator implements domain format and TLD validation
type DomainValidator struct {
	supportedTLDs map[string]bool
	domainRegex   *regexp.Regexp
}

// NewDomainValidator creates a new domain validator with supported TLDs
func NewDomainValidator() *DomainValidator {
	// Supported TLDs as specified in requirements
	supportedTLDs := map[string]bool{
		"com":  true,
		"net":  true,
		"org":  true,
		"io":   true,
		"co":   true,
		"info": true,
		"biz":  true,
		"name": true,
		"me":   true,
		"tv":   true,
		"cc":   true,
		"ws":   true,
		"mobi": true,
		"tel":  true,
		"asia": true,
		"us":   true,
		"uk":   true,
		"ca":   true,
		"au":   true,
		"de":   true,
		"fr":   true,
		"it":   true,
		"es":   true,
		"nl":   true,
		"be":   true,
		"ch":   true,
		"at":   true,
		"se":   true,
		"no":   true,
		"dk":   true,
		"fi":   true,
		"pl":   true,
		"cz":   true,
		"ru":   true,
		"jp":   true,
		"cn":   true,
		"in":   true,
		"br":   true,
		"mx":   true,
	}

	// Domain regex pattern:
	// - Must start with alphanumeric character
	// - Can contain hyphens but not at start/end of labels
	// - Each label must be 1-63 characters
	// - Total length must be <= 253 characters
	// - Must have at least one dot separating domain and TLD
	// - No consecutive hyphens allowed
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$`)

	return &DomainValidator{
		supportedTLDs: supportedTLDs,
		domainRegex:   domainRegex,
	}
}

// ValidateDomain validates domain format and checks if TLD is supported
func (v *DomainValidator) ValidateDomain(domain string) error {
	if domain == "" {
		return errors.NewValidationError(domain, "domain", "domain cannot be empty", nil)
	}

	// Convert to lowercase for consistent processing
	originalDomain := domain
	domain = strings.ToLower(strings.TrimSpace(domain))

	// Check for empty after trimming
	if domain == "" {
		return errors.NewValidationError(originalDomain, "domain", "domain cannot be empty", nil)
	}

	// Check overall length (RFC 1035)
	if len(domain) > 253 {
		return errors.NewValidationError(domain, "length", "domain name too long: maximum 253 characters allowed", nil)
	}

	// Check minimum length
	if len(domain) < 3 {
		return errors.NewValidationError(domain, "length", "domain name too short: minimum 3 characters required", nil)
	}

	// Additional validation checks first (more specific errors)
	if err := v.validateLabels(domain); err != nil {
		return errors.WrapValidationError(domain, err)
	}

	// Check for consecutive hyphens
	if strings.Contains(domain, "--") {
		return errors.NewValidationError(domain, "format", "consecutive hyphens not allowed in domain", nil)
	}

	// Validate domain format using regex
	if !v.domainRegex.MatchString(domain) {
		return errors.NewValidationError(domain, "format", "invalid domain format", nil)
	}

	// Extract and validate TLD
	tld := v.extractTLD(domain)
	if tld == "" {
		return errors.NewValidationError(domain, "tld", "unable to extract TLD from domain", nil)
	}

	if !v.supportedTLDs[tld] {
		return errors.NewValidationError(domain, "tld", fmt.Sprintf("unsupported TLD: .%s", tld), nil)
	}

	return nil
}

// extractTLD extracts the top-level domain from a domain name
func (v *DomainValidator) extractTLD(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-1]
}

// validateLabels performs additional validation on domain labels
func (v *DomainValidator) validateLabels(domain string) error {
	labels := strings.Split(domain, ".")

	for i, label := range labels {
		if label == "" {
			return fmt.Errorf("empty label in domain")
		}

		if len(label) > 63 {
			return fmt.Errorf("label too long: %s (maximum 63 characters per label)", label)
		}

		// Labels cannot start or end with hyphen
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("label cannot start or end with hyphen: %s", label)
		}

		// Domain name (not TLD) cannot be all numeric
		if i < len(labels)-1 && v.isAllNumeric(label) {
			return fmt.Errorf("domain labels cannot be all numeric: %s", label)
		}
	}

	return nil
}

// isAllNumeric checks if a string contains only numeric characters
func (v *DomainValidator) isAllNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// GetSupportedTLDs returns a slice of all supported TLDs
func (v *DomainValidator) GetSupportedTLDs() []string {
	tlds := make([]string, 0, len(v.supportedTLDs))
	for tld := range v.supportedTLDs {
		tlds = append(tlds, tld)
	}
	return tlds
}

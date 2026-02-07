package configimport

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	MaxImportDenylistEntries   = 1000
	MaxImportAllowlistEntries  = 1000
	MaxImportRewriteEntries    = 500
	MaxImportBlocklistEntries  = 100

	maxDomainLength = 253
)

var domainRegex = regexp.MustCompile(`^(\*\.)?([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// Validate checks the parsed config against CRD constraints. It returns an
// error describing all validation failures, or nil if the config is valid.
func Validate(cfg *ProfileConfigJSON) error {
	var errs []string

	// Validate list entry counts.
	if len(cfg.Denylist) > MaxImportDenylistEntries {
		errs = append(errs, fmt.Sprintf("imported denylist contains %d entries, maximum allowed is %d", len(cfg.Denylist), MaxImportDenylistEntries))
	}
	if len(cfg.Allowlist) > MaxImportAllowlistEntries {
		errs = append(errs, fmt.Sprintf("imported allowlist contains %d entries, maximum allowed is %d", len(cfg.Allowlist), MaxImportAllowlistEntries))
	}
	if len(cfg.Rewrites) > MaxImportRewriteEntries {
		errs = append(errs, fmt.Sprintf("imported rewrites contains %d entries, maximum allowed is %d", len(cfg.Rewrites), MaxImportRewriteEntries))
	}
	if cfg.Privacy != nil && len(cfg.Privacy.Blocklists) > MaxImportBlocklistEntries {
		errs = append(errs, fmt.Sprintf("imported blocklists contains %d entries, maximum allowed is %d", len(cfg.Privacy.Blocklists), MaxImportBlocklistEntries))
	}

	// Validate denylist domains.
	for i, entry := range cfg.Denylist {
		if err := validateDomain(entry.Domain); err != nil {
			errs = append(errs, fmt.Sprintf("denylist[%d]: %s", i, err))
		}
	}

	// Validate allowlist domains.
	for i, entry := range cfg.Allowlist {
		if err := validateDomain(entry.Domain); err != nil {
			errs = append(errs, fmt.Sprintf("allowlist[%d]: %s", i, err))
		}
	}

	// Validate rewrite entries.
	for i, entry := range cfg.Rewrites {
		if err := validateDomain(entry.From); err != nil {
			errs = append(errs, fmt.Sprintf("rewrites[%d].from: %s", i, err))
		}
		if entry.To == "" {
			errs = append(errs, fmt.Sprintf("rewrites[%d].to: must not be empty", i))
		} else if len(entry.To) > maxDomainLength {
			errs = append(errs, fmt.Sprintf("rewrites[%d].to: value %q exceeds maximum length of %d characters", i, entry.To, maxDomainLength))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

func validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain must not be empty")
	}
	if len(domain) > maxDomainLength {
		return fmt.Errorf("domain %q exceeds maximum length of %d characters", domain, maxDomainLength)
	}
	if !domainRegex.MatchString(domain) {
		return fmt.Errorf("domain %q is not a valid domain name", domain)
	}
	return nil
}

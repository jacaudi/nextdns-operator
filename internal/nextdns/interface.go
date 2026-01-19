package nextdns

import (
	"context"

	"github.com/amalucelli/nextdns-go/nextdns"
)

// ClientInterface defines the interface for NextDNS operations
// This allows for mocking in tests
type ClientInterface interface {
	// Profile operations
	CreateProfile(ctx context.Context, name string) (string, error)
	GetProfile(ctx context.Context, profileID string) (*nextdns.Profile, error)
	UpdateProfile(ctx context.Context, profileID, name string) error
	DeleteProfile(ctx context.Context, profileID string) error

	// Security operations
	UpdateSecurity(ctx context.Context, profileID string, config *SecurityConfig) error
	GetSecurity(ctx context.Context, profileID string) (*nextdns.Security, error)

	// Privacy operations
	UpdatePrivacy(ctx context.Context, profileID string, config *PrivacyConfig) error
	GetPrivacy(ctx context.Context, profileID string) (*nextdns.Privacy, error)
	SyncPrivacyBlocklists(ctx context.Context, profileID string, blocklists []string) error
	SyncPrivacyNatives(ctx context.Context, profileID string, natives []string) error

	// Parental control operations
	UpdateParentalControl(ctx context.Context, profileID string, config *ParentalControlConfig) error
	GetParentalControl(ctx context.Context, profileID string) (*nextdns.ParentalControl, error)

	// List operations
	SyncDenylist(ctx context.Context, profileID string, domains []string) error
	SyncAllowlist(ctx context.Context, profileID string, domains []string) error
	SyncSecurityTLDs(ctx context.Context, profileID string, tlds []string) error
	GetDenylist(ctx context.Context, profileID string) ([]*nextdns.Denylist, error)
	GetAllowlist(ctx context.Context, profileID string) ([]*nextdns.Allowlist, error)
	GetSecurityTLDs(ctx context.Context, profileID string) ([]*nextdns.SecurityTlds, error)

	// Settings operations
	UpdateSettings(ctx context.Context, profileID string, config *SettingsConfig) error
}

// Ensure Client implements ClientInterface
var _ ClientInterface = (*Client)(nil)

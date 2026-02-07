package configimport

import (
	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

// MergeIntoSpec merges imported profile configuration into the spec, using the
// spec as the authority. Explicit spec fields always take precedence; imported
// values only fill in nil/empty fields. Lists are appended with deduplication.
func MergeIntoSpec(spec *nextdnsv1alpha1.NextDNSProfileSpec, imported *ProfileConfigJSON) {
	if imported == nil {
		return
	}

	mergeSecurity(spec, imported.Security)
	mergePrivacy(spec, imported.Privacy)
	mergeParentalControl(spec, imported.ParentalControl)
	mergeSettings(spec, imported.Settings)
	mergeDenylist(spec, imported.Denylist)
	mergeAllowlist(spec, imported.Allowlist)
	mergeRewrites(spec, imported.Rewrites)
}

// mergeBoolPtr sets *dst to src only when *dst is nil. This ensures that any
// value already present in the spec is never overwritten by the import.
func mergeBoolPtr(dst **bool, src *bool) {
	if *dst == nil {
		*dst = src
	}
}

// mergeStringField sets *dst to src only when *dst is empty.
func mergeStringField(dst *string, src string) {
	if *dst == "" {
		*dst = src
	}
}

func mergeSecurity(spec *nextdnsv1alpha1.NextDNSProfileSpec, src *SecurityJSON) {
	if src == nil {
		return
	}
	if spec.Security == nil {
		spec.Security = &nextdnsv1alpha1.SecuritySpec{}
	}
	s := spec.Security

	mergeBoolPtr(&s.AIThreatDetection, src.AIThreatDetection)
	mergeBoolPtr(&s.GoogleSafeBrowsing, src.GoogleSafeBrowsing)
	mergeBoolPtr(&s.Cryptojacking, src.Cryptojacking)
	mergeBoolPtr(&s.DNSRebinding, src.DNSRebinding)
	mergeBoolPtr(&s.IDNHomographs, src.IDNHomographs)
	mergeBoolPtr(&s.Typosquatting, src.Typosquatting)
	mergeBoolPtr(&s.DGA, src.DGA)
	mergeBoolPtr(&s.NRD, src.NRD)
	mergeBoolPtr(&s.DDNS, src.DDNS)
	mergeBoolPtr(&s.Parking, src.Parking)
	mergeBoolPtr(&s.CSAM, src.CSAM)

	// Merge ThreatIntelligenceFeeds
	if len(src.ThreatIntelligenceFeeds) > 0 {
		seen := make(map[string]struct{}, len(s.ThreatIntelligenceFeeds))
		for _, f := range s.ThreatIntelligenceFeeds {
			seen[f] = struct{}{}
		}
		for _, f := range src.ThreatIntelligenceFeeds {
			if _, exists := seen[f]; !exists {
				s.ThreatIntelligenceFeeds = append(s.ThreatIntelligenceFeeds, f)
			}
		}
	}
}

func mergePrivacy(spec *nextdnsv1alpha1.NextDNSProfileSpec, src *PrivacyJSON) {
	if src == nil {
		return
	}
	if spec.Privacy == nil {
		spec.Privacy = &nextdnsv1alpha1.PrivacySpec{}
	}
	p := spec.Privacy

	mergeBoolPtr(&p.DisguisedTrackers, src.DisguisedTrackers)
	mergeBoolPtr(&p.AllowAffiliate, src.AllowAffiliate)

	// Merge blocklists with dedup by ID
	existing := make(map[string]struct{}, len(p.Blocklists))
	for _, b := range p.Blocklists {
		existing[b.ID] = struct{}{}
	}
	for _, b := range src.Blocklists {
		if _, found := existing[b.ID]; !found {
			p.Blocklists = append(p.Blocklists, nextdnsv1alpha1.BlocklistEntry{
				ID:     b.ID,
				Active: b.Active,
			})
		}
	}

	// Merge natives with dedup by ID
	existingNatives := make(map[string]struct{}, len(p.Natives))
	for _, n := range p.Natives {
		existingNatives[n.ID] = struct{}{}
	}
	for _, n := range src.Natives {
		if _, found := existingNatives[n.ID]; !found {
			p.Natives = append(p.Natives, nextdnsv1alpha1.NativeEntry{
				ID:     n.ID,
				Active: n.Active,
			})
		}
	}
}

func mergeParentalControl(spec *nextdnsv1alpha1.NextDNSProfileSpec, src *ParentalControlJSON) {
	if src == nil {
		return
	}
	if spec.ParentalControl == nil {
		spec.ParentalControl = &nextdnsv1alpha1.ParentalControlSpec{}
	}
	pc := spec.ParentalControl

	mergeBoolPtr(&pc.SafeSearch, src.SafeSearch)
	mergeBoolPtr(&pc.YouTubeRestrictedMode, src.YouTubeRestrictedMode)

	// Merge categories with dedup by ID
	existing := make(map[string]struct{}, len(pc.Categories))
	for _, c := range pc.Categories {
		existing[c.ID] = struct{}{}
	}
	for _, c := range src.Categories {
		if _, found := existing[c.ID]; !found {
			pc.Categories = append(pc.Categories, nextdnsv1alpha1.CategoryEntry{
				ID:     c.ID,
				Active: c.Active,
			})
		}
	}

	// Merge services with dedup by ID
	existingServices := make(map[string]struct{}, len(pc.Services))
	for _, s := range pc.Services {
		existingServices[s.ID] = struct{}{}
	}
	for _, s := range src.Services {
		if _, found := existingServices[s.ID]; !found {
			pc.Services = append(pc.Services, nextdnsv1alpha1.ServiceEntry{
				ID:     s.ID,
				Active: s.Active,
			})
		}
	}
}

func mergeSettings(spec *nextdnsv1alpha1.NextDNSProfileSpec, src *SettingsJSON) {
	if src == nil {
		return
	}
	if spec.Settings == nil {
		spec.Settings = &nextdnsv1alpha1.SettingsSpec{}
	}
	s := spec.Settings

	mergeBoolPtr(&s.Web3, src.Web3)

	// Merge Logs
	if src.Logs != nil {
		if s.Logs == nil {
			s.Logs = &nextdnsv1alpha1.LogsSpec{}
		}
		mergeBoolPtr(&s.Logs.Enabled, src.Logs.Enabled)
		mergeBoolPtr(&s.Logs.LogClientsIPs, src.Logs.LogClientsIPs)
		mergeBoolPtr(&s.Logs.LogDomains, src.Logs.LogDomains)
		mergeStringField(&s.Logs.Retention, src.Logs.Retention)
	}

	// Merge BlockPage
	if src.BlockPage != nil {
		if s.BlockPage == nil {
			s.BlockPage = &nextdnsv1alpha1.BlockPageSpec{}
		}
		mergeBoolPtr(&s.BlockPage.Enabled, src.BlockPage.Enabled)
	}

	// Merge Performance
	if src.Performance != nil {
		if s.Performance == nil {
			s.Performance = &nextdnsv1alpha1.PerformanceSpec{}
		}
		mergeBoolPtr(&s.Performance.ECS, src.Performance.ECS)
		mergeBoolPtr(&s.Performance.CacheBoost, src.Performance.CacheBoost)
		mergeBoolPtr(&s.Performance.CNAMEFlattening, src.Performance.CNAMEFlattening)
	}
}

func mergeDenylist(spec *nextdnsv1alpha1.NextDNSProfileSpec, src []DomainEntryJSON) {
	if len(src) == 0 {
		return
	}

	existing := make(map[string]struct{}, len(spec.Denylist))
	for _, d := range spec.Denylist {
		existing[d.Domain] = struct{}{}
	}
	for _, d := range src {
		if _, found := existing[d.Domain]; !found {
			spec.Denylist = append(spec.Denylist, nextdnsv1alpha1.DomainEntry{
				Domain: d.Domain,
				Active: d.Active,
			})
		}
	}
}

func mergeAllowlist(spec *nextdnsv1alpha1.NextDNSProfileSpec, src []DomainEntryJSON) {
	if len(src) == 0 {
		return
	}

	existing := make(map[string]struct{}, len(spec.Allowlist))
	for _, d := range spec.Allowlist {
		existing[d.Domain] = struct{}{}
	}
	for _, d := range src {
		if _, found := existing[d.Domain]; !found {
			spec.Allowlist = append(spec.Allowlist, nextdnsv1alpha1.DomainEntry{
				Domain: d.Domain,
				Active: d.Active,
			})
		}
	}
}

func mergeRewrites(spec *nextdnsv1alpha1.NextDNSProfileSpec, src []RewriteEntryJSON) {
	if len(src) == 0 {
		return
	}

	existing := make(map[string]struct{}, len(spec.Rewrites))
	for _, r := range spec.Rewrites {
		existing[r.From] = struct{}{}
	}
	for _, r := range src {
		if _, found := existing[r.From]; !found {
			spec.Rewrites = append(spec.Rewrites, nextdnsv1alpha1.RewriteEntry{
				From:   r.From,
				To:     r.To,
				Active: r.Active,
			})
		}
	}
}

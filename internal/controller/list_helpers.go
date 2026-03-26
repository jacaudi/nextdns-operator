package controller

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

// formatProfileRefs formats profile references for display in status messages.
func formatProfileRefs(refs []nextdnsv1alpha1.ResourceReference) string {
	var names []string
	for _, ref := range refs {
		if ref.Namespace != "" {
			names = append(names, fmt.Sprintf("%s/%s", ref.Namespace, ref.Name))
		} else {
			names = append(names, ref.Name)
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(names, ", "))
}

// setListConditions sets the standard Valid, InUse, and DeletionBlocked conditions
// on a list resource's status conditions. The itemLabel describes what is being
// counted (e.g. "domains" or "TLDs") for human-readable messages.
func setListConditions(conditions *[]metav1.Condition, count, refCount int, itemLabel string) {
	// Valid condition
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    "Valid",
		Status:  metav1.ConditionTrue,
		Reason:  "AllDomainsValid",
		Message: fmt.Sprintf("All %d %s are valid", count, itemLabel),
	})

	// InUse condition
	if refCount > 0 {
		meta.SetStatusCondition(conditions, metav1.Condition{
			Type:    "InUse",
			Status:  metav1.ConditionTrue,
			Reason:  "ReferencedByProfiles",
			Message: fmt.Sprintf("Used by %d profile(s)", refCount),
		})
	} else {
		meta.SetStatusCondition(conditions, metav1.Condition{
			Type:    "InUse",
			Status:  metav1.ConditionFalse,
			Reason:  "NotReferenced",
			Message: "Not used by any profiles",
		})
	}

	// Clear DeletionBlocked if it was set
	if refCount == 0 {
		meta.RemoveStatusCondition(conditions, "DeletionBlocked")
	}
}

// countActiveDomains counts the number of DomainEntry items where Active is nil or true.
func countActiveDomains(domains []nextdnsv1alpha1.DomainEntry) int {
	count := 0
	for _, domain := range domains {
		if domain.Active == nil || *domain.Active {
			count++
		}
	}
	return count
}

// countActiveTLDs counts the number of TLDEntry items where Active is nil or true.
func countActiveTLDs(tlds []nextdnsv1alpha1.TLDEntry) int {
	count := 0
	for _, tld := range tlds {
		if tld.Active == nil || *tld.Active {
			count++
		}
	}
	return count
}

// setDeletionBlockedCondition sets the DeletionBlocked condition on a list resource.
func setDeletionBlockedCondition(conditions *[]metav1.Condition, profileRefs []nextdnsv1alpha1.ResourceReference) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:    "DeletionBlocked",
		Status:  metav1.ConditionTrue,
		Reason:  "InUseByProfiles",
		Message: fmt.Sprintf("Cannot delete: used by profiles %s. Remove references first.", formatProfileRefs(profileRefs)),
	})
}

// findRefsForList iterates over all profiles and returns those that reference a given
// list resource. The extractRefs function should return the relevant ListReference
// slice from a profile's spec (e.g. AllowlistRefs, DenylistRefs, or TLDListRefs).
func findRefsForList(
	profiles []nextdnsv1alpha1.NextDNSProfile,
	listName, listNamespace string,
	extractRefs func(*nextdnsv1alpha1.NextDNSProfileSpec) []nextdnsv1alpha1.ListReference,
) []nextdnsv1alpha1.ResourceReference {
	var refs []nextdnsv1alpha1.ResourceReference

	for _, profile := range profiles {
		for _, ref := range extractRefs(&profile.Spec) {
			namespace := ref.Namespace
			if namespace == "" {
				namespace = profile.Namespace
			}

			if ref.Name == listName && namespace == listNamespace {
				refs = append(refs, nextdnsv1alpha1.ResourceReference{
					Name:      profile.Name,
					Namespace: profile.Namespace,
				})
				break
			}
		}
	}

	return refs
}

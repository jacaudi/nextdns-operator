package configimport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

const defaultKey = "config.json"

// ImportResult contains the parsed config and ConfigMap metadata.
type ImportResult struct {
	Config          *ProfileConfigJSON
	ResourceVersion string
	Warnings        []string
}

// ReadAndParse reads a ConfigMap referenced by ConfigImportRef, parses the
// JSON content, and returns the result. Returns an error if the ConfigMap
// or key is not found, or if the JSON is invalid.
func ReadAndParse(ctx context.Context, c client.Client, namespace string, ref *nextdnsv1alpha1.ConfigImportRef) (*ImportResult, error) {
	key := ref.Key
	if key == "" {
		key = defaultKey
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, cm); err != nil {
		return nil, fmt.Errorf("failed to get import ConfigMap %s/%s: %w", namespace, ref.Name, err)
	}

	jsonData, ok := cm.Data[key]
	if !ok {
		return nil, fmt.Errorf("key %q not found in ConfigMap %s/%s", key, namespace, ref.Name)
	}

	// Try strict parsing first to detect unknown fields.
	dec := json.NewDecoder(strings.NewReader(jsonData))
	dec.DisallowUnknownFields()
	var strictCfg ProfileConfigJSON
	var warnings []string
	if err := dec.Decode(&strictCfg); err != nil {
		if strings.Contains(err.Error(), "unknown field") {
			warnings = append(warnings, fmt.Sprintf("import JSON contains unknown fields that will be ignored: %s", err.Error()))
			// Fall back to lenient parsing.
			var cfg ProfileConfigJSON
			if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
				return nil, fmt.Errorf("failed to parse JSON from ConfigMap %s/%s key %q: %w", namespace, ref.Name, key, err)
			}
			if err := Validate(&cfg); err != nil {
				return nil, fmt.Errorf("ConfigMap %s/%s key %q: %w", namespace, ref.Name, key, err)
			}
			return &ImportResult{Config: &cfg, ResourceVersion: cm.ResourceVersion, Warnings: warnings}, nil
		}
		return nil, fmt.Errorf("failed to parse JSON from ConfigMap %s/%s key %q: %w", namespace, ref.Name, key, err)
	}

	if err := Validate(&strictCfg); err != nil {
		return nil, fmt.Errorf("ConfigMap %s/%s key %q: %w", namespace, ref.Name, key, err)
	}

	return &ImportResult{
		Config:          &strictCfg,
		ResourceVersion: cm.ResourceVersion,
	}, nil
}

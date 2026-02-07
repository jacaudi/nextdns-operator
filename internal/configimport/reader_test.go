package configimport

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

func testScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = nextdnsv1alpha1.AddToScheme(scheme)
	return scheme
}

func TestReadAndParse_Success(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-config",
			Namespace:       "default",
			ResourceVersion: "12345",
		},
		Data: map[string]string{
			"config.json": `{
				"security": {"aiThreatDetection": true},
				"denylist": [{"domain": "bad.com", "active": true}]
			}`,
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(testScheme()).
		WithObjects(cm).
		Build()

	ref := &nextdnsv1alpha1.ConfigImportRef{
		Name: "my-config",
		Key:  "config.json",
	}

	result, err := ReadAndParse(context.Background(), client, "default", ref)
	require.NoError(t, err)
	require.NotNil(t, result.Config)
	assert.Equal(t, "12345", result.ResourceVersion)
	assert.NotNil(t, result.Config.Security)
	assert.Len(t, result.Config.Denylist, 1)
}

func TestReadAndParse_CustomKey(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-config",
			Namespace:       "default",
			ResourceVersion: "99",
		},
		Data: map[string]string{
			"profile.json": `{"security": {"nrd": true}}`,
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(testScheme()).
		WithObjects(cm).
		Build()

	ref := &nextdnsv1alpha1.ConfigImportRef{
		Name: "my-config",
		Key:  "profile.json",
	}

	result, err := ReadAndParse(context.Background(), client, "default", ref)
	require.NoError(t, err)
	assert.Equal(t, true, *result.Config.Security.NRD)
}

func TestReadAndParse_DefaultKey(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-config",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		Data: map[string]string{
			"config.json": `{"security": {"csam": true}}`,
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(testScheme()).
		WithObjects(cm).
		Build()

	ref := &nextdnsv1alpha1.ConfigImportRef{
		Name: "my-config",
	}

	result, err := ReadAndParse(context.Background(), client, "default", ref)
	require.NoError(t, err)
	assert.Equal(t, true, *result.Config.Security.CSAM)
}

func TestReadAndParse_ConfigMapNotFound(t *testing.T) {
	client := fake.NewClientBuilder().
		WithScheme(testScheme()).
		Build()

	ref := &nextdnsv1alpha1.ConfigImportRef{
		Name: "nonexistent",
	}

	_, err := ReadAndParse(context.Background(), client, "default", ref)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReadAndParse_KeyNotFound(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"other-key": `{}`,
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(testScheme()).
		WithObjects(cm).
		Build()

	ref := &nextdnsv1alpha1.ConfigImportRef{
		Name: "my-config",
		Key:  "config.json",
	}

	_, err := ReadAndParse(context.Background(), client, "default", ref)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key \"config.json\" not found")
}

func TestReadAndParse_InvalidJSON(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"config.json": `{invalid json`,
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(testScheme()).
		WithObjects(cm).
		Build()

	ref := &nextdnsv1alpha1.ConfigImportRef{
		Name: "my-config",
	}

	_, err := ReadAndParse(context.Background(), client, "default", ref)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
	"github.com/jacaudi/nextdns-operator/internal/coredns"
)

const (
	// CoreDNSFinalizerName is the finalizer used by the CoreDNS controller
	CoreDNSFinalizerName = "nextdns.io/coredns-finalizer"

	// ConditionTypeProfileResolved indicates the referenced profile is resolved
	ConditionTypeProfileResolved = "ProfileResolved"

	// CorefileKey is the key in the ConfigMap for the Corefile
	CorefileKey = "Corefile"

	// maxResourceNameLength is the maximum length for Kubernetes resource names
	maxResourceNameLength = 63
)

// NextDNSCoreDNSReconciler reconciles a NextDNSCoreDNS object
type NextDNSCoreDNSReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	SyncPeriod time.Duration
}

// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnscorednses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnscorednses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnscorednses/finalizers,verbs=update
// +kubebuilder:rbac:groups=nextdns.io,resources=nextdnsprofiles,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *NextDNSCoreDNSReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the NextDNSCoreDNS instance
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{}
	if err := r.Get(ctx, req.NamespacedName, coreDNS); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("NextDNSCoreDNS resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get NextDNSCoreDNS")
		return ctrl.Result{}, err
	}

	// Check if the resource is being deleted
	if !coreDNS.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, coreDNS)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(coreDNS, CoreDNSFinalizerName) {
		logger.Info("Adding finalizer to NextDNSCoreDNS")
		controllerutil.AddFinalizer(coreDNS, CoreDNSFinalizerName)
		if err := r.Update(ctx, coreDNS); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	// Resolve the referenced NextDNSProfile
	profile, err := r.resolveProfile(ctx, coreDNS)
	if err != nil {
		logger.Error(err, "Failed to resolve NextDNSProfile reference")
		r.setCondition(coreDNS, ConditionTypeProfileResolved, metav1.ConditionFalse, "ProfileNotFound", err.Error())
		r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionFalse, "ProfileNotResolved", "Failed to resolve profile reference")
		coreDNS.Status.Ready = false
		if updateErr := r.Status().Update(ctx, coreDNS); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Check if profile is ready
	if !r.isProfileReady(profile) {
		logger.Info("Referenced NextDNSProfile is not ready", "profile", profile.Name)
		r.setCondition(coreDNS, ConditionTypeProfileResolved, metav1.ConditionFalse, "ProfileNotReady", "Referenced profile is not in Ready state")
		r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionFalse, "ProfileNotReady", "Waiting for profile to become ready")
		coreDNS.Status.Ready = false
		if updateErr := r.Status().Update(ctx, coreDNS); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Profile is resolved and ready
	r.setCondition(coreDNS, ConditionTypeProfileResolved, metav1.ConditionTrue, "ProfileResolved", "Referenced profile found and ready")

	// Store profile information in status
	coreDNS.Status.ProfileID = profile.Status.ProfileID
	coreDNS.Status.Fingerprint = profile.Status.Fingerprint

	// Reconcile the ConfigMap with Corefile
	if err := r.reconcileConfigMap(ctx, coreDNS, profile); err != nil {
		logger.Error(err, "Failed to reconcile ConfigMap")
		r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionFalse, "ConfigMapFailed", err.Error())
		coreDNS.Status.Ready = false
		if updateErr := r.Status().Update(ctx, coreDNS); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Reconcile the workload (Deployment or DaemonSet)
	if err := r.reconcileWorkload(ctx, coreDNS); err != nil {
		logger.Error(err, "Failed to reconcile workload")
		r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionFalse, "WorkloadFailed", err.Error())
		coreDNS.Status.Ready = false
		if updateErr := r.Status().Update(ctx, coreDNS); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Reconcile the Service
	if err := r.reconcileService(ctx, coreDNS); err != nil {
		logger.Error(err, "Failed to reconcile Service")
		r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionFalse, "ServiceFailed", err.Error())
		coreDNS.Status.Ready = false
		if updateErr := r.Status().Update(ctx, coreDNS); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Update status with current state
	if err := r.updateStatus(ctx, coreDNS, profile); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled NextDNSCoreDNS",
		"profileID", coreDNS.Status.ProfileID,
		"dnsIP", coreDNS.Status.DNSIP,
		"ready", coreDNS.Status.Ready)

	// Schedule next sync with jitter
	syncInterval := CalculateSyncInterval(r.SyncPeriod)
	if syncInterval > 0 {
		logger.V(1).Info("Scheduling next sync", "interval", syncInterval)
	}

	return ctrl.Result{RequeueAfter: syncInterval}, nil
}

// handleDeletion handles the deletion of a NextDNSCoreDNS resource
func (r *NextDNSCoreDNSReconciler) handleDeletion(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(coreDNS, CoreDNSFinalizerName) {
		logger.Info("Handling deletion of NextDNSCoreDNS")

		// Resources will be cleaned up automatically via OwnerReferences
		// Just remove the finalizer
		controllerutil.RemoveFinalizer(coreDNS, CoreDNSFinalizerName)
		if err := r.Update(ctx, coreDNS); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// resolveProfile fetches the referenced NextDNSProfile
func (r *NextDNSCoreDNSReconciler) resolveProfile(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) (*nextdnsv1alpha1.NextDNSProfile, error) {
	profileRef := coreDNS.Spec.ProfileRef
	ns := profileRef.Namespace
	if ns == "" {
		ns = coreDNS.Namespace
	}

	profile := &nextdnsv1alpha1.NextDNSProfile{}
	if err := r.Get(ctx, types.NamespacedName{Name: profileRef.Name, Namespace: ns}, profile); err != nil {
		return nil, fmt.Errorf("failed to get NextDNSProfile %s/%s: %w", ns, profileRef.Name, err)
	}

	return profile, nil
}

// isProfileReady checks if the profile has a Ready condition set to True
func (r *NextDNSCoreDNSReconciler) isProfileReady(profile *nextdnsv1alpha1.NextDNSProfile) bool {
	for _, cond := range profile.Status.Conditions {
		if cond.Type == ConditionTypeReady && cond.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

// reconcileConfigMap creates or updates the ConfigMap containing the Corefile
func (r *NextDNSCoreDNSReconciler) reconcileConfigMap(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, profile *nextdnsv1alpha1.NextDNSProfile) error {
	logger := log.FromContext(ctx)
	resourceName := r.getResourceName(coreDNS, profile)

	// Build Corefile configuration
	cfg := r.buildCorefileConfig(coreDNS, profile)
	corefileContent := coredns.GenerateCorefile(cfg)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: coreDNS.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		// Set labels
		configMap.Labels = r.buildLabels(coreDNS, profile)

		// Set data
		configMap.Data = map[string]string{
			CorefileKey: corefileContent,
		}

		// Set owner reference
		return controllerutil.SetControllerReference(coreDNS, configMap, r.Scheme)
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile ConfigMap: %w", err)
	}

	if op != controllerutil.OperationResultNone {
		logger.Info("ConfigMap reconciled", "operation", op, "name", resourceName)
	}

	return nil
}

// buildCorefileConfig builds the CorefileConfig from the CR spec
func (r *NextDNSCoreDNSReconciler) buildCorefileConfig(coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, profile *nextdnsv1alpha1.NextDNSProfile) *coredns.CorefileConfig {
	cfg := &coredns.CorefileConfig{
		ProfileID:       profile.Status.ProfileID,
		PrimaryProtocol: coredns.ProtocolDoT, // default
		CacheTTL:        3600,                // default
		LoggingEnabled:  false,
		MetricsEnabled:  true,
	}

	// Override primary protocol if specified
	if coreDNS.Spec.Upstream != nil {
		cfg.PrimaryProtocol = string(coreDNS.Spec.Upstream.Primary)
		if coreDNS.Spec.Upstream.Fallback != nil {
			cfg.FallbackProtocol = string(*coreDNS.Spec.Upstream.Fallback)
		}
	}

	// Override cache settings if specified
	if coreDNS.Spec.Cache != nil {
		if coreDNS.Spec.Cache.Enabled != nil && !*coreDNS.Spec.Cache.Enabled {
			cfg.CacheTTL = 0
		} else if coreDNS.Spec.Cache.SuccessTTL != nil {
			cfg.CacheTTL = *coreDNS.Spec.Cache.SuccessTTL
		}
	}

	// Override logging settings if specified
	if coreDNS.Spec.Logging != nil && coreDNS.Spec.Logging.Enabled != nil {
		cfg.LoggingEnabled = *coreDNS.Spec.Logging.Enabled
	}

	// Override metrics settings if specified
	if coreDNS.Spec.Metrics != nil && coreDNS.Spec.Metrics.Enabled != nil {
		cfg.MetricsEnabled = *coreDNS.Spec.Metrics.Enabled
	}

	return cfg
}

// reconcileWorkload dispatches to Deployment or DaemonSet reconciliation based on mode
func (r *NextDNSCoreDNSReconciler) reconcileWorkload(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) error {
	mode := nextdnsv1alpha1.DeploymentModeDeployment // default
	if coreDNS.Spec.Deployment != nil && coreDNS.Spec.Deployment.Mode != "" {
		mode = coreDNS.Spec.Deployment.Mode
	}

	switch mode {
	case nextdnsv1alpha1.DeploymentModeDaemonSet:
		// Clean up any existing Deployment before creating DaemonSet
		if err := r.cleanupDeployment(ctx, coreDNS); err != nil {
			return err
		}
		return r.reconcileDaemonSet(ctx, coreDNS)
	default:
		// Clean up any existing DaemonSet before creating Deployment
		if err := r.cleanupDaemonSet(ctx, coreDNS); err != nil {
			return err
		}
		return r.reconcileDeployment(ctx, coreDNS)
	}
}

// cleanupDeployment removes any existing Deployment when switching to DaemonSet mode
func (r *NextDNSCoreDNSReconciler) cleanupDeployment(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) error {
	profile, err := r.resolveProfile(ctx, coreDNS)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil // Profile not found, nothing to clean up
		}
		return err // Unexpected error, propagate it
	}

	resourceName := r.getResourceName(coreDNS, profile)
	deployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: coreDNS.Namespace}, deployment)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return r.Delete(ctx, deployment)
}

// cleanupDaemonSet removes any existing DaemonSet when switching to Deployment mode
func (r *NextDNSCoreDNSReconciler) cleanupDaemonSet(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) error {
	profile, err := r.resolveProfile(ctx, coreDNS)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil // Profile not found, nothing to clean up
		}
		return err // Unexpected error, propagate it
	}

	resourceName := r.getResourceName(coreDNS, profile)
	daemonSet := &appsv1.DaemonSet{}
	err = r.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: coreDNS.Namespace}, daemonSet)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return r.Delete(ctx, daemonSet)
}

// reconcileDeployment creates or updates the CoreDNS Deployment
func (r *NextDNSCoreDNSReconciler) reconcileDeployment(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) error {
	logger := log.FromContext(ctx)

	profile, err := r.resolveProfile(ctx, coreDNS)
	if err != nil {
		return err
	}

	resourceName := r.getResourceName(coreDNS, profile)
	labels := r.buildLabels(coreDNS, profile)

	// Determine replicas
	replicas := int32(2) // default
	if coreDNS.Spec.Deployment != nil && coreDNS.Spec.Deployment.Replicas != nil {
		replicas = *coreDNS.Spec.Deployment.Replicas
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: coreDNS.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Labels = labels
		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: r.buildPodAnnotations(coreDNS),
				},
				Spec: r.buildPodSpec(coreDNS, resourceName),
			},
		}

		return controllerutil.SetControllerReference(coreDNS, deployment, r.Scheme)
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile Deployment: %w", err)
	}

	if op != controllerutil.OperationResultNone {
		logger.Info("Deployment reconciled", "operation", op, "name", resourceName)
	}

	return nil
}

// reconcileDaemonSet creates or updates the CoreDNS DaemonSet
func (r *NextDNSCoreDNSReconciler) reconcileDaemonSet(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) error {
	logger := log.FromContext(ctx)

	profile, err := r.resolveProfile(ctx, coreDNS)
	if err != nil {
		return err
	}

	resourceName := r.getResourceName(coreDNS, profile)
	labels := r.buildLabels(coreDNS, profile)

	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: coreDNS.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, daemonSet, func() error {
		daemonSet.Labels = labels
		daemonSet.Spec = appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: r.buildPodAnnotations(coreDNS),
				},
				Spec: r.buildPodSpec(coreDNS, resourceName),
			},
		}

		return controllerutil.SetControllerReference(coreDNS, daemonSet, r.Scheme)
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile DaemonSet: %w", err)
	}

	if op != controllerutil.OperationResultNone {
		logger.Info("DaemonSet reconciled", "operation", op, "name", resourceName)
	}

	return nil
}

// buildPodSpec builds the pod spec for CoreDNS containers
func (r *NextDNSCoreDNSReconciler) buildPodSpec(coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, configMapName string) corev1.PodSpec {
	// Determine image
	image := coredns.DefaultCoreDNSImage
	if coreDNS.Spec.Deployment != nil && coreDNS.Spec.Deployment.Image != "" {
		image = coreDNS.Spec.Deployment.Image
	}

	// Build security context
	allowPrivilegeEscalation := false
	readOnlyRootFilesystem := true
	runAsNonRoot := true

	podSpec := corev1.PodSpec{
		// ServiceAccountName is intentionally left empty to use the namespace default
		// ServiceAccount. The controller does not create a dedicated ServiceAccount,
		// so hardcoding one would cause pods to fail scheduling. Users can configure
		// a dedicated ServiceAccount via pod security policies if needed.
		Containers: []corev1.Container{
			{
				Name:  "coredns",
				Image: image,
				Args:  []string{"-conf", "/etc/coredns/Corefile"},
				Ports: []corev1.ContainerPort{
					{
						Name:          "dns",
						ContainerPort: 53,
						Protocol:      corev1.ProtocolUDP,
					},
					{
						Name:          "dns-tcp",
						ContainerPort: 53,
						Protocol:      corev1.ProtocolTCP,
					},
					{
						Name:          "metrics",
						ContainerPort: 9153,
						Protocol:      corev1.ProtocolTCP,
					},
				},
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: &allowPrivilegeEscalation,
					ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
					Capabilities: &corev1.Capabilities{
						Add:  []corev1.Capability{"NET_BIND_SERVICE"},
						Drop: []corev1.Capability{"ALL"},
					},
				},
				LivenessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path:   "/health",
							Port:   intstr.FromInt(8080),
							Scheme: corev1.URISchemeHTTP,
						},
					},
					InitialDelaySeconds: 5,
					PeriodSeconds:       10,
					TimeoutSeconds:      5,
					SuccessThreshold:    1,
					FailureThreshold:    3,
				},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path:   "/ready",
							Port:   intstr.FromInt(8181),
							Scheme: corev1.URISchemeHTTP,
						},
					},
					InitialDelaySeconds: 5,
					PeriodSeconds:       10,
					TimeoutSeconds:      5,
					SuccessThreshold:    1,
					FailureThreshold:    3,
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "config-volume",
						MountPath: "/etc/coredns",
						ReadOnly:  true,
					},
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: "config-volume",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: configMapName,
						},
						Items: []corev1.KeyToPath{
							{
								Key:  CorefileKey,
								Path: "Corefile",
							},
						},
					},
				},
			},
		},
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot: &runAsNonRoot,
		},
	}

	// Apply deployment-specific settings
	if coreDNS.Spec.Deployment != nil {
		if coreDNS.Spec.Deployment.NodeSelector != nil {
			podSpec.NodeSelector = coreDNS.Spec.Deployment.NodeSelector
		}
		if coreDNS.Spec.Deployment.Affinity != nil {
			podSpec.Affinity = coreDNS.Spec.Deployment.Affinity
		}
		if coreDNS.Spec.Deployment.Tolerations != nil {
			podSpec.Tolerations = coreDNS.Spec.Deployment.Tolerations
		}
		if coreDNS.Spec.Deployment.Resources != nil {
			podSpec.Containers[0].Resources = *coreDNS.Spec.Deployment.Resources
		}
	}

	return podSpec
}

// reconcileService creates or updates the CoreDNS Service
func (r *NextDNSCoreDNSReconciler) reconcileService(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) error {
	logger := log.FromContext(ctx)

	profile, err := r.resolveProfile(ctx, coreDNS)
	if err != nil {
		return err
	}

	serviceName := r.getServiceName(coreDNS, profile)
	labels := r.buildLabels(coreDNS, profile)

	// Determine service type
	serviceType := corev1.ServiceTypeClusterIP // default
	if coreDNS.Spec.Service != nil && coreDNS.Spec.Service.Type != "" {
		switch coreDNS.Spec.Service.Type {
		case nextdnsv1alpha1.ServiceTypeLoadBalancer:
			serviceType = corev1.ServiceTypeLoadBalancer
		case nextdnsv1alpha1.ServiceTypeNodePort:
			serviceType = corev1.ServiceTypeNodePort
		}
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: coreDNS.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		service.Labels = labels

		// Apply additional annotations if specified
		if coreDNS.Spec.Service != nil && coreDNS.Spec.Service.Annotations != nil {
			if service.Annotations == nil {
				service.Annotations = make(map[string]string)
			}
			for k, v := range coreDNS.Spec.Service.Annotations {
				service.Annotations[k] = v
			}
		}

		service.Spec = corev1.ServiceSpec{
			Type:     serviceType,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "dns",
					Port:       53,
					TargetPort: intstr.FromInt(53),
					Protocol:   corev1.ProtocolUDP,
				},
				{
					Name:       "dns-tcp",
					Port:       53,
					TargetPort: intstr.FromInt(53),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "metrics",
					Port:       9153,
					TargetPort: intstr.FromInt(9153),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		}

		// Apply LoadBalancer IP if specified
		if serviceType == corev1.ServiceTypeLoadBalancer && coreDNS.Spec.Service != nil && coreDNS.Spec.Service.LoadBalancerIP != "" {
			service.Spec.LoadBalancerIP = coreDNS.Spec.Service.LoadBalancerIP
		}

		return controllerutil.SetControllerReference(coreDNS, service, r.Scheme)
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile Service: %w", err)
	}

	if op != controllerutil.OperationResultNone {
		logger.Info("Service reconciled", "operation", op, "name", serviceName)
	}

	return nil
}

// buildLabels returns standard Kubernetes labels for the CoreDNS resources
func (r *NextDNSCoreDNSReconciler) buildLabels(coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, profile *nextdnsv1alpha1.NextDNSProfile) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "coredns",
		"app.kubernetes.io/instance":   coreDNS.Name,
		"app.kubernetes.io/component":  "dns",
		"app.kubernetes.io/managed-by": "nextdns-operator",
		"nextdns.io/profile-id":        profile.Status.ProfileID,
	}
}

// buildPodAnnotations returns annotations for CoreDNS pods
func (r *NextDNSCoreDNSReconciler) buildPodAnnotations(coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) map[string]string {
	if coreDNS.Spec.Deployment == nil || coreDNS.Spec.Deployment.PodAnnotations == nil {
		return nil
	}
	// Return a copy to avoid modifying the original
	annotations := make(map[string]string, len(coreDNS.Spec.Deployment.PodAnnotations))
	for k, v := range coreDNS.Spec.Deployment.PodAnnotations {
		annotations[k] = v
	}
	return annotations
}

// getResourceName returns the name for managed resources.
// Names are truncated with a hash suffix if they exceed 63 characters.
func (r *NextDNSCoreDNSReconciler) getResourceName(coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, profile *nextdnsv1alpha1.NextDNSProfile) string {
	name := fmt.Sprintf("%s-%s-coredns", coreDNS.Name, profile.Status.ProfileID)
	if len(name) <= maxResourceNameLength {
		return name
	}
	// Truncate and add hash for uniqueness
	hash := sha256.Sum256([]byte(name))
	hashSuffix := hex.EncodeToString(hash[:3]) // 6 hex chars
	// Leave room for dash and hash: 63 - 1 - 6 = 56
	return name[:56] + "-" + hashSuffix
}

// getServiceName returns the service name, respecting nameOverride
func (r *NextDNSCoreDNSReconciler) getServiceName(coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, profile *nextdnsv1alpha1.NextDNSProfile) string {
	if coreDNS.Spec.Service != nil && coreDNS.Spec.Service.NameOverride != "" {
		return coreDNS.Spec.Service.NameOverride
	}
	return r.getResourceName(coreDNS, profile)
}

// updateStatus updates the status of the NextDNSCoreDNS resource
func (r *NextDNSCoreDNSReconciler) updateStatus(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, profile *nextdnsv1alpha1.NextDNSProfile) error {
	// Get upstream endpoint URL
	primaryProtocol := coredns.ProtocolDoT
	if coreDNS.Spec.Upstream != nil {
		primaryProtocol = string(coreDNS.Spec.Upstream.Primary)
	}
	upstreamURL := coredns.GetUpstreamEndpoint(profile.Status.ProfileID, primaryProtocol)

	// Update upstream status
	coreDNS.Status.Upstream = &nextdnsv1alpha1.UpstreamStatus{
		URL: upstreamURL,
	}

	// Get service to determine DNS IP
	serviceName := r.getServiceName(coreDNS, profile)
	service := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: coreDNS.Namespace}, service); err == nil {
		// Build endpoints from service
		var endpoints []nextdnsv1alpha1.DNSEndpoint

		switch service.Spec.Type {
		case corev1.ServiceTypeLoadBalancer:
			// Get LoadBalancer IP
			for _, ingress := range service.Status.LoadBalancer.Ingress {
				ip := ingress.IP
				if ip == "" {
					ip = ingress.Hostname
				}
				if ip != "" {
					endpoints = append(endpoints,
						nextdnsv1alpha1.DNSEndpoint{IP: ip, Port: 53, Protocol: "UDP"},
						nextdnsv1alpha1.DNSEndpoint{IP: ip, Port: 53, Protocol: "TCP"},
					)
					coreDNS.Status.DNSIP = ip
				}
			}
		case corev1.ServiceTypeNodePort:
			// For NodePort, we'd need node IPs which requires more work
			// Just use ClusterIP for now
			if service.Spec.ClusterIP != "" && service.Spec.ClusterIP != "None" {
				endpoints = append(endpoints,
					nextdnsv1alpha1.DNSEndpoint{IP: service.Spec.ClusterIP, Port: 53, Protocol: "UDP"},
					nextdnsv1alpha1.DNSEndpoint{IP: service.Spec.ClusterIP, Port: 53, Protocol: "TCP"},
				)
				coreDNS.Status.DNSIP = service.Spec.ClusterIP
			}
		default:
			// ClusterIP
			if service.Spec.ClusterIP != "" && service.Spec.ClusterIP != "None" {
				endpoints = append(endpoints,
					nextdnsv1alpha1.DNSEndpoint{IP: service.Spec.ClusterIP, Port: 53, Protocol: "UDP"},
					nextdnsv1alpha1.DNSEndpoint{IP: service.Spec.ClusterIP, Port: 53, Protocol: "TCP"},
				)
				coreDNS.Status.DNSIP = service.Spec.ClusterIP
			}
		}

		coreDNS.Status.Endpoints = endpoints
	}

	// Get replica status
	mode := nextdnsv1alpha1.DeploymentModeDeployment
	if coreDNS.Spec.Deployment != nil && coreDNS.Spec.Deployment.Mode != "" {
		mode = coreDNS.Spec.Deployment.Mode
	}

	resourceName := r.getResourceName(coreDNS, profile)
	var ready bool

	switch mode {
	case nextdnsv1alpha1.DeploymentModeDaemonSet:
		daemonSet := &appsv1.DaemonSet{}
		if err := r.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: coreDNS.Namespace}, daemonSet); err == nil {
			coreDNS.Status.Replicas = &nextdnsv1alpha1.ReplicaStatus{
				Desired:   daemonSet.Status.DesiredNumberScheduled,
				Ready:     daemonSet.Status.NumberReady,
				Available: daemonSet.Status.NumberAvailable,
			}
			ready = daemonSet.Status.NumberReady > 0 && daemonSet.Status.NumberReady == daemonSet.Status.DesiredNumberScheduled
		}
	default:
		deployment := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: coreDNS.Namespace}, deployment); err == nil {
			desired := int32(1)
			if deployment.Spec.Replicas != nil {
				desired = *deployment.Spec.Replicas
			}
			coreDNS.Status.Replicas = &nextdnsv1alpha1.ReplicaStatus{
				Desired:   desired,
				Ready:     deployment.Status.ReadyReplicas,
				Available: deployment.Status.AvailableReplicas,
			}
			ready = deployment.Status.ReadyReplicas > 0 && deployment.Status.ReadyReplicas >= desired
		}
	}

	// Update ready status
	coreDNS.Status.Ready = ready
	if ready {
		r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionTrue, "AllResourcesReady", "All CoreDNS resources are ready")
	} else {
		r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionFalse, "ResourcesNotReady", "Waiting for workload to become ready")
	}

	// Update metadata
	now := metav1.Now()
	coreDNS.Status.LastUpdated = &now
	coreDNS.Status.ObservedGeneration = coreDNS.Generation

	return r.Status().Update(ctx, coreDNS)
}

// setCondition sets a condition on the NextDNSCoreDNS resource
func (r *NextDNSCoreDNSReconciler) setCondition(coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, conditionType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&coreDNS.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		ObservedGeneration: coreDNS.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// findCoreDNSForProfile returns reconcile requests for NextDNSCoreDNS resources referencing the profile
func (r *NextDNSCoreDNSReconciler) findCoreDNSForProfile(ctx context.Context, obj client.Object) []reconcile.Request {
	profile, ok := obj.(*nextdnsv1alpha1.NextDNSProfile)
	if !ok {
		return nil
	}

	var coreDNSList nextdnsv1alpha1.NextDNSCoreDNSList
	if err := r.List(ctx, &coreDNSList); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, coreDNS := range coreDNSList.Items {
		refNs := coreDNS.Spec.ProfileRef.Namespace
		if refNs == "" {
			refNs = coreDNS.Namespace
		}
		if coreDNS.Spec.ProfileRef.Name == profile.Name && refNs == profile.Namespace {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      coreDNS.Name,
					Namespace: coreDNS.Namespace,
				},
			})
		}
	}
	return requests
}

// SetupWithManager sets up the controller with the Manager
func (r *NextDNSCoreDNSReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nextdnsv1alpha1.NextDNSCoreDNS{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Watches(
			&nextdnsv1alpha1.NextDNSProfile{},
			handler.EnqueueRequestsFromMapFunc(r.findCoreDNSForProfile),
		).
		Complete(r)
}

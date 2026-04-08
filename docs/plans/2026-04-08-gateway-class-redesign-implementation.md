# GatewayClass Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **For Claude:** REQUIRED EXECUTION WORKFLOW (follow in order):
> 1. `superpowers:using-git-worktrees` -- Isolate work in a dedicated worktree
> 2. `superpowers:subagent-driven-development` -- Dispatch a fresh subagent per task
> 3. `superpowers:test-driven-development` -- All subagents use TDD
> 4. `superpowers:verification-before-completion` -- Verify all tests pass per task
> 5. `superpowers:requesting-code-review` -- Code review after each task (built in)
> 6. After all tasks: comprehensive code review on full diff from branch point (automatic)
> 7. `superpowers:finishing-a-development-branch` -- Complete the branch
>
> Skills carry their own model and effort settings. Do not override them.

**Goal:** Remove the operator's self-created GatewayClass and make it a pure consumer of external gateway controllers, with per-CR class name override and operator-level default.

**Architecture:** The operator stops creating a GatewayClass with a phantom controller name. Instead, `GatewayConfig` gains a `gatewayClassName` field that references an externally-managed GatewayClass. The operator resolves the class name from the CR field or falls back to an operator-level default. A new cleanup function deletes orphaned gateway resources when `spec.gateway` is removed.

**Tech Stack:** Go, controller-runtime, gateway-api v1/v1alpha2, kubebuilder markers, Helm

**Design doc:** `docs/plans/2026-04-08-gateway-class-redesign-design.md`

---

### Task 1: Add GatewayClassName field to API types

**Files:**
- Modify: `api/v1alpha1/nextdnscoredns_types.go:218-229`
- Modify: `api/v1alpha1/zz_generated.deepcopy.go` (auto-generated)

- [ ] **Step 1: Add GatewayClassName field to GatewayConfig**

In `api/v1alpha1/nextdnscoredns_types.go`, add the `GatewayClassName` field to `GatewayConfig` at line 219 (before `Addresses`):

```go
// GatewayConfig configures Gateway API resources for DNS traffic exposure
type GatewayConfig struct {
	// GatewayClassName specifies which GatewayClass to use for the Gateway.
	// This must reference a GatewayClass managed by an external controller
	// (e.g., Envoy Gateway, Cilium, Istio).
	// If omitted, uses the operator's default gateway class name.
	// +optional
	GatewayClassName *string `json:"gatewayClassName,omitempty"`

	// Addresses specifies the IP addresses for the Gateway.
	// These are requested from the Gateway implementation (e.g., Envoy Gateway + Cilium LB IPAM).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Addresses []GatewayAddress `json:"addresses"`

	// Annotations specifies additional annotations for the Gateway resource
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}
```

- [ ] **Step 2: Regenerate deepcopy**

Run: `task generate`

Expected: `zz_generated.deepcopy.go` updated with deepcopy for the new `*string` field.

- [ ] **Step 3: Regenerate CRD manifests**

Run: `task manifests`

Expected: CRD YAML updated with `gatewayClassName` field in the OpenAPI schema.

- [ ] **Step 4: Verify build**

Run: `go build ./...`

Expected: Clean build, no errors.

- [ ] **Step 5: Commit**

```bash
git add api/v1alpha1/nextdnscoredns_types.go api/v1alpha1/zz_generated.deepcopy.go config/crd/ chart/crds/
git commit -m "feat: add gatewayClassName field to GatewayConfig API type"
```

---

### Task 2: Update reconcileGateway to resolve class name from CR or default

**Files:**
- Modify: `internal/controller/gateway_helpers.go:18-56`
- Test: `internal/controller/gateway_helpers_test.go`

- [ ] **Step 1: Write failing test for CR-level gatewayClassName**

Add to `internal/controller/gateway_helpers_test.go`, after the existing `TestReconcileGateway`:

```go
func TestReconcileGateway_CRLevelClassName(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	crClassName := "cilium"
	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				GatewayClassName: &crClassName,
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{Type: &ipType, Value: "192.168.1.53"},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client:           fakeClient,
		Scheme:           scheme,
		GatewayClassName: "envoy-gateway", // operator default should be overridden
	}

	err := reconciler.reconcileGateway(ctx, coreDNS)
	require.NoError(t, err)

	gw := &gatewayv1.Gateway{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns", Namespace: "default"}, gw)
	require.NoError(t, err)

	// CR-level className should win over operator default
	assert.Equal(t, gatewayv1.ObjectName("cilium"), gw.Spec.GatewayClassName)
}

func TestReconcileGateway_NoClassName(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	ipType := "IPAddress"
	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			Gateway: &nextdnsv1alpha1.GatewayConfig{
				Addresses: []nextdnsv1alpha1.GatewayAddress{
					{Type: &ipType, Value: "192.168.1.53"},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client:           fakeClient,
		Scheme:           scheme,
		GatewayClassName: "", // no operator default
	}

	err := reconciler.reconcileGateway(ctx, coreDNS)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no gatewayClassName")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/controller/ -run "TestReconcileGateway_CRLevelClassName|TestReconcileGateway_NoClassName" -v`

Expected: `TestReconcileGateway_CRLevelClassName` fails (GatewayClassName field doesn't exist on GatewayConfig yet in the test struct -- actually it does from Task 1, but reconcileGateway doesn't read it yet, so it'll use the operator default "envoy-gateway" instead of "cilium"). `TestReconcileGateway_NoClassName` fails (reconcileGateway doesn't validate for empty class name).

- [ ] **Step 3: Update reconcileGateway to resolve class name**

In `internal/controller/gateway_helpers.go`, update `reconcileGateway` to resolve the class name before building the Gateway spec. Replace the function body with:

```go
func (r *NextDNSCoreDNSReconciler) reconcileGateway(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) error {
	logger := log.FromContext(ctx)

	// Resolve GatewayClass name: CR-level override > operator default
	gatewayClassName := r.GatewayClassName
	if coreDNS.Spec.Gateway != nil && coreDNS.Spec.Gateway.GatewayClassName != nil {
		gatewayClassName = *coreDNS.Spec.Gateway.GatewayClassName
	}
	if gatewayClassName == "" {
		return fmt.Errorf("no gatewayClassName specified in spec.gateway and no operator default configured")
	}

	gatewayName := coreDNS.Name + "-dns"

	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayName,
			Namespace: coreDNS.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, gw, func() error {
		// Reset annotations to match spec (removes stale annotations from prior reconciles)
		gw.Annotations = make(map[string]string)
		if coreDNS.Spec.Gateway != nil {
			for k, v := range coreDNS.Spec.Gateway.Annotations {
				gw.Annotations[k] = v
			}
		}

		// Build addresses from spec
		var addresses []gatewayv1.GatewaySpecAddress
		if coreDNS.Spec.Gateway != nil {
			for _, addr := range coreDNS.Spec.Gateway.Addresses {
				gwAddr := gatewayv1.GatewaySpecAddress{
					Value: addr.Value,
				}
				if addr.Type != nil {
					addrType := gatewayv1.AddressType(*addr.Type)
					gwAddr.Type = &addrType
				}
				addresses = append(addresses, gwAddr)
			}
		}

		// Build the gateway spec
		gw.Spec = gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(gatewayClassName),
			Listeners: []gatewayv1.Listener{
				{
					Name:     gatewayv1.SectionName("dns-udp"),
					Port:     gatewayv1.PortNumber(53),
					Protocol: gatewayv1.UDPProtocolType,
					AllowedRoutes: &gatewayv1.AllowedRoutes{
						Kinds: []gatewayv1.RouteGroupKind{
							{Kind: gatewayv1.Kind("UDPRoute")},
						},
					},
				},
				{
					Name:     gatewayv1.SectionName("dns-tcp"),
					Port:     gatewayv1.PortNumber(53),
					Protocol: gatewayv1.TCPProtocolType,
					AllowedRoutes: &gatewayv1.AllowedRoutes{
						Kinds: []gatewayv1.RouteGroupKind{
							{Kind: gatewayv1.Kind("TCPRoute")},
						},
					},
				},
			},
			Addresses: addresses,
		}

		return controllerutil.SetControllerReference(coreDNS, gw, r.Scheme)
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile Gateway: %w", err)
	}

	if op != controllerutil.OperationResultNone {
		logger.Info("Gateway reconciled", "operation", op, "name", gatewayName)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/controller/ -run "TestReconcileGateway" -v`

Expected: All `TestReconcileGateway*` tests PASS (existing test still passes since it sets `GatewayClassName: "envoy-gateway"` on the reconciler and the CR has no override).

- [ ] **Step 5: Commit**

```bash
git add internal/controller/gateway_helpers.go internal/controller/gateway_helpers_test.go
git commit -m "feat: resolve gatewayClassName from CR spec with operator default fallback"
```

---

### Task 3: Add gateway resource cleanup function

**Files:**
- Modify: `internal/controller/gateway_helpers.go` (add new function)
- Test: `internal/controller/gateway_helpers_test.go`

- [ ] **Step 1: Write failing test for gateway cleanup**

Add to `internal/controller/gateway_helpers_test.go`:

```go
func TestCleanupGatewayResources(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
			// No gateway config -- simulating removal
		},
	}

	// Pre-create gateway resources as if they were previously reconciled
	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns-dns",
			Namespace: "default",
		},
	}
	tcpRoute := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns-dns-tcp",
			Namespace: "default",
		},
	}
	udpRoute := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns-dns-udp",
			Namespace: "default",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS, gw, tcpRoute, udpRoute).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	err := reconciler.cleanupGatewayResources(ctx, coreDNS)
	require.NoError(t, err)

	// Verify all gateway resources are deleted
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns", Namespace: "default"}, &gatewayv1.Gateway{})
	assert.True(t, apierrors.IsNotFound(err), "Gateway should be deleted")

	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns-tcp", Namespace: "default"}, &gatewayv1alpha2.TCPRoute{})
	assert.True(t, apierrors.IsNotFound(err), "TCPRoute should be deleted")

	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-coredns-dns-udp", Namespace: "default"}, &gatewayv1alpha2.UDPRoute{})
	assert.True(t, apierrors.IsNotFound(err), "UDPRoute should be deleted")
}

func TestCleanupGatewayResources_AlreadyGone(t *testing.T) {
	scheme := newGatewayTestScheme()
	ctx := context.Background()

	coreDNS := &nextdnsv1alpha1.NextDNSCoreDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-coredns",
			Namespace: "default",
			UID:       types.UID("test-uid"),
		},
		Spec: nextdnsv1alpha1.NextDNSCoreDNSSpec{
			ProfileRef: nextdnsv1alpha1.ResourceReference{Name: "test-profile"},
		},
	}

	// No gateway resources exist -- cleanup should be idempotent
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(coreDNS).
		Build()

	reconciler := &NextDNSCoreDNSReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	err := reconciler.cleanupGatewayResources(ctx, coreDNS)
	require.NoError(t, err)
}
```

Note: add `apierrors "k8s.io/apimachinery/pkg/api/errors"` to the test file imports.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/controller/ -run "TestCleanupGateway" -v`

Expected: FAIL -- `cleanupGatewayResources` method does not exist.

- [ ] **Step 3: Implement cleanupGatewayResources**

Add to `internal/controller/gateway_helpers.go`:

```go
// cleanupGatewayResources deletes Gateway, TCPRoute, and UDPRoute resources
// that were previously created for this NextDNSCoreDNS CR. This is called
// when spec.gateway is removed from a CR.
func (r *NextDNSCoreDNSReconciler) cleanupGatewayResources(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) error {
	logger := log.FromContext(ctx)

	// Delete UDPRoute
	udpRoute := &gatewayv1alpha2.UDPRoute{}
	udpRouteName := types.NamespacedName{Name: coreDNS.Name + "-dns-udp", Namespace: coreDNS.Namespace}
	if err := r.Get(ctx, udpRouteName, udpRoute); err == nil {
		if err := r.Delete(ctx, udpRoute); err != nil {
			return fmt.Errorf("failed to delete UDPRoute %s: %w", udpRouteName.Name, err)
		}
		logger.Info("Deleted orphaned UDPRoute", "name", udpRouteName.Name)
	}

	// Delete TCPRoute
	tcpRoute := &gatewayv1alpha2.TCPRoute{}
	tcpRouteName := types.NamespacedName{Name: coreDNS.Name + "-dns-tcp", Namespace: coreDNS.Namespace}
	if err := r.Get(ctx, tcpRouteName, tcpRoute); err == nil {
		if err := r.Delete(ctx, tcpRoute); err != nil {
			return fmt.Errorf("failed to delete TCPRoute %s: %w", tcpRouteName.Name, err)
		}
		logger.Info("Deleted orphaned TCPRoute", "name", tcpRouteName.Name)
	}

	// Delete Gateway
	gw := &gatewayv1.Gateway{}
	gwName := types.NamespacedName{Name: coreDNS.Name + "-dns", Namespace: coreDNS.Namespace}
	if err := r.Get(ctx, gwName, gw); err == nil {
		if err := r.Delete(ctx, gw); err != nil {
			return fmt.Errorf("failed to delete Gateway %s: %w", gwName.Name, err)
		}
		logger.Info("Deleted orphaned Gateway", "name", gwName.Name)
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/controller/ -run "TestCleanupGateway" -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/controller/gateway_helpers.go internal/controller/gateway_helpers_test.go
git commit -m "feat: add cleanupGatewayResources for orphaned gateway resource deletion"
```

---

### Task 4: Wire cleanup and validation into the reconcile loop

**Files:**
- Modify: `internal/controller/nextdnscoredns_controller.go:217-336`

- [ ] **Step 1: Add gatewayClassName validation and cleanup to the reconcile loop**

In `internal/controller/nextdnscoredns_controller.go`, replace the gateway validation block (lines 217-246) and the gateway reconciliation block (lines 296-336) with the updated logic.

Replace the existing gateway validation block at line 217:

```go
	// Validate Gateway configuration
	if coreDNS.Spec.Gateway != nil {
		// Check mutual exclusivity with LoadBalancer
		if coreDNS.Spec.Service != nil && coreDNS.Spec.Service.Type == nextdnsv1alpha1.ServiceTypeLoadBalancer {
			logger.Info("Invalid configuration: gateway and LoadBalancer service are mutually exclusive")
			r.setCondition(coreDNS, ConditionTypeGatewayReady, metav1.ConditionFalse, "InvalidConfiguration",
				"spec.gateway and spec.service.type=LoadBalancer are mutually exclusive; use one or the other")
			r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionFalse, "InvalidConfiguration",
				"Gateway and LoadBalancer service are mutually exclusive")
			coreDNS.Status.Ready = false
			if updateErr := r.Status().Update(ctx, coreDNS); updateErr != nil {
				logger.Error(updateErr, "Failed to update status")
			}
			return ctrl.Result{}, nil
		}

		// Check if Gateway API CRDs are available
		if !r.GatewayAPIAvailable {
			logger.Info("Gateway API CRDs not available but spec.gateway is set")
			r.setCondition(coreDNS, ConditionTypeGatewayReady, metav1.ConditionFalse, "GatewayAPICRDsMissing",
				"Gateway API CRDs are not installed in the cluster; install them or remove spec.gateway")
			r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionFalse, "GatewayAPICRDsMissing",
				"Gateway API CRDs are not available")
			coreDNS.Status.Ready = false
			if updateErr := r.Status().Update(ctx, coreDNS); updateErr != nil {
				logger.Error(updateErr, "Failed to update status")
			}
			return ctrl.Result{}, nil
		}
	}
```

With:

```go
	// Validate Gateway configuration
	if coreDNS.Spec.Gateway != nil {
		// Check mutual exclusivity with LoadBalancer
		if coreDNS.Spec.Service != nil && coreDNS.Spec.Service.Type == nextdnsv1alpha1.ServiceTypeLoadBalancer {
			logger.Info("Invalid configuration: gateway and LoadBalancer service are mutually exclusive")
			r.setCondition(coreDNS, ConditionTypeGatewayReady, metav1.ConditionFalse, "InvalidConfiguration",
				"spec.gateway and spec.service.type=LoadBalancer are mutually exclusive; use one or the other")
			r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionFalse, "InvalidConfiguration",
				"Gateway and LoadBalancer service are mutually exclusive")
			coreDNS.Status.Ready = false
			if updateErr := r.Status().Update(ctx, coreDNS); updateErr != nil {
				logger.Error(updateErr, "Failed to update status")
			}
			return ctrl.Result{}, nil
		}

		// Check if Gateway API CRDs are available
		if !r.GatewayAPIAvailable {
			logger.Info("Gateway API CRDs not available but spec.gateway is set")
			r.setCondition(coreDNS, ConditionTypeGatewayReady, metav1.ConditionFalse, "GatewayAPICRDsMissing",
				"Gateway API CRDs are not installed in the cluster; install them or remove spec.gateway")
			r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionFalse, "GatewayAPICRDsMissing",
				"Gateway API CRDs are not available")
			coreDNS.Status.Ready = false
			if updateErr := r.Status().Update(ctx, coreDNS); updateErr != nil {
				logger.Error(updateErr, "Failed to update status")
			}
			return ctrl.Result{}, nil
		}

		// Validate that a GatewayClass name is resolvable
		resolvedClassName := r.GatewayClassName
		if coreDNS.Spec.Gateway.GatewayClassName != nil {
			resolvedClassName = *coreDNS.Spec.Gateway.GatewayClassName
		}
		if resolvedClassName == "" {
			logger.Info("No gatewayClassName specified and no operator default configured")
			r.setCondition(coreDNS, ConditionTypeGatewayReady, metav1.ConditionFalse, "NoGatewayClassName",
				"No gatewayClassName specified in spec.gateway and no operator default configured")
			r.setCondition(coreDNS, ConditionTypeReady, metav1.ConditionFalse, "NoGatewayClassName",
				"No gatewayClassName available")
			coreDNS.Status.Ready = false
			if updateErr := r.Status().Update(ctx, coreDNS); updateErr != nil {
				logger.Error(updateErr, "Failed to update status")
			}
			return ctrl.Result{}, nil
		}
	} else if r.GatewayAPIAvailable {
		// spec.gateway was removed -- clean up any orphaned gateway resources
		if err := r.cleanupGatewayResources(ctx, coreDNS); err != nil {
			logger.Error(err, "Failed to clean up gateway resources")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		// Clear gateway-related conditions
		meta.RemoveStatusCondition(&coreDNS.Status.Conditions, ConditionTypeGatewayReady)
		meta.RemoveStatusCondition(&coreDNS.Status.Conditions, ConditionTypeTCPRouteReady)
		meta.RemoveStatusCondition(&coreDNS.Status.Conditions, ConditionTypeUDPRouteReady)
		coreDNS.Status.GatewayReady = false
	}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`

Expected: Clean build.

- [ ] **Step 3: Run full test suite**

Run: `go test ./internal/controller/ -v`

Expected: All tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/controller/nextdnscoredns_controller.go
git commit -m "feat: add gatewayClassName validation and gateway resource cleanup to reconcile loop"
```

---

### Task 5: Remove GatewayClass creation from main.go and update RBAC

**Files:**
- Modify: `cmd/main.go:19,66-68,127-155`
- Modify: `internal/controller/nextdnscoredns_controller.go:87`

- [ ] **Step 1: Remove GatewayClass creation from main.go**

In `cmd/main.go`, replace the `if gatewayAPIAvailable` block (lines 127-155) with:

```go
	if gatewayAPIAvailable {
		setupLog.Info("Gateway API CRDs detected, enabling gateway support")
	} else {
		setupLog.Info("Gateway API CRDs not detected, gateway support disabled")
	}
```

- [ ] **Step 2: Update the flag default and description**

In `cmd/main.go`, replace line 67-68:

```go
	flag.StringVar(&gatewayClassName, "gateway-class-name", lookupEnvOrString("GATEWAY_CLASS_NAME", "nextdns-coredns"),
		"The name of the GatewayClass to create. Can also be set via GATEWAY_CLASS_NAME environment variable.")
```

With:

```go
	flag.StringVar(&gatewayClassName, "gateway-class-name", lookupEnvOrString("GATEWAY_CLASS_NAME", ""),
		"Default GatewayClass name to reference for Gateway API resources. "+
			"Can be overridden per-CR via spec.gateway.gatewayClassName. "+
			"Can also be set via GATEWAY_CLASS_NAME environment variable.")
```

- [ ] **Step 3: Remove unused imports from main.go**

Remove these imports from `cmd/main.go` that are no longer needed:

- `"context"` (only used by GatewayClass RunnableFunc)
- `metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"` (only used by GatewayClass ObjectMeta)
- `"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"` (only used by GatewayClass CreateOrUpdate)
- `"sigs.k8s.io/controller-runtime/pkg/manager"` (only used by manager.RunnableFunc)

Keep: `gatewayv1` and `gatewayv1alpha2` (still needed for scheme registration).

Run: `go build ./...` to verify which imports are actually unused after the removal. Only remove imports that the compiler flags as unused.

- [ ] **Step 4: Remove GatewayClass RBAC marker**

In `internal/controller/nextdnscoredns_controller.go`, delete line 87:

```go
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=get;list;watch;create;update;patch
```

- [ ] **Step 5: Regenerate RBAC manifests**

Run: `task manifests`

Expected: ClusterRole YAML no longer includes `gatewayclasses` resource.

- [ ] **Step 6: Sync Helm RBAC**

Run: `task generate-helm-rbac`

Expected: Helm chart RBAC templates updated.

- [ ] **Step 7: Verify build and tests**

Run: `go build ./... && go test ./... -count=1`

Expected: Clean build, all tests pass.

- [ ] **Step 8: Commit**

```bash
git add cmd/main.go internal/controller/nextdnscoredns_controller.go config/rbac/ chart/
git commit -m "fix: remove GatewayClass creation, operator is consumer not controller"
```

---

### Task 6: Update Helm chart and sample CR

**Files:**
- Modify: `chart/values.yaml:58-61`
- Modify: `config/samples/nextdns_v1alpha1_nextdnscoredns_gateway.yaml`

- [ ] **Step 1: Update Helm values**

In `chart/values.yaml`, replace lines 58-61:

```yaml
# -- Gateway API configuration
gatewayAPI:
  # -- Name of the GatewayClass to create (requires Gateway API CRDs installed)
  gatewayClassName: nextdns-coredns
```

With:

```yaml
# -- Gateway API configuration
gatewayAPI:
  # -- Default GatewayClass name to reference for Gateway API resources.
  # -- Must reference an existing GatewayClass managed by an external controller
  # -- (e.g., Envoy Gateway, Cilium, Istio). Can be overridden per-CR via spec.gateway.gatewayClassName.
  # -- Leave empty if all CRs specify their own gatewayClassName.
  gatewayClassName: ""
```

- [ ] **Step 2: Update sample CR**

Replace the contents of `config/samples/nextdns_v1alpha1_nextdnscoredns_gateway.yaml`:

```yaml
apiVersion: nextdns.io/v1alpha1
kind: NextDNSCoreDNS
metadata:
  name: home-dns-gateway
  namespace: dns
spec:
  profileRef:
    name: my-profile
  upstream:
    primary: DoT
    deviceName: Home DNS
  deployment:
    replicas: 2
  gateway:
    # gatewayClassName references an external GatewayClass managed by a gateway
    # controller (e.g., Envoy Gateway, Cilium, Istio). If omitted, the operator's
    # default --gateway-class-name is used.
    gatewayClassName: envoy-gateway
    addresses:
      - value: "192.168.1.53"
    annotations:
      external-dns.alpha.kubernetes.io/hostname: dns.example.com
  cache:
    enabled: true
  metrics:
    enabled: true
```

- [ ] **Step 3: Verify Helm template renders**

Run: `helm template test chart/`

Expected: Clean render with no errors. The `--gateway-class-name=` arg should show empty string.

- [ ] **Step 4: Commit**

```bash
git add chart/values.yaml config/samples/nextdns_v1alpha1_nextdnscoredns_gateway.yaml
git commit -m "fix: update Helm values and sample CR for external GatewayClass reference"
```

---

### Task 7: Full verification

**Files:** None (verification only)

- [ ] **Step 1: Run full test suite with coverage**

Run: `task test`

Expected: All tests pass. No regressions.

- [ ] **Step 2: Verify CRD includes new field**

Run: `grep -A3 gatewayClassName config/crd/bases/nextdns.io_nextdnscorednses.yaml`

Expected: `gatewayClassName` appears in the CRD OpenAPI schema under `gateway` properties with type `string`.

- [ ] **Step 3: Verify RBAC does not include gatewayclasses**

Run: `grep gatewayclasses config/rbac/role.yaml`

Expected: No matches. `gatewayclasses` should not appear in the generated RBAC.

- [ ] **Step 4: Verify Helm chart RBAC matches**

Run: `grep gatewayclasses chart/templates/*.yaml chart/templates/*.tpl`

Expected: No matches.

- [ ] **Step 5: Verify Helm template with custom class name**

Run: `helm template test chart/ --set gatewayAPI.gatewayClassName=envoy-gateway | grep gateway-class-name`

Expected: `--gateway-class-name=envoy-gateway`

- [ ] **Step 6: Build binary**

Run: `go build -o /dev/null ./cmd/...`

Expected: Clean build.

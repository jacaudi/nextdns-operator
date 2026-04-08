package controller

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

// reconcileGateway creates or updates the Gateway resource for DNS traffic exposure.
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

// reconcileTCPRoute creates or updates the TCPRoute for DNS TCP traffic.
func (r *NextDNSCoreDNSReconciler) reconcileTCPRoute(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, serviceName string) error {
	logger := log.FromContext(ctx)

	routeName := coreDNS.Name + "-dns-tcp"
	gatewayName := coreDNS.Name + "-dns"
	sectionName := gatewayv1.SectionName("dns-tcp")
	port := gatewayv1.PortNumber(53)

	route := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: coreDNS.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, route, func() error {
		route.Spec = gatewayv1alpha2.TCPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:        gatewayv1.ObjectName(gatewayName),
						SectionName: &sectionName,
					},
				},
			},
			Rules: []gatewayv1alpha2.TCPRouteRule{
				{
					BackendRefs: []gatewayv1.BackendRef{
						{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Name: gatewayv1.ObjectName(serviceName),
								Port: &port,
							},
						},
					},
				},
			},
		}

		return controllerutil.SetControllerReference(coreDNS, route, r.Scheme)
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile TCPRoute: %w", err)
	}

	if op != controllerutil.OperationResultNone {
		logger.Info("TCPRoute reconciled", "operation", op, "name", routeName)
	}

	return nil
}

// reconcileUDPRoute creates or updates the UDPRoute for DNS UDP traffic.
func (r *NextDNSCoreDNSReconciler) reconcileUDPRoute(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS, serviceName string) error {
	logger := log.FromContext(ctx)

	routeName := coreDNS.Name + "-dns-udp"
	gatewayName := coreDNS.Name + "-dns"
	sectionName := gatewayv1.SectionName("dns-udp")
	port := gatewayv1.PortNumber(53)

	route := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: coreDNS.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, route, func() error {
		route.Spec = gatewayv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:        gatewayv1.ObjectName(gatewayName),
						SectionName: &sectionName,
					},
				},
			},
			Rules: []gatewayv1alpha2.UDPRouteRule{
				{
					BackendRefs: []gatewayv1.BackendRef{
						{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Name: gatewayv1.ObjectName(serviceName),
								Port: &port,
							},
						},
					},
				},
			},
		}

		return controllerutil.SetControllerReference(coreDNS, route, r.Scheme)
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile UDPRoute: %w", err)
	}

	if op != controllerutil.OperationResultNone {
		logger.Info("UDPRoute reconciled", "operation", op, "name", routeName)
	}

	return nil
}

// cleanupGatewayResources deletes Gateway, TCPRoute, and UDPRoute resources
// that were previously created for this NextDNSCoreDNS CR. This is called
// when spec.gateway is removed from a CR.
func (r *NextDNSCoreDNSReconciler) cleanupGatewayResources(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) error {
	logger := log.FromContext(ctx)

	// Delete UDPRoute
	udpRoute := &gatewayv1alpha2.UDPRoute{}
	udpRouteName := types.NamespacedName{Name: coreDNS.Name + "-dns-udp", Namespace: coreDNS.Namespace}
	if err := r.Get(ctx, udpRouteName, udpRoute); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get UDPRoute %s: %w", udpRouteName.Name, err)
		}
	} else {
		if err := r.Delete(ctx, udpRoute); err != nil {
			return fmt.Errorf("failed to delete UDPRoute %s: %w", udpRouteName.Name, err)
		}
		logger.Info("Deleted orphaned UDPRoute", "name", udpRouteName.Name)
	}

	// Delete TCPRoute
	tcpRoute := &gatewayv1alpha2.TCPRoute{}
	tcpRouteName := types.NamespacedName{Name: coreDNS.Name + "-dns-tcp", Namespace: coreDNS.Namespace}
	if err := r.Get(ctx, tcpRouteName, tcpRoute); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get TCPRoute %s: %w", tcpRouteName.Name, err)
		}
	} else {
		if err := r.Delete(ctx, tcpRoute); err != nil {
			return fmt.Errorf("failed to delete TCPRoute %s: %w", tcpRouteName.Name, err)
		}
		logger.Info("Deleted orphaned TCPRoute", "name", tcpRouteName.Name)
	}

	// Delete Gateway
	gw := &gatewayv1.Gateway{}
	gwName := types.NamespacedName{Name: coreDNS.Name + "-dns", Namespace: coreDNS.Namespace}
	if err := r.Get(ctx, gwName, gw); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get Gateway %s: %w", gwName.Name, err)
		}
	} else {
		if err := r.Delete(ctx, gw); err != nil {
			return fmt.Errorf("failed to delete Gateway %s: %w", gwName.Name, err)
		}
		logger.Info("Deleted orphaned Gateway", "name", gwName.Name)
	}

	return nil
}

// updateGatewayStatus populates the NextDNSCoreDNS status fields from the Gateway status
func (r *NextDNSCoreDNSReconciler) updateGatewayStatus(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) {
	logger := log.FromContext(ctx)

	if coreDNS.Spec.Gateway == nil {
		coreDNS.Status.GatewayReady = false
		return
	}

	gatewayName := coreDNS.Name + "-dns"
	gw := &gatewayv1.Gateway{}
	if err := r.Get(ctx, types.NamespacedName{Name: gatewayName, Namespace: coreDNS.Namespace}, gw); err != nil {
		logger.V(1).Info("Gateway not found for status update", "name", gatewayName)
		coreDNS.Status.GatewayReady = false
		return
	}

	// Reset endpoints to avoid accumulation across reconcile cycles
	coreDNS.Status.Endpoints = nil

	// Check if Gateway has status addresses (indicates it's programmed)
	if len(gw.Status.Addresses) > 0 {
		coreDNS.Status.GatewayReady = true
		for _, addr := range gw.Status.Addresses {
			coreDNS.Status.DNSIP = addr.Value
			coreDNS.Status.Endpoints = append(coreDNS.Status.Endpoints,
				nextdnsv1alpha1.DNSEndpoint{IP: addr.Value, Port: 53, Protocol: "UDP"},
				nextdnsv1alpha1.DNSEndpoint{IP: addr.Value, Port: 53, Protocol: "TCP"},
			)
		}
	} else {
		// Fall back to requested addresses from spec
		coreDNS.Status.GatewayReady = false
		for _, addr := range coreDNS.Spec.Gateway.Addresses {
			coreDNS.Status.DNSIP = addr.Value
			coreDNS.Status.Endpoints = append(coreDNS.Status.Endpoints,
				nextdnsv1alpha1.DNSEndpoint{IP: addr.Value, Port: 53, Protocol: "UDP"},
				nextdnsv1alpha1.DNSEndpoint{IP: addr.Value, Port: 53, Protocol: "TCP"},
			)
		}
	}
}

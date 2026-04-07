package controller

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
)

// reconcileGateway creates or updates the Gateway resource for DNS traffic exposure.
func (r *NextDNSCoreDNSReconciler) reconcileGateway(ctx context.Context, coreDNS *nextdnsv1alpha1.NextDNSCoreDNS) error {
	logger := log.FromContext(ctx)

	gatewayName := coreDNS.Name + "-dns"

	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayName,
			Namespace: coreDNS.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, gw, func() error {
		// Set annotations from spec
		if coreDNS.Spec.Gateway != nil && coreDNS.Spec.Gateway.Annotations != nil {
			if gw.Annotations == nil {
				gw.Annotations = make(map[string]string)
			}
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
			GatewayClassName: gatewayv1.ObjectName(r.GatewayClassName),
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

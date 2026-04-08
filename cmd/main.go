package main

import (
	"flag"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	nextdnsv1alpha1 "github.com/jacaudi/nextdns-operator/api/v1alpha1"
	"github.com/jacaudi/nextdns-operator/internal/controller"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(nextdnsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(gatewayv1.Install(scheme))
	utilruntime.Must(gatewayv1alpha2.Install(scheme))
}

// lookupEnvOrString looks up an environment variable or returns a default string
func lookupEnvOrString(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var syncPeriod string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&syncPeriod, "sync-period", lookupEnvOrString("SYNC_PERIOD", "1h"),
		"The period at which resources are resynced for drift detection. "+
			"Set to 0 to disable periodic syncing. Can also be set via SYNC_PERIOD environment variable.")

	var gatewayClassName string
	flag.StringVar(&gatewayClassName, "gateway-class-name", lookupEnvOrString("GATEWAY_CLASS_NAME", ""),
		"Default GatewayClass name to reference for Gateway API resources. "+
			"Can be overridden per-CR via spec.gateway.gatewayClassName. "+
			"Can also be set via GATEWAY_CLASS_NAME environment variable.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Parse sync period
	syncDuration, err := time.ParseDuration(syncPeriod)
	if err != nil {
		setupLog.Error(err, "invalid sync period", "syncPeriod", syncPeriod)
		os.Exit(1)
	}

	setupLog.Info("drift detection configuration", "syncPeriod", syncDuration)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "nextdns-operator.nextdns.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Detect Gateway API CRDs
	gatewayAPIAvailable := false
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create discovery client")
		os.Exit(1)
	}

	_, apiResourceList, err := discoveryClient.ServerGroupsAndResources()
	if err != nil {
		setupLog.Info("Warning: could not fully discover API resources", "error", err)
	}
	for _, resourceList := range apiResourceList {
		if resourceList.GroupVersion == "gateway.networking.k8s.io/v1" {
			for _, resource := range resourceList.APIResources {
				if resource.Kind == "GatewayClass" {
					gatewayAPIAvailable = true
					break
				}
			}
		}
		if gatewayAPIAvailable {
			break
		}
	}

	if gatewayAPIAvailable {
		setupLog.Info("Gateway API CRDs detected, enabling gateway support")
	} else {
		setupLog.Info("Gateway API CRDs not detected, gateway support disabled")
	}

	if err = (&controller.NextDNSProfileReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		SyncPeriod: syncDuration,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NextDNSProfile")
		os.Exit(1)
	}

	if err = (&controller.NextDNSAllowlistReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		SyncPeriod: syncDuration,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NextDNSAllowlist")
		os.Exit(1)
	}

	if err = (&controller.NextDNSDenylistReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		SyncPeriod: syncDuration,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NextDNSDenylist")
		os.Exit(1)
	}

	if err = (&controller.NextDNSTLDListReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		SyncPeriod: syncDuration,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NextDNSTLDList")
		os.Exit(1)
	}

	if err = (&controller.NextDNSCoreDNSReconciler{
		Client:              mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		SyncPeriod:          syncDuration,
		GatewayAPIAvailable: gatewayAPIAvailable,
		GatewayClassName:    gatewayClassName,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NextDNSCoreDNS")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

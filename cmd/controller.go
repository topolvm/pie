package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/spf13/cobra"
	piev1alpha1 "github.com/topolvm/pie/api/pie/v1alpha1"
	"github.com/topolvm/pie/internal/controller"
	"github.com/topolvm/pie/internal/controller/pie"
	"github.com/topolvm/pie/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var controllerCmd = &cobra.Command{
	Use: "controller",
	RunE: func(cmd *cobra.Command, args []string) error {
		return subMain()
	},
}

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	metricsAddr              string
	enableLeaderElection     bool
	healthProbeAddr          string
	containerImage           string
	monitoringStorageClasses []string
	nodeSelectorString       string
	namespace                string
	controllerURL            string
	probePeriod              int
	createProbeThreshold     time.Duration
	enablePProf              bool

	opts zap.Options
)

func init() {
	flags := controllerCmd.Flags()
	flags.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flags.StringVar(&healthProbeAddr, "health-probe-bind-address", ":8081", "The address the health probe endpoint binds to.")
	flags.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flags.StringVar(&containerImage, "container-image", "", "The container image for pie.")
	flags.StringArrayVar(&monitoringStorageClasses, "monitoring-storage-class", nil, "Monitoring target StorageClasses.")
	flags.StringVar(&nodeSelectorString, "node-selector", "", "The node selector for monitor nodes.")
	flags.StringVar(&namespace, "namespace", "", "The namespace which the controller uses.")
	flags.StringVar(&controllerURL, "controller-url", "", "The controller URL which probe pods access")
	flags.IntVar(&probePeriod, "probe-period", 1, "The period[minute] for CronJob to create a probe pod.")
	flags.DurationVar(&createProbeThreshold, "create-probe-threshold", time.Minute, "The threshold of probe creation.")
	flags.BoolVar(&enablePProf, "enable-pprof", false, "Enable PProf function")
	opts.Development = true

	goflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(goflags)
	opts.BindFlags(goflags)
	flags.AddGoFlagSet(goflags)

	rootCmd.AddCommand(controllerCmd)
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(piev1alpha1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func subMain() error {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	webhookServer := webhook.NewServer(webhook.Options{Port: 9443})
	metricsOption := metricsserver.Options{
		BindAddress: metricsAddr,
	}
	if enablePProf {
		metricsOption.ExtraHandlers = map[string]http.Handler{
			"/debug/pprof/": http.HandlerFunc(pprof.Index),
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsOption,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: healthProbeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "650e0359.topolvm.io", // This is just a unique string. The value itself has no meaning.
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	exporter := metrics.NewMetrics()
	err = mgr.Add(makeReceiveRunner(exporter))
	if err != nil {
		setupLog.Error(err, "unable to start receiverRunner")
		return err
	}

	if containerImage == "" {
		err = errors.New("container image empty")
		setupLog.Error(err, "the container image should be specified")
		return err
	}
	nodeSelector, err := metav1.ParseToLabelSelector(nodeSelectorString)
	if err != nil {
		setupLog.Error(err, "invalid node selector")
		return err
	}
	if probePeriod < 1 || probePeriod > 59 {
		err = errors.New("invalid probe period")
		setupLog.Error(err, "the probe period should be between 1 and 59", "probe period", probePeriod)
		return err
	}
	if time.Duration(probePeriod)*time.Minute < createProbeThreshold {
		err = errors.New("invalid large/small relation")
		setupLog.Error(err, "probe period should be larger than create probe threshold",
			"probe period", probePeriod, "create probe threshold", createProbeThreshold)
		return err
	}
	podReconciler := controller.NewPodReconciler(
		mgr.GetClient(),
		createProbeThreshold,
		exporter,
		monitoringStorageClasses,
		namespace,
	)
	err = podReconciler.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to start podReconciler")
		return err
	}

	probePodReconciler := controller.NewProbePodReconciler(
		mgr.GetClient(),
		exporter,
	)
	err = probePodReconciler.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to start probePodReconciler")
		return err
	}

	if controllerURL == "" {
		err = errors.New("empty controllerURL")
		setupLog.Error(err, "the controllerURL should be specified")
		return err
	}
	nodeReconciler := controller.NewNodeReconciler(
		mgr.GetClient(),
		containerImage,
		namespace,
		controllerURL,
		monitoringStorageClasses,
		nodeSelector,
		probePeriod,
	)
	err = nodeReconciler.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to start nodeReconciler")
		return err
	}

	storageClassReconciler := controller.NewStorageClassReconciler(
		mgr.GetClient(),
		namespace,
		monitoringStorageClasses,
	)
	err = storageClassReconciler.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to start storageClassReconciler")
		return err
	}

	pieProbeController := pie.NewPieProbeController(mgr.GetClient())
	err = pieProbeController.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to start pieProbeController")
		return err
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}

	return nil
}

func makeReceiveRunner(exporter metrics.MetricsExporter) manager.Runnable {
	return manager.RunnableFunc(func(ctx context.Context) error {
		handler := metrics.NewReceiver(exporter)
		s := &http.Server{
			Addr:           ":8082",
			Handler:        handler,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		go func() {
			<-ctx.Done()
			s.Close()
		}()

		return s.ListenAndServe()
	})
}

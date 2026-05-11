package main

import (
	"flag"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	// Set up structured logging
	zapLogger := zap.New()
	ctrl.SetLogger(zapLogger)
	setupLog := ctrl.Log.WithName("setup")

	// Parse flags
	var port int
	var certDir string
	flag.IntVar(&port, "port", 9443, "Webhook server port")
	flag.StringVar(&certDir, "cert-dir", "/tmp/k8s-webhook-server/serving-certs", "Directory containing TLS certificates (tls.crt and tls.key)")
	flag.Parse()

	// Validate port range
	if port < 1024 || port > 65535 {
		setupLog.Error(nil, "invalid port: must be between 1024 and 65535", "port", port)
		os.Exit(1)
	}

	// Create controller-runtime Manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    port,
			CertDir: certDir,
		}),
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	// Register the PodMutator handler at /mutate-v1-pod
	webhookServer := mgr.GetWebhookServer()
	webhookServer.Register("/mutate-v1-pod", &admission.Webhook{
		Handler: &PodMutator{
			Decoder: admission.NewDecoder(mgr.GetScheme()),
		},
	})

	// Add healthz and readyz checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Start the Manager
	setupLog.Info("starting manager", "port", port, "cert-dir", certDir)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "manager exited with error")
		os.Exit(1)
	}
}

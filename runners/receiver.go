package runners

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/topolvm/pie/controller"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	receiverLogger = ctrl.Log.WithName("receiver")
)

type receiverRunner struct {
	exporter controller.MetricsExporter
}

func NewReceiverRunner(exporter controller.MetricsExporter) manager.Runnable {
	return &receiverRunner{
		exporter: exporter,
	}
}

func (r *receiverRunner) Start(ctx context.Context) error {
	receiverLogger.Info("receiver Start")
	defer receiverLogger.Info("receiver End")
	rh := controller.NewReceiverHandler(r.exporter)

	mux := http.NewServeMux()
	mux.Handle("/", rh)

	s := &http.Server{
		Addr:           ":8082",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		receiverLogger.Info("goroutine func")
		defer receiverLogger.Info("goroutine func end")
		err := s.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	receiverLogger.Info("now context Done")
	s.Close()
	receiverLogger.Info("now context Done end")
	return nil
}

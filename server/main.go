package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cprofiler/pkg/apiserver"
	"cprofiler/pkg/collector"
	"cprofiler/pkg/storage"
	"cprofiler/pkg/storage/badger"

	log "github.com/sirupsen/logrus"
)

var (
	configPath     string
	dataPath       string
	dataGCInternal time.Duration
	uiGCInternal   time.Duration
)

func main() {
	flag.StringVar(&configPath, "config-path", "./conf/cprofiler.yml", "Collector configuration file path")
	flag.StringVar(&dataPath, "data-path", "./data/cprofiler/badger", "Collector Data file path")
	flag.DurationVar(&dataGCInternal, "data-gc-internal", 5*time.Minute, "Collector Data gc internal")
	flag.DurationVar(&uiGCInternal, "ui-gc-internal", 2*time.Minute, "Trace and pprof ui gc internal, must be greater than or equal to 1m")

	flag.Parse()

	log.WithFields(log.Fields{"configPath": configPath, "dataPath": dataPath, "dataGCInternal": dataGCInternal.String(), "uiGCInternal": uiGCInternal.String()}).
		Info("flag parse")

	if uiGCInternal < time.Minute {
		log.Fatal("ui-gc-internal must be greater than or equal to 1m")
		return
	}

	startHttpServe()

	// New Store
	store := badger.NewStore(badger.DefaultOptions(dataPath).WithGCInternal(dataGCInternal))
	// Run collector
	collectorManger := runCollector(configPath, store)
	// Run api server
	apiServer := runAPIServer(store, uiGCInternal)

	// receive signal exit
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	s := <-quit
	log.Info("signal receive exit ", s)
	collectorManger.Stop()
	apiServer.Stop()
	store.Release()
}

// runAPIServer Run apis ,pprof ui ,trace ui
func runAPIServer(store storage.Store, gcInternal time.Duration) *apiserver.APIServer {
	apiServer := apiserver.NewAPIServer(
		apiserver.DefaultOptions(store).
			WithAddr(":8080").
			WithGCInternal(gcInternal))

	apiServer.Run()
	return apiServer
}

// runCollector Run collector manger
func runCollector(configPath string, store storage.Store) *collector.Manger {
	m := collector.NewManger(store)
	err := collector.LoadConfig(configPath, func(config collector.CollectorConfig) {
		log.Info("config change, reload collector!!!")
		m.Load(config)
	})
	if err != nil {
		panic(err)
	}
	return m
}

func startHttpServe() {
	go func() {
		http.ListenAndServe("0.0.0.0:9000", nil)
	}()
}

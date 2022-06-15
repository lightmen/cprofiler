package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
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

var buildstamp = ""
var githash = ""
var goversion = fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)

func main() {
	version()

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

func version() {
	args := os.Args
	if len(args) == 2 && (args[1] == "--version" || args[1] == "-v") {
		fmt.Printf("Git Commit Hash: %s\n", githash)
		fmt.Printf("UTC Build Time : %s\n", buildstamp)
		fmt.Printf("Golang Version : %s\n", goversion)
		os.Exit(0)
	}
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

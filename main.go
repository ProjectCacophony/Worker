package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"

	"github.com/emicklei/go-restful"
	"gitlab.com/Cacophony/Worker/api"
	"gitlab.com/Cacophony/Worker/modules"
	"gitlab.com/Cacophony/dhelpers"
	"gitlab.com/Cacophony/dhelpers/cache"
	"gitlab.com/Cacophony/dhelpers/components"
)

var (
	started = time.Now()
)

func init() {
}

func main() {
	var err error

	// Set up components
	components.InitMetrics()
	components.InitLogger("Worker")
	err = components.InitSentry()
	dhelpers.CheckErr(err)
	components.InitTranslator(nil)
	components.InitRedis()
	err = components.InitMongoDB()
	dhelpers.CheckErr(err)
	components.InitCron()
	components.InitLastFm()

	// Setup all modules
	modules.Init()

	cache.GetLogger().Infoln("Worker booting completed, took", time.Since(started).String())

	// start api
	go func() {
		restful.Add(api.New())
		cache.GetLogger().Fatal(http.ListenAndServe(os.Getenv("API_ADDRESS"), nil))
	}()
	cache.GetLogger().Infoln("started API on", os.Getenv("API_ADDRESS"))

	// channel for bot shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	cache.GetCron().Stop()

	// Uninit all modules
	modules.Uninit()
}

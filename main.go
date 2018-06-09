package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"

	"strings"

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
	err = components.InitTracer("Worker")
	dhelpers.CheckErr(err)

	// Setup all modules
	modules.Init()

	// start api server
	apiServer := &http.Server{
		Addr: os.Getenv("API_ADDRESS"),
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      api.New(),
	}
	go func() {
		apiServerListenAndServeErr := apiServer.ListenAndServe()
		if err != nil && !strings.Contains(err.Error(), "http: Server closed") {
			cache.GetLogger().Fatal(apiServerListenAndServeErr)
		}
	}()
	cache.GetLogger().Infoln("started API on", os.Getenv("API_ADDRESS"))

	cache.GetLogger().Infoln("Worker booting completed, took", time.Since(started).String())

	// channel for bot shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// shutdown api server
	apiServerShutdownContext, apiServerCancel := context.WithTimeout(context.Background(), time.Second*15)
	defer apiServerCancel()
	err = apiServer.Shutdown(apiServerShutdownContext)
	dhelpers.LogError(err)

	// stop cron manager
	cache.GetCron().Stop()

	// Uninit all modules
	modules.Uninit()
}

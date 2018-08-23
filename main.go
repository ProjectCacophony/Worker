package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"

	"strings"

	"sync"

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
	if os.Getenv("HEALTHCHECKSIO_API_KEY") != "" {
		components.InitHealthchecksIO()
	}

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
	// create sync.WaitGroup for all shutdown goroutines
	var exitGroup sync.WaitGroup

	// cron manager stopping goroutine
	exitGroup.Add(1)
	go func() {
		// stop cron manager
		cache.GetLogger().Infoln("Stopping cron manager…")
		cache.GetCron().Stop()
		cache.GetLogger().Infoln("Stopped cron manager")
		exitGroup.Done()
	}()

	// API Server shutdown goroutine
	exitGroup.Add(1)
	go func() {
		// shutdown api server
		cache.GetLogger().Infoln("Shutting API server down…")
		err = apiServer.Shutdown(context.TODO())
		dhelpers.LogError(err)
		cache.GetLogger().Infoln("Shut API server down")
		exitGroup.Done()
	}()

	// wait for all shutdown goroutines to finish and then close channel finished
	finished := make(chan bool)
	go func() {
		exitGroup.Wait()
		close(finished)
	}()

	// wait 60 second for everything to finish, or shut it down anyway
	select {
	case <-finished:
		cache.GetLogger().Infoln("shutdown successful")
	case <-time.After(60 * time.Second):
		cache.GetLogger().Infoln("forcing shutdown after 60 seconds")
	}
}

package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.com/project-d-collab/Worker/metrics"
	"gitlab.com/project-d-collab/Worker/modules"
	"gitlab.com/project-d-collab/dhelpers"
	"gitlab.com/project-d-collab/dhelpers/cache"
	"gitlab.com/project-d-collab/dhelpers/components"
)

var (
	started = time.Now()
)

func init() {
}

func main() {
	var err error

	// Set up components
	components.InitLogger("Worker")
	err = components.InitSentry()
	dhelpers.CheckErr(err)
	components.InitTranslator(nil)
	components.InitRedis()
	err = components.InitMongoDB()
	dhelpers.CheckErr(err)
	components.InitCron()

	// start metrics
	metrics.Init()

	// Setup all modules
	modules.Init()

	cache.GetLogger().Infoln("Worker booting completed, took", time.Since(started).String())

	// channel for bot shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	cache.GetCron().Stop()

	// Uninit all modules
	modules.Uninit()
}

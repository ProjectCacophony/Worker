package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"gitlab.com/Cacophony/Worker/pkg/scheduler"
	"gitlab.com/Cacophony/Worker/plugins"
	"gitlab.com/Cacophony/go-kit/api"
	"gitlab.com/Cacophony/go-kit/discord"
	"gitlab.com/Cacophony/go-kit/errortracking"
	"gitlab.com/Cacophony/go-kit/events"
	"gitlab.com/Cacophony/go-kit/featureflag"
	"gitlab.com/Cacophony/go-kit/logging"
	"gitlab.com/Cacophony/go-kit/state"
	"go.uber.org/zap"
)

const (
	// ServiceName is the name of the service
	ServiceName = "worker"
)

func main() {
	// init config
	var config config
	err := envconfig.Process("", &config)
	if err != nil {
		panic(errors.Wrap(err, "unable to load configuration"))
	}
	config.FeatureFlag.Environment = config.ClusterEnvironment
	config.ErrorTracking.Version = config.Hash
	config.ErrorTracking.Environment = config.ClusterEnvironment

	discord.SetAPIBase(config.DiscordAPIBase)

	// init logger
	logger, err := logging.NewLogger(
		config.Environment,
		ServiceName,
		config.LoggingDiscordWebhook,
		&http.Client{
			Timeout: 10 * time.Second,
		},
	)
	if err != nil {
		panic(errors.Wrap(err, "unable to initialise logger"))
	}
	defer logger.Sync()

	// init raven
	err = errortracking.Init(&config.ErrorTracking)
	if err != nil {
		logger.Error("unable to initialise errortracking",
			zap.Error(err),
		)
	}

	// init redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
	})
	_, err = redisClient.Ping().Result()
	if err != nil {
		logger.Fatal("unable to connect to Redis",
			zap.Error(err),
		)
	}

	// init GORM
	gormDB, err := gorm.Open("postgres", config.DBDSN)
	if err != nil {
		logger.Fatal("unable to initialise GORM session",
			zap.Error(err),
		)
	}
	gormDB.DB().SetMaxIdleConns(1)
	gormDB.DB().SetMaxOpenConns(5)
	// gormDB.SetLogger(logger) TODO: write logger
	defer gormDB.Close()

	// init state
	stateClient := state.NewState(redisClient, gormDB)

	// init feature flagger
	featureFlagger, err := featureflag.New(&config.FeatureFlag)
	if err != nil {
		logger.Fatal("unable to initialise feature flagger",
			zap.Error(err),
		)
	}

	// init publisher
	publisher, err := events.NewPublisher(config.AMQPDSN, gormDB)
	if err != nil {
		logger.Fatal("unable to initialise Publisher",
			zap.Error(err),
		)
	}

	// init plugins
	plugins.StartPlugins(
		logger.With(zap.String("feature", "start_plugins")),
		gormDB,
		redisClient,
		config.DiscordTokens,
		stateClient,
		featureFlagger,
		publisher,
	)

	// init scheduler
	sched := scheduler.NewScheduler(
		logger.With(zap.String("feature", "scheduler")),
		featureFlagger,
	)
	go func() {
		sched.Start()
	}()

	// init http server
	httpRouter := api.NewRouter()
	httpServer := api.NewHTTPServer(config.Port, httpRouter)

	go func() {
		err := httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			logger.Fatal("http server error",
				zap.Error(err),
				zap.String("feature", "http-server"),
			)
		}
	}()

	logger.Info("service is running",
		zap.Int("port", config.Port),
	)

	// wait for CTRL+C to stop the service
	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-quitChannel

	// shutdown features

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	plugins.StopPlugins(
		logger.With(zap.String("feature", "stop_plugins")),
		gormDB,
		redisClient,
		config.DiscordTokens,
		stateClient,
		featureFlagger,
		publisher,
	)

	err = httpServer.Shutdown(ctx)
	if err != nil {
		logger.Error("unable to shutdown HTTP Server",
			zap.Error(err),
		)
	}
}

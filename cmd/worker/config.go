package main

import (
	"gitlab.com/Cacophony/go-kit/errortracking"
	"gitlab.com/Cacophony/go-kit/featureflag"
	"gitlab.com/Cacophony/go-kit/logging"
)

// nolint: lll
type config struct {
	Port                  int                  `envconfig:"PORT" default:"8000"`
	Hash                  string               `envconfig:"HASH"`
	Environment           logging.Environment  `envconfig:"ENVIRONMENT" default:"development"`
	ClusterEnvironment    string               `envconfig:"CLUSTER_ENVIRONMENT" default:"development"`
	DiscordTokens         map[string]string    `envconfig:"DISCORD_TOKENS"`
	LoggingDiscordWebhook string               `envconfig:"LOGGING_DISCORD_WEBHOOK"`
	RedisAddress          string               `envconfig:"REDIS_ADDRESS" default:"localhost:6379"`
	RedisPassword         string               `envconfig:"REDIS_PASSWORD"`
	DBDSN                 string               `envconfig:"DB_DSN" default:"postgres://postgres:postgres@localhost:5432/?sslmode=disable"`
	FeatureFlag           featureflag.Config   `envconfig:"FEATUREFLAG"`
	ErrorTracking         errortracking.Config `envconfig:"ERRORTRACKING"`
}

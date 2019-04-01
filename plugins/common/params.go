package common

import (
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"gitlab.com/Cacophony/go-kit/amqp"
	"gitlab.com/Cacophony/go-kit/featureflag"
	"gitlab.com/Cacophony/go-kit/state"
	"go.uber.org/zap"
)

type StartParameters struct {
	Logger         *zap.Logger
	DB             *gorm.DB
	Redis          *redis.Client
	Tokens         map[string]string
	State          *state.State
	FeatureFlagger *featureflag.FeatureFlagger
	Publisher      *amqp.Publisher
}

type StopParameters struct {
	Logger         *zap.Logger
	DB             *gorm.DB
	Redis          *redis.Client
	Tokens         map[string]string
	State          *state.State
	FeatureFlagger *featureflag.FeatureFlagger
	Publisher      *amqp.Publisher
}

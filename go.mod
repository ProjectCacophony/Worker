module gitlab.com/Cacophony/Worker

require (
	github.com/Seklfreak/ginside v0.0.0-20190304181936-3c6d866dc362
	github.com/Seklfreak/ginsta v0.0.0-20190505161125-9c0af4b10e02
	github.com/Unleash/unleash-client-go v0.0.0-20190225211619-9febc6ff26f4 // indirect
	github.com/bsm/redis-lock v8.0.0+incompatible
	github.com/bwmarrin/discordgo v0.19.0
	github.com/getsentry/raven-go v0.2.0
	github.com/go-chi/chi v4.0.2+incompatible // indirect
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/golang/protobuf v1.3.0 // indirect
	github.com/jinzhu/gorm v1.9.2
	github.com/kelseyhightower/envconfig v1.3.0
	github.com/lib/pq v1.0.0
	github.com/mmcdole/gofeed v1.0.0-beta2
	github.com/mmcdole/goxpp v0.0.0-20181012175147-0068e33feabf // indirect
	github.com/pkg/errors v0.8.1
	gitlab.com/Cacophony/go-kit v0.0.0-20190515171420-83e6790ae390
	go.uber.org/zap v1.9.1
)

// replace gitlab.com/Cacophony/go-kit => ../../go-kit

package patrons

import (
	"encoding/json"
	"time"

	lock "github.com/bsm/redis-lock"
	"github.com/go-redis/redis"
)

const (
	lockKey    = "cacophony:worker:stocks-symbols:run-lock"
	lastRunKey = "cacophony:worker:stocks-symbols:run-last"
)

func (p *Plugin) getRunLock() *lock.Locker {
	return lock.New(
		p.redis,
		lockKey,
		&lock.Options{
			LockTimeout: 1 * time.Hour,
			RetryCount:  0, // do not retry
		},
	)
}

func (p *Plugin) shouldRun() (bool, error) {
	raw, err := p.redis.Get(lastRunKey).Bytes()
	if err == redis.Nil {
		return true, nil
	} else if err != nil {
		return false, err
	}

	var lastRun time.Time
	err = json.Unmarshal(raw, &lastRun)
	if err != nil {
		return false, err
	}

	if time.Since(lastRun) < checkInterval {
		return false, nil
	}

	return true, nil
}

func (p *Plugin) setRun() error {
	run := time.Now()

	raw, err := json.Marshal(run)
	if err != nil {
		return err
	}

	return p.redis.Set(lastRunKey, raw, checkInterval).Err()
}

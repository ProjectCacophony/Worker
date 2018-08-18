package modules

import (
	"strings"

	"time"

	"errors"

	"github.com/kristofferahl/go-healthchecksio"
	"gitlab.com/Cacophony/dhelpers/cache"
)

func checkName(job Job) string {
	return "Cacophony Worker " + job.GetJob().Name
}

func checkSchedule(job Job) (period int, schedule string) {
	if strings.HasPrefix(job.GetJob().Cron, "@every ") {
		jobIntervalText := strings.TrimSpace(strings.Replace(job.GetJob().Cron, "@every ", "", 1))
		jobInterval, err := time.ParseDuration(jobIntervalText)
		if err == nil {
			return int(jobInterval.Seconds()), ""
		}
	}

	return 0, job.GetJob().Cron
}

func checkCheck(job Job) healthchecksio.Healthcheck {
	period, schedule := checkSchedule(job)

	return healthchecksio.Healthcheck{
		Channels: "*",
		Grace:    300, // 5 minutes
		Name:     checkName(job),
		Schedule: schedule,
		Tags:     `Cacophony Worker`,
		Timeout:  period,
		Timezone: "UTC",
	}
}

func checkExists(existingChecks []*healthchecksio.HealthcheckResponse, job Job) *healthchecksio.HealthcheckResponse {
	for _, existingCheck := range existingChecks {
		if existingCheck == nil {
			continue
		}
		if existingCheck.Name != checkName(job) {
			continue
		}

		return existingCheck
	}

	return nil
}

func checkCreate(job Job) (*healthchecksio.HealthcheckResponse, error) {
	return cache.GetHealthchecksIO().Create(checkCheck(job))
}

func checkUpdate(check *healthchecksio.HealthcheckResponse, job Job) error {
	if check == nil {
		return errors.New("check passed is nil")
	}

	_, err := cache.GetHealthchecksIO().Update(check.ID(), checkCheck(job))
	return err
}

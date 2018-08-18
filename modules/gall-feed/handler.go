package gallfeed

import (
	"github.com/sirupsen/logrus"
	"gitlab.com/Cacophony/dhelpers"
	"gitlab.com/Cacophony/dhelpers/cache"
)

// Job is a struct for the module
type Job struct{}

var (
	jobName        = "Gall:Feed"
	healthcheckURL string
)

// GetJob defines all jobs
func (j *Job) GetJob() dhelpers.Job {
	return dhelpers.Job{
		Name:     jobName,
		Cron:     "@every 5m",
		Job:      JobFeed,
		AtLaunch: false,
	}
}

// GetTranslationFiles defines all translation files for the module
func (j *Job) GetTranslationFiles() []string {
	return []string{
		"gall.en.toml",
	}
}

// Init is called on bot startup
func (j *Job) Init() {
}

// Uninit is called on normal bot shutdown
func (j *Job) Uninit() {
}

// SetHealthcheckURL is called to inject the healthcheck URL if exists
func (j *Job) SetHealthcheckURL(URL string) {
	healthcheckURL = URL
}

func logger() *logrus.Entry {
	return cache.GetLogger().WithField("module", jobName)
}

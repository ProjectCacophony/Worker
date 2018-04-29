package gallfeed

import (
	"github.com/sirupsen/logrus"
	"gitlab.com/project-d-collab/dhelpers"
	"gitlab.com/project-d-collab/dhelpers/cache"
)

// Job is a struct for the module
type Job struct{}

var (
	jobName = "Gall:Feed"
)

// GetJob defines all jobs
func (j *Job) GetJob() dhelpers.Job {
	return dhelpers.Job{
		Name:     jobName,
		Cron:     "@every 5m",
		Job:      JobFeed,
		AtLaunch: true,
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

func logger() *logrus.Entry {
	return cache.GetLogger().WithField("module", jobName)
}

package modules

import (
	"github.com/kristofferahl/go-healthchecksio"
	"gitlab.com/Cacophony/dhelpers"
	"gitlab.com/Cacophony/dhelpers/cache"
)

// Init initializes all Jobs
func Init() {
	var err error
	cache.GetLogger().Infoln("Initializing Jobs....")

	// get existing checks from healthchecks.io
	healthchecksIOClient := cache.GetHealthchecksIO()
	var existingChecks []*healthchecksio.HealthcheckResponse
	if healthchecksIOClient != nil {
		existingChecks, err = healthchecksIOClient.GetAll()
		if err != nil {
			cache.GetLogger().WithError(err).Errorln("error getting existing checks from healthchecks.io")
		}
	}

	for _, job := range jobList {
		// initialise jobs
		job.Init()
		// load jobs translations
		for _, translationFileName := range job.GetTranslationFiles() {
			_, err = cache.GetLocalizationBundle().LoadMessageFile("./translations/" + translationFileName)
			if err != nil {
				panic(err)
			}
			cache.GetLogger().Infoln("Loaded " + translationFileName)
		}
		// register v
		err = cache.GetCron().AddFunc(job.GetJob().Cron, job.GetJob().Job)
		dhelpers.CheckErr(err)
		// run jobs if requested on launch
		if job.GetJob().AtLaunch {
			go job.GetJob().Job()
		}
		// register at healthchecks.io if possible
		if healthchecksIOClient != nil {
			check := checkExists(existingChecks, job)
			if check == nil {
				// create check
				check, err = checkCreate(job)
				if err != nil {
					cache.GetLogger().WithError(err).WithField("job", job.GetJob().Name).
						Errorln("error creating healthchecks.io check")
				} else {
					cache.GetLogger().WithField("job", job.GetJob().Name).
						Infoln("created new healthchecks.io check")
				}
			} else {
				// update check
				err := checkUpdate(check, job)
				if err != nil {
					cache.GetLogger().WithError(err).
						WithField("job", job.GetJob().Name).WithField("check", check.ID()).
						Errorln("error updating healthchecks.io check")
				} else {
					cache.GetLogger().WithField("job", job.GetJob()).
						WithField("job", job.GetJob().Name).WithField("check", check.ID()).
						Infoln("updated existing healthchecks.io check")
				}
			}
			if check != nil {
				job.SetHealthcheckURL(check.PingURL)
			}
		}
		cache.GetLogger().Infoln("Initialized Job", job.GetJob().Name, "["+job.GetJob().Cron+"]")
	}
}

// Uninit uninitialize all jobs on succesfull shutdown
func Uninit() {
	cache.GetLogger().Infoln("Uninitializing Jobs....")

	for _, job := range jobList {
		job.Uninit()
		cache.GetLogger().Infoln("Uninitialized Job", job.GetJob().Name, "["+job.GetJob().Cron+"]")
	}
}

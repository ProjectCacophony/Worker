package modules

import (
	"gitlab.com/Cacophony/dhelpers"
	"gitlab.com/Cacophony/dhelpers/cache"
)

// Init initializes all Jobs
func Init() {
	var err error
	cache.GetLogger().Infoln("Initializing Jobs....")

	for _, job := range jobList {
		job.Init()
		for _, translationFileName := range job.GetTranslationFiles() {
			_, err = cache.GetLocalizationBundle().LoadMessageFile("./translations/" + translationFileName)
			if err != nil {
				panic(err)
			}
			cache.GetLogger().Infoln("Loaded " + translationFileName)
		}
		err = cache.GetCron().AddFunc(job.GetJob().Cron, job.GetJob().Job)
		dhelpers.CheckErr(err)
		if job.GetJob().AtLaunch {
			go job.GetJob().Job()
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

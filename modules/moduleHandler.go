package modules

import (
	"strings"

	"gitlab.com/project-d-collab/dhelpers"
	"gitlab.com/project-d-collab/dhelpers/cache"
)

// Init initializes all plugins
func Init() {
	var err error
	cache.GetLogger().Infoln("Initializing Modules....")

	for _, module := range moduleList {
		module.Init()
		for _, translationFileName := range module.GetTranslationFiles() {
			_, err = cache.GetLocalizationBundle().LoadMessageFile("./translations/" + translationFileName)
			if err != nil {
				panic(err)
			}
			cache.GetLogger().Infoln("Loaded " + translationFileName)
		}
		jobs := module.GetJobs()
		for _, job := range jobs {
			err = cache.GetCron().AddFunc(job.Cron, job.Job)
			dhelpers.CheckErr(err)
			if job.AtLaunch {
				job.Job()
			}
			cache.GetLogger().Infoln("Initialized Module for Jobs", "["+job.Name+" ("+job.Cron+")"+"]")
		}
	}
}

// Uninit uninitialize all plugins on succesfull shutdown
func Uninit() {
	cache.GetLogger().Infoln("Uninitializing Modules....")

	for _, module := range moduleList {
		module.Uninit()
		jobs := module.GetJobs()
		jobNames := make([]string, 0)
		for _, job := range jobs {
			jobNames = append(jobNames, job.Name+" ("+job.Cron+")")
		}
		cache.GetLogger().Infoln("Uninitialized Module for Jobs", "["+strings.Join(jobNames, ", ")+"]")
	}
}

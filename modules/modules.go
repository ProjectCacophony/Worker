package modules

import (
	"gitlab.com/project-d-collab/Worker/modules/lastfm-servertoptracks"
	"gitlab.com/project-d-collab/dhelpers"
)

// Job is an interface for all modules
type Job interface {

	// GetJob returns the job information
	GetJob() dhelpers.Job

	// GetTranslationFiles returns all translation files for the module
	GetTranslationFiles() []string

	// Init runs at worker startup
	Init()

	// Unit runs at worker shutdown
	Uninit()
}

var (
	jobList = []Job{
		&lastfmservertoptracks.Job{},
	}
)

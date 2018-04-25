package modules

import (
	"gitlab.com/project-d-collab/Worker/modules/lastfm"
	"gitlab.com/project-d-collab/dhelpers"
)

// Module is an interface for all modules
type Module interface {

	// GetJobs returns all jobs of a module
	GetJobs() []dhelpers.Job

	// GetTranslationFiles returns all translation files for the module
	GetTranslationFiles() []string

	// Init runs at worker startup
	Init()

	// Unit runs at worker shutdown
	Uninit()
}

var (
	moduleList = []Module{
		&lastfm.Module{},
	}
)

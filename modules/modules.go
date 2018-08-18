package modules

import (
	"gitlab.com/Cacophony/Worker/modules/feed-feed"
	"gitlab.com/Cacophony/Worker/modules/gall-feed"
	"gitlab.com/Cacophony/Worker/modules/lastfm-servertoptracks"
	"gitlab.com/Cacophony/dhelpers"
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

	SetHealthcheckURL(string)
}

var (
	jobList = []Job{
		&lastfmservertoptracks.Job{},
		&gallfeed.Job{},
		&feedfeed.Job{},
	}
)

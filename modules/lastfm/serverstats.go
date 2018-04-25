package lastfm

import (
	"time"

	"gitlab.com/project-d-collab/dhelpers"
)

// JobServerStats builds server Last.FM stats
func JobServerStats() {
	// init variables
	jobName := "LastFm:ServerStats"
	logger := logger().WithField("job", jobName)
	duration := time.Minute * 1

	// start job if none is running yet
	start, locker, err := dhelpers.JobStart(jobName, duration)
	if locker != nil {
		defer locker.Unlock() // nolint: errcheck
	}
	dhelpers.CheckErr(err)
	if !start {
		logger.Warnln("skipped running job", jobName, "because it is still running")
		return
	}

	logger.Infoln("I'm running!")
}

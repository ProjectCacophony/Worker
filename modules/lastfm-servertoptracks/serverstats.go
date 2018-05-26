package lastfmservertoptracks

import (
	"time"

	"strings"

	"context"

	"github.com/bradfitz/slice"
	"github.com/json-iterator/go"
	"github.com/opentracing/opentracing-go"
	"gitlab.com/Cacophony/SqsProcessor/models"
	"gitlab.com/Cacophony/dhelpers"
	"gitlab.com/Cacophony/dhelpers/cache"
	"gitlab.com/Cacophony/dhelpers/mdb"
	"gitlab.com/Cacophony/dhelpers/state"
)

type lastFmPeriodUserStats struct {
	UserID string
	Period dhelpers.LastFmPeriod
	Tracks []dhelpers.LastfmTrackData
}

// JobServerStats builds server Last.FM stats
func JobServerStats() {
	// Error Handling
	defer dhelpers.JobErrorHandler(jobName)

	// init variables
	duration := time.Minute * 1

	// start span
	span, ctx := opentracing.StartSpanFromContext(context.Background(), jobName)
	defer span.Finish()

	// start job if none is running yet
	start, locker, err := dhelpers.JobStart(jobName, duration)
	dhelpers.CheckErr(err)
	if !start {
		logger().Warnln("skipped running job because it is still running")
		return
	}
	defer locker.Unlock() // nolint: errcheck

	startAt := time.Now()
	logger().Infoln("starting")

	// which periods to look up
	periods := []dhelpers.LastFmPeriod{
		dhelpers.LastFmPeriod7day,
		dhelpers.LastFmPeriod1month,
		dhelpers.LastFmPeriod3month,
		dhelpers.LastFmPeriod6month,
		dhelpers.LastFmPeriod12month,
		dhelpers.LastFmPeriodOverall,
	}

	var entryBucket []models.LastFmEntry
	err = mdb.Iter(models.LastFmTable.DB().Find(nil)).All(&entryBucket)
	dhelpers.CheckErr(err)

	// Get Stats from LastFM
	lastFmUserPeriodStats := make([]lastFmPeriodUserStats, 0)
	var topTracks []dhelpers.LastfmTrackData
	for _, period := range periods {
		for _, entry := range entryBucket {
			topTracks, err = dhelpers.LastFmGetTopTracks(ctx, entry.LastFmUsername, 100, period)
			if err != nil {
				if strings.Contains(err.Error(), "User not found") {
					continue
				}
			}
			dhelpers.CheckErr(err)

			if len(topTracks) <= 0 {
				continue
			}

			var userPeriodStat lastFmPeriodUserStats
			userPeriodStat.UserID = entry.UserID
			userPeriodStat.Period = period

			// add tracks
			userPeriodStat.Tracks = append(userPeriodStat.Tracks, topTracks...)

			lastFmUserPeriodStats = append(lastFmUserPeriodStats, userPeriodStat)

			// renew lock
			locker.Lock() // nolint: errcheck
		}
	}

	// Combine Stats
	combinedGuildStats := make([]dhelpers.LastFmGuildTopTracks, 0)
	allGuildIDs, err := state.AllGuildIDs()
	dhelpers.CheckErr(err)
	for _, period := range periods {
		for _, guildID := range allGuildIDs {
			memberIDs, err := state.GuildUserIDs(guildID)
			dhelpers.CheckErr(err)

			if len(memberIDs) <= 0 {
				continue
			}

			var guildCombinedStat dhelpers.LastFmGuildTopTracks
			guildCombinedStat.GuildID = guildID
			guildCombinedStat.NumberOfUsers = 0
			guildCombinedStat.Period = period

			for _, memberID := range memberIDs {
				for _, userPeriodStat := range lastFmUserPeriodStats {
					if userPeriodStat.UserID != memberID {
						continue
					}
					if userPeriodStat.Period != period {
						continue
					}

					for _, track := range userPeriodStat.Tracks {
						track.Users = 1
						var added bool
						for i := range guildCombinedStat.Tracks {
							if guildCombinedStat.Tracks[i].URL != track.URL {
								continue
							}

							guildCombinedStat.Tracks[i].Users += track.Users
							guildCombinedStat.Tracks[i].Scrobbles += track.Scrobbles
							added = true
						}
						if !added {
							guildCombinedStat.Tracks = append(guildCombinedStat.Tracks, track)
						}
					}
					guildCombinedStat.NumberOfUsers++
				}
			}

			if len(guildCombinedStat.Tracks) <= 0 {
				continue
			}
			combinedGuildStats = append(combinedGuildStats, guildCombinedStat)
		}

		// renew lock
		locker.Lock() // nolint: errcheck
	}

	// sort stats
	for n := range combinedGuildStats {
		slice.Sort(combinedGuildStats[n].Tracks[:], func(i, j int) bool {
			return combinedGuildStats[n].Tracks[i].Scrobbles > combinedGuildStats[n].Tracks[j].Scrobbles
		})

		// renew lock
		locker.Lock() // nolint: errcheck
	}

	// store stats in redis
	for _, combinedGuildStat := range combinedGuildStats {
		combinedGuildStat.CachedAt = time.Now()
		marshalled, err := jsoniter.Marshal(combinedGuildStat)
		dhelpers.CheckErr(err)

		err = cache.GetRedisClient().Set(
			dhelpers.LastFmGuildTopTracksKey(combinedGuildStat.GuildID, combinedGuildStat.Period),
			marshalled,
			time.Hour*24*7,
		).Err()
		dhelpers.CheckErr(err)

		// renew lock
		locker.Lock() // nolint: errcheck
	}

	logger().Infoln("finished, took", time.Since(startAt).String())
}

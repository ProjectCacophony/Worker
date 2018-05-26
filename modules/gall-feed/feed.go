package gallfeed

import (
	"time"

	"context"

	"github.com/Seklfreak/ginside"
	"github.com/bwmarrin/discordgo"
	"github.com/opentracing/opentracing-go"
	"gitlab.com/Cacophony/SqsProcessor/models"
	"gitlab.com/Cacophony/SqsProcessor/modules/gall"
	"gitlab.com/Cacophony/Worker/metrics"
	"gitlab.com/Cacophony/dhelpers"
	"gitlab.com/Cacophony/dhelpers/mdb"
	"gitlab.com/Cacophony/dhelpers/state"
)

type boardCheckBundleInfo struct {
	BoardID     string
	Minor       bool
	Recommended bool
}

// JobFeed posts new Gall Feed posts
func JobFeed() {
	// Error Handling
	defer dhelpers.JobErrorHandler(jobName)

	// start span
	span, _ := opentracing.StartSpanFromContext(context.Background(), jobName)
	defer span.Finish()

	// init variables
	duration := time.Minute * 1

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

	// worker code
	// get all entries to check
	var feedEntries []models.GallFeedEntry
	err = mdb.Iter(models.GallTable.DB().Find(nil)).All(&feedEntries)
	dhelpers.CheckErr(err)

	// renew lock
	locker.Lock() // nolint: errcheck

	// bundle entries
	// don't check channels that don't access anymore, or without necessary permissions
	bundledEntries := make(map[boardCheckBundleInfo][]models.GallFeedEntry)
	var channel *discordgo.Channel
	var botIDForGuild string
	var apermissions int
	var addedToBundle bool
	for _, entry := range feedEntries {
		// channel exists
		channel, err = state.Channel(entry.ChannelID)
		if err != nil {
			logger().Info("skipped", mdb.IDToHuman(entry.ID), "because channel is not available")
			continue
		}

		// get correct bot ID
		botIDForGuild, err = state.BotIDForGuild(channel.GuildID)
		if err != nil {
			logger().Info("skipped", mdb.IDToHuman(entry.ID), "because it was not possible to get a bot ID")
			continue
		}

		// get bot permissions in channel
		apermissions, err = state.UserChannelPermissions(botIDForGuild, entry.ChannelID)
		if err != nil {
			logger().Info("skipped", mdb.IDToHuman(entry.ID), "because it was not possible to access the permissions")
			continue
		}

		// can send messages and embed links?
		if apermissions&discordgo.PermissionSendMessages != discordgo.PermissionSendMessages ||
			apermissions&discordgo.PermissionEmbedLinks != discordgo.PermissionEmbedLinks {
			logger().Info("skipped", mdb.IDToHuman(entry.ID), "because the bot has missings permissions")
			continue
		}

		// bundle feed entry if everything is good
		addedToBundle = false
		for key := range bundledEntries {
			if key.BoardID != entry.BoardID || key.Minor != entry.MinorGallery || key.Recommended != entry.Recommended {
				continue
			}
			bundledEntries[key] = append(bundledEntries[key], entry)
			addedToBundle = true
		}
		if !addedToBundle {
			bundledEntries[boardCheckBundleInfo{
				BoardID:     entry.BoardID,
				Minor:       entry.MinorGallery,
				Recommended: entry.Recommended,
			}] = []models.GallFeedEntry{entry}
		}
	}

	var checkedAt time.Time
	var postedForEntry int
	var alreadyPosted bool

	// check feeds
	for checkInfo, entries := range bundledEntries {
		checkedAt = time.Now()
		// check bundle feeds
		var posts []ginside.Post
		if !checkInfo.Minor {
			posts, err = ginside.BoardPosts(checkInfo.BoardID, checkInfo.Recommended)
			if err != nil {
				logger().Errorln("unable to check feed for", checkInfo.BoardID+":", err.Error())
				continue
			}
		} else {
			posts, err = ginside.BoardMinorPosts(checkInfo.BoardID, checkInfo.Recommended)
			if err != nil {
				logger().Errorln("unable to check feed for", checkInfo.BoardID+":", err.Error())
				continue
			}
		}

		// check entries
		for _, entry := range entries {
			postedForEntry = 0

			for _, post := range posts {
				// skip posts before feed adding check
				if !post.Date.After(entry.AddedAt) {
					continue
				}

				// skip already posted posts
				alreadyPosted = false
				for _, postedPostID := range entry.PostedPostIDs {
					if postedPostID != gall.GetEntryID(post) {
						continue
					}
					alreadyPosted = true
					break
				}
				if alreadyPosted {
					continue
				}

				// skip too old posts
				if time.Since(post.Date) > time.Hour*1 {
					continue
				}
				// don't post more than five per check
				if postedForEntry > 5 {
					continue
				}

				// increase posts per check counter
				postedForEntry++
				// add posted post ID to database entry for deduplication
				entry.PostedPostIDs = append(entry.PostedPostIDs, gall.GetEntryID(post))
				// increase metrics gall posts counter
				metrics.GallFeedPosts.Add(1)

				// start goroutine to post
				go func(gEntry models.GallFeedEntry, gPost ginside.Post) {
					defer dhelpers.JobErrorHandler(jobName)

					err = postPost(gEntry, gPost)
					dhelpers.CheckErr(err)
				}(entry, post)
			}

			// update last checked time
			entry.LastCheck = checkedAt
			err = mdb.UpdateID(models.GallTable, entry.ID, entry)
			dhelpers.CheckErr(err)
		}

		// renew lock
		locker.Lock() // nolint: errcheck
	}

	logger().Infoln("finished, took", time.Since(startAt).String())
}

func postPost(entry models.GallFeedEntry, post ginside.Post) (err error) {
	// get bot id
	botIDForGuild, err := state.BotIDForGuild(entry.GuildID)
	if err != nil {
		return err
	}

	_, err = dhelpers.SendComplexWithBot(botIDForGuild, entry.ChannelID, &discordgo.MessageSend{
		Content: "<" + post.URL + ">",
		Embed: &discordgo.MessageEmbed{
			URL:         post.URL,
			Title:       dhelpers.Tf("GallFeedEmbedTitle", "entry", entry),
			Description: dhelpers.Tf("GallFeedEmbedDescription", "post", post),
			Timestamp:   dhelpers.DiscordTime(post.Date),
			Color:       gall.GallColor,
			Footer: &discordgo.MessageEmbedFooter{
				Text:    dhelpers.T("GallFeedEmbedFooter"),
				IconURL: gall.GallIcon,
			},
		},
	})
	return err
}

package gallfeed

import (
	"time"

	"github.com/Seklfreak/ginside"
	"github.com/bwmarrin/discordgo"
	"gitlab.com/project-d-collab/SqsProcessor/models"
	"gitlab.com/project-d-collab/SqsProcessor/modules/gall"
	"gitlab.com/project-d-collab/dhelpers"
	"gitlab.com/project-d-collab/dhelpers/mdb"
	"gitlab.com/project-d-collab/dhelpers/state"
)

// JobFeed posts new Gall Feed posts
func JobFeed() {
	// Error Handling
	defer dhelpers.JobErrorHandler(jobName)

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
	bundledEntries := make(map[string][]models.GallFeedEntry)
	var channel *discordgo.Channel
	var botIDForGuild string
	var apermissions int
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
		bundledEntries[entry.BoardID] = append(bundledEntries[entry.BoardID], entry)
	}

	// check feeds
	for boardID, entries := range bundledEntries {
		// check bundle feeds
		var posts []ginside.Post
		if !entries[0].MinorGallery {
			posts, err = ginside.BoardRecommendedPosts(boardID)
			if err != nil {
				logger().Errorln("unable to check feed for", boardID+":", err.Error())
				continue
			}
		} else {
			posts, err = ginside.BoardMinorRecommendedPosts(boardID)
			if err != nil {
				logger().Errorln("unable to check feed for", boardID+":", err.Error())
				continue
			}
		}

		// check entries
		for _, entry := range entries {
			for _, post := range posts {
				// skip posts before last check
				if post.Date.Before(entry.LastCheck) {
					continue
				}

				go func(gEntry models.GallFeedEntry, gPost ginside.Post) {
					defer dhelpers.JobErrorHandler(jobName)

					err = postPost(gEntry, gPost)
					dhelpers.CheckErr(err)
				}(entry, post)
			}

			// update last checked time (TODO: possible to update field without updating whole entry?)
			entry.LastCheck = time.Now()
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

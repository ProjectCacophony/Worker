package feedfeed

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mmcdole/gofeed"
	"gitlab.com/Cacophony/SqsProcessor/models"
	feedModule "gitlab.com/Cacophony/SqsProcessor/modules/feed"
	"gitlab.com/Cacophony/dhelpers"
	"gitlab.com/Cacophony/dhelpers/mdb"
	"gitlab.com/Cacophony/dhelpers/state"
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
	var feedEntries []models.FeedEntry
	err = mdb.Iter(models.FeedTable.DB().Find(nil)).All(&feedEntries)
	dhelpers.CheckErr(err)

	// renew lock
	locker.Lock() // nolint: errcheck

	// bundle entries
	// don't check channels that don't access anymore, or without necessary permissions
	bundledEntries := make(map[string][]models.FeedEntry)
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
		bundledEntries[entry.FeedURL] = append(bundledEntries[entry.FeedURL], entry)
	}

	// check feeds
	for feedURL, entries := range bundledEntries {
		// check bundle feeds
		var feed *gofeed.Feed
		feed, err = feedModule.GetFeed(feedURL)
		if err != nil {
			logger().Errorln("unable to check feed for", feedURL+":", err.Error())
			continue
		}

		var latestEntryTime time.Time

		// check entries
		for _, entry := range entries {
			for _, post := range feed.Items {
				// skip feeds with invalid fields
				if post == nil ||
					post.PublishedParsed == nil || post.PublishedParsed.IsZero() ||
					post.Link == "" ||
					post.Title == "" {
					continue
				}

				if post.PublishedParsed.After(latestEntryTime) {
					latestEntryTime = *post.PublishedParsed
				}

				// skip posts before last check
				if !post.PublishedParsed.After(entry.LastCheck) {
					continue
				}

				// TODO: safety check (max one hour old)

				go func(gEntry models.FeedEntry, gPost *gofeed.Item) {
					defer dhelpers.JobErrorHandler(jobName)

					err = postPost(gEntry, *gPost)
					dhelpers.CheckErr(err)
				}(entry, post)
			}

			// update last checked time (TODO: possible to update field without updating whole entry?)
			if !entry.LastCheck.Equal(latestEntryTime) {
				entry.LastCheck = latestEntryTime
				err = mdb.UpdateID(models.FeedTable, entry.ID, entry)
				dhelpers.CheckErr(err)
			}
		}

		// renew lock
		locker.Lock() // nolint: errcheck
	}

	logger().Infoln("finished, took", time.Since(startAt).String())
}

func postPost(entry models.FeedEntry, post gofeed.Item) (err error) {
	// get bot id
	botIDForGuild, err := state.BotIDForGuild(entry.GuildID)
	if err != nil {
		return err
	}

	var author gofeed.Person
	if post.Author != nil {
		author = *post.Author
	}

	message := &discordgo.MessageSend{
		Content: "<" + post.Link + ">",
		Embed: &discordgo.MessageEmbed{
			URL:         post.Link,
			Title:       dhelpers.Tf("FeedEmbedTitle", "entry", entry),
			Description: dhelpers.Tf("FeedEmbedDescription", "post", post, "author", author),
			Timestamp:   dhelpers.DiscordTime(*post.PublishedParsed),
			Footer: &discordgo.MessageEmbedFooter{
				Text: dhelpers.T("FeedEmbedFooter"),
			},
		},
	}

	if post.Image != nil && post.Image.URL != "" {
		message.Embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: post.Image.URL,
		}
	}

	_, err = dhelpers.SendComplexWithBot(botIDForGuild, entry.ChannelID, message)
	return err
}

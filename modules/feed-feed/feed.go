package feedfeed

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/mmcdole/gofeed"
	"github.com/opentracing/opentracing-go"
	"gitlab.com/Cacophony/SqsProcessor/models"
	feedModule "gitlab.com/Cacophony/SqsProcessor/modules/feed"
	"gitlab.com/Cacophony/Worker/metrics"
	"gitlab.com/Cacophony/dhelpers"
	"gitlab.com/Cacophony/dhelpers/mdb"
	"gitlab.com/Cacophony/dhelpers/state"
)

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

	var checkedAt time.Time
	var postedForEntry int
	var alreadyPosted bool

	// check feeds
	for feedURL, entries := range bundledEntries {
		checkedAt = time.Now()
		// check bundle feeds
		var feed *gofeed.Feed
		feed, err = feedModule.GetFeed(feedURL)
		if err != nil {
			logger().Errorln("unable to check feed for", feedURL+":", err.Error())
			continue
		}

		// check entries
		for _, entry := range entries {
			postedForEntry = 0

			for _, post := range feed.Items {
				// skip feeds with invalid fields
				if post == nil ||
					post.PublishedParsed == nil || post.PublishedParsed.IsZero() ||
					post.Link == "" ||
					post.Title == "" {
					continue
				}

				// skip posts before feed adding check
				if !post.PublishedParsed.After(entry.AddedAt) {
					continue
				}

				// skip already posted posts
				alreadyPosted = false
				for _, postedPostID := range entry.PostedPostIDs {
					if postedPostID != feedModule.GetEntryID(post) {
						continue
					}
					alreadyPosted = true
					break
				}
				if alreadyPosted {
					continue
				}

				// skip too old posts
				if time.Since(*post.PublishedParsed) > time.Hour*1 {
					continue
				}
				// don't post more than five per check
				if postedForEntry > 5 {
					continue
				}

				// increase posts per check counter
				postedForEntry++
				// add posted post ID to database entry for deduplication
				entry.PostedPostIDs = append(entry.PostedPostIDs, feedModule.GetEntryID(post))
				// increase metrics gall posts counter
				metrics.FeedFeedPosts.Add(1)

				// start goroutine to post
				go func(gEntry models.FeedEntry, gPost *gofeed.Item) {
					defer dhelpers.JobErrorHandler(jobName)

					err = postPost(gEntry, *gPost)
					dhelpers.CheckErr(err)
				}(entry, post)
			}

			// update last checked time
			entry.LastCheck = checkedAt
			err = mdb.UpdateID(models.FeedTable, entry.ID, entry)
			dhelpers.CheckErr(err)
		}

		// renew lock
		locker.Lock() // nolint: errcheck
	}

	logger().Infoln("finished, took", time.Since(startAt).String())
	err = dhelpers.JobFinishSuccess(healthcheckURL)
	dhelpers.LogError(err)
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

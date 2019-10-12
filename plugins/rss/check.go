package rss

import (
	"database/sql"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gitlab.com/Cacophony/go-kit/permissions"

	"github.com/mmcdole/gofeed"

	"github.com/bwmarrin/discordgo"

	"gitlab.com/Cacophony/go-kit/discord"

	"gitlab.com/Cacophony/Worker/plugins/common"
	"go.uber.org/zap"

	kitFeed "gitlab.com/Cacophony/go-kit/feed"
)

const (
	color              = 15885602           // #F26522
	ageLimit           = time.Hour * 24 * 3 // three days
	postsPerCheckLimit = 5
)

func (p *Plugin) checkBundles(run *common.Run, tx *sql.Tx, bundles boardCheckBundle) {
	var err error

	run.Logger().Info("checking bundles",
		zap.Int("amount", len(bundles)),
	)

	var feed *gofeed.Feed
	for checkInfo, entries := range bundles {
		feed, err = getFeed(p.httpClient, p.parser, checkInfo.FeedURL)
		if err != nil {
			if !isAcceptableError(err) {
				run.Except(err, "feed_url", checkInfo.FeedURL)
			}

			err = checkSet(run.Context(), tx, kitFeed.ErrorStatus, err.Error(), entries...)
			if err != nil {
				run.Except(err, "feed_url", checkInfo.FeedURL)
			}
			continue
		}

		for _, entry := range entries {
			err = p.checkEntry(run, entry, feed)
			if err != nil {
				run.Except(err, "feed_url", checkInfo.FeedURL)

				err = checkSet(run.Context(), tx, kitFeed.ErrorStatus, err.Error(), entry)
				if err != nil {
					run.Except(err, "feed_url", checkInfo.FeedURL)
				}
			} else {
				err = checkSet(run.Context(), tx, kitFeed.SuccessStatus, "", entry)
				if err != nil {
					run.Except(err, "feed_url", checkInfo.FeedURL)
				}
			}
		}
	}
}

func (p *Plugin) checkEntry(run *common.Run, entry Entry, feed *gofeed.Feed) error {
	var posted int

	for _, post := range feed.Items {
		logger := run.Logger().With(
			zap.Uint("entry_id", entry.ID),
			zap.String("post_guid", post.GUID),
			zap.String("post_link", post.Link),
		)

		if post.PublishedParsed == nil && post.UpdatedParsed != nil {
			post.PublishedParsed = post.UpdatedParsed
		}

		if posted > postsPerCheckLimit {
			// logger.Debug("skipping post because of the posts per check limit")
			break
		}

		if post.PublishedParsed == nil {
			// logger.Debug("skipping post because post date is empty")
			continue
		}

		if post.Link == "" {
			// logger.Debug("skipping post because post link is empty")
			continue
		}

		if !post.PublishedParsed.After(entry.CreatedAt) {
			// logger.Debug("skipping post because post date is not after entry creation date")
			continue
		}

		if time.Since(*post.PublishedParsed) > ageLimit {
			// logger.Debug("skipping post because of the age limit")
			continue
		}

		existingPost, err := postFind(p.db,
			"entry_id = ? AND (post_guid = ? OR post_link = ?)",
			entry.ID, post.GUID, post.Link,
		)
		if err != nil && !strings.Contains(err.Error(), "record not found") {
			return err
		}
		if existingPost != nil {
			// logger.Debug("skipping post because it has already been posted")
			continue
		}

		logger.Info("posting post")

		err = p.post(run, entry, post)
		if err != nil {
			return errors.Wrap(err, "error posting")
		}
		posted++
	}

	return nil
}

func (p *Plugin) post(_ *common.Run, entry Entry, post *gofeed.Item) error {
	var err error

	botID := entry.BotID
	if !entry.DM {
		botID, err = p.state.BotForChannel(
			entry.ChannelID,
			permissions.DiscordSendMessages,
			permissions.DiscordEmbedLinks,
		)
		if err != nil {
			return err
		}
	}
	if botID == "" {
		return errors.New("no Bot ID")
	}

	session, err := discord.NewSession(p.tokens, botID)
	if err != nil {
		return err
	}

	channelID := entry.ChannelID
	if entry.DM {
		channelID, err = discord.DMChannel(p.redis, session, channelID)
		if err != nil {
			return err
		}
	}

	messages, err := discord.SendComplexWithVars(
		session,
		p.Localizations(),
		channelID,
		&discordgo.MessageSend{
			Content: "feed.post.content",
			Embed: &discordgo.MessageEmbed{
				URL:         post.Link,
				Title:       "feed.post.embed.title",
				Description: "feed.post.embed.description",
				Timestamp:   post.PublishedParsed.Format(time.RFC3339),
				Color:       color,
				Footer: &discordgo.MessageEmbedFooter{
					Text:    "feed.post.embed.footer.text",
					IconURL: "feed.post.embed.footer.icon-url",
				},
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: getThumbnailURL(post),
				},
			},
		},
		"post", post, "entry", entry,
	)
	if err != nil {
		discord.CheckBlockDMChannel(p.redis, session, channelID, err)
		return err
	}

	messageIDs := make([]string, len(messages))
	for i, message := range messages {
		messageIDs[i] = message.ID
	}

	return postAdd(p.db, entry.ID, post.GUID, post.Link, messageIDs)
}

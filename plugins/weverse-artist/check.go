package weverseartist

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/Seklfreak/geverse"

	"github.com/pkg/errors"

	"gitlab.com/Cacophony/go-kit/feed"
	"gitlab.com/Cacophony/go-kit/permissions"

	"github.com/bwmarrin/discordgo"

	"gitlab.com/Cacophony/go-kit/discord"

	"gitlab.com/Cacophony/Worker/plugins/common"
	"go.uber.org/zap"
)

const (
	ageLimit           = time.Hour * 24 * 3 // three days
	postsPerCheckLimit = 5
)

func (p *Plugin) checkBundles(run *common.Run, tx *sql.Tx, bundles bundle) {
	var err error

	run.Logger().Info("checking bundles",
		zap.Int("amount", len(bundles)),
	)

	var artistFeed *geverse.ArtistPosts
	for checkInfo, entries := range bundles {
		artistFeed, err = p.geverse.GetArtistPosts(run.Context(), checkInfo.ChannelID)
		if err != nil {
			run.Except(err, "channel_id", strconv.FormatInt(checkInfo.ChannelID, 10))

			err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entries...)
			if err != nil {
				run.Except(err, "channel_id", strconv.FormatInt(checkInfo.ChannelID, 10))
			}
			continue
		}

		for _, entry := range entries {
			err = p.checkEntry(run, entry, artistFeed.Posts)
			if err != nil {
				run.Except(err, "channel_id", strconv.FormatInt(checkInfo.ChannelID, 10))

				err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entry)
				if err != nil {
					run.Except(err, "channel_id", strconv.FormatInt(checkInfo.ChannelID, 10))
				}
			} else {
				err = checkSet(run.Context(), tx, feed.SuccessStatus, "", entry)
				if err != nil {
					run.Except(err, "channel_id", strconv.FormatInt(checkInfo.ChannelID, 10))
				}
			}
		}
	}
}

func (p *Plugin) checkEntry(run *common.Run, entry Entry, posts []geverse.ArtistPost) error {
	var posted int

	for _, post := range posts {
		post := post

		logger := run.Logger().With(
			zap.Uint("entry_id", entry.ID),
			zap.Int64("channel_id", entry.WeverseChannelID),
			zap.Int64("post_id", post.ID),
		)

		if posted > postsPerCheckLimit {
			// logger.Debug("skipping post because of the posts per check limit")
			break
		}

		if !post.CreatedAt.After(entry.CreatedAt) {
			// logger.Debug("skipping post because post date is not after entry creation date")
			continue
		}

		if time.Since(post.CreatedAt) > ageLimit {
			// logger.Debug("skipping post because of the age limit")
			continue
		}

		existingPost, err := postFind(p.db,
			"entry_id = ? AND post_id = ? AND post_type = ?",
			entry.ID, post.ID, postType,
		)
		if err != nil && !strings.Contains(err.Error(), "record not found") {
			return err
		}
		if existingPost != nil {
			// logger.Debug("skipping post because it has already been posted")
			continue
		}

		logger.Info("posting post")

		err = p.post(run, entry, &post)
		if err != nil {
			return errors.Wrap(err, "error posting")
		}
		posted++
	}

	return nil
}

func (p *Plugin) post(_ *common.Run, entry Entry, post *geverse.ArtistPost) error {
	var err error

	botID := entry.BotID
	if !entry.DM {
		botID, err = p.state.BotForChannel(
			entry.ChannelOrUserID,
			permissions.DiscordSendMessages,
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

	mediaURLs := splitURLs(extractMediaURLs(post.Photos))
	var mediaURLsFirst []string
	if len(mediaURLs) > 0 {
		mediaURLsFirst = mediaURLs[0]
	}

	channelID := entry.ChannelOrUserID
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
			Content: "weverse-artist.post.content",
		},
		"post", post, "entry", entry, "mediaURLs", mediaURLsFirst,
	)
	if err != nil {
		discord.CheckBlockDMChannel(p.redis, session, entry.ChannelOrUserID, err)
		return err
	}

	messageIDs := make([]string, len(messages))
	for i, message := range messages {
		messageIDs[i] = message.ID
	}

	if len(mediaURLs) > 1 {
		for _, mediaURLsLeftItem := range mediaURLs[1:] {
			messages, err := discord.SendComplexWithVars(
				session,
				p.Localizations(),
				channelID,
				&discordgo.MessageSend{
					Content: "weverse.post.leftover-links",
				},
				"mediaURLs", mediaURLsLeftItem,
			)
			if err != nil {
				discord.CheckBlockDMChannel(p.redis, session, entry.ChannelOrUserID, err)
				return err
			}

			for _, message := range messages {
				messageIDs = append(messageIDs, message.ID)
			}
		}
	}

	return postAdd(p.db, entry.ID, post.ID, messageIDs)
}

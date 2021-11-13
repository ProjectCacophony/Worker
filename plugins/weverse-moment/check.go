package weversemoment

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/Seklfreak/geverse"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/discord"
	"gitlab.com/Cacophony/go-kit/feed"
	"gitlab.com/Cacophony/go-kit/permissions"
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

	resetCache()

	var community *geverse.Community
	for checkInfo, entries := range bundles {
		community, err = p.geverse.GetCommunity(run.Context(), checkInfo.ChannelID)
		if err != nil {
			run.Except(err, "channel_id", strconv.FormatInt(checkInfo.ChannelID, 10))

			err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entries...)
			if err != nil {
				run.Except(err, "channel_id", strconv.FormatInt(checkInfo.ChannelID, 10))
			}
			continue
		}

		for _, entry := range entries {
			err = p.checkEntry(run, entry, community)
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

	resetCache()
}

func (p *Plugin) checkEntry(run *common.Run, entry Entry, community *geverse.Community) error {
	var posted int

	for _, artist := range community.Artists {
		artist := artist

		logger := run.Logger().With(
			zap.Uint("entry_id", entry.ID),
			zap.Int64("channel_id", entry.WeverseChannelID),
			zap.Int64("artist_id", artist.ID),
		)

		if artist.ToFanLastID == 0 {
			// logger.Debug("skipping artist because no moment")
			continue
		}

		existingPost, err := postFind(p.db,
			"entry_id = ? AND post_id = ? AND post_type = ?",
			entry.ID, artist.ToFanLastID, postType,
		)
		if err != nil && !strings.Contains(err.Error(), "record not found") {
			return err
		}
		if existingPost != nil {
			// logger.Debug("skipping post because it has already been posted")
			continue
		}

		moments, err := get(run.Context(), p.geverse, community.ID, artist.ID)
		if err != nil {
			return err
		}

		for _, moment := range moments {
			moment := moment

			logger := logger.With(zap.Int64("moment_id", moment.ID))

			if posted > postsPerCheckLimit {
				// logger.Debug("skipping post because of the posts per check limit")
				break
			}

			if !moment.CreatedAt.After(entry.CreatedAt) {
				// logger.Debug("skipping post because post date is not after entry creation date")
				continue
			}

			if time.Since(moment.CreatedAt) > ageLimit {
				// logger.Debug("skipping post because of the age limit")
				continue
			}

			existingPost, err := postFind(p.db,
				"entry_id = ? AND post_id = ? AND post_type = ?",
				entry.ID, moment.ID, postType,
			)
			if err != nil && !strings.Contains(err.Error(), "record not found") {
				return err
			}
			if existingPost != nil {
				// logger.Debug("skipping post because it has already been posted")
				continue
			}

			logger.Info("posting post")

			err = p.post(run, entry, &moment)
			if err != nil {
				return errors.Wrap(err, "error posting")
			}
			posted++
		}

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
			Content: "weverse-moment.post.content",
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

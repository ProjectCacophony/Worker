package tiktok

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gitlab.com/Cacophony/Worker/plugins/tiktok/client"
	"gitlab.com/Cacophony/go-kit/permissions"

	"gitlab.com/Cacophony/go-kit/feed"

	"github.com/bwmarrin/discordgo"

	"gitlab.com/Cacophony/go-kit/discord"

	"gitlab.com/Cacophony/Worker/plugins/common"
	"go.uber.org/zap"
)

const (
	ageLimit           = time.Hour * 24 * 3 // three days
	postsPerCheckLimit = 5
)

func (p *Plugin) checkBundles(run *common.Run, tx *sql.Tx, bundles boardCheckBundle) {
	var err error

	run.Logger().Info("checking bundles",
		zap.Int("amount", len(bundles)),
	)

	var posts []*client.Post
	for checkInfo, entries := range bundles {
		posts, err = p.tiktokClient.Posts(run.Context(), checkInfo.AccountID)
		if err != nil {
			run.Except(err, "account_id", checkInfo.AccountID)

			err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entries...)
			if err != nil {
				run.Except(err, "account_id", checkInfo.AccountID)
			}
			continue
		}

		for _, entry := range entries {
			err = p.checkEntry(run, tx, entry, posts)
			if err != nil {
				run.Except(err, "account_id", checkInfo.AccountID, "entry_id", strconv.Itoa(int(entry.ID)))

				err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entry)
				if err != nil {
					run.Except(err, "account_id", checkInfo.AccountID, "entry_id", strconv.Itoa(int(entry.ID)))
				}
			} else {
				err = checkSet(run.Context(), tx, feed.SuccessStatus, "", entry)
				if err != nil {
					run.Except(err, "account_id", checkInfo.AccountID, "entry_id", strconv.Itoa(int(entry.ID)))
				}
			}
		}
	}
}

func (p *Plugin) checkEntry(run *common.Run, tx *sql.Tx, entry Entry, posts []*client.Post) error {
	var posted int

	for _, post := range posts {
		logger := run.Logger().With(
			zap.Uint("entry_id", entry.ID),
			zap.String("account_id", entry.TikTokUsername),
			zap.String("post_id", post.ID),
		)

		if posted > postsPerCheckLimit {
			// logger.Debug("skipping post because of the posts per check limit")
			break
		}

		if !post.CreateTimeParsed.After(entry.CreatedAt) {
			// logger.Debug("skipping post because post date is not after entry creation date")
			continue
		}

		if time.Since(post.CreateTimeParsed) > ageLimit {
			// logger.Debug("skipping post because of the age limit")
			continue
		}

		existingPost, err := postFind(p.db,
			"entry_id = ? AND post_id = ?",
			entry.ID, post.ID,
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

func (p *Plugin) post(_ *common.Run, entry Entry, post *client.Post) error {
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

	channelID := entry.ChannelOrUserID
	if entry.DM {
		channelID, err = discord.DMChannel(p.redis, session, channelID)
		if err != nil {
			return errors.Wrap(err, "unable to create dm channel")
		}
	}

	messages, err := discord.SendComplexWithVars(
		session,
		p.Localizations(),
		channelID,
		&discordgo.MessageSend{
			Content: "tiktok.post.content",
		},
		"post", post, "entry", entry,
	)
	if err != nil {
		discord.CheckBlockDMChannel(p.redis, session, entry.ChannelOrUserID, err)
		return errors.Wrap(err, "unable to send main message")
	}

	messageIDs := make([]string, len(messages))
	for i, message := range messages {
		messageIDs[i] = message.ID
	}

	return postAdd(p.db, entry.ID, post.ID, messageIDs)
}

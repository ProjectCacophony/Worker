package instagramigtv

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Seklfreak/ginsta"

	"github.com/pkg/errors"

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

	var user *ginsta.User
	for checkInfo, entries := range bundles {
		user, err = p.ginsta.UserByUsername(run.Context(), checkInfo.Username)
		if err != nil {
			run.Except(err, "account_id", checkInfo.Username)

			err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entries...)
			if err != nil {
				run.Except(err, "account_id", checkInfo.Username)
			}
			continue
		}

		for _, entry := range entries {
			err = p.checkEntry(run, entry, user.Videos)
			if err != nil {
				run.Except(err, "account_id", checkInfo.Username)

				err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entry)
				if err != nil {
					run.Except(err, "account_id", checkInfo.Username)
				}
			} else {
				err = checkSet(run.Context(), tx, feed.SuccessStatus, "", entry)
				if err != nil {
					run.Except(err, "account_id", checkInfo.Username)
				}
			}
		}
	}
}

func (p *Plugin) checkEntry(run *common.Run, entry Entry, posts []*ginsta.Video) error {
	var posted int

	for _, post := range posts {
		logger := run.Logger().With(
			zap.Uint("entry_id", entry.ID),
			zap.String("account_id", entry.InstagramAccountID),
			zap.String("post_id", post.ID),
		)

		if posted > postsPerCheckLimit {
			// logger.Debug("skipping post because of the posts per check limit")
			break
		}

		if !post.TakenAt.After(entry.CreatedAt) {
			// logger.Debug("skipping post because post date is not after entry creation date")
			continue
		}

		if time.Since(post.TakenAt) > ageLimit {
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

func (p *Plugin) post(_ *common.Run, entry Entry, post *ginsta.Video) error {
	var err error

	botID := entry.BotID
	if !entry.DM {
		botID, err = p.state.BotForGuild(entry.GuildID)
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

	url := fmt.Sprintf("https://instagram.com/p/%s/", post.Shortcode)

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
			Content: "instagram-igtv.post.content",
		},
		"post", post, "entry", entry, "url", url,
	)
	if err != nil {
		return err
	}

	messageIDs := make([]string, len(messages))
	for i, message := range messages {
		messageIDs[i] = message.ID
	}

	return postAdd(p.db, entry.ID, post.ID, messageIDs)
}

package instagram

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Seklfreak/ginsta"
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

func (p *Plugin) checkBundles(run *common.Run, tx *sql.Tx, bundles boardCheckBundle) {
	var err error

	run.Logger().Info("checking bundles",
		zap.Int("amount", len(bundles)),
	)

	var posts []*ginsta.Post
	for checkInfo, entries := range bundles {
		posts, err = p.ginsta.PostsByID(run.Context(), checkInfo.AccountID)
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

func (p *Plugin) checkEntry(run *common.Run, tx *sql.Tx, entry Entry, posts []*ginsta.Post) error {
	var posted int
	var updatedInstagramUsername bool

	for _, post := range posts {
		// update stored instagram username if required
		if post.AuthorID == entry.InstagramAccountID &&
			post.AuthorUsername != "" &&
			post.AuthorUsername != entry.InstagramUsername &&
			!updatedInstagramUsername {

			err := updateInstagramUsername(run.Context(), tx, entry.ID, post.AuthorUsername)
			if err != nil {
				return err
			}
			updatedInstagramUsername = true
		}

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

func (p *Plugin) post(_ *common.Run, entry Entry, post *ginsta.Post) error {
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

	mediaURLsFirst := post.MediaURLs
	var mediaURLsLeft [][]string
	if len(mediaURLsFirst) > 5 {
		// divide > 5 links into chunks of five
		toDivide := mediaURLsFirst[5:]
		chunkSize := 5
		for i := 0; i < len(toDivide); i += chunkSize {
			end := i + chunkSize

			if end > len(toDivide) {
				end = len(toDivide)
			}

			mediaURLsLeft = append(mediaURLsLeft, toDivide[i:end])
		}

		mediaURLsFirst = mediaURLsFirst[0:5]
	}

	url := fmt.Sprintf("https://instagram.com/p/%s/", post.Shortcode)

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
			Content: "instagram.post.content",
		},
		"post", post, "entry", entry, "url", url, "mediaURLs", mediaURLsFirst,
	)
	if err != nil {
		discord.CheckBlockDMChannel(p.redis, session, entry.ChannelOrUserID, err)
		return errors.Wrap(err, "unable to send main message")
	}

	messageIDs := make([]string, len(messages))
	for i, message := range messages {
		messageIDs[i] = message.ID
	}

	for _, mediaURLsLeftItem := range mediaURLsLeft {
		messages, err := discord.SendComplexWithVars(
			session,
			p.Localizations(),
			channelID,
			&discordgo.MessageSend{
				Content: "instagram.post.leftover-links",
			},
			"mediaURLs", mediaURLsLeftItem,
		)
		if err != nil {
			discord.CheckBlockDMChannel(p.redis, session, entry.ChannelOrUserID, err)
			return errors.Wrap(err, "unable to send media url message")
		}

		for _, message := range messages {
			messageIDs = append(messageIDs, message.ID)
		}
	}

	return postAdd(p.db, entry.ID, post.ID, messageIDs)
}

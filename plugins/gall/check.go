package gall

import (
	"database/sql"
	"strings"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/Cacophony/go-kit/feed"
	"gitlab.com/Cacophony/go-kit/permissions"

	"github.com/bwmarrin/discordgo"

	"gitlab.com/Cacophony/go-kit/discord"

	"github.com/Seklfreak/ginside"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"go.uber.org/zap"
)

const (
	color              = 4220112            // #4064D0
	ageLimit           = time.Hour * 24 * 3 // three days
	postsPerCheckLimit = 5
)

func (p *Plugin) checkBundles(run *common.Run, tx *sql.Tx, bundles boardCheckBundle) {
	var err error

	run.Logger().Info("checking bundles",
		zap.Int("amount", len(bundles)),
	)

	var posts []ginside.Post
	for checkInfo, entries := range bundles {
		if !checkInfo.Minor {
			posts, err = p.gall.BoardPosts(run.Context(), checkInfo.BoardID, checkInfo.Recommended)
			if err != nil {
				run.Except(err, "board_id", checkInfo.BoardID)

				err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entries...)
				if err != nil {
					run.Except(err, "board_id", checkInfo.BoardID)
				}
				continue
			}
		} else {
			posts, err = p.gall.BoardMinorPosts(run.Context(), checkInfo.BoardID, checkInfo.Recommended)
			if err != nil {
				run.Except(err, "board_id", checkInfo.BoardID)

				err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entries...)
				if err != nil {
					run.Except(err, "board_id", checkInfo.BoardID)
				}
				continue
			}
		}

		for _, entry := range entries {
			err = p.checkEntry(run, entry, posts)
			if err != nil {
				run.Except(err, "board_id", checkInfo.BoardID)

				err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entry)
				if err != nil {
					run.Except(err, "board_id", checkInfo.BoardID)
				}
			} else {
				err = checkSet(run.Context(), tx, feed.SuccessStatus, "", entry)
				if err != nil {
					run.Except(err, "board_id", checkInfo.BoardID)
				}
			}
		}
	}
}

func (p *Plugin) checkEntry(run *common.Run, entry Entry, posts []ginside.Post) error {
	var posted int

	for _, post := range posts {
		logger := run.Logger().With(
			zap.Uint("entry_id", entry.ID),
			zap.String("board_id", entry.BoardID),
			zap.Bool("board_recommended", entry.Recommended),
			zap.Bool("board_minor", entry.MinorGallery),
			zap.String("post_id", post.ID),
		)

		if posted > postsPerCheckLimit {
			// logger.Debug("skipping post because of the posts per check limit")
			break
		}

		if !post.Date.After(entry.CreatedAt) {
			// logger.Debug("skipping post because post date is not after entry creation date")
			continue
		}

		if time.Since(post.Date) > ageLimit {
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

func (p *Plugin) post(_ *common.Run, entry Entry, post ginside.Post) error {
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
			Content: "gall.post.content",
			Embed: &discordgo.MessageEmbed{
				URL:         post.URL,
				Title:       "gall.post.embed.title",
				Description: "gall.post.embed.description",
				Timestamp:   post.Date.Format(time.RFC3339),
				Color:       color,
				Footer: &discordgo.MessageEmbedFooter{
					Text:    "gall.post.embed.footer.text",
					IconURL: "gall.post.embed.footer.icon-url",
				},
			},
		},
		"post", post, "board", entry,
	)
	if err != nil {
		discord.CheckBlockDMChannel(p.redis, session, channelID, err)
		return err
	}

	messageIDs := make([]string, len(messages))
	for i, message := range messages {
		messageIDs[i] = message.ID
	}

	return postAdd(p.db, entry.ID, post.ID, messageIDs)
}

package gall

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"gitlab.com/Cacophony/go-kit/discord"

	"github.com/Seklfreak/ginside"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"go.uber.org/zap"
)

const (
	color              = 4220112
	ageLimit           = time.Hour * 24 * 3 // three days
	postsPerCheckLimit = 5
)

func (p *Plugin) checkBundles(run *common.Run, bundles boardCheckBundle) {
	var err error

	run.Logger().Info("checking bundles",
		zap.Int("amount", len(bundles)),
	)

	var posts []ginside.Post
	for checkInfo, entries := range bundles {
		if !checkInfo.Minor {
			posts, err = p.gall.BoardPosts(run.Context(), checkInfo.BoardID, checkInfo.Recommended)
			if err != nil {
				run.Except(err)
				continue
			}
		} else {
			posts, err = p.gall.BoardMinorPosts(run.Context(), checkInfo.BoardID, checkInfo.Recommended)
			if err != nil {
				run.Except(err)
				continue
			}
		}

		for _, entry := range entries {
			p.checkEntry(run, entry, posts)
		}
	}
}

func (p *Plugin) checkEntry(run *common.Run, entry Entry, posts []ginside.Post) {
	var posted int

	for _, post := range posts {
		logger := run.Logger().With(
			zap.String("board_id", entry.BoardID),
			zap.Bool("board_recommended", entry.Recommended),
			zap.Bool("board_minor", entry.MinorGallery),
			zap.String("post_id", post.ID),
		)

		if posted > postsPerCheckLimit {
			logger.Debug("skipping post because of the posts per check limit")
			break
		}

		if !post.Date.After(entry.CreatedAt) {
			logger.Debug("skipping post because post date is not after entry creation date")
			continue
		}

		if time.Since(post.Date) > ageLimit {
			logger.Debug("skipping post because of the age limit")
			continue
		}

		existingPost, err := postFind(p.db,
			"entry_id = ? AND post_id = ?",
			entry.ID, post.ID,
		)
		if err != nil && !strings.Contains(err.Error(), "record not found") {
			logger.Debug("skipping post because of error",
				zap.Error(err),
			)
			continue
		}
		if existingPost != nil {
			logger.Debug("skipping post because it has already been posted")
			continue
		}

		logger.Info("posting post")

		err = p.post(run, entry, post)
		if err != nil {
			run.Except(err)
		}
		posted++
	}
}

func (p *Plugin) post(_ *common.Run, entry Entry, post ginside.Post) error {

	botID, err := p.state.BotForGuild(entry.GuildID)
	if err != nil {
		return err
	}

	session, err := discord.NewSession(p.tokens, botID)
	if err != nil {
		return err
	}

	messages, err := discord.SendComplexWithVars(
		session,
		p.Localisations(),
		entry.ChannelID,
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
		return err
	}

	messageIDs := make([]string, len(messages))
	for i, message := range messages {
		messageIDs[i] = message.ID
	}

	return postAdd(p.db, entry.ID, post.ID, messageIDs)
}

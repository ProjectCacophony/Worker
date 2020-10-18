package vlive

import (
	vlive_go "github.com/Seklfreak/vlive-go"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"gitlab.com/Cacophony/go-kit/discord"
	"gitlab.com/Cacophony/go-kit/permissions"
	"go.uber.org/zap"
)

func (p *Plugin) handleVideo(video *vlive_go.Video) error {
	entries, err := findEntriesForChannel(p.db, video.ChannelId)
	if err != nil {
		return errors.Wrap(err, "unable to find entries for channel")
	}

	for _, entry := range entries {
		err = p.postVideo(video, entry)
		if err != nil {
			p.logger.Error("failure posting video to entry", zap.Error(err), zap.String("video_seq", video.Seq), zap.Uint("entry_id", entry.ID))
		}
	}

	return nil
}

func (p *Plugin) postVideo(video *vlive_go.Video, entry *Entry) error {
	if entry.ID <= 0 || video.Seq == "" {
		return errors.New("missing fields on video or entry")
	}

	var count int
	err := p.db.Model(Post{}).Where("entry_id = ? AND post_id = ?", entry.ID, video.Seq).Count(&count).Error
	if err != nil {
		p.logger.Error("failure counting posts for entry", zap.Error(err), zap.String("video_seq", video.Seq), zap.Uint("entry_id", entry.ID))
		return nil
	}
	if count > 0 {
		return nil
	}

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
			Content: "vlive.post.content",
		},
		"video", video, "entry", entry,
	)
	if err != nil {
		discord.CheckBlockDMChannel(p.redis, session, entry.ChannelOrUserID, err)
		return errors.Wrap(err, "unable to send main message")
	}

	messageIDs := make([]string, len(messages))
	for i, message := range messages {
		messageIDs[i] = message.ID
	}

	err = p.db.Create(&Post{
		EntryID:    entry.ID,
		PostID:     video.Seq,
		MessageIDs: messageIDs,
	}).Error
	if err != nil {
		return errors.Wrap(err, "failure storing post for entry")
	}

	return nil
}

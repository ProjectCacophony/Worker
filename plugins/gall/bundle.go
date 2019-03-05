package gall

import (
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/discord"
	"go.uber.org/zap"
)

type boardCheckBundleInfo struct {
	BoardID     string
	Minor       bool
	Recommended bool
}

type boardCheckBundle map[boardCheckBundleInfo][]Entry

func (p *Plugin) bundleEntries(run *common.Run, entries []Entry) boardCheckBundle {
	var err error

	run.Logger().Debug("bundling entries",
		zap.Int("amount", len(entries)),
	)

	// don't check channels that don't exist anymore, or without necessary permissions
	bundledEntries := make(map[boardCheckBundleInfo][]Entry)
	var addedToBundle bool
	for _, entry := range entries {
		logger := run.Logger().With(zap.Uint("id", entry.ID))

		if !entry.DM {

			_, err = p.state.Channel(entry.ChannelID)
			if err != nil {
				logger.Debug("skipped entry because of channel state error",
					zap.Error(err),
				)
				continue
			}

			botID, err := p.state.BotForGuild(entry.GuildID)
			if err != nil {
				logger.Debug("skipped entry because of bot for guild error",
					zap.Error(err),
				)
				continue
			}
			if !discord.UserHasPermission(p.state, botID, entry.ChannelID,
				discordgo.PermissionSendMessages,
				discordgo.PermissionEmbedLinks,
			) {
				logger.Debug("skipped entry because of user permissions error",
					zap.Error(errors.New("missing required permissions")),
				)
				continue
			}

		} else {

			_, err = p.state.User(entry.ChannelID)
			if err != nil {
				logger.Debug("skipped entry because of user state error",
					zap.Error(err),
				)
				continue
			}
		}

		// bundle feed entry if everything is good
		addedToBundle = false

		for key := range bundledEntries {
			if key.BoardID != entry.BoardID ||
				key.Minor != entry.MinorGallery ||
				key.Recommended != entry.Recommended {
				continue
			}
			bundledEntries[key] = append(bundledEntries[key], entry)
			addedToBundle = true
		}

		if !addedToBundle {
			bundledEntries[boardCheckBundleInfo{
				BoardID:     entry.BoardID,
				Minor:       entry.MinorGallery,
				Recommended: entry.Recommended,
			}] = []Entry{entry}
		}
	}

	return bundledEntries
}

package rss

import (
	"errors"

	"github.com/bwmarrin/discordgo"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/discord"
	"go.uber.org/zap"
)

type boardCheckBundleInfo struct {
	FeedURL string
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

		_, err = p.state.Channel(entry.ChannelID)
		if err != nil {
			logger.Debug("skipped entry because of error",
				zap.Error(err),
			)
			continue
		}

		botID, err := p.state.BotForGuild(entry.GuildID)
		if err != nil {
			logger.Debug("skipped entry because of error",
				zap.Error(err),
			)
			continue
		}
		if !discord.UserHasPermission(p.state, botID, entry.ChannelID,
			discordgo.PermissionSendMessages,
			discordgo.PermissionEmbedLinks,
		) {
			logger.Debug("skipped entry because of error",
				zap.Error(errors.New("missing required permissions")),
			)
			continue
		}

		// bundle feed entry if everything is good
		addedToBundle = false

		for key := range bundledEntries {
			if key.FeedURL != entry.FeedURL {
				continue
			}
			bundledEntries[key] = append(bundledEntries[key], entry)
			addedToBundle = true
		}

		if !addedToBundle {
			bundledEntries[boardCheckBundleInfo{
				FeedURL: entry.FeedURL,
			}] = []Entry{entry}
		}
	}

	return bundledEntries
}

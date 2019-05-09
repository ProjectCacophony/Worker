package instagram

import (
	"database/sql"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/discord"
	"gitlab.com/Cacophony/go-kit/feed"
	"go.uber.org/zap"
)

type boardCheckBundleInfo struct {
	AccountID string
}

type boardCheckBundle map[boardCheckBundleInfo][]Entry

func (p *Plugin) bundleEntries(run *common.Run, tx *sql.Tx, entries []Entry) boardCheckBundle {
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

			_, err = p.state.Channel(entry.ChannelOrUserID)
			if err != nil {
				logger.Debug("skipped entry because of channel state error",
					zap.Error(err),
				)

				err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entry)
				if err != nil {
					run.Except(err, "entry_id", strconv.FormatUint(uint64(entry.ID), 10))
				}
				continue
			}

			botID, err := p.state.BotForGuild(entry.GuildID)
			if err != nil {
				logger.Debug("skipped entry because of bot for guild error",
					zap.Error(err),
				)

				err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entry)
				if err != nil {
					run.Except(err, "entry_id", strconv.FormatUint(uint64(entry.ID), 10))
				}
				continue
			}
			if !discord.UserHasPermission(p.state, botID, entry.ChannelOrUserID,
				discordgo.PermissionSendMessages,
				discordgo.PermissionEmbedLinks,
			) {
				logger.Debug("skipped entry because of user permissions error",
					zap.Error(errors.New("missing required permissions")),
				)

				err = checkSet(run.Context(), tx, feed.ErrorStatus, "missing permissions", entry)
				if err != nil {
					run.Except(err, "entry_id", strconv.FormatUint(uint64(entry.ID), 10))
				}
				continue
			}

		} else {

			_, err = p.state.User(entry.ChannelOrUserID)
			if err != nil {
				logger.Debug("skipped entry because of user state error",
					zap.Error(err),
				)

				err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entry)
				if err != nil {
					run.Except(err, "entry_id", strconv.FormatUint(uint64(entry.ID), 10))
				}
				continue
			}
		}

		// bundle feed entry if everything is good
		addedToBundle = false

		for key := range bundledEntries {
			if key.AccountID != entry.InstagramAccountID {
				continue
			}
			bundledEntries[key] = append(bundledEntries[key], entry)
			addedToBundle = true
		}

		if !addedToBundle {
			bundledEntries[boardCheckBundleInfo{
				AccountID: entry.InstagramAccountID,
			}] = []Entry{entry}
		}
	}

	return bundledEntries
}

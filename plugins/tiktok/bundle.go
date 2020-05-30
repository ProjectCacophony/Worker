package tiktok

import (
	"database/sql"
	"strconv"

	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/feed"
	"gitlab.com/Cacophony/go-kit/permissions"
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

			_, err = p.state.BotForChannel(
				entry.ChannelOrUserID,
				permissions.DiscordSendMessages,
			)
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
			if key.AccountID != entry.TikTokUsername {
				continue
			}
			bundledEntries[key] = append(bundledEntries[key], entry)
			addedToBundle = true
		}

		if !addedToBundle {
			bundledEntries[boardCheckBundleInfo{
				AccountID: entry.TikTokUsername,
			}] = []Entry{entry}
		}
	}

	return bundledEntries
}

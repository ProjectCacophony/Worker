package eventlogauditlog

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"go.uber.org/zap"
)

type bundleInfo struct {
	GuildID    string
	ActionType actionType
	BotID      string
}

type boardCheckBundle map[bundleInfo][]Item

func (p *Plugin) bundleEntries(run *common.Run, entries []Item) boardCheckBundle {
	run.Logger().Debug("bundling entries",
		zap.Int("amount", len(entries)),
	)

	bundledEntries := make(map[bundleInfo][]Item)
	var addedToBundle bool
	var botID string
	var err error
	for _, entry := range entries {
		botID, err = p.state.BotForGuild(entry.GuildID, discordgo.PermissionViewAuditLogs)
		if err != nil {
			run.Except(err, "entry_id", strconv.Itoa(int(entry.ID)))
			continue
		}

		addedToBundle = false

		for key := range bundledEntries {
			if key.GuildID != entry.GuildID || key.ActionType != entry.ActionType || key.BotID != botID {
				continue
			}
			bundledEntries[key] = append(bundledEntries[key], entry)
			addedToBundle = true
		}

		if !addedToBundle {
			bundledEntries[bundleInfo{
				GuildID:    entry.GuildID,
				ActionType: entry.ActionType,
				BotID:      botID,
			}] = []Item{entry}
		}
	}

	return bundledEntries
}

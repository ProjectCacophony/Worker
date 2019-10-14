package eventlogauditlog

import (
	"database/sql"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"gitlab.com/Cacophony/Worker/plugins/common"
	"gitlab.com/Cacophony/go-kit/discord"
	"go.uber.org/zap"
)

func (p *Plugin) checkBundles(run *common.Run, tx *sql.Tx, bundles boardCheckBundle) []Item {
	var updated []Item

	run.Logger().Info("checking bundles",
		zap.Int("amount", len(bundles)),
	)

	var limit int
	var changed bool
	for info, bundle := range bundles {
		session, err := discord.NewSession(p.tokens, info.BotID)
		if err != nil {
			run.Except(err, "guild_id", info.GuildID, "action_type", string(info.ActionType), "bot_id", info.BotID)
			continue
		}

		limit = len(bundle) + 5 // buffer
		if limit > 100 {
			limit = 100
		}

		var auditLogActionTypes []int
		switch info.ActionType {
		case "discord_ban":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionMemberBanAdd)
		case "discord_unban":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionMemberBanRemove)
		case "discord_leave":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionMemberBanAdd, discordgo.AuditLogActionMemberKick)
		}

		if len(auditLogActionTypes) <= 0 {
			run.Except(fmt.Errorf("unsupported action type: %s", info.ActionType))
		}

		var bigAuditLog discordgo.GuildAuditLog
		for _, auditLogActionType := range auditLogActionTypes {
			resp, err := session.Client.GuildAuditLog(info.GuildID, "", "", auditLogActionType, limit)
			if err != nil {
				run.Except(err, "guild_id", info.GuildID, "action_type", string(info.ActionType), "bot_id", info.BotID)
				continue
			}

			bigAuditLog.AuditLogEntries = append(bigAuditLog.AuditLogEntries, resp.AuditLogEntries...)
			bigAuditLog.Users = append(bigAuditLog.Users, resp.Users...)
			bigAuditLog.Webhooks = append(bigAuditLog.Webhooks, resp.Webhooks...)
		}

		for _, item := range bundle {
			changed = p.handleEntry(run, tx, item, &bigAuditLog)
			if changed {
				updated = append(updated, item)
			}
		}
	}

	return updated
}

func (p *Plugin) handleEntry(run *common.Run, tx *sql.Tx, item Item, auditlog *discordgo.GuildAuditLog) (changed bool) {
	var err error

	for i, entry := range auditlog.AuditLogEntries {

		switch item.ActionType {
		case "discord_ban":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionMemberBanAdd) {
				if entry.Reason != "" {
					err = addReason(run.Context(), tx, item.ID, entry.Reason)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_unban":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionMemberBanRemove) {
				if entry.Reason != "" {
					err = addReason(run.Context(), tx, item.ID, entry.Reason)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_leave":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionMemberBanAdd) {
				if entry.Reason != "" {
					err = addReason(run.Context(), tx, item.ID, entry.Reason)
					if err != nil {
						run.Except(err)
					}
				}

				err = addItemOption(run.Context(), tx, item.ID, "leave_type", "", "ban", "text")
				if err != nil {
					run.Except(err)
				}
			}

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionMemberKick) {
				if entry.Reason != "" {
					err = addReason(run.Context(), tx, item.ID, entry.Reason)
					if err != nil {
						run.Except(err)
					}
				}

				err = addItemOption(run.Context(), tx, item.ID, "leave_type", "", "kick", "text")
				if err != nil {
					run.Except(err)
				}
			}

			changed = true
		}

	}

	return
}

func matchesTarget(auditlog *discordgo.GuildAuditLog, i int, item Item, auditLogType int) bool {
	entry := auditlog.AuditLogEntries[i]

	if entry.ActionType != auditLogType {
		return false
	}

	if entry.TargetID != item.TargetValue {
		return false
	}

	entryTime, err := discord.TimeFromID(entry.ID)
	if err != nil {
		return false
	}

	deltaDuration := item.CreatedAt.Sub(*entryTime).Seconds()
	if deltaDuration > 3 || deltaDuration < -3 {
		return false
	}

	return true
}

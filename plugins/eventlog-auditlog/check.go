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
		case "discord_channel_create":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionChannelCreate)
		case "discord_role_create":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionRoleCreate)
		case "discord_guild_update":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionGuildUpdate)
		case "discord_member_update":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionMemberUpdate)
		case "discord_channel_update":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionChannelUpdate)
		case "discord_channel_delete":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionChannelDelete)
		case "discord_role_update":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionRoleUpdate)
		case "discord_role_delete":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionRoleDelete)
		case "discord_emoji_create":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionEmojiCreate)
		case "discord_emoji_update":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionEmojiUpdate)
		case "discord_emoji_delete":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionEmojiDelete)
		case "discord_webhook_create":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionWebhookCreate)
		case "discord_webhook_update":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionWebhookUpdate)
		case "discord_webhook_delete":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionWebhookDelete)
		case "discord_invite_update":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionInviteUpdate)
		case "discord_invite_delete":
			auditLogActionTypes = append(auditLogActionTypes, discordgo.AuditLogActionInviteDelete)
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
			changed = p.handleEntry(run, tx, info.BotID, item, &bigAuditLog)
			if changed {
				updated = append(updated, item)
			}
		}
	}

	return updated
}

func (p *Plugin) handleEntry(run *common.Run, tx *sql.Tx, botID string, item Item, auditlog *discordgo.GuildAuditLog) (changed bool) {
	var err error

	for i, entry := range auditlog.AuditLogEntries {

		switch item.ActionType {
		case "discord_ban":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionMemberBanAdd) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_unban":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionMemberBanRemove) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_leave":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionMemberBanAdd) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				err = addItemOption(run.Context(), tx, item.ID, "leave_type", "", "ban", "text", botID)
				if err != nil {
					run.Except(err)
				}
			}

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionMemberKick) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				err = addItemOption(run.Context(), tx, item.ID, "leave_type", "", "kick", "text", botID)
				if err != nil {
					run.Except(err)
				}
			}

			changed = true

		case "discord_channel_create":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionChannelCreate) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_role_create":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionRoleCreate) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_guild_update":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionGuildUpdate) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_member_update":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionMemberUpdate) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_channel_update":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionChannelUpdate) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_channel_delete":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionChannelDelete) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_role_update":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionRoleUpdate) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_role_delete":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionRoleDelete) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_emoji_create":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionEmojiCreate) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_emoji_update":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionEmojiUpdate) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_emoji_delete":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionEmojiDelete) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_webhook_create":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionWebhookCreate) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_webhook_update":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionWebhookUpdate) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_webhook_delete":

			if matchesTarget(auditlog, i, item, discordgo.AuditLogActionWebhookDelete) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}
				at, err := discordgo.SnowflakeTimestamp(entry.ID)
				if err != nil {
					run.Except(err)
				}
				if !at.IsZero() {
					err = setCreatedAt(run.Context(), tx, item.ID, at)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_invite_update":

			if matchesInvite(auditlog, i, item, discordgo.AuditLogActionInviteUpdate) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}
				at, err := discordgo.SnowflakeTimestamp(entry.ID)
				if err != nil {
					run.Except(err)
				}
				if !at.IsZero() {
					err = setCreatedAt(run.Context(), tx, item.ID, at)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

		case "discord_invite_delete":

			if matchesInvite(auditlog, i, item, discordgo.AuditLogActionInviteDelete) {
				if entry.Reason != "" {
					err = addItemOption(run.Context(), tx, item.ID, "reason", "", entry.Reason, "text", botID)
					if err != nil {
						run.Except(err)
					}
				}
				if entry.UserID != "" {
					err = setAuthor(run.Context(), tx, item.ID, entry.UserID)
					if err != nil {
						run.Except(err)
					}
				}
				at, err := discordgo.SnowflakeTimestamp(entry.ID)
				if err != nil {
					run.Except(err)
				}
				if !at.IsZero() {
					err = setCreatedAt(run.Context(), tx, item.ID, at)
					if err != nil {
						run.Except(err)
					}
				}

				changed = true
			}

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
	if deltaDuration > 1 || deltaDuration < -5 {
		return false
	}

	return true
}

func matchesInvite(auditlog *discordgo.GuildAuditLog, i int, item Item, auditLogType int) bool {
	entry := auditlog.AuditLogEntries[i]

	if entry.ActionType != auditLogType {
		return false
	}

	for _, change := range entry.Changes {
		if change.Key != "code" {
			continue
		}

		if change.OldValue == item.TargetValue ||
			change.NewValue == item.TargetValue {
			return true
		}
	}

	return false
}

package serverlist

import (
	"database/sql"

	"github.com/bwmarrin/discordgo"

	"github.com/pkg/errors"

	"gitlab.com/Cacophony/go-kit/discord"

	"gitlab.com/Cacophony/Worker/plugins/common"
	"go.uber.org/zap"
)

func (p *Plugin) checkEntries(run *common.Run, tx *sql.Tx, entries []Server) {
	var err error

	run.Logger().Info("checking servers",
		zap.Int("amount", len(entries)),
	)

	for _, entry := range entries {
		logger := p.logger.With(
			zap.Uint("entry_id", entry.ID),
			zap.String("entry_guild_id", entry.GuildID),
			zap.String("entry_invite_code", entry.InviteCode),
		)

		err = p.checkEntry(run, tx, entry)
		if err != nil {
			logger.Error("failure checking entry", zap.Error(err))
			continue
		}

	}
}

func (p *Plugin) checkEntry(run *common.Run, tx *sql.Tx, entry Server) error {
	session, err := discord.NewSession(p.tokens, entry.BotID)
	if err != nil {
		return errors.Wrap(err, "cannot create Discord Session")
	}

	invite, err := discord.Invite(
		p.redis,
		session,
		entry.InviteCode,
	)
	if err != nil {
		if errD, ok := err.(*discordgo.RESTError); ok &&
			errD != nil && errD.Message != nil &&
			errD.Message.Code == discordgo.ErrCodeUnknownInvite {

			err = setExpired(run.Context(), tx, entry.ID)
			if err != nil {
				return errors.Wrap(err, "cannot set order as expired")
			}

			return p.sendExpiredEvent(entry)
		}
		return errors.Wrap(err, "cannot get Invite from the Discord API")
	}

	if invite == nil ||
		invite.Code != entry.InviteCode {
		return errors.New("received invalid Invite from the Discord API")
	}

	if invite.ApproximateMemberCount > 0 {
		err = setMemberCount(run.Context(), tx, entry.ID, invite.ApproximateMemberCount)
		if err != nil {
			return errors.Wrap(err, "cannot update member count")
		}
	}

	return nil
}

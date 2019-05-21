package serverlist

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"gitlab.com/Cacophony/go-kit/events"
)

func (p *Plugin) sendExpiredEvent(server Server) error {
	event, err := events.New(events.CacophonyServerlistServerExpire)
	if err != nil {
		return errors.Wrap(err, "cannot generate new event")
	}

	event.ServerlistServerExpire = &events.ServerlistServerExpire{
		ID:         server.ID,
		GuildID:    server.GuildID,
		InviteCode: server.InviteCode,
		BotID:      server.BotID,
	}
	event.BotUserID = server.BotID

	err, recoverable := p.publisher.Publish(context.TODO(), event)
	if err != nil {
		if !recoverable {
			p.logger.Fatal("unrecoverable publishing error, shutting down",
				zap.Error(err),
			)
		}
		return errors.Wrap(err, "cannot publish event")
	}

	return nil
}

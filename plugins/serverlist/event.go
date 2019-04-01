package serverlist

import (
	"encoding/json"

	"github.com/pkg/errors"

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

	routingKey := events.GenerateRoutingKey(event.Type)

	body, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "cannot marshal event")
	}

	err = p.publisher.Publish(
		routingKey,
		body,
	)
	if err != nil {
		return errors.Wrap(err, "cannot publish event")
	}

	return nil
}

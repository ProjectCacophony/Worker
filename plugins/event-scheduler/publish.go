package eventscheduler

import (
	"strconv"

	"gitlab.com/Cacophony/Worker/plugins/common"
	"go.uber.org/zap"
)

func (p *Plugin) publishEvents(
	run *common.Run,
	events []Entry,
) {
	var err error
	var l *zap.Logger

	for _, event := range events {
		l = run.Logger().With(zap.Uint("event_id", event.ID))

		l.Debug("publishing event")

		err = p.publishEvent(run, event)
		if err != nil {
			l.Error("error publish event", zap.Error(err))
			run.Except(err, "event_id", strconv.FormatUint(uint64(event.ID), 10))
		}
	}
}

func (p *Plugin) publishEvent(
	run *common.Run,
	event Entry,
) error {
	return p.publisher.PublishRaw(run.Context(), event.Body)
}

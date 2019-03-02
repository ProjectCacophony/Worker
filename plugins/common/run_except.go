package common

import (
	"github.com/bwmarrin/discordgo"
	raven "github.com/getsentry/raven-go"
	"gitlab.com/Cacophony/go-kit/state"
	"go.uber.org/zap"
)

func (r *Run) Except(err error) {
	if err == nil {
		return
	}

	doLog := true

	if ignoreError(err) {
		doLog = false
	}

	if doLog {
		r.Logger().Error("error occurred while executing run", zap.Error(err))

		if raven.DefaultClient != nil {
			raven.CaptureError(
				err,
				map[string]string{
					"plugin": r.Plugin,
					"launch": r.Launch.String(),
				},
			)
		}
	}
}

func ignoreError(err error) bool {
	if err == nil {
		return true
	}

	// discord permission errors
	if errD, ok := err.(*discordgo.RESTError); ok && errD != nil && errD.Message != nil {
		if errD.Message.Code == discordgo.ErrCodeMissingPermissions ||
			errD.Message.Code == discordgo.ErrCodeMissingAccess ||
			errD.Message.Code == discordgo.ErrCodeCannotSendMessagesToThisUser {
			return true
		}
	}

	// state errors
	if err == state.ErrStateNotFound ||
		err == state.ErrTargetWrongServer ||
		err == state.ErrTargetWrongType ||
		err == state.ErrUserNotFound ||
		err == state.ErrChannelNotFound ||
		err == state.ErrRoleNotFound {
		return true
	}

	return false
}

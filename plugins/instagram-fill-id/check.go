package instagramfillid

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/jinzhu/gorm"
	"gitlab.com/Cacophony/go-kit/feed"

	"gitlab.com/Cacophony/Worker/plugins/common"
	"go.uber.org/zap"
)

func (p *Plugin) checkBundles(run *common.Run, tx *sql.Tx, bundles checkBundles) {
	run.Logger().Info("checking bundles",
		zap.Int("amount", len(bundles)),
	)

	for checkInfo, entries := range bundles {
		id, err := p.findID(run, tx, checkInfo.Username)
		if err != nil {
			run.Except(err, "username", checkInfo.Username)

			err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entries...)
			if err != nil {
				run.Except(err, "username", checkInfo.Username)
			}
			continue
		}

		for _, entry := range entries {
			if id != "" {
				err = setInstagramID(run.Context(), tx, entry.ID, id)
				if err != nil {
					run.Except(err, "username", checkInfo.Username, "entry_id", strconv.Itoa(int(entry.ID)))

					err = checkSet(run.Context(), tx, feed.ErrorStatus, err.Error(), entry)
					if err != nil {
						run.Except(err, "username", checkInfo.Username, "entry_id", strconv.Itoa(int(entry.ID)))
					}
				}
			} else {
				err = deleteEntry(run.Context(), tx, entry.ID)
				if err != nil {
					run.Except(err, "username", checkInfo.Username, "entry_id", strconv.Itoa(int(entry.ID)))
				}
			}
		}
	}
}

func (p *Plugin) findID(run *common.Run, tx *sql.Tx, username string) (string, error) {
	logger := run.Logger().With(zap.String("username", username))

	logger.Debug("looking up instagram account id via service db")
	userID, err := findIDByUsername(p.db, username)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error("failure finding instagram account id via service db", zap.Error(err))
		}
	} else {
		logger.Debug("found instagram account id via service db")
		return userID, nil
	}

	logger.Debug("looking up instagram account id via instagram api")
	user, err := p.ginsta.UserByUsername(run.Context(), username)
	if err != nil {
		if strings.Contains(err.Error(), "unexpected status code: 404") {
			return "", nil
		}

		logger.Error("failure finding instagram account id via instagram api", zap.Error(err))
	} else {
		logger.Debug("found instagram account id via instagram api")
		return user.ID, nil
	}

	return "", errors.New("unable to lookup account id")
}

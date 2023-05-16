package models

import (
	"fmt"
	"time"

	"github.com/inovex/CalendarSync/internal/config"
)

// TODO: @ljarosch: This might just be a quick-fix. I'm not yet sure if this is an ideal solution

type TimeIdentifier string

const (
	MonthStart TimeIdentifier = "MonthStart"
	MonthEnd   TimeIdentifier = "MonthEnd"
)

func TimeFromConfig(syncTime config.SyncTime) (time.Time, error) {
	now := time.Now()
	curYear, curMonth, _ := now.Date()
	curLocation := now.Location()

	var timeCfg time.Time

	switch TimeIdentifier(syncTime.Identifier) {
	case MonthStart:
		timeCfg = time.Date(curYear, curMonth, 1, 0, 0, 0, 0, curLocation)
		return timeCfg.AddDate(0, syncTime.Offset, 0), nil
	case MonthEnd:
		firstOfMonth := time.Date(curYear, curMonth, 1, 0, 0, 0, 0, curLocation)
		timeCfg = firstOfMonth.AddDate(0, 1, -1)
		return timeCfg.AddDate(0, syncTime.Offset, 0), nil
	default:
		return time.Now(), fmt.Errorf("unknown TimeIdentifier %s", syncTime.Identifier)
	}
}

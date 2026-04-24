package limits

import (
	"fmt"
	"strings"
	"time"
)

type Period string

const (
	PeriodDay   Period = "day"
	PeriodWeek  Period = "week"
	PeriodMonth Period = "month"
)

func ValidatePeriod(p string) bool {
	switch Period(strings.ToLower(p)) {
	case PeriodDay, PeriodWeek, PeriodMonth:
		return true
	default:
		return false
	}
}

func Bucket(now time.Time, period string) (string, error) {
	now = now.UTC()
	switch Period(strings.ToLower(period)) {
	case PeriodDay:
		return now.Format("2006-01-02"), nil
	case PeriodWeek:
		y, w := now.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w), nil
	case PeriodMonth:
		return now.Format("2006-01"), nil
	default:
		return "", fmt.Errorf("unsupported period: %s", period)
	}
}

func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	// простая оценка для MVP: 1 токен ~= 4 символа
	n := len([]rune(text)) / 4
	if n < 1 {
		return 1
	}
	return n
}

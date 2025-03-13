package utils

import (
	"errors"
	"time"
)

type TimeInterval struct {
	StartTime time.Time
	EndTime   time.Time
}

func NewTimeInterval(startTime time.Time, endTime time.Time) (*TimeInterval, error) {
	if startTime.After(endTime) {
		return nil, errors.New("start time must be after end time")
	}

	return &TimeInterval{
		StartTime: startTime,
		EndTime: endTime,
	}, nil
}

func (t *TimeInterval) IsActive() bool {
	now := time.Now()
	return now.After(t.StartTime) && now.Before(t.EndTime)
}
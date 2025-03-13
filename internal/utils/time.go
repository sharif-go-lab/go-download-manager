package utils

import (
	"errors"
	"time"
)

type TimeInterval struct {
	startTime time.Time
	endTime   time.Time
}

func NewTimeInterval(start, end string) (*TimeInterval, error) {
	startTime, err := time.Parse("12:00:00", start)
	if err != nil {
		return nil, err
	}
	endTime, err := time.Parse("12:00:00", end)
	if err != nil {
		return nil, err
	}

	if startTime.After(endTime) {
		return nil, errors.New("start time must be after end time")
	}

	return &TimeInterval{
		startTime: startTime,
		endTime: endTime,
	}, nil
}

func (t *TimeInterval) StartTime() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), t.startTime.Hour(), t.startTime.Minute(), t.startTime.Second(), 0, now.Location())
}

func (t *TimeInterval) EndTime() time.Time {
	now := time.Now()
	start := t.StartTime()
	end := time.Date(now.Year(), now.Month(), now.Day(), t.endTime.Hour(), t.endTime.Minute(), t.endTime.Second(), 0, now.Location())
	if start.After(end) {
		end = end.Add(24 * time.Hour)
	}
	return end
}

func (t *TimeInterval) WaitUntil() {
	start, end := t.StartTime(), t.EndTime()
	if time.Now().Before(start) {
		time.Sleep(start.Sub(time.Now()))
	} else if time.Now().After(end) {
		time.Sleep(24 * time.Hour - time.Now().Sub(start))
	}
}

func CreateLimiter(speedLimit uint64) <-chan time.Time {
	if speedLimit == 0 {
		return nil
	}
	return time.Tick(time.Second / time.Duration(speedLimit))
}
package preconditions

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const WindowedPreconditionName = "time_window"

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (r *realClock) Now() time.Time {
	return time.Now()
}

type Delimiter struct {
	hour   int
	minute int
}

func (b *Delimiter) String() string {
	return fmt.Sprintf("%02d:%02d", b.hour, b.minute)
}

func (b *Delimiter) Equals(another *Delimiter) bool {
	if another == nil {
		return false
	}

	return another.hour == b.hour && another.minute == b.minute
}

func (b *Delimiter) validate() error {
	if b.hour < 0 || b.hour > 23 {
		return errors.New("hour must not be in the interval [0, 23]")
	}

	if b.minute < 0 || b.minute > 59 {
		return errors.New("minute must not be in the interval [0, 60]")
	}

	return nil
}

type WindowedPrecondition struct {
	startTime string
	endTime   string

	clock Clock
}

func NewWindowPrecondition(start, end *Delimiter) (*WindowedPrecondition, error) {
	if start == nil {
		return nil, errors.New("no 'start' parameter supplied")
	}

	if err := start.validate(); err != nil {
		return nil, err
	}

	if end == nil {
		return nil, errors.New("no 'start' parameter supplied")
	}

	if err := end.validate(); err != nil {
		return nil, err
	}

	if start.Equals(end) {
		return nil, errors.New("'start' and 'end' must not be equal")
	}

	return &WindowedPrecondition{
		startTime: start.String(),
		endTime:   end.String(),
		clock:     &realClock{},
	}, nil
}

func WindowPreconditionFromMap(args map[string]any) (*WindowedPrecondition, error) {
	if args == nil {
		return nil, errors.New("empty args provided")
	}

	fromStr, ok := args["from"].(string)
	if !ok {
		return nil, errors.New("no 'from' specified")
	}
	fromParsed, err := extractHourAndMinute(fromStr)
	if err != nil {
		return nil, err
	}

	toStr, ok := args["to"].(string)
	if !ok {
		return nil, errors.New("no 'to' specified")
	}
	toParsed, err := extractHourAndMinute(toStr)
	if err != nil {
		return nil, err
	}

	return NewWindowPrecondition(fromParsed, toParsed)
}

func (c *WindowedPrecondition) PerformCheck() bool {
	layout := "15:04"
	startLocalTime, _ := time.ParseInLocation(layout, c.startTime, time.Local)
	endLocalTime, _ := time.ParseInLocation(layout, c.endTime, time.Local)
	now, _ := time.ParseInLocation(layout, c.clock.Now().Format(layout), time.Local)

	// adjust overlapping dates
	if endLocalTime.Before(startLocalTime) {
		return !(now.After(endLocalTime) && now.Before(startLocalTime))
	}

	return startLocalTime.Before(now) && endLocalTime.After(now)
}

func extractHourAndMinute(input string) (*Delimiter, error) {
	if len(input) != 5 || strings.IndexRune(input, ':') != 2 {
		return nil, errors.New("invalid format, expected HH:MM")
	}

	parts := strings.Split(input, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid time format: %s", input)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		err = fmt.Errorf("hour or minute out of valid range")
		return nil, err
	}

	return &Delimiter{hour: hour, minute: minute}, nil
}

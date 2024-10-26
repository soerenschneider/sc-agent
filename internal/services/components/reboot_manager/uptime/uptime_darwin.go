package uptime

import (
	"time"
)

var start = time.Now()

func Uptime() (time.Duration, error) {
	return time.Since(start), nil
}

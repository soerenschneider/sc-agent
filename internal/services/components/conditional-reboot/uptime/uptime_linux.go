package uptime

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	rawUptimeImpl UptimeSource = &LinuxUptime{}
)

type UptimeSource interface {
	RawUptime() (float64, error)
}

func Uptime() (time.Duration, error) {
	seconds, err := rawUptimeImpl.RawUptime()
	if err != nil {
		return time.Duration(0), fmt.Errorf("could not read raw uptime: %w", err)
	}

	return time.Second * time.Duration(seconds), nil
}

type LinuxUptime struct {
}

func (p *LinuxUptime) RawUptime() (float64, error) {
	uptime, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return math.MaxFloat64, err
	}
	return parseLinuxUptime(string(uptime))
}

func parseLinuxUptime(uptime string) (float64, error) {
	parts := strings.Split(string(uptime), " ")
	secondsStr := strings.TrimSpace(parts[0])
	return strconv.ParseFloat(secondsStr, 64)
}

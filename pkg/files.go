package pkg

import (
	"fmt"
	"math"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/domain"
	"go.uber.org/multierr"
)

func GetExpandedFile(filename string) string {
	usr, _ := user.Current()
	dir := usr.HomeDir

	if strings.HasPrefix(filename, "~/") {
		return filepath.Join(dir, filename[2:])
	}

	if strings.HasPrefix(filename, "$HOME/") {
		return filepath.Join(dir, filename[6:])
	}

	return filename
}

func RunPostIssueHooks(hooks []domain.PostHook) error {
	var errs error
	for _, hook := range hooks {
		log.Info().Str("component", "hooks").Str("hook_name", hook.Name).Msgf("Running post issue hook %s", hook.Name)
		parsed := strings.Split(hook.Cmd, " ")
		cmd := exec.Command(parsed[0], parsed[1:]...) // #nosec G204
		cmdErr := cmd.Run()
		if cmdErr != nil {
			errs = multierr.Append(errs, fmt.Errorf("error running post-hook %q: %w", hook.Name, cmdErr))
		}
	}

	return errs
}

func GetPercentage(from, to time.Time) float32 {
	total := to.Sub(from).Seconds()
	if total == 0 {
		return 0.
	}

	left := time.Until(to).Seconds()
	return float32(math.Max(0, left*100/total))
}

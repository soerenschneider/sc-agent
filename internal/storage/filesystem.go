package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"runtime"
	"syscall"

	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/spf13/afero"
	"go.uber.org/multierr"

	"golang.org/x/sys/unix"
)

type FilesystemStorage struct {
	FilePath  string
	FileOwner string
	FileGroup string
	Mode      os.FileMode
	fs        afero.Fs
}

const (
	FsScheme   = "file"
	ParamChmod = "chmod"
)

var (
	defaultFileMode os.FileMode = 0600
	defaultDirMode  os.FileMode = 0750
)

func NewFilesystemStorageFromUri(uri string) (*FilesystemStorage, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	path, err := expandHomeDir(parsed)
	if err != nil {
		return nil, err
	}

	var username, group string = "root", getOsDependentGroup()
	userData := parsed.User
	if userData != nil {
		username = userData.Username()

		userDataGroup, ok := userData.Password()
		if ok {
			group = userDataGroup
		}
	} else {
		// no username is given, try to detect and use the username running this process
		// otherwise, just use root
		u, err := user.Current()
		if err == nil {
			username = u.Username
		}
	}

	mode := defaultFileMode
	params, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("could not parse queries")
	}
	for key, val := range params {
		if key == ParamChmod {
			parsed, err := strconv.ParseInt(val[0], 8, 32)
			if err != nil {
				return nil, fmt.Errorf("could not parse value for 'chmod' param: %v", val)
			}

			mode = os.FileMode(parsed) //#nosec:G115
			if err != nil {
				return nil, fmt.Errorf("invalid file mode supplied: %v", val[0])
			}
		}
	}

	if len(path) == 0 {
		return nil, errors.New("empty path provided")
	}

	// usually, uid and gid are resolved dynamically to support users/groups that are added after sc-agent has started
	// by trying to resolve it now, we make sure to fail fast on systems that we don't support, e.g. Windows
	_, _, err = resolveUidAndGid(username, group)
	if err != nil {
		return nil, fmt.Errorf("could not resolve uid and gid for user '%s' and group '%s': %w", username, group, err)
	}

	return &FilesystemStorage{
		FilePath:  path,
		FileOwner: username,
		FileGroup: group,
		Mode:      mode,
		fs:        afero.NewOsFs(),
	}, nil
}

func (fss *FilesystemStorage) Read() ([]byte, error) {
	data, err := afero.ReadFile(fss.fs, fss.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoCertFound
		}
		return nil, err
	}

	// Check and fix file permissions, ownership, and group
	if err := fss.checkAndFixFileState(); err != nil {
		// Log the error but don't fail the read operation
		log.Error().Err(err).Str("component", "cert_storage").Str("file", fss.FilePath).Msg("could not fix file state")
	}

	return data, nil
}

func (fss *FilesystemStorage) CanRead() error {
	_, err := os.Stat(fss.FilePath)
	return err
}

func (fss *FilesystemStorage) Write(signedData []byte) error {
	if len(signedData) == 0 || signedData[len(signedData)-1] != '\n' {
		signedData = append(signedData, '\n')
	}

	uid, gid, err := resolveUidAndGid(fss.FileOwner, fss.FileGroup)
	if err != nil {
		return fmt.Errorf("could not resolve uid and gid for file '%s': %v", fss.FilePath, err)
	}

	dir := filepath.Dir(fss.FilePath)
	if dir != "" && dir != "." && dir != "/" {
		// Check if directory already exists
		if _, err := fss.fs.Stat(dir); errors.Is(err, fs.ErrNotExist) {
			log.Warn().Str("component", "cert_storage").Str("file", fss.FilePath).Msg("directory base path does not exist, creating it")
			// Directory doesn't exist, create it
			if err := fss.fs.MkdirAll(dir, defaultDirMode); err != nil {
				return fmt.Errorf("could not create directories for '%s': %v", fss.FilePath, err)
			}

			// Set ownership on newly created directories if specified
			if err := fss.fs.Chown(dir, uid, gid); err != nil {
				return fmt.Errorf("could not chown directory '%s': %v", dir, err)
			}
		} else {
			// Some other error occurred during stat
			return fmt.Errorf("could not check directory '%s': %v", dir, err)
		}
	}

	if err := afero.WriteFile(fss.fs, fss.FilePath, signedData, fss.Mode); err != nil {
		return fmt.Errorf("could not write file '%s' to disk: %v", fss.FilePath, err)
	}

	if err := fss.fs.Chown(fss.FilePath, uid, gid); err != nil {
		return fmt.Errorf("could not chown file '%s': %v", fss.FilePath, err)
	}

	return nil
}

func (fss *FilesystemStorage) CanWrite() error {
	dir := filepath.Dir(fss.FilePath)
	return unix.Access(dir, unix.W_OK)
}

func expandHomeDir(parsed *url.URL) (string, error) {
	path := parsed.Path
	if parsed.Host == "~" || parsed.Host == "$HOME" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("tried to expand '%s' but homeDir could not be detected: %v", parsed.Host, err)
		}

		orig := filepath.Join(parsed.Host, path)
		path = filepath.Join(homeDir, orig[len(parsed.Host):])
		log.Info().Str("component", "cert_storage").Str("old_path", orig).Str("expanded_path", path).Msg("Expanded path")
		return path, nil
	}

	if len(parsed.Host) > 0 {
		return "", fmt.Errorf("invalid syntax for uri, no host expected (did you forget the leading '/'?)")
	}

	return path, nil
}

// checkAndFixFileState verifies and corrects file mode, owner, and group
func (fss *FilesystemStorage) checkAndFixFileState() error {
	fileInfo, err := fss.fs.Stat(fss.FilePath)
	if err != nil {
		return fmt.Errorf("could not stat file '%s': %v", fss.FilePath, err)
	}

	currentMode := fileInfo.Mode().Perm()
	wantedMode := fss.Mode
	log.Debug().Str("file", fss.FilePath).Any("wanted", wantedMode).Any("current", currentMode).Msg("Checking file mode for differences")
	if currentMode != wantedMode {
		log.Info().Str("component", "cert_storage").Str("file", fss.FilePath).Str("current_mode", fmt.Sprintf("%o", currentMode)).Str("desired_mode", fmt.Sprintf("%o", wantedMode)).Msg("file mode mismatch detected")
		if err := fss.fs.Chmod(fss.FilePath, wantedMode); err != nil {
			return fmt.Errorf("could not chmod file '%s': %v", fss.FilePath, err)
		}
	}

	wantedUid, wantedGid, err := resolveUidAndGid(fss.FileOwner, fss.FileGroup)
	if err != nil {
		return fmt.Errorf("could not resolve uid and gid for file '%s': %v", fss.FilePath, err)
	}

	// For afero, we need to check if it's an OS filesystem to get detailed stat info
	if osFs, ok := fss.fs.(*afero.OsFs); ok {
		stat, err := osFs.Stat(fss.FilePath)
		if err != nil {
			return fmt.Errorf("could not get detailed stat for file '%s': %v", fss.FilePath, err)
		}

		// Get system-specific stat info
		if sysStat, ok := stat.Sys().(*syscall.Stat_t); ok {
			currentUid := int(sysStat.Uid)
			currentGid := int(sysStat.Gid)
			log.Debug().Str("file", fss.FilePath).Any("wanted", currentGid).Any("current", currentUid).Msg("Checking file owner for differences")
			if currentUid != wantedUid || currentGid != wantedGid {
				log.Info().Str("component", "cert_storage").Str("file", fss.FilePath).Int("current_uid", currentUid).Int("current_gid", currentGid).Int("wanted_uid", wantedUid).Int("desired_gid", wantedGid).Msg("file ownership mismatch detected")
				if err := fss.fs.Chown(fss.FilePath, wantedUid, wantedGid); err != nil {
					return fmt.Errorf("could not chown file '%s': %v", fss.FilePath, err)
				}
			}
		} else {
			// Fallback: always try to set ownership if we can't determine current values
			log.Info().Str("component", "cert_storage").Str("file", fss.FilePath).Msg("unable to determine current ownership, setting desired ownership")
			if err := fss.fs.Chown(fss.FilePath, wantedUid, wantedGid); err != nil {
				return fmt.Errorf("could not chown file '%s': %v", fss.FilePath, err)
			}
		}

		return nil
	} else {
		// For non-OS filesystems (like memory fs), we might not be able to check ownership
		// but we can still try to set it if the filesystem supports it
		log.Info().Str("component", "cert_storage").Str("file", fss.FilePath).Msg("non-OS filesystem detected, attempting to set ownership")
		if err := fss.fs.Chown(fss.FilePath, wantedUid, wantedGid); err != nil {
			// Don't return error for non-OS filesystems as they might not support ownership
			log.Warn().Err(err).Str("component", "cert_storage").Str("file", fss.FilePath).Msg("could not set ownership on non-OS filesystem")
		}
	}

	return nil
}

func resolveUidAndGid(owner, group string) (int, int, error) {
	var errs error

	uid, err := resolveUid(owner)
	errs = multierr.Append(errs, err)

	gid, err := resolveGid(group)
	errs = multierr.Append(errs, err)

	return uid, gid, errs
}

func resolveUid(owner string) (int, error) {
	localUser, err := user.Lookup(owner)
	if err != nil {
		log.Error().Str("component", "cert_storage").Str("owner", owner).Msg("could not lookup user, falling back to root")
		metrics.CertStorageErrors.WithLabelValues("user_lookup_failed").Inc()
		return 0, nil
	}

	cuid, err := strconv.Atoi(localUser.Uid)
	if err != nil {
		return -1, fmt.Errorf("was expecting a numerical uid, got '%s'", localUser.Uid)
	}

	return cuid, nil
}

func resolveGid(group string) (int, error) {
	localGroup, err := user.LookupGroup(group)
	if err != nil {
		log.Error().Str("component", "cert_storage").Str("group", group).Msg("could not lookup group, falling back to root")
		metrics.CertStorageErrors.WithLabelValues("group_lookup_failed").Inc()
		return 0, nil
	}

	cgid, err := strconv.Atoi(localGroup.Gid)
	if err != nil {
		return -1, fmt.Errorf("was expecting a numerical gid, got '%s'", localGroup.Gid)
	}

	return cgid, nil
}

func getOsDependentGroup() string {
	if runtime.GOOS == "linux" {
		return "root"
	}
	return "wheel"
}

package storage

import (
	"cmp"
	"errors"
	"fmt"
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

	var username, group string
	userData := parsed.User
	if userData != nil {
		username = userData.Username()

		var ok bool
		group, ok = userData.Password()
		if !ok {
			group = ""
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

	return &FilesystemStorage{
		FilePath:  path,
		FileOwner: cmp.Or(username, "root"),
		FileGroup: cmp.Or(group, "root"),
		Mode:      mode,
		fs:        afero.NewOsFs(),
	}, nil
}

func resolveUidAndGid(owner, group string) (int, int, error) {
	var errs error

	uid, err := resolveUid(owner)
	multierr.Append(errs, err)

	gid, err := resolveUid(group)
	multierr.Append(errs, err)

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

func (fs *FilesystemStorage) Read() ([]byte, error) {
	data, err := afero.ReadFile(fs.fs, fs.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoCertFound
		}
		return nil, err
	}

	// Check and fix file permissions, ownership, and group
	if err := fs.checkAndFixFileState(); err != nil {
		// Log the error but don't fail the read operation
		log.Error().Err(err).Str("component", "cert_storage").Str("file", fs.FilePath).Msg("could not fix file state")
	}

	return data, nil
}

func (fs *FilesystemStorage) CanRead() error {
	_, err := os.Stat(fs.FilePath)
	return err
}

func (fs *FilesystemStorage) Write(signedData []byte) error {
	if len(signedData) == 0 || signedData[len(signedData)-1] != '\n' {
		signedData = append(signedData, '\n')
	}

	dir := filepath.Dir(fs.FilePath)
	if dir != "" && dir != "." && dir != "/" {
		// Check if directory already exists
		if _, err := fs.fs.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				log.Warn().Str("component", "cert_storage").Str("file", fs.FilePath).Msg("directory base path does not exist, creating it")
				// Directory doesn't exist, create it
				dirMode := os.FileMode(0700)
				err = fs.fs.MkdirAll(dir, dirMode)
				if err != nil {
					return fmt.Errorf("could not create directories for '%s': %v", fs.FilePath, err)
				}

				// Set ownership on newly created directories if specified
				uid, gid, err := resolveUidAndGid(fs.FileOwner, fs.FileGroup)
				if err == nil {
					if err := fs.chownDirectoryRecursive(dir, uid, gid); err != nil {
						return fmt.Errorf("could not set ownership on directories for '%s': %v", fs.FilePath, err)

					}
				}
			} else {
				// Some other error occurred during stat
				return fmt.Errorf("could not check directory '%s': %v", dir, err)
			}
		}
		// If no error from Stat(), directory already exists - do nothing
	}

	err := afero.WriteFile(fs.fs, fs.FilePath, signedData, fs.Mode)
	if err != nil {
		return fmt.Errorf("could not write file '%s' to disk: %v", fs.FilePath, err)
	}

	uid, gid, err := resolveUidAndGid(fs.FileOwner, fs.FileGroup)
	if err == nil {
		err = fs.fs.Chown(fs.FilePath, uid, gid)
		if err != nil {
			return fmt.Errorf("could not chown file '%s': %v", fs.FilePath, err)
		}
	}

	return nil
}

func (fs *FilesystemStorage) CanWrite() error {
	dir := filepath.Dir(fs.FilePath)
	return unix.Access(dir, unix.W_OK)
}

// chownDirectoryRecursive sets ownership on all directories that were created
func (fs *FilesystemStorage) chownDirectoryRecursive(targetDir string, uid, gid int) error {
	// We need to chown each directory level that was potentially created
	// Start from the root and work our way down to ensure proper ownership

	// Build a list of directory components
	var dirsToChown []string
	currentDir := targetDir

	for currentDir != "" && currentDir != "." && currentDir != "/" {
		dirsToChown = append([]string{currentDir}, dirsToChown...) // prepend
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // reached root
		}
		currentDir = parentDir
	}

	// Apply ownership to each directory level
	for _, dir := range dirsToChown {
		// Check if this directory exists before trying to chown
		if _, err := fs.fs.Stat(dir); err == nil {
			err = fs.fs.Chown(dir, uid, gid)
			if err != nil {
				return fmt.Errorf("could not chown directory '%s': %v", dir, err)
			}
		}
	}

	return nil
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
func (fs *FilesystemStorage) checkAndFixFileState() error {
	fileInfo, err := fs.fs.Stat(fs.FilePath)
	if err != nil {
		return fmt.Errorf("could not stat file '%s': %v", fs.FilePath, err)
	}

	currentMode := fileInfo.Mode().Perm()
	desiredMode := fs.Mode
	if currentMode != desiredMode {
		log.Info().Str("component", "cert_storage").Str("file", fs.FilePath).Str("current_mode", fmt.Sprintf("%o", currentMode)).Str("desired_mode", fmt.Sprintf("%o", desiredMode)).Msg("file mode mismatch detected")
		if err := fs.fs.Chmod(fs.FilePath, desiredMode); err != nil {
			return fmt.Errorf("could not chmod file '%s': %v", fs.FilePath, err)
		}
	}

	// For afero, we need to check if it's an OS filesystem to get detailed stat info
	if osFs, ok := fs.fs.(*afero.OsFs); ok {
		stat, err := osFs.Stat(fs.FilePath)
		if err != nil {
			return fmt.Errorf("could not get detailed stat for file '%s': %v", fs.FilePath, err)
		}

		desiredUID, desiredGID, err := resolveUidAndGid(fs.FileOwner, fs.FileGroup)
		if err != nil {
			return fmt.Errorf("could not resolve uid and gid for file '%s': %v", fs.FilePath, err)
		}

		// Get system-specific stat info
		if sysStat, ok := stat.Sys().(*syscall.Stat_t); ok {
			currentUID := int(sysStat.Uid)
			currentGID := int(sysStat.Gid)

			if currentUID != desiredUID || currentGID != desiredGID {
				log.Info().Str("component", "cert_storage").Str("file", fs.FilePath).Int("current_uid", currentUID).Int("current_gid", currentGID).Int("desired_uid", desiredUID).Int("desired_gid", desiredGID).Msg("file ownership mismatch detected")
				if err := fs.fs.Chown(fs.FilePath, desiredUID, desiredGID); err != nil {
					return fmt.Errorf("could not chown file '%s': %v", fs.FilePath, err)
				}
			}
			return nil
		} else {
			// Fallback: always try to set ownership if we can't determine current values
			log.Info().Str("component", "cert_storage").Str("file", fs.FilePath).Msg("unable to determine current ownership, setting desired ownership")
			if err := fs.fs.Chown(fs.FilePath, desiredUID, desiredGID); err != nil {
				return fmt.Errorf("could not chown file '%s': %v", fs.FilePath, err)
			}

			return nil
		}

		// For non-OS filesystems (like memory fs), we might not be able to check ownership
		// but we can still try to set it if the filesystem supports it
		log.Info().Str("component", "cert_storage").Str("file", fs.FilePath).Msg("non-OS filesystem detected, attempting to set ownership")
		if err := fs.fs.Chown(fs.FilePath, desiredUID, desiredGID); err != nil {
			// Don't return error for non-OS filesystems as they might not support ownership
			log.Warn().Err(err).Str("component", "cert_storage").Str("file", fs.FilePath).Msg("could not set ownership on non-OS filesystem")
		}
	}

	return nil
}

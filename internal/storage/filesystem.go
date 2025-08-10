package storage

import (
	"errors"
	"fmt"

	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/metrics"
	"github.com/spf13/afero"

	"golang.org/x/sys/unix"
)

type FilesystemStorage struct {
	FilePath  string
	FileOwner *int
	FileGroup *int
	Mode      os.FileMode
	fs        afero.Fs
}

const (
	FsScheme   = "file"
	ParamChmod = "chmod"
)

var defaultMode os.FileMode = 0600

func NewFilesystemStorageFromUri(uri string) (*FilesystemStorage, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	path, err := expandHomeDir(parsed)
	if err != nil {
		return nil, err
	}

	var username, pass string
	userData := parsed.User
	if userData != nil {
		username = userData.Username()

		var ok bool
		pass, ok = userData.Password()
		if !ok {
			pass = ""
		}
	}

	mode := defaultMode
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

	return newFilesystemStorage(path, username, pass, mode)
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
		log.Info().Msgf("Expanded path '%s' to '%s'", orig, path)
		return path, nil
	}

	if len(parsed.Host) > 0 {
		return "", fmt.Errorf("invalid syntax for uri, no host expected (did you forget the leading '/'?)")
	}

	return path, nil
}

func newFilesystemStorage(path, owner, group string, mode os.FileMode) (*FilesystemStorage, error) {
	if len(path) == 0 {
		return nil, errors.New("empty path provided")
	}

	var uid, gid *int
	if len(owner) > 0 && len(group) > 0 {
		localUser, err := user.Lookup(owner)
		if err != nil {
			log.Error().Str("component", "cert_storage").Str("owner", owner).Msg("could not lookup user, falling back to root")
			metrics.CertStorageErrors.WithLabelValues("user_lookup_failed").Inc()
			localUser = &user.User{
				Uid: "0",
			}
		}

		cuid, err := strconv.Atoi(localUser.Uid)
		if err != nil {
			return nil, fmt.Errorf("was expecting a numerical uid, got '%s'", localUser.Uid)
		}
		uid = &cuid

		localGroup, err := user.LookupGroup(group)
		if err != nil {
			log.Error().Str("component", "cert_storage").Str("group", group).Msg("could not lookup group, falling back to root")
			metrics.CertStorageErrors.WithLabelValues("group_lookup_failed").Inc()
			localUser = &user.User{
				Gid: "0",
			}
		}

		cgid, err := strconv.Atoi(localGroup.Gid)
		if err != nil {
			return nil, fmt.Errorf("was expecting a numerical gid, got '%s'", localGroup.Gid)
		}
		gid = &cgid
	}

	return &FilesystemStorage{
		FilePath:  path,
		FileOwner: uid,
		FileGroup: gid,
		Mode:      mode,
		fs:        afero.NewOsFs(),
	}, nil
}

func (fs *FilesystemStorage) Read() ([]byte, error) {
	data, err := afero.ReadFile(fs.fs, fs.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoCertFound
		}
		return nil, err
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

	err := afero.WriteFile(fs.fs, fs.FilePath, signedData, fs.Mode)
	if err != nil {
		return fmt.Errorf("could not write file '%s' to disk: %v", fs.FilePath, err)
	}

	if fs.FileOwner != nil && fs.FileGroup != nil {
		err = fs.fs.Chown(fs.FilePath, *fs.FileOwner, *fs.FileGroup)
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

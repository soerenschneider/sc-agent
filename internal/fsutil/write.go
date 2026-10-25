package fsutil

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/spf13/afero"
)

var ErrFileOwnership = errors.New("could not update file ownership")

// WriteFileAtomic replaces the file atomically: the data is written to a temporary file in the same directory that
// gets the file's mode and ownership before it is renamed, so readers never see partially written data or a file they
// are not allowed to read. If owner is set, the file is owned by that user, otherwise the ownership and mode of an
// existing file are kept. New files are created with mode 0600.
func WriteFileAtomic(fs afero.Fs, name string, data []byte, owner string) error {
	mode := os.FileMode(0600)
	uid, gid, chown := -1, -1, false
	if info, err := fs.Stat(name); err == nil {
		mode = info.Mode().Perm()
		uid, gid, chown = fileOwner(info)
	}

	if owner != "" && runtime.GOOS != "windows" {
		usr, err := user.Lookup(owner)
		if err != nil {
			return fmt.Errorf("%w: could not lookup user %s", ErrFileOwnership, owner)
		}
		uid, _ = strconv.Atoi(usr.Uid)
		gid, chown = 0, true
	}

	tmp, err := afero.TempFile(fs, filepath.Dir(name), "."+filepath.Base(name)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// no-op after the rename succeeded
	defer func() {
		_ = fs.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := fs.Chmod(tmpName, mode); err != nil {
		return err
	}
	if chown && !ownedBy(fs, tmpName, uid, gid) {
		if err := fs.Chown(tmpName, uid, gid); err != nil {
			return fmt.Errorf("%w: %w", ErrFileOwnership, err)
		}
	}

	return fs.Rename(tmpName, name)
}

// ownedBy reports whether the file is already owned by uid and gid, so no privileges are needed to keep its ownership.
func ownedBy(fs afero.Fs, file string, uid, gid int) bool {
	info, err := fs.Stat(file)
	if err != nil {
		return false
	}
	fileUid, fileGid, ok := fileOwner(info)
	return ok && fileUid == uid && fileGid == gid
}

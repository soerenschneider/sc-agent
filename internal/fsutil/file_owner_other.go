//go:build !unix

package fsutil

import (
	"os"
)

// fileOwner returns the uid and gid of the file, if the platform supports it.
func fileOwner(_ os.FileInfo) (uid, gid int, ok bool) {
	return 0, 0, false
}

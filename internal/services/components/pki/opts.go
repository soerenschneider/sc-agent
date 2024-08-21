package pki

import (
	"errors"
)

func WithMountPath(mountPath string) VaultClientOpts {
	return func(v *VaultX509Client) error {
		if len(mountPath) == 0 {
			return errors.New("empty pki mount path")
		}

		v.mountPath = mountPath
		return nil
	}
}

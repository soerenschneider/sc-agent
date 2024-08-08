package vault_x509

import (
	"errors"
)

func WithMountPath(mountPath string) VaultX509PkiOpts {
	return func(v *VaultX509Client) error {
		if len(mountPath) == 0 {
			return errors.New("empty pki mount path")
		}

		v.mountPath = mountPath
		return nil
	}
}

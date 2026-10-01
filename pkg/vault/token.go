package vault

import (
	"os"
	"strings"
)

func ContainsFileWrappedToken(file string) (bool, error) {
	token, err := os.ReadFile(file)
	if err != nil {
		return false, err
	}
	return IsWrappedToken(string(token)), nil
}

const (
	openbaoPrefix = "s."
	vaultPrefix   = "hvs."
)

func IsWrappedToken(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), openbaoPrefix) || strings.HasPrefix(strings.TrimSpace(token), vaultPrefix)
}

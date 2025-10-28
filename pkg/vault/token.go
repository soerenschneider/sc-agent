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

func IsWrappedToken(token string) bool {
	return strings.HasPrefix(strings.TrimSpace(token), "hvs.")
}

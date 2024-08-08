package vault

import (
	"os"
	"strings"
)

func IsWrappedToken(file string) bool {
	content, err := os.ReadFile(file)
	if err != nil {
		return false
	}
	return strings.HasPrefix(string(content), "hvs.")
}

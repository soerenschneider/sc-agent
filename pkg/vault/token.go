package vault

import (
	"os"
	"strings"
)

func IsWrappedToken(file string) (bool, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return false, err
	}
	return strings.HasPrefix(string(content), "hvs."), nil
}

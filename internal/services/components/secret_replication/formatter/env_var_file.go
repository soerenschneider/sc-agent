package formatter

import (
	"bytes"
	"fmt"
	"maps"
	"slices"
	"strings"
)

type EnvVarFormatter struct {
	uppercaseKeys bool
}

func NewEnvVarFormatter(uppercaseKeys bool) *EnvVarFormatter {
	return &EnvVarFormatter{uppercaseKeys: uppercaseKeys}
}

func (y *EnvVarFormatter) Format(data map[string]any) ([]byte, error) {
	var buffer bytes.Buffer

	// sort the keys, otherwise the random map order changes the file on every replication
	for _, key := range slices.Sorted(maps.Keys(data)) {
		name := key
		if y.uppercaseKeys {
			name = strings.ToUpper(key)
		}
		_, _ = fmt.Fprintf(&buffer, "%s=%v\n", name, data[key])
	}

	return buffer.Bytes(), nil
}

package formatter

import (
	"bytes"
	"fmt"
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

	for key, value := range data {
		if y.uppercaseKeys {
			key = strings.ToUpper(key)
		}
		_, _ = buffer.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	}

	return buffer.Bytes(), nil
}

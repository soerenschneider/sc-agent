package formatter

import (
	"encoding/json"
)

type JsonFormatter struct {
}

func (y *JsonFormatter) Format(data map[string]any) ([]byte, error) {
	marshalled, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}

	return marshalled, nil
}

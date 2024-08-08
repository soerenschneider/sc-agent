package formatter

import "gopkg.in/yaml.v3"

type YamlFormatter struct {
}

func (y *YamlFormatter) Format(data map[string]any) ([]byte, error) {
	marshalled, err := yaml.Marshal(&data)
	if err != nil {
		return nil, err
	}

	return marshalled, nil
}

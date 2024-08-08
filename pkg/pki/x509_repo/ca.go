package stores

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// CaSink accepts CA data to write to the configured storage implementation.
type CaSink struct {
	storage StorageImplementation
}

func CaSinkFromConfig(storageConfig []map[string]string) (*CaSink, error) {
	var caVal string
	for _, conf := range storageConfig {
		val, ok := conf[caId]
		if ok {
			caVal = val
		} else {
			log.Info().Msgf("No storage config given for '%s', writing to stdout", caId)
		}
	}

	if len(caVal) > 0 {
		storageImpl, err := BuildFromUri(caVal)
		if err != nil {
			return nil, err
		}
		return &CaSink{storageImpl}, nil
	}

	return &CaSink{nil}, nil
}

func (out *CaSink) WriteCa(certData []byte) error {
	if out.storage == nil {
		fmt.Println(string(certData))
		return nil
	}

	return out.storage.Write(certData)
}

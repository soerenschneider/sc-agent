package stores

import (
	"fmt"
	"net/url"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/storage"
)

type CrlSink struct {
	storage StorageImplementation
}

const crlId = "crl"

func CrlSinkFromConfig(storageConfig []map[string]string) (*CrlSink, error) {
	var crlVal string
	for _, conf := range storageConfig {
		val, ok := conf[crlId]
		if ok {
			crlVal = val
		} else {
			log.Info().Msgf("No storage config given for '%s', writing to stdout", crlId)
		}
	}

	if len(crlVal) > 0 {
		storageImpl, err := BuildFromUri(crlVal)
		if err != nil {
			return nil, err
		}
		return &CrlSink{storageImpl}, nil
	}

	return &CrlSink{nil}, nil
}

func (out *CrlSink) WriteCrl(crlData []byte) error {
	if out.storage == nil {
		fmt.Println(string(crlData))
		return nil
	}

	return out.storage.Write(crlData)
}

func BuildFromUri(uri string) (StorageImplementation, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		log.Error().Msgf("can not build storage from uri '%s': %v", uri, err)
		return nil, err
	}

	switch parsed.Scheme {
	case storage.FsScheme:
		return storage.NewFilesystemStorageFromUri(uri)
	default:
		return nil, fmt.Errorf("can not build storage impl for unknown scheme '%s'", parsed.Scheme)
	}
}

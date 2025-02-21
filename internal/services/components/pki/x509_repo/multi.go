package stores

import (
	"crypto/x509"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/storage"
	pki2 "github.com/soerenschneider/sc-agent/pkg/pki"
	"go.uber.org/multierr"
)

type MultiKeyPairSink struct {
	sinks []*KeyPairSink
}

func NewMultiKeyPairSink(sinks ...*KeyPairSink) (*MultiKeyPairSink, error) {
	if nil == sinks {
		return nil, errors.New("no sinks provided")
	}

	return &MultiKeyPairSink{sinks: sinks}, nil
}

func (fs *MultiKeyPairSink) WriteCert(certData *pki2.CertData) error {
	var errs error
	for _, sink := range fs.sinks {
		if err := sink.WriteCert(certData); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return errs
}

// sanitize checks whether all configured storage backends contain data.
// This is needed in cases where a new storage backend is added to the configuration after data was already
// written to existing backends. The new backend would not receive existing data due to the way the Read() method
// is implemented.
func (fs *MultiKeyPairSink) sanitize() error {
	certData := map[int]bool{}
	for idx := range fs.sinks {
		_, err := fs.sinks[idx].ReadCert()
		if err == nil {
			certData[idx] = true
		}
	}

	if len(certData) > 0 && len(certData) != len(fs.sinks) {
		log.Warn().Str("component", "storage").Msg("not all configured files are present")
		return storage.ErrFilesMissing
	}

	return nil
}

func (fs *MultiKeyPairSink) ReadCert() (*x509.Certificate, error) {
	// try to detect whether storages are configured that do not have any data written to them, yet
	if err := fs.sanitize(); err != nil && errors.Is(err, storage.ErrFilesMissing) {
		// due to inhomogeneity of written data and the combinations of possible storage configurations, it's
		// hard to reconstruct the data from storage in order to write it to new storage backends.
		// As this case is not going to happen often, we just request a new certificate and write it to all
		// backends.
		return nil, err
	}

	var errs error
	for _, sink := range fs.sinks {
		cert, err := sink.ReadCert()
		if err == nil {
			return cert, nil
		}
		errs = multierr.Append(errs, err)
	}

	return nil, errs
}

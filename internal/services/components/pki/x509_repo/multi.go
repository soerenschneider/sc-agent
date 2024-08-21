package stores

import (
	"crypto/x509"

	"github.com/pkg/errors"
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

func (f *MultiKeyPairSink) WriteCert(certData *pki2.CertData) error {
	var errs error
	for _, sink := range f.sinks {
		if err := sink.WriteCert(certData); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return errs
}

func (f *MultiKeyPairSink) ReadCert() (*x509.Certificate, error) {
	var errs error
	for _, sink := range f.sinks {
		cert, err := sink.ReadCert()
		if err == nil {
			return cert, nil
		}
		errs = multierr.Append(errs, err)
	}

	return nil, errs
}

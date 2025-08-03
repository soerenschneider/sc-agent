package stores

import (
	"bytes"
	"crypto/x509"
	"regexp"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/pkg/pki"
)

// KeyPairSink offers an interface to read/write keypair data (certificate and private key) and optional ca data.
type KeyPairSink struct {
	ca         StorageImplementation
	caChain    StorageImplementation
	cert       StorageImplementation
	privateKey StorageImplementation
}

const (
	certId    = "cert"
	keyId     = "key"
	caId      = "ca"
	caChainId = "ca_chain"
)

var lineBreaksRegex = regexp.MustCompile(`(\r\n?|\n){2,}`)

func NewKeyPairSink(cert, privateKey, ca, chain StorageImplementation) (*KeyPairSink, error) {
	if nil == privateKey {
		return nil, errors.New("empty private key storage provided")
	}

	return &KeyPairSink{cert: cert, privateKey: privateKey, ca: ca, caChain: chain}, nil
}

func (f *KeyPairSink) ReadCert() (*x509.Certificate, error) {
	var source StorageImplementation
	if f.cert != nil {
		source = f.cert
	} else {
		source = f.privateKey
	}

	data, err := source.Read()
	if err != nil {
		return nil, err
	}

	return pki.ParseCertPem(data)
}

func (f *KeyPairSink) WriteCert(certData *pki.CertData) error {
	if nil == certData {
		return errors.New("got nil as certData")
	}

	// case 1: write cert, ca and private key to same storage
	if f.cert == nil && f.ca == nil && f.caChain == nil {
		return f.writeToPrivateSlot(certData)
	}

	// case 2: write cert and private to a same storage, write ca (if existent) to dedicated storage
	if f.cert == nil && (f.caChain != nil || f.ca != nil) {
		return f.writeToCertAndCaSlot(certData)
	}

	// case 3: write to individual storage
	return f.writeToIndividualSlots(certData)
}

func endsWithNewline(data []byte) bool {
	return bytes.HasSuffix(data, []byte("\n"))
}

func (f *KeyPairSink) writeToPrivateSlot(certData *pki.CertData) error {
	var data = certData.Certificate
	if !endsWithNewline(data) {
		data = append(data, "\n"...)
	}

	if certData.HasCaData() {
		data = append(data, certData.CaData...)
		if !endsWithNewline(data) {
			data = append(data, "\n"...)
		}
	} else if certData.HasCaChain() {
		data = append(data, certData.CaChain...)
		if !endsWithNewline(data) {
			data = append(data, "\n"...)
		}
	}

	data = append(data, certData.PrivateKey...)
	return f.privateKey.Write(data)
}

func (f *KeyPairSink) writeToCertAndCaSlot(certData *pki.CertData) error {
	var data = certData.Certificate
	if !endsWithNewline(data) {
		data = append(data, "\n"...)
	}

	data = append(data, certData.PrivateKey...)
	if !endsWithNewline(data) {
		data = append(data, "\n"...)
	}

	if err := f.privateKey.Write(data); err != nil {
		return err
	}

	if certData.HasCaData() && f.ca != nil {
		caData := certData.CaData
		if !endsWithNewline(caData) {
			caData = append(caData, "\n"...)
		}
		if err := f.ca.Write(caData); err != nil {
			return err
		}
	}

	if f.caChain != nil && (certData.HasCaChain() || certData.HasCaData()) {
		var caData []byte
		if !certData.HasCaChain() {
			log.Warn().Str("component", "pki").Msg("ca-chain data absent, writing ca data")
			caData = certData.CaData
		} else {
			caData = certData.CaChain
		}

		if !endsWithNewline(caData) {
			caData = append(caData, "\n"...)
		}
		if err := f.caChain.Write(caData); err != nil {
			return err
		}
	}

	return nil
}

func fixLineBreaks(input []byte) (ret []byte) {
	ret = []byte(lineBreaksRegex.ReplaceAll(input, []byte("$1")))
	return
}

func (f *KeyPairSink) writeToIndividualSlots(certData *pki.CertData) error {
	var certRaw = certData.Certificate
	if certData.HasCaData() && f.ca == nil {
		if !endsWithNewline(certRaw) {
			certRaw = append(certRaw, "\n"...)
		}

		certRaw = append(certRaw, certData.CaData...)
	}

	if err := f.cert.Write(certRaw); err != nil {
		return err
	}

	if certData.HasCaData() && f.ca != nil {
		if err := f.ca.Write(certData.CaData); err != nil {
			return err
		}
	}

	if certData.HasCaChain() && f.caChain != nil {
		if err := f.caChain.Write(certData.CaChain); err != nil {
			return err
		}
	}

	if certData.HasPrivateKey() {
		return f.privateKey.Write(fixLineBreaks(certData.PrivateKey))
	}

	return nil
}

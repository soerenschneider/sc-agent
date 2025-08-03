package pki

type CertData struct {
	PrivateKey  []byte
	Certificate []byte
	CaData      []byte
	CaChain     []byte
	Serial      string
	Csr         []byte
}

func (c *CertData) HasPrivateKey() bool {
	return len(c.PrivateKey) > 0
}

func (c *CertData) HasCertificate() bool {
	return len(c.Certificate) > 0
}

func (c *CertData) HasCaData() bool {
	return len(c.CaData) > 0
}

func (c *CertData) HasCaChain() bool {
	return len(c.CaChain) > 0
}

type Signature struct {
	Certificate []byte
	CaData      []byte
	Serial      string
}

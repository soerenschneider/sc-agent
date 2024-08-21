package checkers

import "errors"

func UseTLS(certFile, keyFile string) KafkaOpts {
	return func(c *KafkaChecker) error {
		c.certFile = certFile
		c.keyFile = keyFile
		return nil
	}
}

func AcceptedKeys(keys []string) KafkaOpts {
	return func(c *KafkaChecker) error {
		if len(keys) == 0 {
			return errors.New("empty slice provided as kafka keys")
		}
		c.acceptedKeys = keys
		return nil
	}
}

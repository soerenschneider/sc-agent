package metrics

import "errors"

func WithTLS(certFile, keyFile string) func(w *MetricsServer) error {
	return func(w *MetricsServer) error {
		if len(certFile) == 0 {
			return errors.New("empty certfile")
		}

		if len(keyFile) == 0 {
			return errors.New("empty keyfile")
		}

		w.certFile = certFile
		w.keyFile = keyFile
		return nil
	}
}

func WithTLSClientVerification(certFile, keyFile, caFile string) func(w *MetricsServer) error {
	return func(w *MetricsServer) error {
		if len(certFile) == 0 {
			return errors.New("empty certfile")
		}

		if len(keyFile) == 0 {
			return errors.New("empty keyfile")
		}

		if len(caFile) == 0 {
			return errors.New("empty ca-file")
		}

		w.certFile = certFile
		w.keyFile = keyFile
		w.clientCa = caFile
		return nil
	}
}

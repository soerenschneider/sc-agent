package http_server

import (
	"bytes"
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc-agent/internal/metrics"
)

type contextKey string

const (
	clientCnKey contextKey = "clientCn"
	emailsKey   contextKey = "emails"
)

type CustomResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

// WriteHeader captures the status code
func (crw *CustomResponseWriter) WriteHeader(code int) {
	crw.statusCode = code
	crw.ResponseWriter.WriteHeader(code)
}

// Write captures the response body
func (crw *CustomResponseWriter) Write(b []byte) (int, error) {
	crw.body.Write(b)
	return crw.ResponseWriter.Write(b)
}

// NewCustomResponseWriter initializes CustomResponseWriter with status code 200 and an empty body buffer
func NewCustomResponseWriter(w http.ResponseWriter) *CustomResponseWriter {
	return &CustomResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

// Middleware to log the HTTP status code
func logStatusCodeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		crw := NewCustomResponseWriter(w)
		next.ServeHTTP(crw, r)
		metrics.StatusCode.WithLabelValues(strconv.Itoa(crw.statusCode)).Inc()
		if crw.statusCode >= 500 {
			log.Error().Int("status", crw.statusCode).Str("path", r.URL.Path).Msg(strings.TrimSpace(crw.body.String()))
		} else if crw.statusCode >= 400 {
			log.Warn().Int("status", crw.statusCode).Str("path", r.URL.Path).Msg("Bad request")
		}
	})
}

type void struct{}

type TlsClientPrincipalFilter struct {
	userCns map[string]void
	emails  map[string]void
}

func NewPrincipalFilter(userCns []string, emails []string) *TlsClientPrincipalFilter {
	allowedUsers := map[string]void{}
	for _, user := range userCns {
		allowedUsers[user] = void{}
	}

	allowedEmails := map[string]void{}
	for _, email := range emails {
		allowedEmails[email] = void{}
	}

	return &TlsClientPrincipalFilter{
		userCns: allowedUsers,
		emails:  allowedEmails,
	}
}

func (t *TlsClientPrincipalFilter) CanProceed(userCn string, emails []string) bool {
	_, foundValidPrincipal := t.userCns[userCn]
	if foundValidPrincipal {
		return true
	}

	for _, email := range emails {
		_, foundValidPrincipal := t.emails[email]
		if foundValidPrincipal {
			return true
		}
	}

	return false
}

func (t *TlsClientPrincipalFilter) tlsClientCertMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connState := r.TLS
		if connState == nil {
			http.Error(w, "No TLS connection state", http.StatusInternalServerError)
			return
		}

		if len(connState.PeerCertificates) == 0 {
			http.Error(w, "No client certificate provided", http.StatusUnauthorized)
			return
		}

		clientCert := connState.PeerCertificates[0]

		clientCN := clientCert.Subject.CommonName
		emails := clientCert.EmailAddresses

		if !t.CanProceed(clientCN, emails) {
			http.Error(w, "No valid auth", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), clientCnKey, clientCN)
		ctx = context.WithValue(ctx, emailsKey, emails)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

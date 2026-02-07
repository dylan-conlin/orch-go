package certs_test

import (
	"crypto/tls"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/certs"
)

func TestEmbeddedCerts_ValidKeyPair(t *testing.T) {
	// Verify embedded certs can form a valid X509 key pair
	_, err := tls.X509KeyPair(certs.CertPEM, certs.KeyPEM)
	if err != nil {
		t.Fatalf("embedded certs do not form valid key pair: %v", err)
	}
}

func TestEmbeddedCerts_NotEmpty(t *testing.T) {
	if len(certs.CertPEM) == 0 {
		t.Error("CertPEM is empty")
	}
	if len(certs.KeyPEM) == 0 {
		t.Error("KeyPEM is empty")
	}
}

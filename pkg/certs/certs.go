// Package certs provides embedded TLS certificates for the orch-go server.
// This allows the binary to be self-contained without requiring external cert files.
package certs

import (
	_ "embed"
)

// CertPEM contains the embedded TLS certificate.
//
//go:embed cert.pem
var CertPEM []byte

// KeyPEM contains the embedded TLS private key.
//
//go:embed key.pem
var KeyPEM []byte

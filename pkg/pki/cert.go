/*
Copyright 2023 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pki

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

type PEMBlockType string

const (
	CertificateBlock PEMBlockType = "CERTIFICATE"
	PrivateKeyBlock  PEMBlockType = "PRIVATE KEY"
)

const DefaultExpiry = 365 * 24 * time.Hour

type CertificateKeyPair struct {
	PrivateKeyPem  []byte
	CertificatePem []byte
	Certificate    *x509.Certificate
	PrivateKey     ed25519.PrivateKey
}

func (c *CertificateKeyPair) CreateCertificate(hostName string, expiry time.Duration) (*CertificateKeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	template, err := populateTemplate(hostName, expiry)
	if err != nil {
		return nil, err
	}
	template.DNSNames = []string{hostName}
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
	certDer, err := x509.CreateCertificate(rand.Reader, template, c.Certificate, pub, c.PrivateKey)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(certDer)
	if err != nil {
		return nil, err
	}
	certPem, privPem, err := encodeKeyPair(cert, priv)
	if err != nil {
		return nil, err
	}
	return &CertificateKeyPair{
		PrivateKeyPem:  privPem,
		CertificatePem: certPem,
		Certificate:    cert,
		PrivateKey:     priv,
	}, nil
}

func encodeKeyPair(cert *x509.Certificate, priv crypto.Signer) (certPem []byte, privPem []byte, err error) {
	certDer := cert.Raw
	privDer, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return
	}
	certPem = pem.EncodeToMemory(&pem.Block{Type: string(CertificateBlock), Bytes: certDer})
	privPem = pem.EncodeToMemory(&pem.Block{Type: string(PrivateKeyBlock), Bytes: privDer})
	return
}

func populateTemplate(commonName string, expiry time.Duration) (*x509.Certificate, error) {
	if expiry == 0 {
		expiry = DefaultExpiry
	}
	sn, err := serialNumber()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &x509.Certificate{
		SerialNumber: sn,
		Subject:      pkix.Name{CommonName: commonName},

		NotBefore: now.Add(-5 * time.Minute),
		NotAfter:  now.Add(expiry),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}, nil
}

// serialNumber generates a new random serial number.
// "serial number is a not greater than 20 bytes long non-negative integer."
func serialNumber() (*big.Int, error) {
	b := make([]byte, 20)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	b[0] = b[0] & 0x7f // make it non-negative, b[0] because of Big-endian
	return new(big.Int).SetBytes(b), nil
}

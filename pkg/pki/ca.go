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
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"time"
)

type CAConfig struct {
	CommonName          string
	PermittedDNSDomains []string
	Expiry              time.Duration
}

func GenerateCA(config *CAConfig) (*CertificateKeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	template, err := populateTemplate(config.CommonName, config.Expiry)
	if err != nil {
		return nil, err
	}
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign
	if len(config.PermittedDNSDomains) > 0 {
		template.PermittedDNSDomainsCritical = true
		template.PermittedDNSDomains = config.PermittedDNSDomains
	}
	certDer, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
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
		Certificate:    cert,
		PrivateKey:     priv,
		CertificatePem: certPem,
		PrivateKeyPem:  privPem,
	}, nil
}

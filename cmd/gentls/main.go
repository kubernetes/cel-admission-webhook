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

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"k8s.io/cel-admission-webhook/pkg/pki"
)

const oneYear = time.Hour * 24 * 365

func generateAndWriteCertificates(hostName string) error {
	ca, err := pki.GenerateCA(&pki.CAConfig{
		CommonName: "SelfSigned",
	})
	if err != nil {
		return err
	}
	err = os.WriteFile("ca.pem", ca.CertificatePem, 0666)
	if err != nil {
		return err
	}
	keyPair, err := ca.CreateCertificate(hostName, oneYear)
	if err != nil {
		return err
	}
	err = os.WriteFile("server.pem", keyPair.CertificatePem, 0666)
	if err != nil {
		return err
	}
	return os.WriteFile("server-key.pem", keyPair.PrivateKeyPem, 0666)
}

func main() {
	var hostName string
	flag.StringVar(&hostName, "host", "example.com", "TLS hostname")
	flag.Parse()
	err := generateAndWriteCertificates(hostName)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

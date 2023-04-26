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
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// TestServing tests the CA and issued certificate against a go
// http server. We skip TLS-level testing in favor of a real-world
// validation logic.
func TestServing(t *testing.T) {
	ca, err := GenerateCA(&CAConfig{
		CommonName:          "ca.local",
		PermittedDNSDomains: []string{"localhost"},
		Expiry:              0,
	})
	if err != nil {
		t.Fatalf("fail to generate CA: %v", err)
	}
	serverCert, err := ca.CreateCertificate("localhost", time.Hour)
	if err != nil {
		t.Fatalf("fail to generate server cert: %v", err)
	}

	ln, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 0,
	})
	if err != nil {
		t.Fatal(err)
	}

	cert, _ := tls.X509KeyPair(serverCert.CertificatePem, serverCert.PrivateKeyPem)
	tlsLn := tls.NewListener(ln, &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2", "http/1.1"},
	})

	// check the status and body for a more "real-world" situation.
	body := "114514"

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(body))
	})

	server := &http.Server{
		Addr:    ln.Addr().String(),
		Handler: mux,
	}
	defer func(server *http.Server, ctx context.Context) {
		_ = server.Shutdown(ctx)
	}(server, context.Background())

	go func() {
		err := server.Serve(tlsLn)
		if err != http.ErrServerClosed {
			panic(err)
		}
	}()

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: NewCertPoolFromCA(ca.Certificate),
			},
			ForceAttemptHTTP2: true,
		},
	}

	// no need to wait for the server to be ready
	// because we manually created the listener
	// the server would Accept() when ready
	resp, err := client.Get(fmt.Sprintf("https://localhost:%d", ln.Addr().(*net.TCPAddr).Port))
	if err != nil {
		t.Fatalf("fail to get: %v", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("fail to read body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %v", resp.StatusCode)
	}
	if string(b) != body {
		t.Errorf("unexpected body: expected %q but got %q", body, b)
	}
}

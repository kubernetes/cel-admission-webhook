package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func runServer(certFile, keyFile, addr string) error {
	config, err := loadClientConfig()
	if err != nil {
		return err
	}
	t, _ := config.TransportConfig()
	u, err := url.Parse(config.Host)
	if err != nil {
		return err
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}
	if len(t.TLS.CAData) > 0 {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(t.TLS.CAData)
		transport.TLSClientConfig.RootCAs = pool
	}
	if len(t.TLS.CertData) > 0 && len(t.TLS.KeyData) > 0 {
		cert, err := tls.X509KeyPair(t.TLS.CertData, t.TLS.KeyData)
		if err != nil {
			return err
		}
		transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}
	r := &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(request *httputil.ProxyRequest) {
			log.Printf("%6s %s", request.In.Method, request.In.URL)
			request.SetURL(u)
			if !strings.Contains(request.In.URL.Path, "admissionregistration.k8s.io/v1alpha1") {
				return
			}
			reader, writer := io.Pipe()
			request.Out.Body = reader
			request.Out.URL.Path = strings.ReplaceAll(request.Out.URL.Path, "admissionregistration.k8s.io/v1alpha1", "admissionregistration.x-k8s.io/v1alpha1")
			request.Out.URL.RawPath = strings.ReplaceAll(request.Out.URL.RawPath, "admissionregistration.k8s.io/v1alpha1", "admissionregistration.x-k8s.io/v1alpha1")
			request.Out.Header.Set("Accept", "application/json")
			request.Out.ContentLength = -1
			go func() {
				scanner := bufio.NewScanner(request.In.Body)
				defer writer.Close()
				for scanner.Scan() {
					b := scanner.Bytes()
					b = bytes.ReplaceAll(b, []byte("admissionregistration.k8s.io/v1alpha1"), []byte("admissionregistration.x-k8s.io/v1alpha1"))
					_, _ = writer.Write(b)
				}
			}()
		},
		ModifyResponse: func(response *http.Response) error {
			if !strings.Contains(response.Request.URL.Path, "admissionregistration.x-k8s.io/v1alpha1") {
				return nil
			}
			b, err := io.ReadAll(response.Body)
			_ = response.Body.Close()
			b = bytes.ReplaceAll(b, []byte("admissionregistration.x-k8s.io/v1alpha1"), []byte("admissionregistration.k8s.io/v1alpha1"))
			response.Body = io.NopCloser(bytes.NewReader(b))
			response.ContentLength = int64(len(b))
			response.Header.Set("Content-Length", strconv.Itoa(len(b)))
			if err != nil {
				return err
			}
			return nil
		},
	}
	return http.ListenAndServeTLS(addr, certFile, keyFile, r)
}

func main() {
	var certFile string
	var keyFile string
	var addr string
	flag.StringVar(&certFile, "cert", "server.pem", "Path to TLS certificate file.")
	flag.StringVar(&keyFile, "key", "server-key.pem", "Path to TLS key file.")
	flag.StringVar(&addr, "addr", "0.0.0.0:8443", "Address to listen on")
	flag.Parse()
	err := runServer(certFile, keyFile, addr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func loadClientConfig() (*rest.Config, error) {
	// Connect to k8s
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here

	configOverrides := &clientcmd.ConfigOverrides{}
	// if you want to change override values or bind them to flags, there are methods to help you

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	if config, err := kubeConfig.ClientConfig(); err == nil {
		return config, nil
	}

	// untested. assuming this is how it might work when run from inside clsuter
	return rest.InClusterConfig()
}

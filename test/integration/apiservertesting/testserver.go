package apiservertesting

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
)

// This key is for testing purposes only and is not considered secure.
const ecdsaPrivateKey = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIEZmTmUhuanLjPA2CLquXivuwBDHTt5XYwgIr/kA1LtRoAoGCCqGSM49
AwEHoUQDQgAEH6cuzP8XuD5wal6wf9M6xDljTOPLX2i8uIp/C/ASqiIGUeeKQtX0
/IR3qCXyThP/dbCiHrF3v1cuhBOHY8CLVg==
-----END EC PRIVATE KEY-----`

// TearDownFunc is to be called to tear down a test server.
type TearDownFunc func()

// TestServerInstanceOptions Instance options the TestServer
type TestServerInstanceOptions struct {
}

// TestServer return values supplied by kube-test-ApiServer
type TestServer struct {
	ClientConfig *restclient.Config // Rest client config
	TearDownFn   TearDownFunc       // TearDown function
	TmpDir       string             // Temp Dir used, by the apiserver
}

// Logger allows t.Testing and b.Testing to be passed to StartTestServer and StartTestServerOrDie
type Logger interface {
	Helper()
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Logf(format string, args ...interface{})
}

type errOutput struct {
	logger Logger
}

func (e errOutput) Write(p []byte) (n int, err error) {
	e.logger.Logf("%v", string(p))
	return len(p), nil
}

type logOutput struct {
	logger Logger
}

func (e logOutput) Write(p []byte) (n int, err error) {
	e.logger.Logf("%v", string(p))
	return len(p), nil
}

func StartTestServer(t Logger, instanceOptions *TestServerInstanceOptions, customFlags []string, storageConfig *storagebackend.Config) (result TestServer, err error) {
	saSigningKeyFile, err := os.CreateTemp("/tmp", "insecure_test_key")
	if err != nil {
		t.Fatalf("create temp file failed: %v", err)
	}
	defer os.RemoveAll(saSigningKeyFile.Name())
	if err = os.WriteFile(saSigningKeyFile.Name(), []byte(ecdsaPrivateKey), 0666); err != nil {
		t.Fatalf("write file %s failed: %v", saSigningKeyFile.Name(), err)
	}

	tokenAuthFile, err := os.CreateTemp("/tmp", "token_auth_file")
	if err != nil {
		t.Fatalf("create token auth file failed: %v", err)
	}
	defer os.RemoveAll(tokenAuthFile.Name())
	token := string(uuid.NewUUID())
	if err = os.WriteFile(tokenAuthFile.Name(), []byte(fmt.Sprintf("%v,system:apiserver,0,\"system:masters\"", token)), 0666); err != nil {
		t.Fatalf("write file %s failed: %v", tokenAuthFile.Name(), err)
	}

	certDir, err := os.MkdirTemp("", "kubernetes-kube-apiserver")
	if err != nil {
		t.Fatalf("create apiserver temp dir failed: %v", err)
	}
	// defer os.RemoveAll(certDir)

	apiServerBinary := "/Users/alex/go/src/k8s.io/kubernetes/_output/local/bin/darwin/arm64/kube-apiserver"
	args := append([]string{
		"--bind-address=0.0.0.0",
		"--v=3",
		"--vmodule=",
		// "--authorization-mode=Node,RBAC",
		"--storage-backend=" + storageConfig.Type,
		"--etcd-prefix=" + storageConfig.Prefix,
		"--etcd-cafile=" + storageConfig.Transport.TrustedCAFile,
		"--etcd-certfile=" + storageConfig.Transport.CertFile,
		"--etcd-keyfile=" + storageConfig.Transport.KeyFile,
		"--etcd-servers=" + strings.Join(storageConfig.Transport.ServerList, ","),
		// "--kubelet-client-certificate=/Users/alex/cluster-pki/apiserver-kubelet-client.crt",
		// "--kubelet-client-key=/Users/alex/cluster-pki/apiserver-kubelet-client.key",
		// "--kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname",
		// "--proxy-client-cert-file=/Users/alex/cluster-pki/front-proxy-client.crt",
		// "--proxy-client-key-file=/Users/alex/cluster-pki/front-proxy-client.key",
		// "--requestheader-allowed-names=front-proxy-client",
		// "--requestheader-client-ca-file=/Users/alex/cluster-pki/front-proxy-ca.crt",
		// "--requestheader-extra-headers-prefix=X-Remote-Extra-",
		// "--requestheader-group-headers=X-Remote-Group",
		// "--requestheader-username-headers=X-Remote-User",
		// "--runtime-config=",
		"--secure-port=6009",
		"--service-account-issuer=https://foo.bar.example.com",
		"--service-account-key-file=" + saSigningKeyFile.Name(),
		"--service-account-signing-key-file=" + saSigningKeyFile.Name(),
		"--service-cluster-ip-range=10.96.0.0/16",
		"--token-auth-file=" + tokenAuthFile.Name(),
		"--cert-dir=" + certDir,
		"--feature-gates=ValidatingAdmissionPolicy=true",
		"--runtime-config=admissionregistration.k8s.io/v1alpha1=true",
		// "--client-ca-file=/Users/alex/cluster-pki/ca.crt",
		// "--tls-cert-file=/Users/alex/cluster-pki/apiserver.crt",
		// "--tls-private-key-file=/Users/alex/cluster-pki/apiserver.key",
	}, customFlags...)

	ctx, canceller := context.WithCancel(context.TODO())
	invocation := exec.CommandContext(ctx, apiServerBinary, args...)
	invocation.Stdout = logOutput{t}
	invocation.Stderr = errOutput{t}

	if err := invocation.Start(); err != nil {
		canceller()
		return result, err
	}

	//TODO: hookup pipes to logger

	errCh := make(chan error)
	go func() {
		defer canceller()
		defer close(errCh)

		errCh <- invocation.Wait()
	}()

	t.Logf("Waiting for /healthz to be ok...")
	config := &rest.Config{
		Host: "https://0.0.0.0:6009",
		// ContentConfig: restclient.ContentConfig{
		// 	ContentType: "",
		// },
		BearerToken: token,
		TLSClientConfig: restclient.TLSClientConfig{
			Insecure: true,
		},
		DisableCompression: true,
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return result, fmt.Errorf("failed to create a client: %v", err)
	}

	// wait until healthz endpoint returns ok
	err = wait.Poll(100*time.Millisecond, time.Minute, func() (bool, error) {
		select {
		case err := <-errCh:
			return false, err
		default:
		}

		req := client.CoreV1().RESTClient().Get().AbsPath("/healthz")
		// // The storage version bootstrap test wraps the storage version post-start
		// // hook, so the hook won't become health when the server bootstraps
		// if instanceOptions.StorageVersionWrapFunc != nil {
		// 	// We hardcode the param instead of having a new instanceOptions field
		// 	// to avoid confusing users with more options.
		// 	storageVersionCheck := fmt.Sprintf("poststarthook/%s", apiserver.StorageVersionPostStartHookName)
		// 	req.Param("exclude", storageVersionCheck)
		// }
		result := req.Do(context.TODO())
		status := 0
		result.StatusCode(&status)
		if status == 200 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return result, fmt.Errorf("failed to wait for /healthz to return ok: %v", err)
	}

	// wait until default namespace is created
	err = wait.Poll(100*time.Millisecond, 30*time.Second, func() (bool, error) {
		select {
		case err := <-errCh:
			return false, err
		default:
		}

		if _, err := client.CoreV1().Namespaces().Get(context.TODO(), "default", metav1.GetOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				t.Logf("Unable to get default namespace: %v", err)
			}
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return result, fmt.Errorf("failed to wait for default namespace to be created: %v", err)
	}

	// from here the caller must call tearDown
	result.ClientConfig = restclient.CopyConfig(config)
	result.ClientConfig.QPS = 1000
	result.ClientConfig.Burst = 10000
	result.TearDownFn = func() {
		canceller()
	}

	return result, nil
}

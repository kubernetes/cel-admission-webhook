//go:build tools

package celadmissionwebhook

// Used to force direct dependency on code-generator so that it stays in vendor
// directory for go run.
import (
	_ "k8s.io/code-generator"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"

	// yq is used for patching CRDs after they are generated
	_ "github.com/mikefarah/yq/v4"
)

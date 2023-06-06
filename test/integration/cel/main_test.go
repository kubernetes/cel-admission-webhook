package cel

import (
	"testing"

	"k8s.io/cel-admission-webhook/test/integration/framework"
)

func TestMain(m *testing.M) {
	framework.EtcdMain(m.Run)
}

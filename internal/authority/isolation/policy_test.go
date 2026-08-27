/*
FactoryDocSync:
docs:
- docs/features/F-002-work-authority.md
- docs/design-docs/ADR-001-git-beads-authority.md
- docs/design-docs/ADR-003-rule-of-two.md
- docs/code-documentation-map.md
*/

package isolation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthorityDeploymentFailsClosedAgainstAgentAndHostAccess(t *testing.T) {
	root := filepath.Join("..", "..", "..", "deploy", "authority")
	namespaces := readFixture(t, filepath.Join(root, "namespaces.yaml"))
	policies := readFixture(t, filepath.Join(root, "network-policies.yaml"))
	gateway := readFixture(t, filepath.Join(root, "gateway.yaml"))

	for _, required := range []string{
		"name: mars3-authority", "name: mars3-agent-sandboxes",
		"name: authority-gateway", "automountServiceAccountToken: false",
	} {
		assertContains(t, namespaces, required)
	}
	for _, required := range []string{
		"name: deny-all-agent-sandbox-traffic", "name: deny-all-authority-traffic",
		"name: allow-control-plane-to-gateway", "mars3.io/authority-client: trusted",
		"name: allow-gateway-to-authority-stores", "app.kubernetes.io/name: mars3-authority-postgres",
		"app.kubernetes.io/name: mars3-authority-beads", "port: 5432", "port: 7443",
	} {
		assertContains(t, policies, required)
	}
	for _, forbidden := range []string{
		"namespace: mars3-agent-sandboxes\n  podSelector:\n    matchLabels:\n      app.kubernetes.io/name: mars3-authority",
		"0.0.0.0/0", "ipBlock:", "except:", "NodePort", "LoadBalancer",
	} {
		assertAbsent(t, policies, forbidden)
	}
	for _, required := range []string{
		"replicas: 2", "serviceAccountName: authority-gateway", "automountServiceAccountToken: false",
		"hostNetwork: false", "hostPID: false", "hostIPC: false", "type: RuntimeDefault",
		"allowPrivilegeEscalation: false", "readOnlyRootFilesystem: true", "runAsNonRoot: true",
		"drop:\n                - ALL", "type: ClusterIP",
	} {
		assertContains(t, gateway, required)
	}
	for _, forbidden := range []string{"secretKeyRef:", "envFrom:", "hostPath:", "privileged: true", "serviceAccountToken:", "api-token", "password"} {
		assertAbsent(t, gateway, forbidden)
	}
	if strings.Count(gateway, "@sha256:") != 1 || strings.Contains(gateway, ":latest") {
		t.Fatal("gateway image must have exactly one immutable digest and no mutable tag")
	}
}

func readFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, content, value string) {
	t.Helper()
	if !strings.Contains(content, value) {
		t.Fatalf("deployment is missing required bounded control %q", value)
	}
}

func assertAbsent(t *testing.T, content, value string) {
	t.Helper()
	if strings.Contains(content, value) {
		t.Fatalf("deployment contains forbidden authority surface %q", value)
	}
}

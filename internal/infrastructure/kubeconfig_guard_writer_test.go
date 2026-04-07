package infrastructure

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestGuardKubeconfigWriterRewritesClusterServer(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "source.yaml")
	caPath := filepath.Join(tempDir, "ca.crt")
	clientCertPath := filepath.Join(tempDir, "client.crt")
	clientKeyPath := filepath.Join(tempDir, "client.key")

	for _, path := range []string{caPath, clientCertPath, clientKeyPath} {
		if err := os.WriteFile(path, []byte("test"), filePermission); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}

	config := clientcmdapi.NewConfig()
	config.CurrentContext = "prod"
	config.Clusters["prod-cluster"] = &clientcmdapi.Cluster{
		Server:               "https://prod.example.com",
		CertificateAuthority: "ca.crt",
	}
	config.AuthInfos["prod-user"] = &clientcmdapi.AuthInfo{
		Token:             "secret-token",
		ClientCertificate: "client.crt",
		ClientKey:         "client.key",
		Exec: &clientcmdapi.ExecConfig{
			Command: "aws",
			Args:    []string{"eks", "get-token"},
		},
	}
	config.Contexts["prod"] = &clientcmdapi.Context{
		Cluster:   "prod-cluster",
		AuthInfo:  "prod-user",
		Namespace: "payments",
	}

	data, err := clientcmd.Write(*config)
	if err != nil {
		t.Fatalf("clientcmd.Write() error = %v", err)
	}

	if err := os.WriteFile(sourcePath, data, filePermission); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}

	writer := NewGuardKubeconfigWriter()
	outputPath := filepath.Join(tempDir, "guard", "config")
	result, err := writer.Write(sourcePath, "prod", "http://127.0.0.1:41111", outputPath)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if result.Path != outputPath {
		t.Fatalf("Write() path = %q, want %q", result.Path, outputPath)
	}

	guarded, err := clientcmd.LoadFromFile(outputPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if guarded.CurrentContext != "prod" {
		t.Fatalf("CurrentContext = %q, want %q", guarded.CurrentContext, "prod")
	}

	if len(guarded.Contexts) != 1 || len(guarded.Clusters) != 1 || len(guarded.AuthInfos) != 1 {
		t.Fatalf("guarded config counts = contexts:%d clusters:%d authinfos:%d", len(guarded.Contexts), len(guarded.Clusters), len(guarded.AuthInfos))
	}

	if guarded.Clusters["prod-cluster"].Server != "http://127.0.0.1:41111" {
		t.Fatalf("cluster server = %q, want %q", guarded.Clusters["prod-cluster"].Server, "http://127.0.0.1:41111")
	}

	if guarded.AuthInfos["prod-user"].Token != "secret-token" {
		t.Fatalf("token = %q, want %q", guarded.AuthInfos["prod-user"].Token, "secret-token")
	}

	if guarded.AuthInfos["prod-user"].Exec == nil || guarded.AuthInfos["prod-user"].Exec.Command != "aws" {
		t.Fatalf("exec config = %+v", guarded.AuthInfos["prod-user"].Exec)
	}

	if !filepath.IsAbs(guarded.Clusters["prod-cluster"].CertificateAuthority) {
		t.Fatalf("certificate authority path = %q, want absolute path", guarded.Clusters["prod-cluster"].CertificateAuthority)
	}

	if !filepath.IsAbs(guarded.AuthInfos["prod-user"].ClientCertificate) {
		t.Fatalf("client certificate path = %q, want absolute path", guarded.AuthInfos["prod-user"].ClientCertificate)
	}
}

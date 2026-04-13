package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type GuardKubeconfigWriter struct{}

func NewGuardKubeconfigWriter() *GuardKubeconfigWriter {
	return &GuardKubeconfigWriter{}
}

func (w *GuardKubeconfigWriter) Write(sourcePath, contextName, proxyAddress, outputPath string) (*domain.GuardedKubeconfigResult, error) {
	config, err := clientcmd.LoadFromFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	if contextName == "" {
		contextName = config.CurrentContext
	}

	contextEntry, ok := config.Contexts[contextName]
	if !ok {
		return nil, domain.ErrContextNotFound
	}

	clusterEntry, ok := config.Clusters[contextEntry.Cluster]
	if !ok {
		return nil, domain.ErrClusterNotFound
	}

	authInfoEntry, ok := config.AuthInfos[contextEntry.AuthInfo]
	if !ok {
		return nil, domain.ErrUserNotFound
	}

	guarded := clientcmdapi.NewConfig()
	guarded.CurrentContext = contextName
	guarded.Contexts[contextName] = contextEntry.DeepCopy()
	guarded.Clusters[contextEntry.Cluster] = clusterEntry.DeepCopy()
	guarded.AuthInfos[contextEntry.AuthInfo] = authInfoEntry.DeepCopy()
	guarded.Clusters[contextEntry.Cluster].Server = proxyAddress

	resolveKubeconfigPaths(sourcePath, guarded.Clusters[contextEntry.Cluster], guarded.AuthInfos[contextEntry.AuthInfo])

	if err := os.MkdirAll(filepath.Dir(outputPath), dirPermission); err != nil {
		return nil, fmt.Errorf("create guard kubeconfig directory: %w", err)
	}

	data, err := clientcmd.Write(*guarded)
	if err != nil {
		return nil, fmt.Errorf("serialize guarded kubeconfig: %w", err)
	}

	if err := os.WriteFile(outputPath, data, filePermission); err != nil {
		return nil, fmt.Errorf("write guarded kubeconfig: %w", err)
	}

	return &domain.GuardedKubeconfigResult{
		Path:      outputPath,
		Context:   contextName,
		Namespace: contextEntry.Namespace,
	}, nil
}

func (w *GuardKubeconfigWriter) Cleanup(outputPath string) error {
	if outputPath == "" {
		return nil
	}

	if err := os.RemoveAll(filepath.Dir(outputPath)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove guard artifacts: %w", err)
	}

	return nil
}

func resolveKubeconfigPaths(sourcePath string, cluster *clientcmdapi.Cluster, authInfo *clientcmdapi.AuthInfo) {
	baseDir := filepath.Dir(sourcePath)

	cluster.CertificateAuthority = resolvePath(baseDir, cluster.CertificateAuthority)
	authInfo.ClientCertificate = resolvePath(baseDir, authInfo.ClientCertificate)
	authInfo.ClientKey = resolvePath(baseDir, authInfo.ClientKey)
	authInfo.TokenFile = resolvePath(baseDir, authInfo.TokenFile)
}

func resolvePath(baseDir, value string) string {
	if value == "" || filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(baseDir, value)
}

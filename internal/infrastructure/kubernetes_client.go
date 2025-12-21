package infrastructure

import (
	"context"
	"os/exec"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type KubernetesClient struct {
	kubeconfigPath string
}

func NewKubernetesClient(kubeconfigPath string) *KubernetesClient {
	return &KubernetesClient{kubeconfigPath: kubeconfigPath}
}

func (k *KubernetesClient) ListNamespaces() ([]string, error) {
	// Try using the native Kubernetes client first
	namespaces, err := k.listNamespacesWithClient()
	if err == nil {
		return namespaces, nil
	}

	// If native client fails (e.g., due to auth issues), fall back to kubectl
	return k.listNamespacesWithKubectl()
}

// listNamespacesWithClient uses the native Kubernetes Go client
func (k *KubernetesClient) listNamespacesWithClient() ([]string, error) {
	config, err := clientcmd.BuildConfigFromFlags("", k.kubeconfigPath)
	if err != nil {
		return nil, err
	}

	config.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	namespaceList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	namespaces := make([]string, len(namespaceList.Items))
	for i, ns := range namespaceList.Items {
		namespaces[i] = ns.Name
	}

	return namespaces, nil
}

func (k *KubernetesClient) listNamespacesWithKubectl() ([]string, error) {
	cmd := exec.Command("kubectl", "get", "namespaces", "-o", "name", "--kubeconfig", k.kubeconfigPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	namespaces := make([]string, 0, len(lines))

	for _, line := range lines {
		// kubectl returns "namespace/name" format, extract just the name
		if strings.HasPrefix(line, "namespace/") {
			nsName := strings.TrimPrefix(line, "namespace/")
			if nsName != "" {
				namespaces = append(namespaces, nsName)
			}
		}
	}

	return namespaces, nil
}

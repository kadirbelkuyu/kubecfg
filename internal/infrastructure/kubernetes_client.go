package infrastructure

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type KubernetesClient struct {
	kubeconfigPath string
}

func NewKubernetesClient(kubeconfigPath string) *KubernetesClient {
	return &KubernetesClient{kubeconfigPath: kubeconfigPath}
}

func (k *KubernetesClient) ListNamespaces() ([]string, error) {
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

package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var (
	kubeconfigPath string
	v              *viper.Viper
)

func Init() {
	v = viper.New()
	v.SetEnvPrefix("kubecfg")
	v.AutomaticEnv()

	v.SetDefault("kubeconfig", getDefaultKubeconfigPath())
}

func getDefaultKubeconfigPath() string {
	if envPath := os.Getenv("KUBECONFIG"); envPath != "" {
		return envPath
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}

func SetKubeconfigPath(path string) {
	kubeconfigPath = path
}

func GetKubeconfigPath() string {
	if kubeconfigPath != "" {
		return kubeconfigPath
	}
	return v.GetString("kubeconfig")
}

package config

import (
	"os"
	"path/filepath"
	"time"

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
	v.SetDefault("guard.state_dir", getDefaultGuardStateDir())
	v.SetDefault("guard.session_path", filepath.Join(getDefaultGuardStateDir(), "session.json"))
	v.SetDefault("guard.default_ttl", "30m")
	v.SetDefault("audit.enabled", true)
	v.SetDefault("audit.path", filepath.Join(getDefaultGuardStateDir(), "audit.log"))
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

func GetGuardStateDir() string {
	return v.GetString("guard.state_dir")
}

func GetGuardSessionPath() string {
	return v.GetString("guard.session_path")
}

func GetGuardDefaultTTL() time.Duration {
	ttl, err := time.ParseDuration(v.GetString("guard.default_ttl"))
	if err != nil {
		return 30 * time.Minute
	}
	return ttl
}

func IsAuditEnabled() bool {
	return v.GetBool("audit.enabled")
}

func GetAuditPath() string {
	return v.GetString("audit.path")
}

func getDefaultGuardStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kubecfg")
}

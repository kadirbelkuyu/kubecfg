package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// ProfileConfig mirrors a single entry under the `profiles:` YAML key.
type ProfileConfig struct {
	Readonly            bool     `mapstructure:"readonly"`
	ConfirmDestructive  bool     `mapstructure:"confirm_destructive"`
	BlockedVerbs        []string `mapstructure:"blocked_verbs"`
	AllowedNamespaces   []string `mapstructure:"allowed_namespaces"`
	BlockedResources    []string `mapstructure:"blocked_resources"`
	WarnContextPatterns []string `mapstructure:"warn_context_patterns"`
	Description         string   `mapstructure:"description"`
}

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

	// Attempt to load ~/.kubecfg/config.yaml; ignore if missing.
	v.SetConfigType("yaml")
	v.AddConfigPath(getDefaultGuardStateDir())
	v.SetConfigName("config")
	_ = v.ReadInConfig()
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
	// Prefer sessions.default_ttl (new YAML structure) over the legacy guard.default_ttl key.
	if s := v.GetString("sessions.default_ttl"); s != "" {
		if ttl, err := time.ParseDuration(s); err == nil {
			return ttl
		}
	}
	ttl, err := time.ParseDuration(v.GetString("guard.default_ttl"))
	if err != nil {
		return 30 * time.Minute
	}
	return ttl
}

// GetProfiles returns user-defined policy profiles from the config file.
// Returns nil (empty map) when the profiles section is absent or unparseable.
func GetProfiles() map[string]ProfileConfig {
	var profiles map[string]ProfileConfig
	if err := v.UnmarshalKey("profiles", &profiles); err != nil {
		return nil
	}
	return profiles
}

func IsAuditEnabled() bool {
	return v.GetBool("audit.enabled")
}

func GetAuditPath() string {
	return v.GetString("audit.path")
}

func GetConfirmationsDir() string {
	return filepath.Join(getDefaultGuardStateDir(), "confirmations")
}

func GetLastContextPath() string {
	return filepath.Join(getDefaultGuardStateDir(), "last-context")
}

func getDefaultGuardStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kubecfg")
}

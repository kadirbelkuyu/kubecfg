package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
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
	v.SetDefault("kubeconfig_sources.dirs", []string{getDefaultKubeconfigDir()})
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

func getDefaultKubeconfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kube")
}

func SetKubeconfigPath(path string) {
	kubeconfigPath = path
}

func GetKubeconfigPath() string {
	if kubeconfigPath != "" {
		return ExpandPath(kubeconfigPath)
	}
	if activePath := strings.TrimSpace(v.GetString("kubeconfig_sources.active")); activePath != "" {
		return ExpandPath(activePath)
	}
	return ExpandPath(v.GetString("kubeconfig"))
}

func GetKubeconfigSourceDirs() []string {
	dirs := append([]string{getDefaultKubeconfigDir()}, v.GetStringSlice("kubeconfig_sources.dirs")...)
	return normalizePathList(dirs)
}

func SetActiveKubeconfigPath(path string) error {
	normalized, err := normalizeExistingFile(path)
	if err != nil {
		return err
	}

	doc, err := loadConfigDocument()
	if err != nil {
		return err
	}
	root := ensureDocumentMapping(doc)
	sources := ensureMappingValue(root, "kubeconfig_sources")
	setScalarValue(sources, "active", normalized)

	if err := writeConfigDocument(doc); err != nil {
		return err
	}

	v.Set("kubeconfig_sources.active", normalized)
	kubeconfigPath = normalized
	return nil
}

func AddKubeconfigSourceDir(dir string) error {
	normalized, err := normalizeExistingDir(dir)
	if err != nil {
		return err
	}

	dirs := GetKubeconfigSourceDirs()
	if slices.Contains(dirs, normalized) {
		return nil
	}
	dirs = append(dirs, normalized)

	return persistKubeconfigSourceDirs(dirs)
}

func RemoveKubeconfigSourceDir(dir string) error {
	normalized := normalizePath(dir)
	if normalized == "" {
		return fmt.Errorf("source directory is required")
	}

	dirs := GetKubeconfigSourceDirs()
	next := make([]string, 0, len(dirs))
	for _, existing := range dirs {
		if existing != normalized {
			next = append(next, existing)
		}
	}

	return persistKubeconfigSourceDirs(next)
}

func persistKubeconfigSourceDirs(dirs []string) error {
	dirs = normalizePathList(dirs)

	doc, err := loadConfigDocument()
	if err != nil {
		return err
	}
	root := ensureDocumentMapping(doc)
	sources := ensureMappingValue(root, "kubeconfig_sources")
	setStringSliceValue(sources, "dirs", dirs)

	if err := writeConfigDocument(doc); err != nil {
		return err
	}

	v.Set("kubeconfig_sources.dirs", dirs)
	return nil
}

func ExpandPath(path string) string {
	return normalizePath(path)
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

func GetGroupsPath() string {
	return filepath.Join(getDefaultGuardStateDir(), "groups.yaml")
}

func getDefaultGuardStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kubecfg")
}

func getConfigFilePath() string {
	return filepath.Join(getDefaultGuardStateDir(), "config.yaml")
}

func normalizeExistingFile(path string) (string, error) {
	normalized := normalizePath(path)
	if normalized == "" {
		return "", fmt.Errorf("kubeconfig path is required")
	}

	info, err := os.Stat(normalized)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("kubeconfig file not found: %s", normalized)
		}
		return "", fmt.Errorf("stat kubeconfig file: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("kubeconfig path is a directory: %s", normalized)
	}

	return normalized, nil
}

func normalizeExistingDir(path string) (string, error) {
	normalized := normalizePath(path)
	if normalized == "" {
		return "", fmt.Errorf("source directory is required")
	}

	info, err := os.Stat(normalized)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("source directory not found: %s", normalized)
		}
		return "", fmt.Errorf("stat source directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("source path is not a directory: %s", normalized)
	}

	return normalized, nil
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			path = home
		}
	} else if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}

	path = os.ExpandEnv(path)
	if absolute, err := filepath.Abs(path); err == nil {
		path = absolute
	}
	return filepath.Clean(path)
}

func normalizePathList(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		path = normalizePath(path)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		normalized = append(normalized, path)
	}
	return normalized
}

func loadConfigDocument() (*yaml.Node, error) {
	root, err := os.OpenRoot(getDefaultGuardStateDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newConfigDocument(), nil
		}
		return nil, fmt.Errorf("open config directory: %w", err)
	}
	defer func() {
		_ = root.Close()
	}()

	data, err := root.ReadFile("config.yaml")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newConfigDocument(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return newConfigDocument(), nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if doc.Kind == 0 {
		return newConfigDocument(), nil
	}

	return &doc, nil
}

func newConfigDocument() *yaml.Node {
	return &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{{
			Kind: yaml.MappingNode,
		}},
	}
}

func writeConfigDocument(doc *yaml.Node) error {
	path := getConfigFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func ensureDocumentMapping(doc *yaml.Node) *yaml.Node {
	if doc.Kind != yaml.DocumentNode {
		doc.Kind = yaml.DocumentNode
	}
	if len(doc.Content) == 0 || doc.Content[0] == nil {
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		doc.Content[0] = &yaml.Node{Kind: yaml.MappingNode}
	}
	return doc.Content[0]
}

func ensureMappingValue(root *yaml.Node, key string) *yaml.Node {
	if value := findMappingValue(root, key); value != nil {
		if value.Kind != yaml.MappingNode {
			value.Kind = yaml.MappingNode
			value.Tag = "!!map"
			value.Value = ""
			value.Content = nil
		}
		return value
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	valueNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	root.Content = append(root.Content, keyNode, valueNode)
	return valueNode
}

func setScalarValue(root *yaml.Node, key, value string) {
	valueNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	setMappingValue(root, key, valueNode)
}

func setStringSliceValue(root *yaml.Node, key string, values []string) {
	sequence := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, value := range values {
		sequence.Content = append(sequence.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
	}
	setMappingValue(root, key, sequence)
}

func setMappingValue(root *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			root.Content[i+1] = value
			return
		}
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	root.Content = append(root.Content, keyNode, value)
}

func findMappingValue(root *yaml.Node, key string) *yaml.Node {
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i+1]
		}
	}
	return nil
}

package domain

type KubeConfig struct {
	APIVersion     string           `yaml:"apiVersion"`
	Kind           string           `yaml:"kind"`
	CurrentContext string           `yaml:"current-context"`
	Clusters       []ClusterEntry   `yaml:"clusters"`
	Contexts       []ContextEntry   `yaml:"contexts"`
	Users          []UserEntry      `yaml:"users"`
	Preferences    map[string]any   `yaml:"preferences,omitempty"`
}

type ClusterEntry struct {
	Name    string  `yaml:"name"`
	Cluster Cluster `yaml:"cluster"`
}

type Cluster struct {
	Server                   string `yaml:"server"`
	CertificateAuthority     string `yaml:"certificate-authority,omitempty"`
	CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
	InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify,omitempty"`
}

type ContextEntry struct {
	Name    string  `yaml:"name"`
	Context Context `yaml:"context"`
}

type Context struct {
	Cluster   string `yaml:"cluster"`
	User      string `yaml:"user"`
	Namespace string `yaml:"namespace,omitempty"`
}

type UserEntry struct {
	Name string `yaml:"name"`
	User User   `yaml:"user"`
}

type User struct {
	ClientCertificate     string     `yaml:"client-certificate,omitempty"`
	ClientCertificateData string     `yaml:"client-certificate-data,omitempty"`
	ClientKey             string     `yaml:"client-key,omitempty"`
	ClientKeyData         string     `yaml:"client-key-data,omitempty"`
	Token                 string     `yaml:"token,omitempty"`
	Username              string     `yaml:"username,omitempty"`
	Password              string     `yaml:"password,omitempty"`
	Exec                  *ExecConfig `yaml:"exec,omitempty"`
	AuthProvider          *AuthProviderConfig `yaml:"auth-provider,omitempty"`
}

type ExecConfig struct {
	APIVersion         string            `yaml:"apiVersion,omitempty"`
	Command            string            `yaml:"command"`
	Args               []string          `yaml:"args,omitempty"`
	Env                []ExecEnvVar      `yaml:"env,omitempty"`
	InstallHint        string            `yaml:"installHint,omitempty"`
	ProvideClusterInfo bool              `yaml:"provideClusterInfo,omitempty"`
	InteractiveMode    string            `yaml:"interactiveMode,omitempty"`
}

type ExecEnvVar struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type AuthProviderConfig struct {
	Name   string            `yaml:"name"`
	Config map[string]string `yaml:"config,omitempty"`
}

func NewKubeConfig() *KubeConfig {
	return &KubeConfig{
		APIVersion:  "v1",
		Kind:        "Config",
		Clusters:    make([]ClusterEntry, 0),
		Contexts:    make([]ContextEntry, 0),
		Users:       make([]UserEntry, 0),
		Preferences: make(map[string]any),
	}
}

func (k *KubeConfig) FindCluster(name string) (*ClusterEntry, int) {
	for i, c := range k.Clusters {
		if c.Name == name {
			return &k.Clusters[i], i
		}
	}
	return nil, -1
}

func (k *KubeConfig) FindContext(name string) (*ContextEntry, int) {
	for i, c := range k.Contexts {
		if c.Name == name {
			return &k.Contexts[i], i
		}
	}
	return nil, -1
}

func (k *KubeConfig) FindUser(name string) (*UserEntry, int) {
	for i, u := range k.Users {
		if u.Name == name {
			return &k.Users[i], i
		}
	}
	return nil, -1
}

func (k *KubeConfig) RemoveCluster(name string) bool {
	_, idx := k.FindCluster(name)
	if idx < 0 {
		return false
	}
	k.Clusters = append(k.Clusters[:idx], k.Clusters[idx+1:]...)
	return true
}

func (k *KubeConfig) RemoveContext(name string) bool {
	_, idx := k.FindContext(name)
	if idx < 0 {
		return false
	}
	k.Contexts = append(k.Contexts[:idx], k.Contexts[idx+1:]...)
	return true
}

func (k *KubeConfig) RemoveUser(name string) bool {
	_, idx := k.FindUser(name)
	if idx < 0 {
		return false
	}
	k.Users = append(k.Users[:idx], k.Users[idx+1:]...)
	return true
}

func (k *KubeConfig) ContextNames() []string {
	names := make([]string, len(k.Contexts))
	for i, c := range k.Contexts {
		names[i] = c.Name
	}
	return names
}

package domain

// ContextDefinition is the editable slice of a kubeconfig that describes a
// single context: the context entry plus the cluster and user it references.
type ContextDefinition struct {
	Context ContextEntry
	Cluster ClusterEntry
	User    UserEntry
}

// ExtractContextDefinition returns the definition of the named context.
func (k *KubeConfig) ExtractContextDefinition(name string) (*ContextDefinition, error) {
	ctx, idx := k.FindContext(name)
	if idx < 0 {
		return nil, ErrContextNotFound
	}

	cluster, clusterIdx := k.FindCluster(ctx.Context.Cluster)
	if clusterIdx < 0 {
		return nil, ErrClusterNotFound
	}

	user, userIdx := k.FindUser(ctx.Context.User)
	if userIdx < 0 {
		return nil, ErrUserNotFound
	}

	return &ContextDefinition{Context: *ctx, Cluster: *cluster, User: *user}, nil
}

// SoleContextDefinition returns the definition of the only context in the
// config, or of current-context when the config holds more than one.
func (k *KubeConfig) SoleContextDefinition() (*ContextDefinition, error) {
	switch {
	case len(k.Contexts) == 0:
		return nil, ErrInvalidConfig
	case len(k.Contexts) == 1:
		return k.ExtractContextDefinition(k.Contexts[0].Name)
	case k.CurrentContext != "":
		return k.ExtractContextDefinition(k.CurrentContext)
	default:
		return nil, ErrAmbiguousContext
	}
}

// ToKubeConfig renders the definition as a standalone single-context kubeconfig.
func (d *ContextDefinition) ToKubeConfig() *KubeConfig {
	config := NewKubeConfig()
	config.CurrentContext = d.Context.Name
	config.Contexts = append(config.Contexts, d.Context)
	config.Clusters = append(config.Clusters, d.Cluster)
	config.Users = append(config.Users, d.User)

	return config
}

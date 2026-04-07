package application

import (
	"fmt"
	"strings"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type Service struct {
	repo domain.Repository
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetDefaultPath() string {
	return s.repo.GetDefaultPath()
}

func (s *Service) AddConfig(sourcePath, targetPath, contextName string) error {
	sourceConfig, err := s.repo.Load(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to load source config: %w", err)
	}

	var targetConfig *domain.KubeConfig
	if s.repo.Exists(targetPath) {
		if err := s.repo.Backup(targetPath); err != nil {
			return err
		}
		targetConfig, err = s.repo.Load(targetPath)
		if err != nil {
			return fmt.Errorf("failed to load target config: %w", err)
		}
	} else {
		targetConfig = domain.NewKubeConfig()
	}

	if len(sourceConfig.Contexts) == 0 {
		return domain.ErrInvalidConfig
	}

	originalContext := sourceConfig.Contexts[0]
	originalCluster := originalContext.Context.Cluster
	originalUser := originalContext.Context.User

	newClusterName := contextName
	newUserName := contextName + "-user"

	if _, idx := targetConfig.FindContext(contextName); idx >= 0 {
		return domain.ErrContextExists
	}

	for _, cluster := range sourceConfig.Clusters {
		if cluster.Name == originalCluster {
			newCluster := cluster
			newCluster.Name = newClusterName
			targetConfig.Clusters = append(targetConfig.Clusters, newCluster)
			break
		}
	}

	for _, user := range sourceConfig.Users {
		if user.Name == originalUser {
			newUser := user
			newUser.Name = newUserName
			targetConfig.Users = append(targetConfig.Users, newUser)
			break
		}
	}

	newContext := domain.ContextEntry{
		Name: contextName,
		Context: domain.Context{
			Cluster:   newClusterName,
			User:      newUserName,
			Namespace: originalContext.Context.Namespace,
		},
	}
	targetConfig.Contexts = append(targetConfig.Contexts, newContext)

	return s.repo.Save(targetPath, targetConfig)
}

func (s *Service) RemoveContext(targetPath, contextName string) error {
	if !s.repo.Exists(targetPath) {
		return domain.ErrConfigNotFound
	}

	if err := s.repo.Backup(targetPath); err != nil {
		return err
	}

	config, err := s.repo.Load(targetPath)
	if err != nil {
		return err
	}

	ctx, idx := config.FindContext(contextName)
	if idx < 0 {
		return domain.ErrContextNotFound
	}

	clusterName := ctx.Context.Cluster
	userName := ctx.Context.User

	config.RemoveContext(contextName)

	clusterInUse := false
	for _, c := range config.Contexts {
		if c.Context.Cluster == clusterName {
			clusterInUse = true
			break
		}
	}
	if !clusterInUse {
		config.RemoveCluster(clusterName)
	}

	userInUse := false
	for _, c := range config.Contexts {
		if c.Context.User == userName {
			userInUse = true
			break
		}
	}
	if !userInUse {
		config.RemoveUser(userName)
	}

	if config.CurrentContext == contextName {
		config.CurrentContext = ""
		if len(config.Contexts) > 0 {
			config.CurrentContext = config.Contexts[0].Name
		}
	}

	return s.repo.Save(targetPath, config)
}

func (s *Service) ListContexts(targetPath string) ([]ContextInfo, error) {
	if !s.repo.Exists(targetPath) {
		return nil, domain.ErrConfigNotFound
	}

	config, err := s.repo.Load(targetPath)
	if err != nil {
		return nil, err
	}

	contexts := make([]ContextInfo, len(config.Contexts))
	for i, ctx := range config.Contexts {
		var server string
		for _, cluster := range config.Clusters {
			if cluster.Name == ctx.Context.Cluster {
				server = cluster.Cluster.Server
				break
			}
		}

		contexts[i] = ContextInfo{
			Name:      ctx.Name,
			Cluster:   ctx.Context.Cluster,
			User:      ctx.Context.User,
			Namespace: ctx.Context.Namespace,
			Server:    server,
			Current:   ctx.Name == config.CurrentContext,
		}
	}

	return contexts, nil
}

func (s *Service) UseContext(targetPath, contextName, namespace string) error {
	if !s.repo.Exists(targetPath) {
		return domain.ErrConfigNotFound
	}

	config, err := s.repo.Load(targetPath)
	if err != nil {
		return err
	}

	_, idx := config.FindContext(contextName)
	if idx < 0 {
		return domain.ErrContextNotFound
	}

	config.CurrentContext = contextName

	if namespace != "" {
		config.Contexts[idx].Context.Namespace = namespace
	}

	return s.repo.Save(targetPath, config)
}

func (s *Service) GetContextNamespace(targetPath, contextName string) (string, error) {
	if !s.repo.Exists(targetPath) {
		return "", domain.ErrConfigNotFound
	}

	config, err := s.repo.Load(targetPath)
	if err != nil {
		return "", err
	}

	ctx, idx := config.FindContext(contextName)
	if idx < 0 {
		return "", domain.ErrContextNotFound
	}

	return ctx.Context.Namespace, nil
}

func (s *Service) SetNamespace(targetPath, namespace string) error {
	if !s.repo.Exists(targetPath) {
		return domain.ErrConfigNotFound
	}

	config, err := s.repo.Load(targetPath)
	if err != nil {
		return err
	}

	if config.CurrentContext == "" {
		return domain.ErrNoCurrentContext
	}

	_, idx := config.FindContext(config.CurrentContext)
	if idx < 0 {
		return domain.ErrContextNotFound
	}

	config.Contexts[idx].Context.Namespace = namespace

	return s.repo.Save(targetPath, config)
}

func (s *Service) MergeConfigs(sourcePaths []string, outputPath string) error {
	merged := domain.NewKubeConfig()

	for _, path := range sourcePaths {
		config, err := s.repo.Load(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		for _, cluster := range config.Clusters {
			if _, idx := merged.FindCluster(cluster.Name); idx < 0 {
				merged.Clusters = append(merged.Clusters, cluster)
			}
		}

		for _, user := range config.Users {
			if _, idx := merged.FindUser(user.Name); idx < 0 {
				merged.Users = append(merged.Users, user)
			}
		}

		for _, ctx := range config.Contexts {
			if _, idx := merged.FindContext(ctx.Name); idx < 0 {
				merged.Contexts = append(merged.Contexts, ctx)
			}
		}

		if merged.CurrentContext == "" && config.CurrentContext != "" {
			merged.CurrentContext = config.CurrentContext
		}
	}

	return s.repo.Save(outputPath, merged)
}

func (s *Service) ValidateConfig(path string) error {
	config, err := s.repo.Load(path)
	if err != nil {
		return err
	}

	for _, ctx := range config.Contexts {
		if _, idx := config.FindCluster(ctx.Context.Cluster); idx < 0 {
			return fmt.Errorf("context %s references unknown cluster %s", ctx.Name, ctx.Context.Cluster)
		}
		if _, idx := config.FindUser(ctx.Context.User); idx < 0 {
			return fmt.Errorf("context %s references unknown user %s", ctx.Name, ctx.Context.User)
		}
	}

	return nil
}

func (s *Service) RenameContext(targetPath, oldName, newName string) error {
	if !s.repo.Exists(targetPath) {
		return domain.ErrConfigNotFound
	}

	if err := s.repo.Backup(targetPath); err != nil {
		return err
	}

	config, err := s.repo.Load(targetPath)
	if err != nil {
		return err
	}

	ctx, idx := config.FindContext(oldName)
	if idx < 0 {
		return domain.ErrContextNotFound
	}

	if _, existIdx := config.FindContext(newName); existIdx >= 0 {
		return domain.ErrContextExists
	}

	oldClusterName := ctx.Context.Cluster
	oldUserName := ctx.Context.User
	newClusterName := newName
	newUserName := newName + "-user"

	for i, cluster := range config.Clusters {
		if cluster.Name == oldClusterName {
			config.Clusters[i].Name = newClusterName
			break
		}
	}

	for i, user := range config.Users {
		if user.Name == oldUserName {
			config.Users[i].Name = newUserName
			break
		}
	}

	config.Contexts[idx].Name = newName
	config.Contexts[idx].Context.Cluster = newClusterName
	config.Contexts[idx].Context.User = newUserName

	if config.CurrentContext == oldName {
		config.CurrentContext = newName
	}

	return s.repo.Save(targetPath, config)
}

func (s *Service) SearchContexts(targetPath, query string) ([]ContextInfo, error) {
	contexts, err := s.ListContexts(targetPath)
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var results []ContextInfo
	for _, ctx := range contexts {
		if strings.Contains(strings.ToLower(ctx.Name), query) ||
			strings.Contains(strings.ToLower(ctx.Cluster), query) ||
			strings.Contains(strings.ToLower(ctx.Server), query) ||
			strings.Contains(strings.ToLower(ctx.Namespace), query) {
			results = append(results, ctx)
		}
	}

	return results, nil
}

type ContextInfo struct {
	Name      string
	Cluster   string
	User      string
	Namespace string
	Server    string
	Current   bool
}

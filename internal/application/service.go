package application

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

type Service struct {
	repo domain.Repository
}

type MergeConflictStrategy string

const (
	MergeConflictSkip      MergeConflictStrategy = "skip"
	MergeConflictOverwrite MergeConflictStrategy = "overwrite"
	MergeConflictRename    MergeConflictStrategy = "rename"
	MergeConflictFail      MergeConflictStrategy = "fail"
)

type BackupInfo struct {
	Path      string
	Name      string
	CreatedAt time.Time
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

func (s *Service) MergeConfigs(sourcePaths []string, outputPath string, strategy string) error {
	conflictStrategy, err := parseMergeConflictStrategy(strategy)
	if err != nil {
		return err
	}

	merged := domain.NewKubeConfig()

	for _, path := range sourcePaths {
		config, err := s.repo.Load(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		currentContextName := ""

		for _, ctx := range config.Contexts {
			mergedName, imported, err := mergeContext(merged, config, ctx, conflictStrategy)
			if err != nil {
				return fmt.Errorf("failed to merge context %q from %s: %w", ctx.Name, path, err)
			}
			if imported && config.CurrentContext == ctx.Name {
				currentContextName = mergedName
			}
		}

		if merged.CurrentContext == "" && currentContextName != "" {
			merged.CurrentContext = currentContextName
		}
	}

	return s.repo.Save(outputPath, merged)
}

func (s *Service) ValidateConfig(path string) (*ValidationReport, error) {
	config, err := s.repo.Load(path)
	if err != nil {
		return nil, err
	}

	report := &ValidationReport{
		Path:         path,
		ContextCount: len(config.Contexts),
		ClusterCount: len(config.Clusters),
		UserCount:    len(config.Users),
	}

	contextNames := make(map[string]struct{}, len(config.Contexts))
	clusterNames := make(map[string]struct{}, len(config.Clusters))
	userNames := make(map[string]struct{}, len(config.Users))

	for _, cluster := range config.Clusters {
		switch {
		case cluster.Name == "":
			report.Issues = append(report.Issues, ValidationIssue{Resource: "cluster", Message: "name is required"})
		case hasDuplicate(clusterNames, cluster.Name):
			report.Issues = append(report.Issues, ValidationIssue{Resource: "cluster", Name: cluster.Name, Message: "duplicate name"})
		default:
			clusterNames[cluster.Name] = struct{}{}
		}

		if cluster.Cluster.Server == "" {
			report.Issues = append(report.Issues, ValidationIssue{Resource: "cluster", Name: cluster.Name, Message: "server is required"})
		}
	}

	for _, user := range config.Users {
		switch {
		case user.Name == "":
			report.Issues = append(report.Issues, ValidationIssue{Resource: "user", Message: "name is required"})
		case hasDuplicate(userNames, user.Name):
			report.Issues = append(report.Issues, ValidationIssue{Resource: "user", Name: user.Name, Message: "duplicate name"})
		default:
			userNames[user.Name] = struct{}{}
		}
	}

	for _, ctx := range config.Contexts {
		switch {
		case ctx.Name == "":
			report.Issues = append(report.Issues, ValidationIssue{Resource: "context", Message: "name is required"})
		case hasDuplicate(contextNames, ctx.Name):
			report.Issues = append(report.Issues, ValidationIssue{Resource: "context", Name: ctx.Name, Message: "duplicate name"})
		default:
			contextNames[ctx.Name] = struct{}{}
		}

		if ctx.Context.Cluster == "" {
			report.Issues = append(report.Issues, ValidationIssue{Resource: "context", Name: ctx.Name, Message: "cluster reference is required"})
		} else if _, idx := config.FindCluster(ctx.Context.Cluster); idx < 0 {
			report.Issues = append(report.Issues, ValidationIssue{Resource: "context", Name: ctx.Name, Message: fmt.Sprintf("references unknown cluster %q", ctx.Context.Cluster)})
		}

		if ctx.Context.User == "" {
			report.Issues = append(report.Issues, ValidationIssue{Resource: "context", Name: ctx.Name, Message: "user reference is required"})
		} else if _, idx := config.FindUser(ctx.Context.User); idx < 0 {
			report.Issues = append(report.Issues, ValidationIssue{Resource: "context", Name: ctx.Name, Message: fmt.Sprintf("references unknown user %q", ctx.Context.User)})
		}
	}

	if config.CurrentContext != "" {
		if _, idx := config.FindContext(config.CurrentContext); idx < 0 {
			report.Issues = append(report.Issues, ValidationIssue{
				Resource: "config",
				Name:     config.CurrentContext,
				Message:  "current-context does not exist",
			})
		}
	}

	return report, nil
}

func (s *Service) ListBackups(targetPath string) ([]BackupInfo, error) {
	backups, err := s.repo.ListBackups(targetPath)
	if err != nil {
		return nil, err
	}

	result := make([]BackupInfo, 0, len(backups))
	for _, backup := range backups {
		result = append(result, BackupInfo{
			Path:      backup,
			Name:      filepath.Base(backup),
			CreatedAt: parseBackupTime(backup),
		})
	}

	return result, nil
}

func (s *Service) RestoreBackup(targetPath, backupPath string) error {
	backups, err := s.repo.ListBackups(targetPath)
	if err != nil {
		return err
	}

	found := false
	for _, backup := range backups {
		if backup == backupPath {
			found = true
			break
		}
	}

	if !found {
		return domain.ErrBackupNotFound
	}

	if s.repo.Exists(targetPath) {
		if err := s.repo.Backup(targetPath); err != nil {
			return err
		}
	}

	return s.repo.RestoreBackup(targetPath, backupPath)
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

func parseMergeConflictStrategy(strategy string) (MergeConflictStrategy, error) {
	switch MergeConflictStrategy(strings.ToLower(strings.TrimSpace(strategy))) {
	case "", MergeConflictSkip:
		return MergeConflictSkip, nil
	case MergeConflictOverwrite:
		return MergeConflictOverwrite, nil
	case MergeConflictRename:
		return MergeConflictRename, nil
	case MergeConflictFail:
		return MergeConflictFail, nil
	default:
		return "", fmt.Errorf("unsupported merge conflict strategy: %s", strategy)
	}
}

func mergeContext(target, source *domain.KubeConfig, ctx domain.ContextEntry, strategy MergeConflictStrategy) (string, bool, error) {
	cluster, clusterIdx := source.FindCluster(ctx.Context.Cluster)
	if clusterIdx < 0 {
		return "", false, fmt.Errorf("missing cluster %q", ctx.Context.Cluster)
	}

	user, userIdx := source.FindUser(ctx.Context.User)
	if userIdx < 0 {
		return "", false, fmt.Errorf("missing user %q", ctx.Context.User)
	}

	clusterEntry := *cluster
	userEntry := *user
	contextEntry := ctx

	clusterName, importCluster, err := resolveClusterConflict(target, clusterEntry, strategy)
	if err != nil {
		return "", false, err
	}
	if clusterName == "" {
		return "", false, nil
	}
	clusterEntry.Name = clusterName

	userName, importUser, err := resolveUserConflict(target, userEntry, strategy)
	if err != nil {
		return "", false, err
	}
	if userName == "" {
		return "", false, nil
	}
	userEntry.Name = userName

	contextEntry.Context.Cluster = clusterName
	contextEntry.Context.User = userName

	contextName, importContext, err := resolveContextConflict(target, contextEntry, strategy)
	if err != nil {
		return "", false, err
	}
	if !importContext {
		return "", false, nil
	}
	contextEntry.Name = contextName

	if importCluster {
		upsertCluster(target, clusterEntry)
	}
	if importUser {
		upsertUser(target, userEntry)
	}
	upsertContext(target, contextEntry)

	return contextEntry.Name, true, nil
}

func resolveClusterConflict(target *domain.KubeConfig, entry domain.ClusterEntry, strategy MergeConflictStrategy) (string, bool, error) {
	existing, idx := target.FindCluster(entry.Name)
	if idx < 0 {
		return entry.Name, true, nil
	}

	if reflect.DeepEqual(existing.Cluster, entry.Cluster) {
		return entry.Name, false, nil
	}

	switch strategy {
	case MergeConflictSkip:
		return "", false, nil
	case MergeConflictOverwrite:
		return entry.Name, true, nil
	case MergeConflictRename:
		return nextAvailableName(entry.Name, func(name string) bool {
			_, idx := target.FindCluster(name)
			return idx >= 0
		}), true, nil
	case MergeConflictFail:
		return "", false, fmt.Errorf("cluster %q already exists", entry.Name)
	default:
		return "", false, fmt.Errorf("unsupported merge conflict strategy: %s", strategy)
	}
}

func resolveUserConflict(target *domain.KubeConfig, entry domain.UserEntry, strategy MergeConflictStrategy) (string, bool, error) {
	existing, idx := target.FindUser(entry.Name)
	if idx < 0 {
		return entry.Name, true, nil
	}

	if reflect.DeepEqual(existing.User, entry.User) {
		return entry.Name, false, nil
	}

	switch strategy {
	case MergeConflictSkip:
		return "", false, nil
	case MergeConflictOverwrite:
		return entry.Name, true, nil
	case MergeConflictRename:
		return nextAvailableName(entry.Name, func(name string) bool {
			_, idx := target.FindUser(name)
			return idx >= 0
		}), true, nil
	case MergeConflictFail:
		return "", false, fmt.Errorf("user %q already exists", entry.Name)
	default:
		return "", false, fmt.Errorf("unsupported merge conflict strategy: %s", strategy)
	}
}

func resolveContextConflict(target *domain.KubeConfig, entry domain.ContextEntry, strategy MergeConflictStrategy) (string, bool, error) {
	existing, idx := target.FindContext(entry.Name)
	if idx < 0 {
		return entry.Name, true, nil
	}

	if reflect.DeepEqual(existing.Context, entry.Context) {
		return entry.Name, false, nil
	}

	switch strategy {
	case MergeConflictSkip:
		return "", false, nil
	case MergeConflictOverwrite:
		return entry.Name, true, nil
	case MergeConflictRename:
		return nextAvailableName(entry.Name, func(name string) bool {
			_, idx := target.FindContext(name)
			return idx >= 0
		}), true, nil
	case MergeConflictFail:
		return "", false, fmt.Errorf("context %q already exists", entry.Name)
	default:
		return "", false, fmt.Errorf("unsupported merge conflict strategy: %s", strategy)
	}
}

func upsertCluster(config *domain.KubeConfig, entry domain.ClusterEntry) {
	if _, idx := config.FindCluster(entry.Name); idx >= 0 {
		config.Clusters[idx] = entry
		return
	}
	config.Clusters = append(config.Clusters, entry)
}

func upsertUser(config *domain.KubeConfig, entry domain.UserEntry) {
	if _, idx := config.FindUser(entry.Name); idx >= 0 {
		config.Users[idx] = entry
		return
	}
	config.Users = append(config.Users, entry)
}

func upsertContext(config *domain.KubeConfig, entry domain.ContextEntry) {
	if _, idx := config.FindContext(entry.Name); idx >= 0 {
		config.Contexts[idx] = entry
		return
	}
	config.Contexts = append(config.Contexts, entry)
}

func nextAvailableName(base string, exists func(string) bool) string {
	if !exists(base) {
		return base
	}

	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !exists(candidate) {
			return candidate
		}
	}
}

func parseBackupTime(path string) time.Time {
	base := filepath.Base(path)
	idx := strings.LastIndex(base, ".backup.")
	if idx < 0 {
		return time.Time{}
	}

	timestamp := base[idx+len(".backup."):]
	parsed, err := time.Parse("20060102-150405", timestamp)
	if err != nil {
		return time.Time{}
	}

	return parsed
}

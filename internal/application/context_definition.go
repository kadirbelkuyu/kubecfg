package application

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kadirbelkuyu/kubecfg/internal/domain"
)

const editInstructions = `# Edit the definition of context %q and save to apply the changes.
# The names of the context, cluster, and user must stay unchanged;
# rename a context with 'kubecfg rename'.
# Saving the file without changes leaves the context untouched;
# an empty document cancels the edit.
# This file holds cluster credentials and is removed once the editor closes.
`

const (
	editErrorHeader  = "# The previous attempt was rejected:\n"
	editErrorTrailer = "# Fix the definition below and save again.\n"
)

// maxEditAttempts caps how often a rejected definition is handed back to the
// editor before the edit gives up.
const maxEditAttempts = 10

// ContextDefinitionEditor hands a context definition to an interactive editor
// and returns the document the editor saved.
type ContextDefinitionEditor interface {
	Edit(contextName string, document []byte) ([]byte, error)
}

// ContextUpdate reports which parts of a context definition were replaced.
type ContextUpdate struct {
	ContextName string
	Changed     []string
}

// IsEmpty reports whether the definition was left untouched.
func (u *ContextUpdate) IsEmpty() bool {
	return u == nil || len(u.Changed) == 0
}

// GetContextDefinition returns the definition of a context as a standalone
// single-context kubeconfig document. An empty context name selects the
// current context.
func (s *Service) GetContextDefinition(targetPath, contextName string) (string, error) {
	_, definition, err := s.loadContextDefinition(targetPath, contextName)
	if err != nil {
		return "", err
	}

	document, err := marshalContextDefinition(definition)
	if err != nil {
		return "", err
	}

	return string(document), nil
}

// EditContextDefinition opens the definition of a context in the editor and
// writes the edited document back to the kubeconfig. The editor is reopened
// with the reported problem while the saved document keeps changing.
func (s *Service) EditContextDefinition(targetPath, contextName string, editor ContextDefinitionEditor) (*ContextUpdate, error) {
	config, current, err := s.loadContextDefinition(targetPath, contextName)
	if err != nil {
		return nil, err
	}

	document, err := marshalContextDefinition(current)
	if err != nil {
		return nil, err
	}
	document = append([]byte(fmt.Sprintf(editInstructions, current.Context.Name)), document...)

	var lastErr error
	for attempt := 0; attempt < maxEditAttempts; attempt++ {
		edited, err := editor.Edit(current.Context.Name, document)
		if err != nil {
			return nil, err
		}

		next, definitionErr := parseContextDefinition(edited)
		if definitionErr == nil {
			definitionErr = validateContextDefinition(current, next)
		}
		if definitionErr == nil {
			return s.saveContextDefinition(targetPath, config, current, next)
		}
		lastErr = definitionErr

		if bytes.Equal(bytes.TrimSpace(edited), bytes.TrimSpace(document)) {
			break
		}
		document = annotateContextDefinition(edited, definitionErr)
	}

	return nil, lastErr
}

// UpdateContextFromFile replaces the cluster details and credentials of a
// context with the ones found in a kubeconfig file, keeping the local context,
// cluster, and user names.
func (s *Service) UpdateContextFromFile(targetPath, contextName, sourcePath string) (*ContextUpdate, error) {
	config, current, err := s.loadContextDefinition(targetPath, contextName)
	if err != nil {
		return nil, err
	}

	source, err := s.repo.Load(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load source config: %w", err)
	}

	incoming, err := source.SoleContextDefinition()
	if err != nil {
		return nil, err
	}

	next := rebindContextDefinition(current, incoming)
	if err := validateContextDefinition(current, next); err != nil {
		return nil, err
	}

	return s.saveContextDefinition(targetPath, config, current, next)
}

func (s *Service) loadContextDefinition(targetPath, contextName string) (*domain.KubeConfig, *domain.ContextDefinition, error) {
	if !s.repo.Exists(targetPath) {
		return nil, nil, domain.ErrConfigNotFound
	}

	config, err := s.repo.Load(targetPath)
	if err != nil {
		return nil, nil, err
	}

	if contextName == "" {
		if config.CurrentContext == "" {
			return nil, nil, domain.ErrNoCurrentContext
		}
		contextName = config.CurrentContext
	}

	definition, err := config.ExtractContextDefinition(contextName)
	if err != nil {
		return nil, nil, err
	}

	return config, definition, nil
}

func (s *Service) saveContextDefinition(targetPath string, config *domain.KubeConfig, current, next *domain.ContextDefinition) (*ContextUpdate, error) {
	update := &ContextUpdate{
		ContextName: current.Context.Name,
		Changed:     changedContextDefinitionParts(current, next),
	}
	if update.IsEmpty() {
		return update, nil
	}

	if err := s.repo.Backup(targetPath); err != nil {
		return nil, err
	}

	upsertContext(config, next.Context)
	upsertCluster(config, next.Cluster)
	upsertUser(config, next.User)

	if err := s.repo.Save(targetPath, config); err != nil {
		return nil, err
	}

	return update, nil
}

// rebindContextDefinition copies the cluster details and credentials of an
// imported definition onto the identity of an existing context.
func rebindContextDefinition(current, incoming *domain.ContextDefinition) *domain.ContextDefinition {
	next := *current
	next.Cluster.Cluster = incoming.Cluster.Cluster
	next.User.User = incoming.User.User

	if namespace := strings.TrimSpace(incoming.Context.Context.Namespace); namespace != "" {
		next.Context.Context.Namespace = namespace
	}

	return &next
}

func validateContextDefinition(current, next *domain.ContextDefinition) error {
	if next.Context.Name != current.Context.Name {
		return fmt.Errorf("context name cannot be changed from %q to %q, use 'kubecfg rename'", current.Context.Name, next.Context.Name)
	}

	if next.Cluster.Name != current.Cluster.Name {
		return fmt.Errorf("cluster name cannot be changed from %q to %q", current.Cluster.Name, next.Cluster.Name)
	}

	if next.User.Name != current.User.Name {
		return fmt.Errorf("user name cannot be changed from %q to %q", current.User.Name, next.User.Name)
	}

	if next.Context.Context.Cluster != next.Cluster.Name {
		return fmt.Errorf("context %q references unknown cluster %q", next.Context.Name, next.Context.Context.Cluster)
	}

	if next.Context.Context.User != next.User.Name {
		return fmt.Errorf("context %q references unknown user %q", next.Context.Name, next.Context.Context.User)
	}

	if strings.TrimSpace(next.Cluster.Cluster.Server) == "" {
		return fmt.Errorf("cluster %q requires a server URL", next.Cluster.Name)
	}

	return nil
}

func changedContextDefinitionParts(current, next *domain.ContextDefinition) []string {
	changed := make([]string, 0, 3)

	if !reflect.DeepEqual(current.Context, next.Context) {
		changed = append(changed, "context")
	}
	if !reflect.DeepEqual(current.Cluster, next.Cluster) {
		changed = append(changed, "cluster")
	}
	if !reflect.DeepEqual(current.User, next.User) {
		changed = append(changed, "user")
	}

	return changed
}

func marshalContextDefinition(definition *domain.ContextDefinition) ([]byte, error) {
	document, err := yaml.Marshal(definition.ToKubeConfig())
	if err != nil {
		return nil, fmt.Errorf("encode context definition: %w", err)
	}

	return document, nil
}

func parseContextDefinition(document []byte) (*domain.ContextDefinition, error) {
	var config domain.KubeConfig
	if err := yaml.Unmarshal(document, &config); err != nil {
		return nil, fmt.Errorf("parse context definition: %w", err)
	}

	if len(config.Contexts) != 1 {
		return nil, fmt.Errorf("the definition must hold exactly one context, found %d", len(config.Contexts))
	}

	name := config.Contexts[0].Name
	definition, err := config.ExtractContextDefinition(name)
	if err != nil {
		return nil, fmt.Errorf("context %q: %w", name, err)
	}

	return definition, nil
}

func annotateContextDefinition(document []byte, reason error) []byte {
	var annotated bytes.Buffer

	annotated.WriteString(editErrorHeader)
	for _, line := range strings.Split(reason.Error(), "\n") {
		annotated.WriteString("#   " + line + "\n")
	}
	annotated.WriteString(editErrorTrailer)
	annotated.Write(stripEditErrorBlock(document))

	return annotated.Bytes()
}

func stripEditErrorBlock(document []byte) []byte {
	if !bytes.HasPrefix(document, []byte(editErrorHeader)) {
		return document
	}

	trailer := []byte(editErrorTrailer)
	idx := bytes.Index(document, trailer)
	if idx < 0 {
		return document
	}

	return document[idx+len(trailer):]
}

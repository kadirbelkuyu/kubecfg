package application

import "fmt"

type ValidationIssue struct {
	Resource string
	Name     string
	Message  string
}

type ValidationReport struct {
	Path         string
	ContextCount int
	ClusterCount int
	UserCount    int
	Issues       []ValidationIssue
}

func (r *ValidationReport) IsValid() bool {
	return r != nil && len(r.Issues) == 0
}

func hasDuplicate(items map[string]struct{}, name string) bool {
	_, ok := items[name]
	return ok
}

func (i ValidationIssue) String() string {
	if i.Name == "" {
		return fmt.Sprintf("%s: %s", i.Resource, i.Message)
	}
	return fmt.Sprintf("%s %q: %s", i.Resource, i.Name, i.Message)
}

package health

import "context"

type Checker interface {
	Check(ctx context.Context, contextName string) Result
	CheckAll(ctx context.Context, contextNames []string) []Result
}

type Cache interface {
	Get(contextName string) (Result, bool)
	Set(result Result)
	Invalidate(contextName string)
	InvalidateAll()
}

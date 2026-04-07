package domain

type GuardMode string

const (
	GuardModeReadonly GuardMode = "readonly"
)

type GuardRequestPolicy interface {
	Validate(method, path string) error
}

type GuardRuntime interface {
	NextListenAddress() (string, error)
	Start(session *Session) error
	Stop(session *Session) error
	IsRunning(session *Session) bool
}

type GuardedKubeconfigWriter interface {
	Write(sourcePath, contextName, proxyAddress, outputPath string) (*GuardedKubeconfigResult, error)
	Cleanup(outputPath string) error
}

type GuardedKubeconfigResult struct {
	Path      string
	Context   string
	Namespace string
}

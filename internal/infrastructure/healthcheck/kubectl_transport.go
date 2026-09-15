package healthcheck

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure/kubecommand"
)

type kubectlTransport struct {
	kubeconfigPath string
	contextName    string
	timeout        time.Duration
}

func (t kubectlTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	output, err := kubecommand.Run(req.Context(), "get", "--raw", req.URL.RequestURI(),
		"--kubeconfig", t.kubeconfigPath, "--context", t.contextName, "--request-timeout", t.timeout.String())
	statusCode := http.StatusOK
	if err != nil {
		var httpErr *kubecommand.HTTPError
		if !errors.As(err, &httpErr) {
			return nil, err
		}
		statusCode = httpErr.StatusCode
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(output)),
		Request:    req,
	}, nil
}

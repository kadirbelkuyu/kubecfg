package fzf

import (
	"os"
	"os/exec"
	"strings"
)

var lookPath = exec.LookPath

func Available() bool {
	if strings.TrimSpace(os.Getenv("KUBECFG_IGNORE_FZF")) != "" {
		return false
	}

	_, err := lookPath("fzf")
	return err == nil
}

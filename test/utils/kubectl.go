package utils_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type KubectlVersion struct {
	ClientVersion struct {
		Minor string `json:"minor"`
	} `json:"clientVersion"`
}

type Kubectl struct {
	Receiver io.Writer
}

func NewKubectl(receiver io.Writer) *Kubectl {
	if receiver == nil {
		panic("receiver must not be nil")
	}
	return &Kubectl{
		Receiver: receiver,
	}
}

func (k *Kubectl) Execute(ctx context.Context, stdin *bytes.Buffer, args ...string) (string, string, error) {
	cmd := kubectl(ctx, args...)
	if stdin != nil {
		cmd.Stdin = stdin
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(&stdout, k.Receiver)
	cmd.Stderr = io.MultiWriter(&stderr, k.Receiver)
	_, _ = fmt.Fprintf(k.Receiver, "Executing: %s \n", strings.Join(cmd.Args, " "))

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

// ExecuteOn is a wrapper around Execute that allows specifying a kube context
func (k *Kubectl) ExecuteOn(ctx context.Context, kubeContext string, stdin *bytes.Buffer, args ...string) (
	string,
	string,
	error,
) {
	args = append([]string{"--context", kubeContext}, args...)
	return k.Execute(ctx, stdin, args...)
}

func kubectl(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.Env = os.Environ()
	// disable DEBUG=1 from getting through to kube
	for i, pair := range cmd.Env {
		if strings.HasPrefix(pair, "DEBUG") {
			cmd.Env = append(cmd.Env[:i], cmd.Env[i+1:]...)
			break
		}
	}
	return cmd
}

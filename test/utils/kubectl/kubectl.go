package kubectl

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rotisserie/eris"

	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const AllSelector = "*"

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
	fmt.Fprintf(k.Receiver, "Executing: %s \n", strings.Join(cmd.Args, " "))

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

// Wrapper around Execute that allows specifying a kube context
func (k *Kubectl) ExecuteOn(ctx context.Context, kubeContext string, stdin *bytes.Buffer, args ...string) (
	string,
	string,
	error,
) {
	args = append([]string{"--context", kubeContext}, args...)
	return k.Execute(ctx, stdin, args...)
}

func (k *Kubectl) GetVersion(ctx context.Context) (*KubectlVersion, error) {
	args := []string{"version", "--client=true", "-o", "json"}
	stdout, stderr, err := k.Execute(ctx, nil, args...)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to get kubectl version: %s", stderr)
	}
	var version KubectlVersion
	err = json.Unmarshal([]byte(stdout), &version)
	if err != nil {
		return nil, eris.Wrapf(err, "failed to unmarshal kubectl version: %s", stdout)
	}
	return &version, nil
}

func (k *Kubectl) DeleteObjects(ctx context.Context, kubeContext, objectKind, ns string, addedArgs ...string) error {
	args := []string{"delete", objectKind}

	// Accept star to describe objects across all namespaces
	if ns == AllSelector {
		args = append(args, "-A")
	} else {
		args = append(args, "--namespace", ns)
	}
	// Add extra args
	args = append(args, addedArgs...)

	_, stderr, err := k.ExecuteOn(ctx, kubeContext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to delete objects of kind %q in namespace %q: %s", objectKind, ns, stderr)
	}
	return nil
}

func (k *Kubectl) DeleteObject(
	ctx context.Context,
	kubeContext, objectKind, name, ns string,
	addedArgs ...string,
) error {
	args := []string{"delete", objectKind, name, "--namespace", ns}
	// Add extra args
	args = append(args, addedArgs...)

	_, stderr, err := k.ExecuteOn(ctx, kubeContext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to delete object of kind %q in namespace %q: %s", objectKind, ns, stderr)
	}
	return nil
}

func (k *Kubectl) CreateNamespace(ctx context.Context, kubeContext, ns string, addedArgs ...string) error {
	args := []string{"create", "namespace", ns}

	// Add extra args
	args = append(args, addedArgs...)

	stdout, stderr, err := k.ExecuteOn(ctx, kubeContext, nil, args...)

	if err != nil {
		// If the namespace already exists, ignore the error
		if strings.Contains(stderr, fmt.Sprintf("Error from server (AlreadyExists): namespaces %q already exists\n", ns)) {
			return nil
		}
		return eris.Wrapf(err, "failed to create namespace %s: %s", ns, stderr)
	}
	if !strings.Contains(stdout, fmt.Sprintf("namespace/%s created\n", ns)) {
		return eris.Errorf("failed to create namespace %s", ns)
	}
	return nil
}

func (k *Kubectl) DeleteNamespace(ctx context.Context, kubeContext, ns string, addedArgs ...string) error {
	args := []string{"delete", "namespace", ns}

	// Add extra args
	args = append(args, addedArgs...)

	stdout, stderr, err := k.ExecuteOn(ctx, kubeContext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to delete namespace %q: %s", ns, stderr)
	}
	if !strings.Contains(stdout, fmt.Sprintf("namespace %q deleted\n", ns)) {
		return eris.Errorf("failed to delete namespace %q", ns)
	}
	return nil
}

func (k *Kubectl) Apply(ctx context.Context, kubeContext, ns string, content []byte, addedArgs ...string) error {
	args := []string{"apply", "-n", ns, "-f", "-"}

	// Add extra args
	args = append(args, addedArgs...)

	buf := bytes.NewBuffer(content)

	_, stderr, err := k.ExecuteOn(ctx, kubeContext, buf, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to apply resource %s: %s", ns, stderr)
	}
	if stderr != "" {
		return eris.Errorf("failed to apply resource to namespace %q: %s", ns, stderr)
	}
	return nil
}

func (k *Kubectl) Delete(ctx context.Context, kubeContext, ns string, content []byte, addedArgs ...string) error {
	args := []string{"delete", "-n", ns, "-f", "-"}

	// Add extra args
	args = append(args, addedArgs...)

	buf := bytes.NewBuffer(content)

	_, stderr, err := k.ExecuteOn(ctx, kubeContext, buf, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to apply resource %s: %s", ns, stderr)
	}
	if stderr != "" {
		return eris.Errorf("failed to delete resource from namespace %q: %s", ns, stderr)
	}
	return nil
}

func (k *Kubectl) WaitFor(
	ctx context.Context,
	kubecontext, objectKind, ns, condition, jsonPath string,
	addedArgs ...string,
) error {
	args := []string{"wait"}
	if condition != "" {
		args = append(args, fmt.Sprintf("--for=condition=%s", condition))
	}

	if jsonPath != "" {
		args = append(args, fmt.Sprintf("--for=jsonpath=%s", jsonPath))
	}

	args = append(args, objectKind)

	// Accept star to describe objects across all namespaces
	if ns == AllSelector {
		args = append(args, "-A")
	} else {
		args = append(args, "--namespace", ns)
	}
	// Add extra args
	args = append(args, addedArgs...)

	_, stderr, err := k.ExecuteOn(ctx, kubecontext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to wait for condition %s", stderr)
	}
	if stderr != "" {
		return eris.Errorf("failed to wait for condition %s", stderr)
	}
	return nil
}

func (k *Kubectl) WaitForDelete(
	ctx context.Context,
	kubecontext, namespace, objectKind, objectName, timeout string,
) error {
	args := []string{
		"wait",
		"-n",
		namespace,
		"--for=delete",
		fmt.Sprintf("%s/%s", objectKind, objectName),
		"--timeout",
		timeout,
	}

	_, stderr, err := k.ExecuteOn(ctx, kubecontext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to wait for condition %s", stderr)
	}
	if stderr != "" {
		return eris.Errorf("failed to wait for condition %s", stderr)
	}
	return nil
}

func (k *Kubectl) Exists(ctx context.Context, kubecontext, ns, objectKind, name string) (bool, error) {
	args := []string{"get", "--namespace", ns, objectKind, "-o", "name"}

	// By default, expect exact match
	expectedName := fmt.Sprintf("%s/%s", objectKind, name)
	nameMatches := func(line string) bool {
		return line == expectedName
	}

	// But if name is wildcard, use prefix match
	if name == "*" {
		expectedPrefix := fmt.Sprintf("%s/", objectKind)
		nameMatches = func(line string) bool {
			return strings.HasPrefix(line, expectedPrefix)
		}
	}

	stdout, stderr, err := k.ExecuteOn(ctx, kubecontext, nil, args...)
	if err != nil {
		return false, eris.Wrapf(err, "failed to get objects of kind %s from namspace %s: %s", objectKind, ns, stderr)
	}

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		if nameMatches(scanner.Text()) {
			return true, nil
		}
	}

	return false, nil
}

func (k *Kubectl) RolloutStatus(ctx context.Context, kubecontext, ns, deployment string, args ...string) error {
	args = append([]string{"-n", ns, "rollout", "status", "deployment", deployment}, args...)
	stdout, stderr, err := k.ExecuteOn(ctx, kubecontext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to get rollout status for deployment %s: %s", deployment, stderr)
	}
	if !strings.Contains(stdout, fmt.Sprintf("deployment %q successfully rolled out\n", deployment)) {
		return eris.Errorf("deployment %s not rolled out", deployment)
	}
	return nil
}

func (k *Kubectl) RolloutRestart(ctx context.Context, kubecontext, ns, deployment string) error {
	args := []string{"-n", ns, "rollout", "restart", "deployment", deployment}
	stdout, stderr, err := k.ExecuteOn(ctx, kubecontext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to restart deployment %s: %s", deployment, stderr)
	}
	if !strings.Contains(stdout, fmt.Sprintf("deployment.apps/%s restarted\n", deployment)) {
		return eris.Errorf("deployment %s failed to restart", deployment)
	}
	return nil
}

func (k *Kubectl) RolloutDaemonsetStatus(ctx context.Context, kubecontext, ns, daemonset string) error {
	args := []string{"-n", ns, "rollout", "status", "daemonset", daemonset}
	stdout, stderr, err := k.ExecuteOn(ctx, kubecontext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to get rollout status for daemon set %s: %s", daemonset, stderr)
	}
	if !strings.Contains(stdout, fmt.Sprintf("daemon set %q successfully rolled out\n", daemonset)) {
		return eris.Errorf("daemon set %s not rolled out", daemonset)
	}
	return nil
}

func (k *Kubectl) RolloutDaemonsetRestart(ctx context.Context, kubecontext, ns, daemonset string) error {
	args := []string{"-n", ns, "rollout", "restart", "daemonset", daemonset}
	stdout, stderr, err := k.ExecuteOn(ctx, kubecontext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to restart daemonset %s: %s", daemonset, stderr)
	}
	if !strings.Contains(stdout, fmt.Sprintf("daemonset.apps/%s restarted\n", daemonset)) {
		return eris.Errorf("failed to restart daemonset %s", daemonset)
	}
	return nil
}

func (k *Kubectl) ScaleDeployment(ctx context.Context, kubecontext, ns, deployment string, replicas int) error {
	args := []string{"-n", ns, "--replicas", fmt.Sprintf("%d", replicas), "scale", "deployment", deployment}
	stdout, stderr, err := k.ExecuteOn(ctx, kubecontext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to scale deployment %s: %s", deployment, stderr)
	}
	if !strings.Contains(stdout, fmt.Sprintf("deployment.apps/%s scaled\n", deployment)) {
		return eris.Errorf("failed to scale deployment %s: %s", deployment, stderr)
	}
	return nil
}

func (k *Kubectl) SetEnv(
	ctx context.Context,
	kubecontext,
	ns,
	deploymentName,
	envName,
	envValue string,
	args ...string,
) error {
	return k.SetDeploymentEnvVars(
		ctx,
		kubecontext,
		ns,
		deploymentName,
		"",
		map[string]string{envName: envValue},
	)
}

func (k *Kubectl) DeleteEnv(
	ctx context.Context,
	kubecontext,
	ns,
	deploymentName,
	envName string,
	args ...string,
) error {
	args = append([]string{
		"set", "env", fmt.Sprintf("deployment/%s", deploymentName),
		"-n", ns,
		fmt.Sprintf(
			"%s-", envName,
		),
	}, args...)
	stdout, stderr, err := k.ExecuteOn(ctx, kubecontext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to unset env %q for deployment %s: %s", envName, deploymentName, stderr)
	}
	if !strings.Contains(stdout, fmt.Sprintf("deployment.apps/%s env updated\n", deploymentName)) {
		return eris.Errorf("failed to unset env %q for deployment %s: %s", envName, deploymentName, stderr)
	}
	return nil
}

func (k *Kubectl) Cat(ctx context.Context, object k8sClient.Object, kubecontext, file string) (string, error) {
	args := []string{
		"exec", "-n", object.GetNamespace(), fmt.Sprintf("%s/%s", object.GetObjectKind().GroupVersionKind().Kind, object.GetName()), "--", "cat", file,
	}
	stdout, stderr, err := k.ExecuteOn(ctx, kubecontext, nil, args...)
	if err != nil {
		return "", eris.Wrapf(err, "failed to cat objects %s: %s", file, stderr)
	}
	return stdout, nil
}

func (k *Kubectl) Netcat(
	ctx context.Context,
	kubecontext, ns, fromDeployment, fromContainer, host, port, stdinStr string,
) (string, error) {
	cmd := kubectl(ctx, []string{
		"--context", kubecontext,
		"-n", ns,
		"exec", fmt.Sprintf("deployment/%s", fromDeployment),
		"-i",
		"-c", fromContainer,
		"--", "nc", host, port,
	}...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", eris.Wrapf(err, "failed to get stdin pipe")
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(&stdout, k.Receiver)
	cmd.Stderr = io.MultiWriter(&stderr, k.Receiver)

	err = cmd.Start()
	if err != nil {
		return "", eris.Wrapf(err, "failed to start command")
	}

	_, err = io.WriteString(stdin, stdinStr)
	if err != nil {
		return "", eris.Wrapf(err, "failed to write to stdin")
	}

	// TODO (nikolasmatt): Figure out a better way to handle terminating netcat after getting a response.
	time.Sleep(250 * time.Millisecond)
	if err := cmd.Process.Kill(); err != nil {
		return "", eris.Wrapf(err, "failed to wait for command to finish")
	}

	if stderr.String() != "" {
		return "", eris.Errorf("failed to netcat %s/%s: %s", ns, fromDeployment, stderr.String())
	}

	return stdout.String(), nil
}

func (k *Kubectl) DeleteCRDsWithLabel(
	ctx context.Context,
	kubeContext string,
	labelKey string,
	labelValue string,
) error {
	args := []string{"delete", "crd", "-l", fmt.Sprintf("%s=%s", labelKey, labelValue)}
	_, stderr, err := k.ExecuteOn(ctx, kubeContext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to delete crds with label \"%s=%s\": %s", labelKey, labelValue, stderr)
	}
	return nil
}

func (k *Kubectl) SetDeploymentEnvVars(
	ctx context.Context,
	kubeContext string,
	ns string,
	deploymentName string,
	containerName string,
	envVars map[string]string,
	extraArgs ...string,
) error {
	var envVarStrings []string
	for name, value := range envVars {
		envVarStrings = append(envVarStrings, fmt.Sprintf("%s=%s", name, value))
	}
	args := []string{"set", "env", "-n", ns, fmt.Sprintf("deployment/%s", deploymentName)}

	if containerName != "" {
		args = append(args, "-c", containerName)
	}

	args = append(args, envVarStrings...)
	args = append(args, extraArgs...)

	stdout, stderr, err := k.ExecuteOn(ctx, kubeContext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to set env vars for deployment %s: %s", deploymentName, stderr)
	}
	if !strings.Contains(stdout, fmt.Sprintf("deployment.apps/%s env updated\n", deploymentName)) {
		return eris.Errorf("failed to set env for deployment %s:\n env vars: %v\n Error: %s\n", deploymentName, envVars, stderr)
	}
	return nil
}

func (k *Kubectl) DisableContainer(
	ctx context.Context,
	kubeContext string,
	ns string,
	deploymentName string,
	containerName string,
) error {
	args := []string{
		"-n", ns,
		"patch", "deployment", deploymentName,
		"--patch",
		fmt.Sprintf(
			"{\"spec\":{\"template\":{\"spec\":{\"containers\":[{\"name\":\"%s\",\"command\":[\"sleep\",\"20h\"],\"env\":[{\"name\":\"DISABLE_CONTAINER\",\"value\":\"True\"}]}]}}}}",
			containerName,
		),
	}
	_, stderr, err := k.ExecuteOn(ctx, kubeContext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to set disable container %s/%s: %s", deploymentName, containerName, stderr)
	}
	return nil
}

func (k *Kubectl) EnableContainer(
	ctx context.Context,
	kubeContext string,
	ns string,
	deploymentName string,
	containerName string,
) error {
	args := []string{
		"-n", ns,
		"patch", "deployment", deploymentName,
		"--type", "json",
		"-p", "[{\"op\": \"remove\", \"path\": \"/spec/template/spec/containers/0/command\"}]",
	}
	_, stderr, err := k.ExecuteOn(ctx, kubeContext, nil, args...)
	if err != nil {
		return eris.Wrapf(err, "failed to set enable container %s/%s: %s", deploymentName, containerName, stderr)
	}
	return nil
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

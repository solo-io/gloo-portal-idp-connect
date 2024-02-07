package test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/solo-io/gloo-portal-idp-connect/test/utils/kubectl"
)

var scheme = runtime.NewScheme()

func init() {
	runtimeutil.Must(clientgoscheme.AddToScheme(scheme))
}

type KubeContext struct {
	client     client.Client
	kubeClient *kubernetes.Clientset
	kubectl    *kubectl.Kubectl

	Context string
}

func NewKubeContext(kubeCtx string) (*KubeContext, error) {
	client, kubeClient, err := getClient(kubeCtx)
	if err != nil {
		return nil, err
	}

	return &KubeContext{
		client:     client,
		kubeClient: kubeClient,
		kubectl:    kubectl.NewKubectl(GinkgoWriter),
		Context:    kubeCtx,
	}, nil
}

func (k *KubeContext) GetPod(ctx context.Context, ns string, labelSelector map[string]string) (*Pod, error) {
	pods, err := k.GetPods(ctx, ns, labelSelector)
	if err != nil {
		return nil, err
	}
	if len(pods) == 0 {
		return nil, fmt.Errorf("no pods found for label selector %v", labelSelector)
	}
	return pods[0], nil
}

func (k *KubeContext) GetPods(ctx context.Context, ns string, labelSelector map[string]string) ([]*Pod, error) {
	buildSelector := []string{}
	for lKey, lValue := range labelSelector {
		buildSelector = append(buildSelector, lKey+"="+lValue)
	}
	pl, err := k.kubeClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: strings.Join(buildSelector, ","), FieldSelector: "status.phase==Running"})
	if err != nil {
		return nil, err
	}

	var runningPods []*Pod
	for _, pod := range pl.Items {
		// We should only be returning pods that have not been marked for deletion
		if pod.DeletionTimestamp == nil {
			runningPods = append(runningPods, &Pod{
				Pod:     pod,
				Cluster: k,
			})
		}
	}

	return runningPods, nil
}

type Pod struct {
	corev1.Pod
	Cluster *KubeContext
}

// Curl requires a container named 'curl' to exist on the deployment
func (p *Pod) Curl(ctx context.Context, container string, args ...string) (string, error) {
	args = append(
		[]string{
			"-n", p.Namespace,
			"exec", p.Name,
			"-c", container,
			"--", "curl",
		}, args...,
	)

	// TODO (nikolasmatt): stdout and stderr are currently expected to be merged in tests. We need to habdle error in the future.
	stdout, stderr, _ := p.Cluster.kubectl.ExecuteOn(ctx, p.Cluster.Context, nil, args...)

	// HTTP response codes are printed in stderr. It is currently expected in the ouput by many tests.
	return stdout + stderr, nil
}

type CurlFromPod struct {
	Url            string
	Cluster        *KubeContext
	Method         string
	Data           string
	App            string
	Namespace      string
	Headers        []string
	TimeoutSeconds float32
	// Container from which curl is executed, defaults to "curl"
	Container string
}

func (c *CurlFromPod) Execute() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute/2)
	defer cancel()

	args := []string{
		c.Url,
		"-s",
		"--connect-timeout", "5",
	}

	method := "GET"
	if c.Method != "" {
		method = c.Method
	}

	args = append(args, "-X", method)

	if c.Data != "" {
		args = append(args, "-d", c.Data)
	}

	for _, header := range c.Headers {
		args = append(args, "-H", header)
	}

	if c.TimeoutSeconds > 0 {
		args = append(args, "--max-time", fmt.Sprintf("%v", c.TimeoutSeconds))
	}

	pod, err := c.Cluster.GetPod(ctx, c.Namespace, map[string]string{"app": c.App})
	if err != nil {
		return "", err
	}

	container := "curl"
	if c.Container != "" {
		container = c.Container
	}

	return pod.Curl(ctx, container, args...)
}

func getClientConfig(kubeCtx string) (*rest.Config, error) {
	// Let's avoid defaulting, require clients to be explicit.
	if kubeCtx == "" {
		return nil, errors.New("missing cluster name")
	}

	cfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, err
	}

	config := clientcmd.NewNonInteractiveClientConfig(*cfg, kubeCtx, &clientcmd.ConfigOverrides{}, nil)
	restCfg, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}

	// Let's speed up our client when running tests
	restCfg.QPS = 50
	if v := os.Getenv("K8S_CLIENT_QPS"); v != "" {
		qps, err := strconv.ParseFloat(v, 32)
		if err != nil {
			return nil, err
		}
		restCfg.QPS = float32(qps)
	}

	restCfg.Burst = 100
	if v := os.Getenv("K8S_CLIENT_BURST"); v != "" {
		burst, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		restCfg.Burst = burst
	}

	return restCfg, nil
}

func getClient(kubeCtx string) (client.Client, *kubernetes.Clientset, error) {
	restCfg, err := getClientConfig(kubeCtx)
	if err != nil {
		return client.Client(nil), nil, err
	}

	kubeClients, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return client.Client(nil), nil, err
	}

	cluster, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		return client.Client(nil), nil, err
	}

	return cluster, kubeClients, nil
}

func (k *KubeContext) CheckPodsInCluster(ctx context.Context) {
	podsReady := func(g Gomega) bool {
		pl := &corev1.PodList{}

		// get new client every time - avoid caching and rate-limiting issues
		k8Client := k.client

		err := k8Client.List(ctx, pl)
		g.Expect(err).NotTo(HaveOccurred())

		for i := range pl.Items {
			pod := &pl.Items[i]

			// Could capture pods that are pending or in the process of termination
			if !IsPodConditionReady(pod, corev1.PodReady) {
				return false
			}

			if !IsPodConditionReady(pod, corev1.ContainersReady) {
				return false
			}
		}

		return true
	}

	Eventually(func(g Gomega) {
		g.Expect(podsReady(g)).To(BeTrue())
	}, "90s").Should(Succeed())
}

// IsPodConditionReady returns true if the pod condition is ready
func IsPodConditionReady(pod *corev1.Pod, conditionType corev1.PodConditionType) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == conditionType && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

package e2e

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/yaml"
)

var (
	ns string

	//go:embed testdata/dummyStorageClass.yaml
	dummyStorageClassYaml []byte
)

func execAtLocal(cmd string, input []byte, args ...string) ([]byte, []byte, error) {
	var stdout, stderr bytes.Buffer
	command := exec.Command(cmd, args...)
	command.Stdout = &stdout
	command.Stderr = &stderr

	if len(input) != 0 {
		command.Stdin = bytes.NewReader(input)
	}

	err := command.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func kubectl(args ...string) ([]byte, []byte, error) {
	return execAtLocal("kubectl", nil, args...)
}

func kubectlWithInput(input []byte, args ...string) ([]byte, []byte, error) {
	return execAtLocal("kubectl", input, args...)
}

func TestMtest(t *testing.T) {
	if os.Getenv("E2ETEST") == "" {
		t.Skip("Run under e2e/")
		return
	}

	ns = os.Getenv("TEST_NAMESPACE")
	if ns == "" {
		t.Fatal("No TEST_NAMESPACE specified.")
	}

	rand.Seed(time.Now().UnixNano())

	RegisterFailHandler(Fail)

	SetDefaultEventuallyPollingInterval(time.Second)
	SetDefaultEventuallyTimeout(3 * time.Minute)

	RunSpecs(t, "pie test")
}

var _ = BeforeSuite(func() {
	By("[BeforeSuite] Waiting for pie to get ready")
	Eventually(func() error {
		stdout, stderr, err := kubectl("-n", ns, "get", "deploy", "pie", "-o", "json")
		if err != nil {
			return fmt.Errorf("kubectl get deploy failed. stderr: %s, err: %w", string(stderr), err)
		}

		var deploy appsv1.Deployment
		err = yaml.Unmarshal(stdout, &deploy)
		if err != nil {
			return err
		}

		if deploy.Status.AvailableReplicas != 1 {
			return errors.New("pie is not available yet")
		}

		return nil
	}).Should(Succeed())
})

var _ = Describe("pie", func() {
	It("should collect metrics of pie", func() {
		wg := sync.WaitGroup{}
		ctx, cancel := context.WithCancel(context.Background())
		defer func() {
			cancel()
			wg.Wait()
		}()

		_, _, err := kubectlWithInput(dummyStorageClassYaml, "apply", "-f", "-")
		Expect(err).NotTo(HaveOccurred())

		_, _, err = kubectl("rollout", "restart", "-n", ns, "deploy/pie")
		Expect(err).NotTo(HaveOccurred())

		err = portForward(ctx, &wg, ns, "svc/pie", "8080:8080")
		Expect(err).NotTo(HaveOccurred())

		stdout, _, err := kubectl("get", "node", "-o=jsonpath={.items[*].metadata.name}")
		Expect(err).NotTo(HaveOccurred())
		nodeLabelKey := string("node")
		nodeLabelValue := string(stdout)
		nodeLabelPair := io_prometheus_client.LabelPair{Name: &nodeLabelKey, Value: &nodeLabelValue}

		standardSCLabelKey := string("storage_class")
		standardSCLabelValue := string("standard")
		standardSCLabelPair := io_prometheus_client.LabelPair{Name: &standardSCLabelKey, Value: &standardSCLabelValue}

		dummySCLabelKey := string("storage_class")
		dummySCLabelValue := string("dummy")
		dummySCLabelPair := io_prometheus_client.LabelPair{Name: &dummySCLabelKey, Value: &dummySCLabelValue}

		Eventually(func(g Gomega) {
			resp, err := http.Get("http://localhost:8080/metrics")
			g.Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			var parser expfmt.TextParser
			metricFamilies, err := parser.TextToMetricFamilies(resp.Body)
			g.Expect(err).NotTo(HaveOccurred())

			By("checking metrics for standard SC")
			for _, metricName := range []string{
				"csi_driver_io_write_latency_seconds",
				"csi_driver_io_read_latency_seconds",
				"csi_driver_create_probe_fast_total",
			} {
				g.Expect(metricName).Should(BeKeyOf(metricFamilies))
				for _, metric := range metricFamilies[metricName].Metric {
					g.Expect(metric.Label).Should(ContainElement(&nodeLabelPair))
					g.Expect(metric.Label).Should(ContainElement(&standardSCLabelPair))
				}
			}

			By("checking metrics for dummy SC")
			g.Expect("csi_driver_create_probe_slow_total").Should(BeKeyOf(metricFamilies))
			for _, metric := range metricFamilies["csi_driver_create_probe_slow_total"].Metric {
				g.Expect(metric.Label).Should(ContainElement(&nodeLabelPair))
				g.Expect(metric.Label).Should(ContainElement(&dummySCLabelPair))
			}
		}).Should(Succeed())

	})
})

func portForward(ctx context.Context, wg *sync.WaitGroup, ns, target, port string) error {
	var portForwardOutput atomic.Value
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			output, _ := exec.CommandContext(ctx, "kubectl", "port-forward", "-n", ns, target, port).CombinedOutput()
			portForwardOutput.Store(output)
			time.Sleep(time.Second)
		}
	}()

	var err error
	localPort := strings.SplitN(port, ":", 2)[0]
	for i := 0; i < 12; i++ {
		var conn net.Conn
		conn, err = net.Dial("tcp", "localhost:"+localPort)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("failed to port-forward: output=%s: %w", portForwardOutput.Load().([]byte), err)
}

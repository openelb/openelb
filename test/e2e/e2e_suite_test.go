package e2e_test

import (
	"context"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/test/e2eutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	testClient    client.Client
	cfg           *rest.Config
	workspace     string
	testNamespace string
	nginxDeploy   *appsv1.Deployment
)

const (
	managerName    = "porter-manager"
	testDeployName = "test-app"
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func() {
	testNamespace = os.Getenv("TEST_NS")
	Expect(testNamespace).ShouldNot(BeEmpty())
	cfg, err := config.GetConfig()
	Expect(err).ShouldNot(HaveOccurred(), "Error reading kubeconfig")
	Expect(v1alpha1.AddToScheme(scheme.Scheme)).ShouldNot(HaveOccurred())
	c, err := client.New(cfg, client.Options{})
	Expect(err).NotTo(HaveOccurred(), "Error in creating client")
	testClient = c
	Expect(prepareDeployment()).ShouldNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(testClient.Delete(context.TODO(), nginxDeploy)).ShouldNot(HaveOccurred())
	e2eutil.KubectlDelete(os.Getenv("YAML_PATH"))
	Expect(e2eutil.DeleteNamespace(testClient, testNamespace)).ShouldNot(HaveOccurred())
})

func getWorkspace() string {
	_, filename, _, _ := runtime.Caller(0)
	return path.Dir(filename)
}

func prepareDeployment() error {
	deployStr := `{
	"kind": "Deployment",
	"apiVersion": "apps/v1",
	"metadata": {
		"name": "test-app",
		"creationTimestamp": null,
		"labels": {
			"app": "test-app"
		}
	},
	"spec": {
		"replicas": 3,
		"selector": {
			"matchLabels": {
				"app": "test-app"
			}
		},
		"template": {
			"metadata": {
				"labels": {
					"app": "test-app"
				}
			},
			"spec": {
				"containers": [
					{
						"name": "nginx",
						"image": "nginx:alpine"
					}
				]
			}
		}
	}
}`
	nginxDeploy = new(appsv1.Deployment)
	reader := strings.NewReader(deployStr)
	err := yaml.NewYAMLOrJSONDecoder(reader, 10).Decode(nginxDeploy)
	if err != nil {
		return err
	}
	nginxDeploy.Namespace = testNamespace
	nginxDeploy.Name = testDeployName
	return testClient.Create(context.TODO(), nginxDeploy)
}

See [e2e-tests](https://git.k8s.io/community/contributors/devel/sig-testing/e2e-tests.md)

[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/test/e2e/README.md?pixel)]()


export KUBECONFIG=/path/to/kubeconfig

ginkgo -focus=layer2
ginkgo -focus=BGP
ginkgo  -v --focus="LB:OpenELB" ./test/e2e/
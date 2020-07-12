# How to Build Porter Project

### Prerequisites

1. Go 1.11, the plugin uses [gobgp](https://github.com/osrg/gobgp) to create BGP client, and godgp requires Go 1.11.
2. Docker
3. Kustomize，it uses [kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md) to dynamically generate the k8s yaml files needed for the cluster.
4. If you need to push the plugin image to the remote private repository, you need to execute `docker login` in advance.

### Steps

1. Execute `git clone https://github.com/kubesphere/porter.git`, then enter into the folder.
2. Following with the above guides to modify the config.toml (Under `config/bgp/`).
3. (Optional）Modify the code according to your needs.
4. (Optional）Modify the parameters of the image according to your needs (Under `config/manager`).
5. (Optional）Follow the [Simulation Tutorial](doc/simulate_with_bird.md) to deploy a Bird node, then modify the BirdIP in `hack/test.sh`, and run `make e2e-test` for e2e testing.
6. Modify the IMG name in the Makefile, then run `make release`, and the final yaml file is under `/deploy`.
7. Execute `kubectl apply -f deploy/release.yaml` to deploy porter as a plugin.

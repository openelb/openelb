# Build the Porter Project

## Prerequisites

* You need to prepare a Linux environment.
* You need to install [Go 1.12 or later](https://github.com/kubesphere/porter/blob/master/doc/how-to-build.md).
* You need to install [Docker](https://www.docker.com/get-started).
* You need to install [Docker Buildx](https://www.docker.com/blog/getting-started-with-docker-for-arm-on-linux/).

## Procedure

1. Log in to your environment, and run the following commands to clone the Porter project and go to the `porter` directory:
   ```
   git clone https://github.com/kubesphere/porter.git
   ```
   
   ```bash
   cd porter
   ```

2. Run the following command to install Kustomize and Kubebuilder:

   ```bash
   ./hack/install_tools.sh
   ```

3. Run the following command to install controller-gen:

   ```bash
   go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.0
   ```

4. Run the following command to configure the environment variable for controller-gen:

   ```bash
   export PATH=/root/go/bin/:$PATH
   ```

   {{< notice note >}}

   You need to change `/root/go/bin/` to the actual path of controller-gen.

   {{</ notice >}}

5. Run the following command to generate CRDs and webhooks:

   ```
   make generate
   ```

6. Customize the values of `IMG_MANAGER` and `IMG_AGENT` in `Makefile` and run the following command to generate a YAML release file in the `deploy` directory:

   ```bash
   make release
   ```

   {{< notice note >}}

   * `IMG_MANAGER` specifies the repository and tag of the porter-manager image.

   * `IMG_AGENT` specifies the repository and tag of the porter-agent image.
   * Currently, Porter uses only the porter-manager image. The porter-agent image will be used in future versions.

   {{</ notice >}}

7. Run the following command to deploy Porter as a plugin:

   ```bash
   kubectl apply -f deploy/release.yaml
   ```
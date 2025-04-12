# openelb chart

---

## Introduction

OpenELB is an open source load balancer designed for bare metal Kubernetes clusters. It's implemented by physical switch, and uses BGP and ECMP to achieve the best performance and high availability.

---

## Prerequistes

- kubernetes >= 1.15

- helm3

---

## Installing the Chart

> Note: this chart is only supported by helm3

```bash
$ helm repo add openelb https://openelb.io/openelb
"openelb" has been added to your repositories

$ helm repo list
NAME    URL                                 
openelb https://openelb.io/openelb
$ 
$ helm repo update
Hang tight while we grab the latest from your chart repositories...
...Successfully got an update from the "openelb" chart repository
Update Complete. ⎈Happy Helming!⎈
$ 
$ helm search repo openelb
NAME            CHART VERSION   APP VERSION     DESCRIPTION                                    
openelb/openelb 0.6.0           0.6.0           Bare Metal Load-balancer for Kubernetes Cluster
$ 
$ helm install openelb openelb/openelb -n openelb-system --create-namespace --set speaker.layer2=true --set speaker.vip=true
NAME: openelbLAST DEPLOYED: Fri May 31 18:04:37 2024
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE: None
NOTES:
The OpenELB has been installed.

More info on the official site: https://openelb.io
```

## Uninstalling the Chart

To uninstall/delete the `openelb` release:

```bash
helm del openelb -n openelb-system
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following table lists the configurable parameters of the OpenELB chart and their default values.


| Parameter                     | Description                                                  | Default                           |
| ----------------------------- | ------------------------------------------------------------ | --------------------------------- |
| `global.imageRegistry`        | The default image registry to pull images from.              | `docker.io`                       |
| `global.tag`                  | The global tag for images.                                   |                                   |
| `global.imagePullSecrets`     | Secrets for pulling images from private registries.          | `[]`                              |
| `admission.image.repository`  | The repository for the admission webhook image.              | `kubesphere/kube-webhook-certgen` |
| `admission.image.tag`         | The tag for the admission webhook image.                     | `v1.1.1`                          |
| `admission.image.pullPolicy`  | The image pull policy for the admission webhook image.       | `IfNotPresent`                    |
| `controller.monitorEnable`    | Enable or disable monitoring for the controller              | `false`                           |
| `controller.monitorPort`      | The port to use for monitoring the controller.               | `50052`                           |
| `controller.webhookPort`      | The port to use for the webhook server.                      | `443`                             |
| `controller.image.repository` | The repository for the openelb-controller image.             | `kubesphere/openelb-controller`   |
| `controller.image.tag`        | The tag for the openelb-controller image.                    | `master`                          |
| `controller.image.pullPolicy` | The image pull policy for the openelb-controller image.      | `IfNotPresent`                    |
| `controller.resources`        | The resource limits and requests for the openelb-controller. |                                   |
| `controller.affinity`         | The affinity settings  for the openelb-controller.           |                                   |
| `controller.tolerations`      | The tolerations for the openelb-controller.                  |                                   |
| `controller.nodeSelector`     | The node selector for the openelb-controller.                |                                   |
| `controller.priorityClass`    | Priority Class name for the openelb-controller               |                                   |
| `speaker.enable`              | Enable or disable the speaker component.                     | `true`                            |
| `speaker.vip`                 | Enable or disable VIP mode for the speaker.                  | `false`                           |
| `speaker.layer2`              | Enable or disable Layer2 mode for the speaker.               | `false`                            |
| `speaker.memberlistSecret`    | The secret for the member list, if any.                      |                                   |
| `speaker.apiHosts`            | The API hosts for the speaker.                               | `:50051`                          |
| `speaker.monitorEnable`       | Enable or disable monitoring for the speaker.                | `false`                           |
| `speaker.monitorPort`         | The port to use for monitoring the speaker.                  | `50052`                           |
| `speaker.image.repository`    | The repository for the openelb-speaker image.                | `kubesphere/openelb-speaker`      |
| `speaker.image.tag`           | The tag for the openelb-speaker image.                       | `master`                          |
| `speaker.image.pullPolicy`    | The image pull policy for the openelb-speaker image.         | `IfNotPresent`                    |
| `speaker.resources`           | The resource limits and requests for the openelb-speaker.    |                                   |
| `speaker.affinity`            | The affinity settings for the openelb-speaker.               |                                   |
| `speaker.tolerations`         | The tolerations for the openelb-speaker.                     |                                   |
| `speaker.nodeSelector`        | The node selector for the openelb-speaker                    |                                   |
| `speaker.priorityClass`       | Priority Class Name for the openelb-speaker                  |                                   |
| `customImage.enable`          | Enable or disable the use of custom images.                  | `false`                           |
| `customImage.forwardImage`    | The custom image for the openelb-forward component.          |                                   |
| `customImage.proxyImage`      | The custom image for the openelb-proxy component.            |                                   |


Specify parameters using `--set key=value[,key=value]` argument to `helm install`

Alternatively a YAML file that specifies the values for the parameters can be provided like this:

```bash
$ helm install --name openelb -f values.yaml openelb/openelb -n openelb-system --create-namespace
```

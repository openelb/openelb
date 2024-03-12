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
helm repo add stable https://charts.kubesphere.io/stable
help repo update
helm install openelb stable/openelb
```

## Uninstalling the Chart

To uninstall/delete the `openelb` release:

```bash
helm del openelb
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following table lists the configurable parameters of the OpenELB chart and their default values.

| Parameter | Description  | Default              |
| -----------------------    | -----------------------|----------------------|
| `manager.image.repository`| `manager` image name.        | `kubesphere/openelb` |
| `manager.image.tag`       | `manager` image tag.         | `v0.6.0`             |
| `manager.image.pullPolicy`| `manager` image pull Policy. | `IfNotPresent`       |
| `manager.resources`       | openelb manager resource requests and limits      | `{}`                 |
| `manager.nodeSelector`     | Node labels for pod assignment             | `{}`                 |
| `manager.terminationGracePeriodSeconds`  | Wait up to this many seconds for a broker to shut down gracefully, after which it is killed   | `10`                 |
| `manager.tolerations` | List of node tolerations for the pods. https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/  | `[]`                 |
| `manager.serviceAccount.name`    | Name of Kubernetes serviceAccount.   | `default`            |
| `manager.serviceAccount.create`    | Whether to create a serviceaccount   | `false`              |
| `manager.apiHosts`    | GoBGP will listen to the address.   | `:50051`             |
| `manager.readinessPort`    | The openelb manager readinessprobe listens to addresses.   | `8000`               |

Specify parameters using `--set key=value[,key=value]` argument to `helm install`

Alternatively a YAML file that specifies the values for the parameters can be provided like this:

```bash
$ helm install --name my-openelb -f values.yaml stable/openelb
```




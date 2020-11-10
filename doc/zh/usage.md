# Usage

Porter安装并配置好之后， 可以通过创建LoadBalancer Service来使用Porter暴露。

## 配置LoadBalancer Service

LoadBalancer Service必须要添加annotations `lb.kubesphere.io/v1alpha1: porter`，type也要指定为`LoadBalancer`。

```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
  annotations:
    lb.kubesphere.io/v1alpha1: porter
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  ports:
    - name:  http
      port:  8088
      targetPort:  80
```

**其他不符合条件的Service， Porter不会处理。**

### 指定协议

可以通过annotation "protocol.porter.kubesphere.io/v1alpha1"指定Porter从何种协议的Eip中分配地址。

以下示例指定annotation "protocol.porter.kubesphere.io/v1alpha1: bgp", 表示Porter将从BGP协议的Eip中分配地址。它也是Porter的默认值， 可以省略。
```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: bgp
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  ports:
    - name:  http
      port:  8088
      targetPort:  80
```

以下示例指定annotation "protocol.porter.kubesphere.io/v1alpha1: layer2", 表示Porter将从layer2协议的Eip中分配地址。
```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: layer2
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  ports:
    - name:  http
      port:  8088
      targetPort:  80
```

### 指定Eip

假设我们有如下示例中的Eip

```yaml
apiVersion: network.kubesphere.io/v1alpha2
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    address: 192.168.0.0/24
```

Porter 处理LoadBalancer Service时会遍历所有`enable`的Eip， **从第一个匹配协议并且有空闲地址的Eip中分配地址**， 这意味着分配的地址会是不可预期的， 如果你需要使用固定的地址，那么可以通过为LoadBalancer Service 指定参数`spec.loadBalancerIP`来固定地址。

```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: bgp
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  loadBalancerIP: 192.168.0.100
  ports:
    - name:  http
      port:  8088
      targetPort:  80
```

如果你不需要固定地址， 只需要从特定地址池中分配地址， 那么可以指定annotation "eip.porter.kubesphere.io/v1alpha2", 它的值为Eip的Name。

```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: bgp
    eip.porter.kubesphere.io/v1alpha2: eip-sample-pool
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  ports:
    - name:  http
      port:  8088
      targetPort:  80
```

### 配置spec.externalTrafficPolicy

默认LoadBalancer Service的`externalTrafficPolicy`值为`Cluster`, 这意味着Kubernetes集群中所有Node都可以作为Nexthop来转发流量, 区别只是在于：
* BGP模式下， Porter会发布包含所有Node作为Nexthop的等价路由
* layer2摸下， Porter会随机选择一个Node作为Nexthop来转发流量

Kubernetes还为`externalTrafficPolicy`提供另外一个值`Local`，这意味着Kubernetes集群中只有包含在Endpoints中的Node才能作为Nexthop来转发流量， 这相较于前者来说有一点好处就是： **流量转发路径短，不用再通过其他Node中的kube-proxy来多一次转发**

### 共享Eip

一般情况下， 对外暴露的公网IP地址会有限， 当需要对外暴露的服务比较多时， 这个时候Eip地址就不够用， **这里Porter目前支持BGP模式的Eip共享**。 在使用的时候， 你只需要参照上面**指定Eip**一节中指定想要共享的Eip即可。

**注意：两个Service ports端口不能重复, 并且需要externalTrafficPolicy不能为Local**

如下是一个已经存在的Service， 并且指定`loadBalancerIP: 192.168.0.100`
```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: bgp
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  loadBalancerIP: 192.168.0.100
  ports:
    - name:  http
      port:  8088
      targetPort:  80
```
这里，如果想复用192.168.0.100这个IP，你可以直接创建Service并且同时指定该IP即可
```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc2
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: bgp
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  loadBalancerIP: 192.168.0.100
  ports:
    - name:  http
      port:  8089
      targetPort:  80
```
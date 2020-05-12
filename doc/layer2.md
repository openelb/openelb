# 与MetalLB Layer2的区别
MetalLB目前Layer2的信息通告都是通过Daemonset中的speaker进行的，并且通过Memberlist维护所有speaker关系，
保证同一个Service EIP的ARP请求只会被一个speaker应答。这么做的原因有二：通过Daemonset保证speaker的高可用；
通过speaker分散应答ARP请求，减轻speaker压力。

考虑到Layer2场景下，其实ARP请求并不会是一个非常高频的请求，并且请求之后也会有cache，另外ARP请求本身就是一个广播
请求，尽管MetalLB的部分speaker不应答请求，但还是会收到广播数据包，只是最后被speaker中Drop掉，相对于应答也并没有
减轻什么压力。所以Porter最终实现Layer2将其ARP请求接受应答处理都放在Porter manager中， 通过manager的主备策略实现
高可用，同时可以复用代码。

在构造ARP应答包的时候，Porter会采用manager所在NODE的MAC应答， 这样可以避免一些云平台的ARP Spoof引起的丢包。

# Porter Layer2的使用
为了支持Layer2，以及后续不同的LB策略，在EIP中引入字段lbTye，用以表示是使用bgp，layer2或者其他。

例如创建一个layer2的EIP
```yaml
kubectl apply -f - <<EOF
apiVersion: network.kubesphere.io/v1alpha1
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    address: 10.11.11.0/24
    protocol: layer2
    disable: false
EOF
```

在service中使用layer2的时候，我们需要使用”protocol.porter.kubesphere.io/v1alpha1: layer2”指定使用layer2.
如果service中通过”eip.porter.kubesphere.io/v1alpha1“指定了EIP， 那么我们可以省略.
```yaml
kind: Service
apiVersion: v1
metadata:
    name:  mylbapp
    annotations:
        lb.kubesphere.io/v1alpha1: porter
        #eip.porter.kubesphere.io/v1alpha1: 1.1.1.1 如果需要手动指定eip，可以添加这个注记
        #protocol.porter.kubesphere.io/v1alpha1: layer2  如果没有指定eip，那么必须添加这个标记
spec:
    selector:
        app:  mylbapp
    type:  LoadBalancer 
    ports:
      - name:  http
        port:  8088
        targetPort:  80
    ```

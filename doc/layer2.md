#Preparation

Normally, Nic will answer ARP request which ip not reside on it. In order to let porter-manager
control all arp reply for layer2 eip, we should config kube-proxy strictARP.
```yaml
kubectl edit configmap -n kube-system kube-proxy

apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: "ipvs"
ipvs:
  strictARP: true
```

# Porter Layer2 Usage
Porter now support bgp and layer2, and will support non-bgp switch in future. In
order to distinguish them, filed `protocol` was added. 

An example of how to use it will follow.

Create a layer2 EIP
```yaml
kubectl apply -f - <<EOF
apiVersion: network.kubesphere.io/v1alpha1
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    address: 10.11.11.0/24
    protocol: layer2
EOF
```

when we use layer2 EIP in service, annotation "protocol.porter.kubesphere.io/v1alpha1: layer2"
should be provided
```yaml
kind: Service
apiVersion: v1
metadata:
    name:  mylbapp
    annotations:
        lb.kubesphere.io/v1alpha1: porter
        #eip.porter.kubesphere.io/v1alpha1: 1.1.1.1 
        #protocol.porter.kubesphere.io/v1alpha1: layer2 
spec:
    selector:
        app:  mylbapp
    type:  LoadBalancer 
    ports:
      - name:  http
        port:  8088
        targetPort:  80
    ```

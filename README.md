Helm chart for OpenELB

1、Add OpenELB repo information locally
```bash
% helm repo add openelb https://openelb.github.io/openelb
"openelb" has been added to your repositories

% helm repo list
NAME    URL                                 
openelb https://openelb.github.io/openelb

% helm search repo openelb
NAME            CHART VERSION   APP VERSION     DESCRIPTION                                    
openelb/openelb 0.6.0           0.6.0           Bare Metal Load-balancer for Kubernetes Cluster
```

2、Install OpenELB
```bash
$ helm install openelb openelb/openelb -n openelb-system --create-namespace --set speaker.layer2=true --set speaker.vip=true
NAME: openelb
LAST DEPLOYED: Fri May 31 18:04:37 2024
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE: None
NOTES:
The OpenELB has been installed.

More info on the official site: https://openelb.io
```
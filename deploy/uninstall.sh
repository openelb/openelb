#!/usr/bin/env bash
set -euo pipefail

NAMESPACE=${NAMESPACE:-'default'}                                          # the namespace where openelb is installed
APPLICATION_NAME=${APPLICATION_NAME:-'openelb'}                            # when you installed by helm, The APPLICATION_NAME should be the same as you specified.
MANAGER_SERVICEACCOUNT=${MANAGER_SERVICEACCOUNT:-'openelb-manager'}        # openelb-manager serviceAccount
ADMISSION_SERVICEACCOUNT=${ADMISSION_SERVICEACCOUNT:-'openelb-admission'}  # openelb-admission serviceAccount

echo "=================================="
echo "Start uninstall openelb"
kubectl delete --ignore-not-found mutatingwebhookconfiguration $APPLICATION_NAME-admission
kubectl delete --ignore-not-found validatingwebhookconfiguration $APPLICATION_NAME-admission
kubectl delete --ignore-not-found svc $APPLICATION_NAME-admission -n $NAMESPACE
kubectl delete --ignore-not-found deploy $APPLICATION_NAME-manager -n $NAMESPACE
kubectl delete --ignore-not-found job $APPLICATION_NAME-admission-create -n $NAMESPACE
kubectl delete --ignore-not-found job $APPLICATION_NAME-admission-patch -n $NAMESPACE
kubectl delete --ignore-not-found ds openelb-keepalive-vip -n $NAMESPACE
kubectl delete --ignore-not-found cm openelb-vip-configmap -n $NAMESPACE

# delete role rolebinding / clusterrole clusterrolebinding / sa
kubectl delete --ignore-not-found role openelb-admission
kubectl delete --ignore-not-found role leader-election-role
kubectl delete --ignore-not-found rolebinding openelb-admission
kubectl delete --ignore-not-found rolebinding leader-election-rolebinding

kubectl delete --ignore-not-found clusterrole kube-keepalived-vip
kubectl delete --ignore-not-found clusterrole openelb-admission
kubectl delete --ignore-not-found clusterrole openelb-manager-role
kubectl delete --ignore-not-found clusterrolebinding kube-keepalived-vip
kubectl delete --ignore-not-found clusterrolebinding openelb-admission
kubectl delete --ignore-not-found clusterrolebinding openelb-manager-rolebinding

kubectl delete --ignore-not-found sa kube-keepalived-vip -n $NAMESPACE
kubectl delete --ignore-not-found sa $MANAGER_SERVICEACCOUNT -n $NAMESPACE
kubectl delete --ignore-not-found sa $ADMISSION_SERVICEACCOUNT -n $NAMESPACE

echo "=================================="
echo "Patch openelb controlled resources"
for eip in $(kubectl get eips.network.kubesphere.io -o name); do
  kubectl patch $eip -p '{"metadata": {"finalizers": null}}' --type merge
done

for bgppeer in $(kubectl get bgppeers.network.kubesphere.io -o name); do
  kubectl patch $bgppeer -p '{"metadata": {"finalizers": null}}' --type merge
done

for bgpconf in $(kubectl get bgpconfs.network.kubesphere.io -o name); do
  kubectl patch $bgpconf -p '{"metadata": {"finalizers": null}}' --type merge
done


kubectl get svc --all-namespaces -l eip.openelb.kubesphere.io/v1alpha2 | awk '{print $1, $2}' | while read namespace name; do
  if [ $name == "NAME" ]; then
    continue
  fi
  # kubectl patch svc $name -n $namespace -p '{"metadata": {"finalizers": null}}' --type merge
  kubectl patch svc $name -n $namespace -p '{"metadata": {"$deleteFromPrimitiveList/finalizers": ["finalizer.lb.kubesphere.io/v1alpha1"]}}' --type strategic
done


echo "=================================="
echo "Delete CRD"
kubectl delete --ignore-not-found crd eips.network.kubesphere.io
kubectl delete --ignore-not-found crd bgpconfs.network.kubesphere.io
kubectl delete --ignore-not-found crd bgppeers.network.kubesphere.io

echo "=================================="
if [[ $NAMESPACE == "openelb-system" ]]; then
  echo "Delete namespace"
  kubectl delete --ignore-not-found all --all -n $NAMESPACE
  kubectl delete --ignore-not-found ns $NAMESPACE
fi

echo "Uninstall openelb successfully"
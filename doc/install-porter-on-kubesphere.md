# Install Porter on KubeSphere (Web Console)

This document describes how to install and delete Porter on the [KubeSphere](https://kubesphere.io/) web console. For details about how to install and delete Porter in a Kubernetes cluster without KubeSphere, see [Install Porter on Kubernetes (kubectl and Helm)](./install-porter-on-kubernetes.md).

## Prerequisites

You need to prepare a Kubernetes cluster with KubeSphere, and ensure that the Kubernetes version is 1.15 or later. Porter requires CustomResourceDefinition (CRD) v1, which is only supported by Kubernetes 1.15 or later. You can use the following methods to install KubeSphere:

* [Deploy a new Kubernetes cluster with KubeSphere](https://kubesphere.io/docs/installing-on-linux/).
* [Install KubeSphere in an existing Kubernetes cluster](https://kubesphere.io/docs/installing-on-kubernetes/).

Porter is designed to be used in bare-metal Kubernetes environments. However, you can also use a cloud-based Kubernetes cluster for learning and testing.

## Install Porter on the KubeSphere Web Console

1. Log in to the KubeSphere console and go to your workspace.

   ![enter-workspace](./img/install-porter-on-kubesphere/enter-workspace.jpg)

2. On the left navigation bar, choose **Apps Management** > **App Repos**, and click **Add Repo** on the right.

   ![add-repo](./img/install-porter-on-kubesphere/add-repo.jpg)

3. In the displayed dialog box, set **App Repository Name** (for example, `KubeSphere-test`), set **URL** to `https://charts.kubesphere.io/test`, click **Validate** to check the URL, and click **OK**.

   ![repo-spec](./img/install-porter-on-kubesphere/repo-spec.jpg)

4. Go to your project, choose **Application Workloads** > **Applications** on the left navigation bar, and click **Deploy New Application** on the right.

   ![deploy-new-app](./img/install-porter-on-kubesphere/deploy-new-app.jpg)

5. In the displayed dialog box, click **From App Templates**.

   ![from-app-templates](./img/install-porter-on-kubesphere/from-app-templates.jpg)

6. Select **KubeSphere-test** from the drop-down list and click **porter**.

   ![porter-template](./img/install-porter-on-kubesphere/porter-template.jpg)

7. Click **Deploy** and follow the wizard instructions to complete the installation. You can customize the chart configuration in the YAML file based on your requirements.

   ![deploy-porter](./img/install-porter-on-kubesphere/deploy-porter.jpg)

   ![porter-yaml](./img/install-porter-on-kubesphere/porter-yaml.jpg)

8. Choose **Application Workloads** > **Pods** on the left navigation bar to check whether the status of porter-manager is **running**. If yes, porter has been installed successfully.

   ![verify-porter](./img/install-porter-on-kubesphere/verify-porter.jpg)

## Delete Porter on the KubeSphere Web Console

To delete Porter on the KubeSphere web console, go to your project, choose **Application Workloads** > **Applications** on the left navigation bar, click ![porter-operation](./img/install-porter-on-kubesphere/porter-operation.jpg) on the right of the Porter application, and choose **Delete** from the drop-down list.

![delete-porter](./img/install-porter-on-kubesphere/delete-porter.jpg)

{{< notice note >}}

Before deleting Porter, you must first delete all services that use Porter.

{{</ notice >}}
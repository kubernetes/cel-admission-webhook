# Installation Guide

This document describes steps necessary to install the CEL Admission Webhook Shim
on your local cluister

# Easy Installation

## For Staging/Production using Kpt

> 🚧 WORK IN PROGRESS 🚧

## For Testing using Kind

This method uses a local [`kind`](https://kind.sigs.k8s.io) cluster and is recommended only for testing/development. 

Run the following script to build the image from source, and generate a minimum working webhook configuration:

```sh
cd $REPO_PATH
# Build container image to run the controller
docker build -t kubernetes-x/cel-shim-webhook:v1 ./
# Make container image available to Pods in the Kind cluster
# NOTE: If your kind cluster is named, this will also need the --name argument with the name of your cluster
kind load docker-image kubernetes-x/cel-shim-webhook:v1
# Generate deployment, serviceaccount, certificates, etc into ./_output/manifests
./hack/gen-manifests.sh
# Apply required manifests for webhook
kubectl apply -f ./_output/manifests --server-side=true
```

This script creates a self-signed CA and a server certificate, and outputs manifests containing webhook configurations and TLS secrets.

After running the final `kubectl apply` the CEL Admission Webhook should
be active on your kind cluster and ready to enforce policies.


# Manual Installation

> 🚧 WORK IN PROGRESS 🚧

Requirements:
  1. Kubernetes 1.16+
  2. kubectl configured to cluster

## Install CRDs

The `cel-admission-webhook` requires CRDs to properly function. Lets begin by making sure those CRDs are available in our cluster.

### From GitHub

```sh
kubectl apply -f <https://github.com/URL/to/unified/crds/yaml/in/repo> --server-side=true
```
```console
customresourcedefinition.apiextensions.k8s.io/validatingadmissionpolicies.admissionregistration.x-k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/validatingadmissionpolicybindings.admissionregistration.x-k8s.io serverside-applied
```

### From Source

```sh
kubectl apply -f ./artifacts/crds --server-side=true
```
```console
customresourcedefinition.apiextensions.k8s.io/validatingadmissionpolicies.admissionregistration.x-k8s.io serverside-applied
customresourcedefinition.apiextensions.k8s.io/validatingadmissionpolicybindings.admissionregistration.x-k8s.io serverside-applied
```

## Create Namespace

For organizational purposes, we will apply all objects to the `celshim` namespace. Feel free to change this to suit your needs.

```sh
kubectl create namespace celshim
```
```console
namespace/celshim created
```

## Create TLS Secret

To authenticate the webhook to the cluster, we require a TLS server certificate and CA trusted by the cluster.

There are a number of ways to generate certificates, or sign a certificate with a pre-existing CA.
This guide includes a way to create a self-signed cert specific to the webhook.

If you have your own PKI, refer to your own processes for generating a certificate for the webhook. The only requirement is that the SANs in the certificate match the URL used to reach the webhook service. For our service named `cel-shim-webhook` in `celshim` namespace that will be `cel-shim-webhook.celshim.svc`

### Generate-Signed Certificate with `gentls`

You can quickly get a self-signed CA and server certificate through
the bundled `gentls` command:

```sh
go run k8s.io/cel-admission-webhook/cmd/gentls --host=cel-shim-webhook.celshim.svc
```
> Change the host argument if your webhook service will be named differently.

The output of this script are three files:
1. `ca.pem`: Self-Signed Certificate Authority. Set this aside for later.
2. `server.pem`: Server Certificate signed by `ca.pem`. This should be placed in a TLS Secret.
3. `server-key.pem`. Private key for `server.pem`. This should be placed in a TLS Secret.

### Create Secret

Run the following command after you have a key and a certificate for the webhook server. This will create a secret in the `celshim` namespace
to be injected into the shim deployment.

```sh
kubectl create secret tls cel-shim-webhook --cert=server.pem --key=server-key.pem --namespace celshim
```
```console
secret/cel-shim-webhook created
```

## Setup RBAC

ServiceAccount and RBAC configuration is often very cluster specific. Below is an example of a ServiceAccount, ClusterRole, and ClusterRoleBinding that can be used with the cel shim webhook.

If you already have a service account for the webhook to use, feel free to 
skip this step and use substitute it into the Deployment configuration.

### Create Service Account

```sh
kubectl create serviceaccount cel-webhook -n celshim
```
```console
serviceaccount/cel-webhook created
```
<details> 
  <summary>Equivalent YAML</summary>

```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cel-webhook
  namespace: celshim
```
</details>

### Create ClusterRole

```sh
kubectl create clusterrole cel-webhook --verb="*" --resource="*.*"
```
```console
clusterrole.rbac.authorization.k8s.io/cel-webhook created
```
<details> 
  <summary>Equivalent YAML</summary>

```yaml
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cel-webhook
rules:
  - verbs: ["*"]
    apiGroups: ["*"]
    resources: ['*']
```
</details>

### Create ClusterRoleBinding

```sh
kubectl create clusterrolebinding cel-webhook --clusterrole=cel-webhook --serviceaccount=celshim:cel-webhook
```
```console
clusterrolebinding.rbac.authorization.k8s.io/cel-webhook created
```
<details> 
  <summary>Equivalent YAML</summary>

```yaml
---
kind: ClusterRoleBinding
metadata:
  name: cel-webhook
  namespace: celshim
apiVersion: rbac.authorization.k8s.io/v1
subjects:
  - kind: ServiceAccount
    name: cel-webhook
    namespace: celshim
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cel-webhook
  namespace: celshim
```
</details>

## Create Deployment
### From Public Container Registry

This project does not yet have any releases in public registry, but that is the long term plan.

### From Source

Alternatively, you may also choose to build the container image from source yourself:

#### 1. Build Container Image

```sh
git clone git@github.com:kubernetes/cel-admission-webhook.git
cd $REPO_PATH
docker build -t kubernetes-x/cel-shim-webhook:v1 ./
```

#### 2. Publish To Registry

To make the container image available to run in Kubernetes, it must be published
to a container registry trusted and accessible by your cluster.

##### Local Kind Cluster

Images can be loaded into a kind cluster with a single command.

```sh
kind load docker-image kubernetes-x/cel-shim-webhook:v1
```
> NOTE: `kind` assumes your cluster is named `kind` by default. If your cluster 
is named differently, you must also supply the `--name` argument.


##### Google Container Registry

TODO

##### Other Cloud Providers

TODO

#### 3. Install Deployment

Apply the following deployment configuration.

```sh
kubectl apply --server-side=true -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cel-shim-webhook
  namespace: celshim
  labels:
    app: cel-shim-webhook
spec:
  selector:
    matchLabels:
      app: cel-shim-webhook
  template:
    metadata:
      labels:
        app: cel-shim-webhook
    spec:
      serviceAccountName: cel-webhook
      containers:
        - name: cel-shim-webhook
          image: kubernetes-x/cel-shim-webhook:v1
          args:
            - -cert=/etc/tls/tls.crt
            - -key=/etc/tls/tls.key
            - -addr=:443
          volumeMounts:
            - mountPath: "/etc/tls"
              name: tls
              readOnly: true
      volumes:
        - name: tls
          secret:
            # kubectl create secret tls cel-shim-webhook --cert=server.pem --key=server-key.pem
            secretName: cel-shim-webhook
EOF
```
```console
deployment.apps/cel-shim-webhook serverside-applied
```
> NOTE: If you named your service account or TLS secret differently, this configuration must be edited to reflect that.

## Create Service

Now that the controller is standing up, expose the deployment to the apiserver
with a service:

```sh
kubectl apply --server-side=true -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: cel-shim-webhook
  namespace: celshim
  labels:
    app: cel-shim-webhook
spec:
  selector:
    app: cel-shim-webhook
  ports:
    - port: 443
      protocol: TCP
EOF
```
```console
service/cel-shim-webhook serverside-applied
```

This service matches all webhook Pods with label `app: cel-shim-webhook`. 
Requests to this service on port `443` will be directed to a running webhook container.

## Install Webhook Configuration

With deployment running and service properly pointing to it, it is time to do the final step of configuring kubernetes to use the webhook. ValidatingWebhookConfiguration informs Kubernetes that you'd like to use a webhook, on which resources, and how to connect to the webhook.

### Encode CA Certificate

Find the CA pem file used to sign the wbehook server certificate. Encode it into
base64 with the following command:

```sh
export CA_BUNDLE=$(cat ca.pem | base64 -w0) && echo $CA_BUNDLE
```
```console
LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJMVENCNEtBREFnRUNBaFIzV3huK3JScmdueC9BeWNOK1hraURsWDBKRkRBRkJnTXJaWEF3RlRFVE1CRUcKQTFVRUF4TUtVMlZzWmxOcFoyNWxaREFlRncweU16QTFNREV4T1RReE5UZGFGdzB5TkRBME16QXhPVFEyTlRkYQpNQlV4RXpBUkJnTlZCQU1UQ2xObGJHWlRhV2R1WldRd0tqQUZCZ01yWlhBRElRQkljS2RCTktJYm52ZThQRWw2CnpvRm5Ud0wveWc0WjlwdTZjZlR2dGFUdzBhTkNNRUF3RGdZRFZSMFBBUUgvQkFRREFnS2tNQThHQTFVZEV3RUIKL3dRRk1BTUJBZjh3SFFZRFZSME9CQllFRkVMcEh2ZXFIbkJDVWxKSDR1WFk3RG5lZXZrR01BVUdBeXRsY0FOQgpBSjVNOVd3L1BEcW40SXVQVjViK05NR05ocXZlK2tyNUYzV0s2RTZZSE80WXRwTFhlL2dTOUlNUlZnTW14Sm5LCks2U3IzTGVMQmxXVzFFN29kbmY4QXd3PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
```

### Create ValidatingWebhookConfiguration

```sh
kubectl apply --server-side=true -f -<<EOF
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: "cel-shim.example.com"
  namespace: "celshim"
webhooks:
  - name: "cel-shim.example.com"
    rules:
      - apiGroups: ["*"]
        apiVersions: ["*"]
        operations: ["*"]
        resources: ["*"]
        scope: "*"
    clientConfig:
      service:
        namespace: celshim
        name: cel-shim-webhook
        path: /validate
        port: 443
      caBundle: |
        $CA_BUNDLE
    admissionReviewVersions: ["v1"]
    sideEffects: None
    timeoutSeconds: 2
    namespaceSelector:
      matchExpressions:
      - key: kubernetes.io/metadata.name
        operator: NotIn
        values: ["kube-system","kube-node-lease","kube-public","celshim"]
    objectSelector:
      matchExpressions:
      - key: app
        operator: NotIn
        values: ["cel-shim-webhook"]
EOF
```
```console
validatingwebhookconfiguration.admissionregistration.k8s.io/cel-shim.example.com serverside-applied
```

> NOTE: We use $CA_BUNDLE environment variable in the deployment. This inserts the base64-encoded CA certificate into the resource to be created.

This configuration ignores anything in `celshim` namespace and
some common internal Kubernetes objects. If you use a different namespace for 
your deployment, you should also add it to the ignore list.

# Test Policy

## Create Policy

Create an admission policy  to match all operations on ConfigMaps whose names 
do not end with `k8s`:

```sh
kubectl apply --server-side=true -f - <<EOF
apiVersion: admissionregistration.x-k8s.io/v1alpha1
kind: ValidatingAdmissionPolicy
metadata:
  name: k8s-policy
spec:
  matchConstraints:
    resourceRules:
    - resourceNames: [ ]
      operations: [ "*" ]
      apiGroups: [ "" ]
      apiVersions: [ "v1" ]
      resources: [ "configmaps" ]

  paramKind:
    apiVersion: v1
    kind: ConfigMap
  failurePolicy: Fail
  validations:
  - expression: object.metadata.name.endsWith('k8s')
EOF
```

## Create Binding

Create a binding to begin enforcing the policy:

```sh
kubectl apply --server-side -f - <<EOF
apiVersion: admissionregistration.x-k8s.io/v1alpha1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: k8s-policy-binding
spec:
  policyName: k8s-policy
  validationActions:
  - Deny
EOF
```

## Test Error Case

```sh
kubectl apply --server-side=true -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  key: value
EOF
```
```console
The request is invalid: admission webhook "cel-shim.example.com" denied the request: configmaps "my-config" is forbidden: ValidatingAdmissionPolicy 'k8s-policy' with binding 'k8s-policy-binding' denied request: failed expression: object.metadata.name.endsWith('k8s')
```

## Test Success Case

```sh
kubectl apply --server-side=true -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config-k8s
data:
  key: value
EOF
```
```console
configmap/my-config-k8s serverside-applied
```

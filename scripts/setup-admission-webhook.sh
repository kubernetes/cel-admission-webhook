#!/bin/bash
set -ex
kubectl apply --server-side=true -f https://raw.githubusercontent.com/kubernetes/cel-admission-webhook/main/artifacts/crds/admissionregistration.x-k8s.io_validatingadmissionpolicies.yaml
kubectl apply --server-side=true -f https://raw.githubusercontent.com/kubernetes/cel-admission-webhook/main/artifacts/crds/admissionregistration.x-k8s.io_validatingadmissionpolicybindings.yaml
kubectl create namespace celshim
go run k8s.io/cel-admission-webhook/cmd/gentls --host=cel-shim-webhook.celshim.svc
kubectl create secret tls cel-shim-webhook --cert=server.pem --key=server-key.pem --namespace celshim
kubectl create serviceaccount cel-webhook -n celshim
kubectl create clusterrole cel-webhook --verb="*" --resource="*.*"
kubectl create clusterrolebinding cel-webhook --clusterrole=cel-webhook --serviceaccount=celshim:cel-webhook
docker build -t kubernetes-x/cel-shim-webhook:v1 ./
kind load docker-image kubernetes-x/cel-shim-webhook:v1
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
export CA_BUNDLE=$(cat ca.pem | base64 -w0) && echo $CA_BUNDLE

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
        resources: ["*", "pods/*"]
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
EOF

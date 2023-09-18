# Kubeenforcer
This project aims to provide a simple way to enforce policies on Kubernetes clusters. It is based on the [cel-admission-webhook](https://github.com/kubernetes/cel-admission-webhook) project.

## How it works
Kubeenforcer is a Kubernetes admission webhook that intercepts requests to the Kubernetes API server and evaluates a [CEL](https://github.com/google/cel-spec) expression. If the expression evaluates to true, the request is allowed to proceed. If the expression evaluates to false, the request is denied/audited.
Using this set of rules, you can enforce policies on your Kubernetes cluster and monitor on suspicious activity such as a pod trying to mount a hostPath volume or an exec request to a pod.

## Alerting
Kubeenforcer can be configured to send alerts to multiple destinations. Currently, the following destinations are supported:
- Alertmanager
    - Slack
    - Email
    - Pagerduty
    - Opsgenie
    - Victorops
    - Webhook
    - Wechat
    - Discord
    - Telegram

## Installation

### Using Helm:

```bash
git clone https://github.com/kubescape/kubeenforcer.git && cd kubeenforcer
kubectl create namespace kubescape
helm install kubeenforcer -n kubescape ./charts/kubeenforcer
```

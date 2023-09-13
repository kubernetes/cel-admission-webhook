# Kubeenforcer
This project aims to provide a simple way to enforce policies on Kubernetes clusters. It is based on the [cel-admission-webhook](https://github.com/kubernetes/cel-admission-webhook) project.

## How it works
Kubeenforcer is a Kubernetes admission webhook that intercepts requests to the Kubernetes API server and evaluates a [CEL](https://github.com/google/cel-spec) expression. If the expression evaluates to true, the request is allowed to proceed. If the expression evaluates to false, the request is denied/audited.
Using this set of rules, you can enforce policies on your Kubernetes cluster and monitor on suspicious activity such as a pod trying to mount a hostPath volume or an exec request to a pod.

## Alerting
Kubeenforcer can be configured to send alerts to multiple destinations. Currently, the following destinations are supported:
- Alertmanager
- Slack (WIP)


## Installation
Using Helm:
```bash
...
```

## Configuration
The following table lists the configurable parameters of the Kubeenforcer chart and their default values.

| Parameter | Description | Default |
| --------- | ----------- | ------- |

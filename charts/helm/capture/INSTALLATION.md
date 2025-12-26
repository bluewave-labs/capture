# Kubernetes Installation Guide for Capture

This guide walks you through deploying Capture on a Kubernetes cluster using Helm.

## Prerequisites

- A running Kubernetes cluster
- Helm CLI installed and configured
- `kubectl` configured to access your cluster

## Steps

### 1. Clone and navigate to the Helm chart

```shell
git clone https://github.com/bluewave-labs/capture.git
cd capture/charts/helm/capture
```

### 2. Customize values.yaml

Edit `values.yaml` and set the required `secret.apiSecret`. Adjust image, service, and other settings as needed.

### 3. Deploy the Helm chart

```shell
helm install capture .
```

### 4. Verify deployment

```shell
kubectl get pods
kubectl get svc
```

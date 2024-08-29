# ResourceFungibility Plugin

Support resource fungibility like GPU types with at most 8 alternative choices.

## How to use

### Set kube-scheduler.yaml

Generally looks like:

```yaml
  containers:
  - args:
    - --authentication-kubeconfig=/etc/kubernetes/scheduler.conf
    - --authorization-kubeconfig=/etc/kubernetes/scheduler.conf
    - --bind-address=127.0.0.1
    - --feature-gates=MaxUnavailableStatefulSet=true # this is require by lws
    - --kubeconfig=/etc/kubernetes/scheduler.conf
    - --leader-elect=true
    - --config=/etc/kubernetes/kube-scheduler-config.yaml # set the kube-scheduler-config.yaml
    image: inftyai/vscheduler:<version> # set the right version of vscheduler image
```

### Set KubeSchedulerConfiguration

A minimal `kube-scheduler-config.yaml` looks like:

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
leaderElection:
  leaderElect: true
clientConnection:
  kubeconfig: /etc/kubernetes/scheduler.conf
profiles:
- schedulerName: default-scheduler
  plugins:
    multiPoint:
      enabled:
      - name: ResourceFungibility
        weight: 10 # make sure this plugin dominates the scheduling since GPU is scarce
```

### Set ClusterRole

Edit clusterRole `system:kube-scheduler` to make sure it has the privilege to get Models, mostly like:

```yaml
- apiGroups:
  - llmaz.io
  resources:
  - openmodels
  verbs:
  - get
```

## Build your own scheduler

If you want to import resourceFungibility plugin to build a customized scheduler, what you need to do is quite similar to [main.go](../../cmd/main.go).

## Limits

However, it only supports GPUs with the same number.

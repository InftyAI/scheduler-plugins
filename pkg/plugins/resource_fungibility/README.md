# ResourceFungibility Plugin

Support resource fungibility like GPU types with at most 8 alternative choices.

## Build your own scheduler

Run `IMG=<registry>/<repo>:<tag> make image-push` to build scheduler image.

## How to use

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

Run `chmod +r /etc/kubernetes/scheduler.conf`.

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

### Set kube-scheduler.yaml

Generally looks like:

```yaml
  containers:
  - commands:
    - --feature-gates=MaxUnavailableStatefulSet=true # this is require by lws
    - --config=/etc/kubernetes/kube-scheduler-config.yaml # set the kube-scheduler-config.yaml
    # - --v=6 # debugging only.
    image: inftyai/scheduler:<version> # set the right version of scheduler image
  - mountPath: /etc/kubernetes/kube-scheduler-config.yaml
      name: kube-scheduler-config
      readOnly: true
  volumes:
  - hostPath:
      path: /etc/kubernetes/kube-scheduler-config.yaml
      type: FileOrCreate
    name: kube-scheduler-config
```

## Limits

It only supports GPUs with the same number now.

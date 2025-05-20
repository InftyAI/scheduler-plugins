# Test Resource Fungibility Plugin

## Install Scheduler

Replace the default scheduler with the custom one, following [README.md](../README.md).

## Setup a test cluster with KWOK

- Install [Kind](https://github.com/kubernetes-sigs/kind).
- Install [Kwok](https://kwok.sigs.k8s.io/docs/user/kwok-in-cluster/) in a cluster.
- Run with `kubectl delete stage pod-complete` to make sure Pods are always consuming the GPUs.
- Run `init.sh` to setup the environment with different kinds of GPUs prepared.
- Run `kubectl apply -f workload.yaml` to create the workloads.

## Cleanup

Run `gc.sh` to clean up the environment.

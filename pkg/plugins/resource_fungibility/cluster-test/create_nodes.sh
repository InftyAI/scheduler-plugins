#!/bin/bash

# usage: ./create_nodes.sh kwok-node 0 10 "topology-key/supernode=s1,topology-key/rdma=r1" "topology-key/supernode=s1,topology-key/rdma=r1"

if [[ $# -lt 5 ]]; then
  echo "Usage: $0 <node-name-prefix> <node-start-count> <node-count> <node-labels> <node-annotations>"
  echo "Example: $0 kwok-node 0 3 \"type=kwok,kubernetes.io/role=agent\" \"kwok.x-k8s.io/node=fake\""
  exit 1
fi

NODE_PREFIX=$1
NODE_START_COUNT=$2
NODE_COUNT=$3
NODE_LABELS=$4          # Labels: key1=value1,key2=value2
NODE_ANNOTATIONS=$5     # Annotations: key1=value1,key2=value2

parse_key_value() {
  local input=$1
  local output=""
  IFS=',' read -ra pairs <<< "$input"
  for pair in "${pairs[@]}"; do
    IFS='=' read -r key value <<< "$pair"
    output+="    $key: $value\n"
  done
  echo -e "$output"
}

LABELS_YAML=$(parse_key_value "$NODE_LABELS")
ANNOTATIONS_YAML=$(parse_key_value "$NODE_ANNOTATIONS")

for ((i = $NODE_START_COUNT; i < $NODE_COUNT; i++)); do
  NODE_NAME="${NODE_PREFIX}-${i}"

  echo "Creating Node: $NODE_NAME"

  kubectl apply -f - <<EOF
apiVersion: v1
kind: Node
metadata:
  annotations:
    kwok.x-k8s.io/node: fake
$ANNOTATIONS_YAML
  labels:
$LABELS_YAML
    kubernetes.io/hostname: $NODE_NAME
  name: $NODE_NAME
spec:
#   taints: # Avoid scheduling actual running pods to fake Node
#   - effect: NoSchedule
#     key: kwok.x-k8s.io/node
#     value: fake
status:
  allocatable:
    cpu: 32
    memory: 256Gi
    pods: 110
    nvidia.com/gpu: 8
  capacity:
    cpu: 32
    memory: 256Gi
    pods: 110
    nvidia.com/gpu: 8
  nodeInfo:
    architecture: amd64
    bootID: ""
    containerRuntimeVersion: ""
    kernelVersion: ""
    kubeProxyVersion: fake
    kubeletVersion: fake
    machineID: ""
    operatingSystem: linux
    osImage: ""
    systemUUID: ""
  phase: Running
EOF
done

echo "All nodes created successfully!"

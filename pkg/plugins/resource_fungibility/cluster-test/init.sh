#!/bin/bash

SCRIPT_DIR=$(dirname "$0")
echo $SCRIPT_DIR

"$SCRIPT_DIR"/create_nodes.sh kwok-node 0 2 "karpenter.k8s.aws/instance-gpu-name=t4" ""
"$SCRIPT_DIR"/create_nodes.sh kwok-node 2 4 "karpenter.k8s.aws/instance-gpu-name=a20" ""
"$SCRIPT_DIR"/create_nodes.sh kwok-node 4 6 "karpenter.k8s.aws/instance-gpu-name=a100" ""
"$SCRIPT_DIR"/create_nodes.sh kwok-node 6 8 "karpenter.k8s.aws/instance-gpu-name=h100" ""
"$SCRIPT_DIR"/create_nodes.sh kwok-node 8 10 "karpenter.k8s.aws/instance-gpu-name=h800" ""

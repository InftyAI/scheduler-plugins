#!/bin/bash

echo "Finding nodes with annotation 'kwok.x-k8s.io/node: fake'..."
NODES=$(kubectl get nodes -o json | jq -r '.items[] | select(.metadata.annotations["kwok.x-k8s.io/node"] == "fake") | .metadata.name')

if [[ -z "$NODES" ]]; then
  echo "No nodes found with the annotation 'kwok.x-k8s.io/node: fake'."
  exit 0
fi

echo "Deleting nodes:"
for NODE in $NODES; do
  echo "Deleting node/$NODE..."
  kubectl delete node "$NODE"
done

echo "All matching nodes have been deleted."

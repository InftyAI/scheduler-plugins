# vScheduler

A Kubernetes scheduler designed for smart scheduling with llmaz.

## Plugins

vScheduler maintains multiple plugins for llm workloads scheduling.

### ResourceFungibility

A llama2-70B model can be run on 2xA100-80GB GPUs, can also be run on 4xA100-40GB GPUs, this is what we called fungibility.

With resourceFungibility plugin, we can simply achieve with at most 8 alternatives.

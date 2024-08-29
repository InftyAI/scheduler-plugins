# vScheduler

A Kubernetes scheduler designed for smart scheduling with llmaz.

## Plugins

vScheduler maintains multiple plugins for llm workloads scheduling.

### ResourceFungibility Plugin

A `llama2-7B` model can be run on __1xA100__ GPU, can also be run on __1xA10__ GPU, this is what we called fungibility.

With [resourceFungibility](./docs/plugins/resource_fungibility.md) plugin, we can simply achieve this with at most 8 alternative GPU types.

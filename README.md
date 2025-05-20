# Scheduler Plugins

Scheduler Plugins maintains multiple plugins used to differentiate the scheduling strategies for different workloads.

## Plugin List

### ResourceFungibility Plugin

A `llama2-7B` model can be running on __1xA100__ GPU, also on __1xA10__ GPU, even on __1x4090__ and a variety of other types of GPUs as well, that's what we called resource fungibility. In practical scenarios, we may have a heterogeneous cluster with different GPU types, and high-end GPUs will stock out a lot, to meet the SLOs of the service as well as the cost, we need to schedule the workloads on different GPU types.

With [resourceFungibility](./pkg/plugins/resource_fungibility/README.md) plugin, we can simply achieve this with at most 8 alternative GPU types.

In the future, we need to explore the GPU usage dynamically, not only for the availability and cost, but also the performance. See related paper about [Mélange: Cost Efficient Large Language Model
Serving by Exploiting GPU Heterogeneity](https://arxiv.org/pdf/2404.14527).

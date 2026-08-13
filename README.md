# Gardener Extension for Provider metal-stack

[![GitHub License](https://img.shields.io/github/license/metal-stack/gardener-extension-provider-metal)](https://github.com/metal-stack/gardener-extension-provider-metal/blob/master/LICENCE)
[![Build](https://github.com/metal-stack/gardener-extension-provider-metal/actions/workflows/build.yaml/badge.svg)](https://github.com/metal-stack/gardener-extension-provider-metal/actions/workflows/build.yaml)

[Project Gardener](https://gardener.cloud/) implements the automated management and operation of [Kubernetes](https://kubernetes.io/) clusters as a service. This controller implements [Gardener's extension contract](https://github.com/gardener/gardener/blob/master/docs/extensions/overview.md) for the **metal-stack** provider.

It reconciles the `Infrastructure`, `ControlPlane`, and `Worker` resources of `type: metal`, and additionally contains a validator for all metal-specific provider configs as well as mutating webhooks.

The `Worker` resource will also create a `FirewallDeployment` resource reconciled by the [firewall-controller-manager](https://github.com/metal-stack/firewall-controller-manager).

For the shoot `ControlPlane` the extension provider also deploys [MetalLB](https://metallb.io/) into the cluster, which gets dynamically configured by the [metal-ccm](https://github.com/metal-stack/metal-ccm).

## Example

An example `ControllerRegistration` resource that can be used to register this controller to Gardener can be found [here](example/controller-registration.yaml).

## Development

Development currently needs to happen against a real environment because there are many dependencies to external APIs for reconciliation. It is planned to allow development in the [mini-lab](https://github.com/metal-stack/mini-lab) soon.

## Feedback and Support

Feedback and contributions are always welcome! Please report bugs or suggestions as [GitHub issues](https://github.com/metal-stack/gardener-extension-provider-metal/issues) or reach out to our [community](https://metal-stack.io/community).

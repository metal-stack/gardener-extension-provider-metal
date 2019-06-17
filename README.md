# [Gardener Extension for AWS provider](https://gardener.cloud)

[![Go Report Card](https://goreportcard.com/badge/github.com/metal-pod/gardener-extension-provider-metal)](https://goreportcard.com/report/github.com/metal-pod/gardener-extension-provider-metal)

Project Gardener implements the automated management and operation of [Kubernetes](https://kubernetes.io/) clusters as a service. Its main principle is to leverage Kubernetes concepts for all of its tasks.

Recently, most of the vendor specific logic has been developed [in-tree](https://github.com/gardener/gardener). However, the project has grown to a size where it is very hard to extend, maintain, and test. With [GEP-1](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md) we have proposed how the architecture can be changed in a way to support external controllers that contain their very own vendor specifics. This way, we can keep Gardener core clean and independent.

This controller operates on the `Infrastructure` resource in the `extensions.gardener.cloud/v1alpha1` API group. It manages those objects that are requesting an AWS infrastructure (`.spec.type=aws`):

```yaml
---
apiVersion: extensions.gardener.cloud/v1alpha1
kind: Infrastructure
metadata:
  name: infrastructure
spec:
  type: aws
  region: eu-west-1
  secretRef:
    name: cloudprovider
    namespace: shoot--foo--bar
  providerConfig:
    apiVersion: aws.provider.extensions.gardener.cloud/v1alpha1
    kind: InfrastructureConfig
    networks:
      vpc: # specify either 'id' or 'cidr'
      # id: vpc-123456
        cidr: 10.250.0.0/16
      zones:
      - name: eu-west-1a
        internal: 10.250.112.0/22
        public: 10.250.96.0/22
        workers: 10.250.0.0/19
  sshPublicKey: ...

```

Please find [a concrete example](example/infrastructure.yaml) in the `example` folder.

After reconciliation the resulting data will be stored in the resource's `.status` field:

```yaml
...
status:
  ...
  providerStatus:
    apiVersion: aws.provider.extensions.gardener.cloud/v1alpha1
    kind: InfrastructureStatus
    ...
```

An example for a `ControllerRegistration` resource that can be used to register this controller to Gardener can be found [here](example/controller-registration.yaml).

Please find more information regarding the extensibility concepts and a detailed proposal [here](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md).

----

## How to start using or developing this extension controller locally

You can run the controller locally on your machine by executing `make start-provider-metal`.

Static code checks and tests can be executed by running `VERIFY=true make all`. We are using [dep](https://github.com/golang/dep) for Golang package dependency management and [Ginkgo](https://github.com/onsi/ginkgo)/[Gomega](https://github.com/onsi/gomega) for testing.

## Feedback and Support

Feedback and contributions are always welcome. Please report bugs or suggestions as [GitHub issues](https://github.com/gardener/gardener-extensions/issues) or join our [Slack channel #gardener](https://kubernetes.slack.com/messages/gardener) (please invite yourself to the Kubernetes workspace [here](http://slack.k8s.io)).

## Learn more!

Please find further resources about out project here:

* [Our landing page gardener.cloud](https://gardener.cloud/)
* ["Gardener, the Kubernetes Botanist" blog on kubernetes.io](https://kubernetes.io/blog/2018/05/17/gardener/)
* [GEP-1 (Gardener Enhancement Proposal) on extensibility](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md)

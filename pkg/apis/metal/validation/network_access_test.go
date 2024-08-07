package validation_test

import (
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"

	. "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("CloudProfileConfig validation", func() {
	Describe("#ValidateControlPlaneConfigNetworkAccess", func() {
		var (
			controlPlaneConfig *apismetal.ControlPlaneConfig
			cloudProfileConfig *apismetal.CloudProfileConfig
			partitionName      string
			path               *field.Path
		)

		BeforeEach(func() {
			partitionName = "partition-b"
			controlPlaneConfig = &apismetal.ControlPlaneConfig{
				NetworkAccessType: pointer.Pointer(apismetal.NetworkAccessBaseline),
			}
			cloudProfileConfig = &apismetal.CloudProfileConfig{}
			path = field.NewPath("test")
		})

		Describe("with network access type baseline", func() {
			It("should not pass with missing partition", func() {
				errorList := ValidateControlPlaneConfigNetworkAccess(controlPlaneConfig, cloudProfileConfig, partitionName, path)

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":     Equal(field.ErrorTypeInvalid),
						"Field":    Equal("test.metalControlPlanes"),
						"BadValue": HaveLen(0),
						"Detail":   Equal("missing partition with name \"partition-b\""),
					})),
				))
			})

			It("should pass for empty network isolation", func() {
				cloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
					"prod": {
						Partitions: map[string]apismetal.Partition{
							"partition-b": {},
						},
					},
				}

				errorList := ValidateControlPlaneConfigNetworkAccess(controlPlaneConfig, cloudProfileConfig, partitionName, path)

				Expect(errorList).To(BeEmpty())
			})
		})

		Describe("with network access type forbidden", func() {
			BeforeEach(func() {
				controlPlaneConfig.NetworkAccessType = pointer.Pointer(apismetal.NetworkAccessForbidden)
			})

			It("should fail without network isolation", func() {
				cloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
					"prod": {
						Partitions: map[string]apismetal.Partition{
							"partition-b": {},
						},
					},
				}

				errorList := ValidateControlPlaneConfigNetworkAccess(controlPlaneConfig, cloudProfileConfig, partitionName, path)

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":     Equal(field.ErrorTypeInvalid),
						"Field":    Equal("test.networkAccessType"),
						"BadValue": PointTo(Equal(apismetal.NetworkAccessForbidden)),
						"Detail":   Equal("network access type requires partition's networkIsolation to be set"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":     Equal(field.ErrorTypeRequired),
						"Field":    Equal("test.metalControlPlanes[prod].partitions[partition-b].networkIsolation"),
						"BadValue": Equal(""),
						"Detail":   Equal("network isolation required if control plane config networkAccess is not baseline"),
					})),
				))
			})

			It("should pass with valid network isolation", func() {
				cloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
					"prod": {
						Partitions: map[string]apismetal.Partition{
							"partition-b": {
								NetworkIsolation: &apismetal.NetworkIsolation{
									AllowedNetworks: apismetal.AllowedNetworks{
										Ingress: []string{"10.0.0.1/24"},
										Egress:  []string{"100.0.0.1/24"},
									},
									DNSServers: []string{"1.1.1.1", "1.0.0.1"},
									NTPServers: []string{"134.60.1.27", "134.60.111.110"},
									RegistryMirrors: []apismetal.RegistryMirror{
										{
											Name:     "metal-stack registry",
											Endpoint: "https://r.metal-stack.dev",
											IP:       "1.2.3.4",
											Port:     443,
											MirrorOf: []string{
												"ghcr.io",
												"quay.io",
											},
										},
									},
								},
							},
						},
					},
				}

				errorList := ValidateControlPlaneConfigNetworkAccess(controlPlaneConfig, cloudProfileConfig, partitionName, path)

				Expect(errorList).To(BeEmpty())
			})
		})

		Describe("with network access type restricted", func() {
			BeforeEach(func() {
				controlPlaneConfig.NetworkAccessType = pointer.Pointer(apismetal.NetworkAccessRestricted)
			})

			It("should fail without network isolation", func() {
				cloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
					"prod": {
						Partitions: map[string]apismetal.Partition{
							"partition-b": {},
						},
					},
				}

				errorList := ValidateControlPlaneConfigNetworkAccess(controlPlaneConfig, cloudProfileConfig, partitionName, path)

				Expect(errorList).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":     Equal(field.ErrorTypeInvalid),
						"Field":    Equal("test.networkAccessType"),
						"BadValue": PointTo(Equal(apismetal.NetworkAccessRestricted)),
						"Detail":   Equal("network access type requires partition's networkIsolation to be set"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":     Equal(field.ErrorTypeRequired),
						"Field":    Equal("test.metalControlPlanes[prod].partitions[partition-b].networkIsolation"),
						"BadValue": Equal(""),
						"Detail":   Equal("network isolation required if control plane config networkAccess is not baseline"),
					})),
				))
			})

			It("should pass with valid network isolation", func() {
				cloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
					"prod": {
						Partitions: map[string]apismetal.Partition{
							"partition-b": {
								NetworkIsolation: &apismetal.NetworkIsolation{
									AllowedNetworks: apismetal.AllowedNetworks{
										Ingress: []string{"10.0.0.1/24"},
										Egress:  []string{"100.0.0.1/24"},
									},
									DNSServers: []string{"1.1.1.1", "1.0.0.1"},
									NTPServers: []string{"134.60.1.27", "134.60.111.110"},
									RegistryMirrors: []apismetal.RegistryMirror{
										{
											Name:     "metal-stack registry",
											Endpoint: "https://r.metal-stack.dev",
											IP:       "1.2.3.4",
											Port:     443,
											MirrorOf: []string{
												"ghcr.io",
												"quay.io",
											},
										},
									},
								},
							},
						},
					},
				}

				errorList := ValidateControlPlaneConfigNetworkAccess(controlPlaneConfig, cloudProfileConfig, partitionName, path)

				Expect(errorList).To(BeEmpty())
			})
		})
	})
})

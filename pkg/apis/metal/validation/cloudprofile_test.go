package validation_test

import (
	"github.com/gardener/gardener/pkg/apis/core"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"

	. "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("CloudProfileConfig validation", func() {
	Describe("#ValidateCloudProfileConfig", func() {
		var (
			cloudProfile       *core.CloudProfile
			cloudProfileConfig *apismetal.CloudProfileConfig
			path               *field.Path
		)

		BeforeEach(func() {
			cloudProfile = &core.CloudProfile{
				Spec: core.CloudProfileSpec{
					Regions: []core.Region{
						{
							Name: "region-a",
							Zones: []core.AvailabilityZone{
								{
									Name: "partition-a",
								},
								{
									Name: "partition-b",
								},
							},
						},
						{
							Name: "region-b",
							Zones: []core.AvailabilityZone{
								{
									Name: "partition-c",
								},
							},
						},
					},
				},
			}

			cloudProfileConfig = &apismetal.CloudProfileConfig{}
			path = field.NewPath("test")
		})

		It("should pass empty configuration", func() {
			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile, path)

			Expect(errorList).To(BeEmpty())
		})

		It("should pass properly configured control plane partitions", func() {
			cloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
				"prod": {
					Partitions: map[string]apismetal.Partition{
						"partition-b": {},
					},
				},
			}

			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile, path)

			Expect(errorList).To(BeEmpty())
		})

		It("should prevent declaring partitions that are not configured in zones", func() {
			cloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
				"prod": {
					Partitions: map[string]apismetal.Partition{
						"random-partition": {},
					},
				},
			}

			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile, path)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":     Equal(field.ErrorTypeInvalid),
				"Field":    Equal("test.metalControlPlanes.prod"),
				"BadValue": Equal("random-partition"),
				"Detail":   Equal("the control plane has a partition that is not a configured zone in any of the cloud profile regions: [partition-a partition-b partition-c]"),
			}))))
		})

		It("should pass properly configured control plane partitions with network isolation", func() {
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
										Endpoint: "https://some.registry",
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

			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile, path)

			Expect(errorList).To(BeEmpty())
		})

		It("should allow up to 3 dns servers", func() {
			cloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
				"prod": {
					Partitions: map[string]apismetal.Partition{
						"partition-b": {
							NetworkIsolation: &apismetal.NetworkIsolation{
								AllowedNetworks: apismetal.AllowedNetworks{
									Ingress: []string{"10.0.0.1/24"},
									Egress:  []string{"100.0.0.1/24"},
								},
								DNSServers: []string{"1.1.1.1", "1.0.0.1", "8.8.8.8"},
								NTPServers: []string{"134.60.1.27", "134.60.111.110"},
								RegistryMirrors: []apismetal.RegistryMirror{
									{
										Name:     "metal-stack registry",
										Endpoint: "https://some.registry",
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

			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile, path)

			Expect(errorList).To(BeEmpty())
		})

		It("should prevent more than 3 dns servers", func() {
			cloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
				"prod": {
					Partitions: map[string]apismetal.Partition{
						"partition-b": {
							NetworkIsolation: &apismetal.NetworkIsolation{
								AllowedNetworks: apismetal.AllowedNetworks{
									Ingress: []string{"10.0.0.1/24"},
									Egress:  []string{"100.0.0.1/24"},
								},
								DNSServers: []string{"1.1.1.1", "1.0.0.1", "8.8.8.8", "8.8.4.4"},
								NTPServers: []string{"134.60.1.27", "134.60.111.110"},
								RegistryMirrors: []apismetal.RegistryMirror{
									{
										Name:     "metal-stack registry",
										Endpoint: "https://some.registry",
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

			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile, path)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.dnsServers"),
					"BadValue": Equal([]string{"1.1.1.1", "1.0.0.1", "8.8.8.8", "8.8.4.4"}),
					"Detail":   Equal("only up to 3 dns servers are allowed"),
				})),
			))
		})

		It("should prevent partitions with empty network isolation registry mirror", func() {
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
										Name:     "",
										Endpoint: "",
										IP:       "",
										Port:     0,
										MirrorOf: []string{},
									},
								},
							},
						},
					},
				},
			}

			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile, path)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.registryMirrors[0].name"),
					"BadValue": Equal(""),
					"Detail":   Equal("name of mirror may not be empty"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.registryMirrors[0].endpoint"),
					"BadValue": Equal(""),
					"Detail":   Equal("url must have the scheme http/s"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.registryMirrors[0].ip"),
					"BadValue": Equal(""),
					"Detail":   Equal("invalid ip address"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.registryMirrors[0].port"),
					"BadValue": Equal(int32(0)),
					"Detail":   Equal("must be a valid port"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.registryMirrors[0].mirrorOf"),
					"BadValue": HaveLen(0),
					"Detail":   Equal("registry mirror must replace existing registries"),
				})),
			))
		})

		It("should prevent partitions with invalid network isolation", func() {
			cloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
				"prod": {
					Partitions: map[string]apismetal.Partition{
						"partition-b": {
							NetworkIsolation: &apismetal.NetworkIsolation{
								AllowedNetworks: apismetal.AllowedNetworks{
									Ingress: []string{"10.0.0.1"},
									Egress:  []string{"100.0.0.1/128"},
								},
								DNSServers: []string{"1.1.1"},
								NTPServers: []string{"134.60.1.272"},
								RegistryMirrors: []apismetal.RegistryMirror{
									{
										Name:     "metal-stack registry",
										Endpoint: "file:///invalid",
										IP:       "1.2.3.4.5",
										Port:     443,
										MirrorOf: []string{
											"https://ghcr.io",
										},
									},
								},
							},
						},
					},
				},
			}

			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile, path)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.dnsServers[0]"),
					"BadValue": Equal("1.1.1"),
					"Detail":   Equal("invalid ip address"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.ntpServers[0]"),
					"BadValue": Equal("134.60.1.272"),
					"Detail":   Equal("invalid ip address"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.allowedNetworks.egress[0]"),
					"BadValue": Equal("100.0.0.1/128"),
					"Detail":   Equal("invalid cidr"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.allowedNetworks.ingress[0]"),
					"BadValue": Equal("10.0.0.1"),
					"Detail":   Equal("invalid cidr"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.registryMirrors[0].endpoint"),
					"BadValue": Equal("file:///invalid"),
					"Detail":   Equal("url must have the scheme http/s"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.registryMirrors[0].ip"),
					"BadValue": Equal("1.2.3.4.5"),
					"Detail":   Equal("invalid ip address"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeInvalid),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.registryMirrors[0].mirrorOf[0]"),
					"BadValue": Equal("https://ghcr.io"),
					"Detail":   Equal("not a valid registry host"),
				})),
			))
		})
	})

	Describe("#ValidateImmutableCloudProfileConfig", func() {
		var (
			newCloudProfileConfig *apismetal.CloudProfileConfig
			oldCloudProfileConfig *apismetal.CloudProfileConfig
			path                  *field.Path
		)

		BeforeEach(func() {
			newCloudProfileConfig = &apismetal.CloudProfileConfig{}
			oldCloudProfileConfig = &apismetal.CloudProfileConfig{}
			path = field.NewPath("test")
		})

		It("should pass empty configuration", func() {
			errorList := ValidateImmutableCloudProfileConfig(newCloudProfileConfig, oldCloudProfileConfig, path)

			Expect(errorList).To(BeEmpty())
		})

		It("should pass when not existing previously", func() {
			newCloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
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
										Endpoint: "https://some.registry",
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
			oldCloudProfileConfig = nil

			errorList := ValidateImmutableCloudProfileConfig(newCloudProfileConfig, oldCloudProfileConfig, path)

			Expect(errorList).To(BeEmpty())
		})

		It("should pass when changing anything except dns", func() {
			newCloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
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
										Endpoint: "https://some.registry",
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
			oldCloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
				"prod": {
					Partitions: map[string]apismetal.Partition{
						"partition-b": {
							NetworkIsolation: &apismetal.NetworkIsolation{
								AllowedNetworks: apismetal.AllowedNetworks{
									Ingress: []string{"192.0.0.1/24"},
									Egress:  []string{"192.0.0.1/24"},
								},
								DNSServers: []string{"1.1.1.1", "1.0.0.1"},
								NTPServers: []string{"134.0.0.1"},
								RegistryMirrors: []apismetal.RegistryMirror{
									{
										Name:     "metal-stack registry2",
										Endpoint: "https://some.other.registry",
										IP:       "1.2.3.5",
										Port:     8443,
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

			errorList := ValidateImmutableCloudProfileConfig(newCloudProfileConfig, oldCloudProfileConfig, path)

			Expect(errorList).To(BeEmpty())
		})

		It("should fail when changing dns", func() {
			newCloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
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
										Endpoint: "https://some.registry",
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
			oldCloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
				"prod": {
					Partitions: map[string]apismetal.Partition{
						"partition-b": {
							NetworkIsolation: &apismetal.NetworkIsolation{
								AllowedNetworks: apismetal.AllowedNetworks{
									Ingress: []string{"10.0.0.1/24"},
									Egress:  []string{"100.0.0.1/24"},
								},
								DNSServers: []string{"8.8.8.8", "8.8.4.4"},
								NTPServers: []string{"134.60.1.27", "134.60.111.110"},
								RegistryMirrors: []apismetal.RegistryMirror{
									{
										Name:     "metal-stack registry",
										Endpoint: "https://some.registry",
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

			errorList := ValidateImmutableCloudProfileConfig(newCloudProfileConfig, oldCloudProfileConfig, path)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeNotSupported),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.dnsServers[0]"),
					"BadValue": Equal("1.1.1.1"),
					"Detail":   Equal("supported values: \"8.8.8.8\""),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":     Equal(field.ErrorTypeNotSupported),
					"Field":    Equal("test.metalControlPlanes.prod.partition-b.networkIsolation.dnsServers[1]"),
					"BadValue": Equal("1.0.0.1"),
					"Detail":   Equal("supported values: \"8.8.4.4\""),
				})),
			))
		})
	})
})

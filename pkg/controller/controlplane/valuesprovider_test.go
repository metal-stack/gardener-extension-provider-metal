package controlplane

import (
	"context"
	"encoding/json"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"

	"github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	namespace = "test"
)

var _ = Describe("ValuesProvider", func() {
	var (
		ctrl *gomock.Controller

		// Build scheme
		scheme = runtime.NewScheme()
		_      = apismetal.AddToScheme(scheme)

		cp = &extensionsv1alpha1.ControlPlane{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "control-plane",
				Namespace: namespace,
			},
			Spec: extensionsv1alpha1.ControlPlaneSpec{
				ProviderConfig: &runtime.RawExtension{
					Raw: encode(&apismetal.ControlPlaneConfig{
						CloudControllerManager: &apismetal.CloudControllerManagerConfig{
							KubernetesConfig: gardenv1beta1.KubernetesConfig{
								FeatureGates: map[string]bool{
									"CustomResourceValidation": true,
								},
							},
						},
					}),
				},
				InfrastructureProviderStatus: &runtime.RawExtension{
					Raw: encode(&apismetal.InfrastructureStatus{
						VPC: apismetal.VPCStatus{
							ID: "vpc-1234",
							Subnets: []apismetal.Subnet{
								{
									ID:      "subnet-acbd1234",
									Purpose: "public",
									Zone:    "eu-west-1a",
								},
							},
						},
					}),
				},
			},
		}

		cidr    = gardencorev1alpha1.CIDR("10.250.0.0/19")
		cluster = &extensionscontroller.Cluster{
			Shoot: &gardenv1beta1.Shoot{
				Spec: gardenv1beta1.ShootSpec{
					Cloud: gardenv1beta1.Cloud{
						metal: &gardenv1beta1.metalCloud{
							Networks: gardenv1beta1.metalNetworks{
								K8SNetworks: gardencorev1alpha1.K8SNetworks{
									Pods: &cidr,
								},
							},
						},
					},
					Kubernetes: gardenv1beta1.Kubernetes{
						Version: "1.13.4",
					},
				},
			},
		}

		checksums = map[string]string{
			gardencorev1alpha1.SecretNameCloudProvider: "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
			metal.CloudProviderConfigName:              "08a7bc7fe8f59b055f173145e211760a83f02cf89635cef26ebb351378635606",
			"cloud-controller-manager":                 "3d791b164a808638da9a8df03924be2a41e34cd664e42231c00fe369e3588272",
			"cloud-controller-manager-server":          "6dff2a2e6f14444b66d8e4a351c049f7e89ee24ba3eaab95dbec40ba6bdebb52",
		}

		configChartValues = map[string]interface{}{
			"vpcID":       "vpc-1234",
			"subnetID":    "subnet-acbd1234",
			"clusterName": namespace,
			"zone":        "eu-west-1a",
		}

		ccmChartValues = map[string]interface{}{
			"replicas":          1,
			"clusterName":       namespace,
			"kubernetesVersion": "1.13.4",
			"podNetwork":        cidr,
			"podAnnotations": map[string]interface{}{
				"checksum/secret-cloud-controller-manager":        "3d791b164a808638da9a8df03924be2a41e34cd664e42231c00fe369e3588272",
				"checksum/secret-cloud-controller-manager-server": "6dff2a2e6f14444b66d8e4a351c049f7e89ee24ba3eaab95dbec40ba6bdebb52",
				"checksum/secret-cloudprovider":                   "8bafb35ff1ac60275d62e1cbd495aceb511fb354f74a20f7d06ecb48b3a68432",
				"checksum/configmap-cloud-provider-config":        "08a7bc7fe8f59b055f173145e211760a83f02cf89635cef26ebb351378635606",
			},
			"featureGates": map[string]bool{
				"CustomResourceValidation": true,
			},
		}

		logger = log.Log.WithName("test")
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#GetConfigChartValues", func() {
		It("should return correct config chart values", func() {
			// Create valuesProvider
			vp := NewValuesProvider(logger)
			err := vp.(inject.Scheme).InjectScheme(scheme)
			Expect(err).NotTo(HaveOccurred())

			// Call GetConfigChartValues method and check the result
			values, err := vp.GetConfigChartValues(context.TODO(), cp, cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(configChartValues))
		})
	})

	Describe("#GetControlPlaneChartValues", func() {
		It("should return correct control plane chart values", func() {
			// Create valuesProvider
			vp := NewValuesProvider(logger)
			err := vp.(inject.Scheme).InjectScheme(scheme)
			Expect(err).NotTo(HaveOccurred())

			// Call GetControlPlaneChartValues method and check the result
			values, err := vp.GetControlPlaneChartValues(context.TODO(), cp, cluster, checksums, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(Equal(ccmChartValues))
		})
	})
})

func encode(obj runtime.Object) []byte {
	data, _ := json.Marshal(obj)
	return data
}

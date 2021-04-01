package e2e

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"math/rand"
	"os"
	"strings"

	awscloud "github.com/kubernetes-sigs/aws-ebs-csi-driver/pkg/cloud"
	ebscsidriver "github.com/kubernetes-sigs/aws-ebs-csi-driver/pkg/driver"
	"github.com/kubernetes-sigs/aws-ebs-csi-driver/tests/e2e/driver"
	"github.com/kubernetes-sigs/aws-ebs-csi-driver/tests/e2e/testsuites"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = Describe("[Performance] Dynamic Provisioning", func() {
	f := framework.NewDefaultFramework("ebs")

	var (
		cs          clientset.Interface
		ns          *v1.Namespace
		ebsDriver   driver.PVTestDriver
		volumeTypes = awscloud.ValidVolumeTypes
		fsTypes     = []string{ebscsidriver.FSTypeXfs}
	)

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
		ebsDriver = driver.InitEbsCSIDriver()
	})

	for _, t := range volumeTypes {
		for _, fs := range fsTypes {
			volumeType := t
			fsType := fs
			Measure(fmt.Sprintf("should create a volume on demand with volume type %q and fs type %q", volumeType, fsType), func(b Benchmarker) {
				pods := []testsuites.PodDetails{
					{
						Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
						Volumes: []testsuites.VolumeDetails{
							{
								VolumeType: volumeType,
								FSType:     fsType,
								ClaimSize:  driver.MinimumSizeForVolumeType(volumeType),
								VolumeMount: testsuites.VolumeMountDetails{
									NameGenerate:      "test-volume-",
									MountPathGenerate: "/mnt/test-",
								},
							},
						},
					},
				}
				test := testsuites.DynamicallyProvisionedCmdVolumeTest{
					CSIDriver: ebsDriver,
					Pods:      pods,
				}
				runtime := b.Time("runtime", func() {
					test.Run(cs, ns)
				})
				print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
			}, 3)
		}
	}

	for _, t := range volumeTypes {
		volumeType := t
		Measure(fmt.Sprintf("should create a volume on demand with volumeType %q and encryption", volumeType), func(b Benchmarker) {
			pods := []testsuites.PodDetails{
				{
					Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
					Volumes: []testsuites.VolumeDetails{
						{
							VolumeType: volumeType,
							FSType:     ebscsidriver.FSTypeExt4,
							Encrypted:  true,
							ClaimSize:  driver.MinimumSizeForVolumeType(volumeType),
							VolumeMount: testsuites.VolumeMountDetails{
								NameGenerate:      "test-volume-",
								MountPathGenerate: "/mnt/test-",
							},
						},
					},
				},
			}
			test := testsuites.DynamicallyProvisionedCmdVolumeTest{
				CSIDriver: ebsDriver,
				Pods:      pods,
			}
			runtime := b.Time("runtime", func() {
				test.Run(cs, ns)
			})
			print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
		}, 3)
	}

	Measure("should create a volume on demand with provided mountOptions", func(b Benchmarker) {
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						VolumeType:   awscloud.VolumeTypeGP2,
						FSType:       ebscsidriver.FSTypeExt4,
						MountOptions: []string{"rw"},
						ClaimSize:    driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: ebsDriver,
			Pods:      pods,
		}
		runtime := b.Time("runtime", func() {
			test.Run(cs, ns)
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 3)
	/*
		Measure("should create multiple PV objects, bind to PVCs and attach all to a single pod", func(b Benchmarker) {
			pods := []testsuites.PodDetails{
				{
					Cmd: "echo 'hello world' > /mnt/test-1/data && echo 'hello world' > /mnt/test-2/data && grep 'hello world' /mnt/test-1/data  && grep 'hello world' /mnt/test-2/data",
					Volumes: []testsuites.VolumeDetails{
						{
							VolumeType: awscloud.VolumeTypeGP2,
							FSType:     ebscsidriver.FSTypeExt3,
							ClaimSize:  driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
							VolumeMount: testsuites.VolumeMountDetails{
								NameGenerate:      "test-volume-",
								MountPathGenerate: "/mnt/test-",
							},
						},
						{
							VolumeType: awscloud.VolumeTypeIO1,
							FSType:     ebscsidriver.FSTypeExt4,
							ClaimSize:  driver.MinimumSizeForVolumeType(awscloud.VolumeTypeIO1),
							VolumeMount: testsuites.VolumeMountDetails{
								NameGenerate:      "test-volume-",
								MountPathGenerate: "/mnt/test-",
							},
						},
					},
				},
			}
			test := testsuites.DynamicallyProvisionedCmdVolumeTest{
				CSIDriver: ebsDriver,
				Pods:      pods,
			}
			runtime := b.Time("runtime", func() {
				test.Run(cs, ns)
			})
			print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
		},10)*/

	Measure("should create multiple PV objects, bind to PVCs and attach all to different pods", func(b Benchmarker) {
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						VolumeType: awscloud.VolumeTypeGP2,
						FSType:     ebscsidriver.FSTypeExt3,
						ClaimSize:  driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						VolumeType: awscloud.VolumeTypeIO1,
						FSType:     ebscsidriver.FSTypeExt4,
						ClaimSize:  driver.MinimumSizeForVolumeType(awscloud.VolumeTypeIO1),
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: ebsDriver,
			Pods:      pods,
		}
		runtime := b.Time("runtime", func() {
			test.Run(cs, ns)
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 3)

	Measure("should create a raw block volume on demand", func(b Benchmarker) {
		pods := []testsuites.PodDetails{
			{
				Cmd: "dd if=/dev/zero of=/dev/xvda bs=1024k count=100",
				Volumes: []testsuites.VolumeDetails{
					{
						VolumeType: awscloud.VolumeTypeGP2,
						FSType:     ebscsidriver.FSTypeExt4,
						ClaimSize:  driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
						VolumeMode: testsuites.Block,
						VolumeDevice: testsuites.VolumeDeviceDetails{
							NameGenerate: "test-block-volume-",
							DevicePath:   "/dev/xvda",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: ebsDriver,
			Pods:      pods,
		}
		runtime := b.Time("runtime", func() {
			test.Run(cs, ns)
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 3)

	Measure("should create multiple PV objects, bind to PVCs and attach all to different pods on the same node", func(b Benchmarker) {
		pods := []testsuites.PodDetails{
			{
				Cmd: "while true; do echo $(date -u) >> /mnt/test-1/data; sleep 1; done",
				Volumes: []testsuites.VolumeDetails{
					{
						VolumeType: awscloud.VolumeTypeGP2,
						FSType:     ebscsidriver.FSTypeExt3,
						ClaimSize:  driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
			{
				Cmd: "while true; do echo $(date -u) >> /mnt/test-1/data; sleep 1; done",
				Volumes: []testsuites.VolumeDetails{
					{
						VolumeType: awscloud.VolumeTypeIO1,
						FSType:     ebscsidriver.FSTypeExt4,
						ClaimSize:  driver.MinimumSizeForVolumeType(awscloud.VolumeTypeIO1),
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedCollocatedPodTest{
			CSIDriver:    ebsDriver,
			Pods:         pods,
			ColocatePods: true,
		}
		runtime := b.Time("runtime", func() {
			test.Run(cs, ns)
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 3)

	// Track issue https://github.com/kubernetes/kubernetes/issues/70505
	Measure("should create a volume on demand and mount it as readOnly in a pod", func(b Benchmarker) {
		pods := []testsuites.PodDetails{
			{
				Cmd: "touch /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						VolumeType: awscloud.VolumeTypeGP2,
						FSType:     ebscsidriver.FSTypeExt4,
						ClaimSize:  driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
							ReadOnly:          true,
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedReadOnlyVolumeTest{
			CSIDriver: ebsDriver,
			Pods:      pods,
		}
		runtime := b.Time("runtime", func() {
			test.Run(cs, ns)
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 10)

	Measure(fmt.Sprintf("should delete PV with reclaimPolicy %q", v1.PersistentVolumeReclaimDelete), func(b Benchmarker) {
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		volumes := []testsuites.VolumeDetails{
			{
				VolumeType:    awscloud.VolumeTypeGP2,
				FSType:        ebscsidriver.FSTypeExt4,
				ClaimSize:     driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
				ReclaimPolicy: &reclaimPolicy,
			},
		}
		test := testsuites.DynamicallyProvisionedReclaimPolicyTest{
			CSIDriver: ebsDriver,
			Volumes:   volumes,
		}
		runtime := b.Time("runtime", func() {
			test.Run(cs, ns)
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 3)

	Measure(fmt.Sprintf("[env] should retain PV with reclaimPolicy %q", v1.PersistentVolumeReclaimRetain), func(b Benchmarker) {
		if os.Getenv(awsAvailabilityZonesEnv) == "" {
			Skip(fmt.Sprintf("env %q not set", awsAvailabilityZonesEnv))
		}
		reclaimPolicy := v1.PersistentVolumeReclaimRetain
		volumes := []testsuites.VolumeDetails{
			{
				VolumeType:    awscloud.VolumeTypeGP2,
				FSType:        ebscsidriver.FSTypeExt4,
				ClaimSize:     driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
				ReclaimPolicy: &reclaimPolicy,
			},
		}
		availabilityZones := strings.Split(os.Getenv(awsAvailabilityZonesEnv), ",")
		availabilityZone := availabilityZones[rand.Intn(len(availabilityZones))]
		region := availabilityZone[0 : len(availabilityZone)-1]
		cloud, err := awscloud.NewCloud(region)
		if err != nil {
			Fail(fmt.Sprintf("could not get NewCloud: %v", err))
		}

		test := testsuites.DynamicallyProvisionedReclaimPolicyTest{
			CSIDriver: ebsDriver,
			Volumes:   volumes,
			Cloud:     cloud,
		}
		runtime := b.Time("runtime", func() {
			test.Run(cs, ns)
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 3)

	Measure("should create a deployment object, write and read to it, delete the pod and write and read to it again", func(b Benchmarker) {
		pod := testsuites.PodDetails{
			Cmd: "echo 'hello world' >> /mnt/test-1/data && while true; do sleep 1; done",
			Volumes: []testsuites.VolumeDetails{
				{
					VolumeType: awscloud.VolumeTypeGP2,
					FSType:     ebscsidriver.FSTypeExt3,
					ClaimSize:  driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedDeletePodTest{
			CSIDriver: ebsDriver,
			Pod:       pod,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:            []string{"cat", "/mnt/test-1/data"},
				ExpectedString: "hello world\nhello world\n", // pod will be restarted so expect to see 2 instances of string
			},
		}
		runtime := b.Time("runtime", func() {
			test.Run(cs, ns)
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 3)

	Measure("should create a volume on demand and resize it ", func(b Benchmarker) {
		allowVolumeExpansion := true
		pod := testsuites.PodDetails{
			Cmd: "echo 'hello world' >> /mnt/test-1/data && grep 'hello world' /mnt/test-1/data && sync",
			Volumes: []testsuites.VolumeDetails{
				{
					VolumeType: awscloud.VolumeTypeGP2,
					FSType:     ebscsidriver.FSTypeExt4,
					ClaimSize:  driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
					AllowVolumeExpansion: &allowVolumeExpansion,
				},
			},
		}
		test := testsuites.DynamicallyProvisionedResizeVolumeTest{
			CSIDriver: ebsDriver,
			Pod:       pod,
		}
		runtime := b.Time("runtime", func() {
			test.Run(cs, ns)
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 3)
})

/*var _ = Describe("[Performance] Snapshot", func() {
	f := framework.NewDefaultFramework("ebs")

	var (
		cs          clientset.Interface
		snapshotrcs restclientset.Interface
		ns          *v1.Namespace
		ebsDriver   driver.PVTestDriver
	)

	BeforeEach(func() {
		cs = f.ClientSet
		var err error
		snapshotrcs, err = restClient(testsuites.SnapshotAPIGroup, testsuites.APIVersionv1beta1)
		if err != nil {
			Fail(fmt.Sprintf("could not get rest clientset: %v", err))
		}
		ns = f.Namespace
		ebsDriver = driver.InitEbsCSIDriver()
	})

	Measure("should create a pod, write and read to it, take a volume snapshot, and create another pod from the snapshot", func(b Benchmarker) {
		pod := testsuites.PodDetails{
			// sync before taking a snapshot so that any cached data is written to the EBS volume
			Cmd: "echo 'hello world' >> /mnt/test-1/data && grep 'hello world' /mnt/test-1/data && sync",
			Volumes: []testsuites.VolumeDetails{
				{
					VolumeType: awscloud.VolumeTypeGP2,
					FSType:     ebscsidriver.FSTypeExt4,
					ClaimSize:  driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}
		restoredPod := testsuites.PodDetails{
			Cmd: "grep 'hello world' /mnt/test-1/data",
			Volumes: []testsuites.VolumeDetails{
				{
					VolumeType: awscloud.VolumeTypeGP2,
					FSType:     ebscsidriver.FSTypeExt4,
					ClaimSize:  driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedVolumeSnapshotTest{
			CSIDriver:   ebsDriver,
			Pod:         pod,
			RestoredPod: restoredPod,
		}
		runtime := b.Time("runtime", func() {
			test.Run(cs, snapshotrcs, ns)
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 10)
})*/

var _ = Describe("[Performance] Dynamic Provisioning", func() {
	f := framework.NewDefaultFramework("ebs")

	var (
		cs        clientset.Interface
		ns        *v1.Namespace
		ebsDriver driver.DynamicPVTestDriver
	)

	BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
		ebsDriver = driver.InitEbsCSIDriver()
	})

	Measure("should allow for topology aware volume scheduling", func(b Benchmarker) {
		volumeBindingMode := storagev1.VolumeBindingWaitForFirstConsumer
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						VolumeType:        awscloud.VolumeTypeGP2,
						FSType:            ebscsidriver.FSTypeExt4,
						ClaimSize:         driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
						VolumeBindingMode: &volumeBindingMode,
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedTopologyAwareVolumeTest{
			CSIDriver: ebsDriver,
			Pods:      pods,
		}
		runtime := b.Time("runtime", func() {
			test.Run(cs, ns)
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 3)

	// Requires env AWS_AVAILABILITY_ZONES, a comma separated list of AZs
	Measure("[env] should allow for topology aware volume with specified zone in allowedTopologies", func(b Benchmarker) {
		if os.Getenv(awsAvailabilityZonesEnv) == "" {
			Skip(fmt.Sprintf("env %q not set", awsAvailabilityZonesEnv))
		}
		allowedTopologyZones := strings.Split(os.Getenv(awsAvailabilityZonesEnv), ",")
		volumeBindingMode := storagev1.VolumeBindingWaitForFirstConsumer
		pods := []testsuites.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: []testsuites.VolumeDetails{
					{
						VolumeType:            awscloud.VolumeTypeGP2,
						FSType:                ebscsidriver.FSTypeExt4,
						ClaimSize:             driver.MinimumSizeForVolumeType(awscloud.VolumeTypeGP2),
						VolumeBindingMode:     &volumeBindingMode,
						AllowedTopologyValues: allowedTopologyZones,
						VolumeMount: testsuites.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
					},
				},
			},
		}
		test := testsuites.DynamicallyProvisionedTopologyAwareVolumeTest{
			CSIDriver: ebsDriver,
			Pods:      pods,
		}
		runtime := b.Time("runtime", func() {
			test.Run(cs, ns)
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 3)
})

func createPVCOnDemand() {
	// uses the current context in kubeconfig
	// path-to-kubeconfig -- for example, /root/.kube/config
	config, _ := clientcmd.BuildConfigFromFlags("", "$HOME/.kube/config")
	// creates the clientset
	clientset, _ := kubernetes.NewForConfig(config)
	// access the API to list pods
	scName := "ebs-sc"
	volumeMode := v1.PersistentVolumeFilesystem
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "pvc-123",
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: &scName,
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse("1Gi"),
				},
			},
			VolumeMode: &volumeMode,
		},
	}
	clientset.CoreV1().PersistentVolumeClaims("default").Create(pvc)
}

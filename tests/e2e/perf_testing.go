package e2e

import (
	"fmt"
	"github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	snapshotclientset "github.com/kubernetes-csi/external-snapshotter/v2/pkg/client/clientset/versioned"
	awscloud "github.com/kubernetes-sigs/aws-ebs-csi-driver/pkg/cloud"
	ebscsidriver "github.com/kubernetes-sigs/aws-ebs-csi-driver/pkg/driver"
	"github.com/kubernetes-sigs/aws-ebs-csi-driver/pkg/util"
	"github.com/kubernetes-sigs/aws-ebs-csi-driver/tests/e2e/driver"
	"github.com/kubernetes-sigs/aws-ebs-csi-driver/tests/e2e/testsuites"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	restclientset "k8s.io/client-go/rest"
	"k8s.io/kubernetes/test/e2e/framework"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
	e2epv "k8s.io/kubernetes/test/e2e/framework/pv"
	"time"
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

	/*Measure("should create a volume on demand with provided mountOptions", func(b Benchmarker) {
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
	}, 3)*/
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

	/*Measure("should create multiple PV objects, bind to PVCs and attach all to different pods", func(b Benchmarker) {
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
	}, 3)*/
})

/*
var _ = Describe("[Performance] Snapshot", func() {
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

/*var _ = Describe("[Performance] Dynamic Provisioning", func() {
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
})*/

var _ = Describe("[PerformanceISO] Dynamic Provisioning", func() {
	f := framework.NewDefaultFramework("ebs")

	var (
		cs clientset.Interface
		//ns          *v1.Namespace
	)

	BeforeEach(func() {
		cs = f.ClientSet
		//ns = f.Namespace
	})

	/*Measure("should iso create volume", func(b Benchmarker) {
	// uses the current context in kubeconfig
	// path-to-kubeconfig -- for example, /root/.kube/config

	// creates the clientset
	runtime := b.Time("runtime", func() {
		createPVCOnDemand(cs, "default")
	})
	print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))	}, 1)*/

	/*Measure("should iso modify volume", func(b Benchmarker) {
		// uses the current context in kubeconfig
		// path-to-kubeconfig -- for example, /root/.kube/config

		// creates the clientset
		runtime := b.Time("runtime", func() {
			UpdatePVCOnDemand(cs, "default")
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 1)*/
	Measure("should iso delete volume", func(b Benchmarker) {
		// uses the current context in kubeconfig
		// path-to-kubeconfig -- for example, /root/.kube/config

		// creates the clientset
		runtime := b.Time("runtime", func() {
			DeletePVCOnDemand(cs, "default")
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 1)

})

func UpdatePVCOnDemand(clientsete2e clientset.Interface, ns string) {

	for i:=0;i<100;i++ {
		pvcName := fmt.Sprintf("pvc-%d", i)
		//scName := "ebs-cs-iso"
		//volumeMode := v1.PersistentVolumeFilesystem
		pvc, err := clientsete2e.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{})
		if err != nil {
			framework.ExpectNoError(err, fmt.Sprintf("fail to get pvc(%s): %v", pvcName, err))
		}
		originalSize := pvc.Spec.Resources.Requests["storage"]
		delta := resource.Quantity{}
		delta.Set(util.GiBToBytes(1))
		originalSize.Add(delta)
		pvc.Spec.Resources.Requests["storage"] = originalSize

		By(fmt.Sprintf("get pvc %v in ns %v", pvc, ns))


		updatedPvc, err := clientsete2e.CoreV1().PersistentVolumeClaims(ns).Update(pvc)
		if err != nil {
			framework.ExpectNoError(err, fmt.Sprintf("fail to resize pvc(%s): %v", pvcName, err))
		}
		WaitForPvToResize(clientsete2e, ns,  updatedPvc.Spec.VolumeName, originalSize,  1*time.Minute, 5*time.Second)
	}


}

// WaitForPvToResize waiting for pvc size to be resized to desired size
func WaitForPvToResize(c clientset.Interface, ns string, pvName string, desiredSize resource.Quantity, timeout time.Duration, interval time.Duration) error {
	By(fmt.Sprintf("Waiting up to %v for pv in namespace %q to be complete", timeout, ns))
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(interval) {
		newPv, _ := c.CoreV1().PersistentVolumes().Get(pvName, metav1.GetOptions{})
		newPvSize := newPv.Spec.Capacity["storage"]
		if desiredSize.Equal(newPvSize) {
			By(fmt.Sprintf("Pv size is updated to %v", newPvSize.String()))
			return nil
		}
	}
	return fmt.Errorf("Gave up after waiting %v for pv %q to complete resizing", timeout, pvName)
}

func DeletePVCOnDemand(clientsete2e clientset.Interface, ns string) {

	for i := 0; i < 100; i++ {
		pvcName := fmt.Sprintf("pvc-%d", i)
		clientsete2e.CoreV1().PersistentVolumeClaims(ns).Delete(pvcName, nil)
		waitForPersistentVolumeClaimDeleted(clientsete2e, ns, pvcName, 5*time.Second, 5*time.Minute)
	}

}
func createPVCOnDemand(clientsete2e clientset.Interface, ns string) {

	scName := "in-tree-iso"
	volumeMode := v1.PersistentVolumeFilesystem
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("pvc-%d", i)
		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
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
		clientsete2e.CoreV1().PersistentVolumeClaims(ns).Create(pvc)
		WaitForSuccess(clientsete2e, ns, name)
	}
}

func WaitForSuccess(clientsete2e clientset.Interface, ns string, pvcName string) v1.PersistentVolumeClaim {
	var err error

	By(fmt.Sprintf("waiting for PVC to be in phase %q", v1.ClaimBound))
	err = e2epv.WaitForPersistentVolumeClaimPhase(v1.ClaimBound, clientsete2e, ns, pvcName, framework.Poll, framework.ClaimProvisionTimeout)
	framework.ExpectNoError(err)

	By("checking the PVC")
	// Get new copy of the claim
	persistentVolumeClaim, err := clientsete2e.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{})
	framework.ExpectNoError(err)
	By("validating provisioned PV")
	persistentVolume, err := clientsete2e.CoreV1().PersistentVolumes().Get(persistentVolumeClaim.Spec.VolumeName, metav1.GetOptions{})
	framework.ExpectNoError(err)

	pvCapacity := persistentVolume.Spec.Capacity[v1.ResourceName(v1.ResourceStorage)]
	claimCapacity := persistentVolumeClaim.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	Expect(pvCapacity.Value()).To(Equal(claimCapacity.Value()), "pvCapacity is not equal to requestedCapacity")

	return *persistentVolumeClaim
}

// waitForPersistentVolumeClaimDeleted waits for a PersistentVolumeClaim to be removed from the system until timeout occurs, whichever comes first.
func waitForPersistentVolumeClaimDeleted(c clientset.Interface, ns string, pvcName string, Poll, timeout time.Duration) error {
	framework.Logf("Waiting up to %v for PersistentVolumeClaim %s to be removed", timeout, pvcName)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(Poll) {
		_, err := c.CoreV1().PersistentVolumeClaims(ns).Get(pvcName, metav1.GetOptions{})
		if err != nil {
			if apierrs.IsNotFound(err) {
				framework.Logf("Claim %q in namespace %q doesn't exist in the system", pvcName, ns)
				return nil
			}
			framework.Logf("Failed to get claim %q in namespace %q, retrying in %v. Error: %v", pvcName, ns, Poll, err)
		}
	}
	return fmt.Errorf("PersistentVolumeClaim %s is not removed from the system within %v", pvcName, timeout)
}

var _ = Describe("[PerformanceSnapshotISO] Dynamic Provisioning", func() {

	var (
		snapshotrcs restclientset.Interface
	)

	BeforeEach(func() {
		var err error
		snapshotrcs, err = restClient(testsuites.SnapshotAPIGroup, testsuites.APIVersionv1beta1)
		if err != nil {
			Fail(fmt.Sprintf("could not get rest clientset: %v", err))
		}
	})

	Measure("should iso create snapshot", func(b Benchmarker) {
		// uses the current context in kubeconfig
		// path-to-kubeconfig -- for example, /root/.kube/config

		// creates the clientset
		runtime := b.Time("runtime", func() {
			createSnapshotOnDemand(snapshotrcs, "ebs-claim", "default")
		})
		print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))
	}, 1)

	/*Measure("should iso delete snapshot", func(b Benchmarker) {
	// uses the current context in kubeconfig
	// path-to-kubeconfig -- for example, /root/.kube/config

	// creates the clientset
	runtime := b.Time("runtime", func() {
		deleteSnapshotOnDemand(snapshotrcs, "default")
	})
	print(fmt.Sprintf("Runtime: %v", runtime.Seconds()))	}, 1)*/

})

const (
	VolumeSnapshotKind        = "VolumeSnapshot"
	VolumeSnapshotContentKind = "VolumeSnapshotContent"
	SnapshotAPIVersion        = "snapshot.storage.k8s.io/v1beta1"
	APIVersionv1beta1         = "v1beta1"
)

func createSnapshotOnDemand(snapshotrcs restclientset.Interface, pvcName string, ns string) {
	By("creating a VolumeSnapshot for " + pvcName)
	vsc := "csi-aws-vsc"
	for i := 0; i < 1; i++ {
		name := fmt.Sprintf("snapshot-%d", i)
		snapshot := &v1beta1.VolumeSnapshot{
			TypeMeta: metav1.TypeMeta{
				Kind:       VolumeSnapshotKind,
				APIVersion: SnapshotAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: v1beta1.VolumeSnapshotSpec{
				VolumeSnapshotClassName: &vsc,
				Source: v1beta1.VolumeSnapshotSource{
					PersistentVolumeClaimName: &pvcName,
				},
			},
		}
		snapshot, err := snapshotclientset.New(snapshotrcs).SnapshotV1beta1().VolumeSnapshots(ns).Create(snapshot)
		framework.ExpectNoError(err)
		WaitForSnapshotReadyToUse(snapshot, snapshotrcs, ns)
	}
}

func WaitForSnapshotReadyToUse(snapshot *v1beta1.VolumeSnapshot, snapshotrcs restclientset.Interface, ns string) {
	By("waiting for VolumeSnapshot to be ready to use - " + snapshot.Name)
	err := wait.Poll(15*time.Second, 5*time.Minute, func() (bool, error) {
		vs, err := snapshotclientset.New(snapshotrcs).SnapshotV1beta1().VolumeSnapshots(ns).Get(snapshot.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("did not see ReadyToUse: %v", err)
		}

		if vs.Status == nil || vs.Status.ReadyToUse == nil {
			return false, nil
		}
		return *vs.Status.ReadyToUse, nil
	})
	framework.ExpectNoError(err)
}

func deleteSnapshotOnDemand(snapshotrcs restclientset.Interface, ns string) {
	for i := 0; i < 100; i++ {
		snapshotName := fmt.Sprintf("snapshot-%d", i)
		err := snapshotclientset.New(snapshotrcs).SnapshotV1beta1().VolumeSnapshots(ns).Delete(snapshotName, nil)
		framework.ExpectNoError(err)
		waitForSnapshotDeleted(snapshotrcs, snapshotName, ns, 5*time.Second, 5*time.Minute)
	}
}

func waitForSnapshotDeleted(snapshotrcs restclientset.Interface, snapshotName string, ns string, poll, timeout time.Duration) error {
	e2elog.Logf("Waiting up to %v for VolumeSnapshot %s to be removed", timeout, snapshotName)
	c := snapshotclientset.New(snapshotrcs).SnapshotV1beta1()
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		_, err := c.VolumeSnapshots(ns).Get(snapshotName, metav1.GetOptions{})
		if err != nil {
			if apierrs.IsNotFound(err) {
				e2elog.Logf("Snapshot %q in namespace %q doesn't exist in the system", snapshotName, ns)
				return nil
			}
			e2elog.Logf("Failed to get snapshot %q in namespace %q, retrying in %v. Error: %v", snapshotName, ns, poll, err)
		}
	}
	return fmt.Errorf("VolumeSnapshot %s is not removed from the system within %v", snapshotName, timeout)
}

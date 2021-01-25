/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
   http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testsuites

import (
	"fmt"
	"github.com/kubernetes-sigs/aws-ebs-csi-driver/pkg/util"
	"github.com/kubernetes-sigs/aws-ebs-csi-driver/tests/e2e/driver"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"time"

	. "github.com/onsi/ginkgo"
	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
)

// DynamicallyProvisionedResizeVolumeTest will provision required StorageClass(es), PVC(s) and Pod(s)
// Waiting for the PV provisioner to create a new PV
// Update pvc storage size
// Waiting for new PVC and PV to be ready
// And finally attach pvc to the pod and wait for pod to be ready.
type DynamicallyProvisionedResizeVolumeTest struct {
	CSIDriver driver.DynamicPVTestDriver
	Pod       PodDetails
}

func (t *DynamicallyProvisionedResizeVolumeTest) Run(client clientset.Interface, namespace *v1.Namespace) {
	volume := t.Pod.Volumes[0]
	tpvc, _ := volume.SetupDynamicPersistentVolumeClaim(client, namespace, t.CSIDriver)
	defer tpvc.Cleanup()

	pvcName := tpvc.persistentVolumeClaim.Name
	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace.Name).Get(pvcName, metav1.GetOptions{})
	By(fmt.Sprintf("Get pvc name: %v", pvc.Name))
	originalSize := pvc.Spec.Resources.Requests["storage"]
	delta := resource.Quantity{}
	delta.Set(util.GiBToBytes(1))
	originalSize.Add(delta)
	pvc.Spec.Resources.Requests["storage"] = originalSize

	By("resizing the pvc")
	updatedPvc, err := client.CoreV1().PersistentVolumeClaims(namespace.Name).Update(pvc)
	if err != nil {
		framework.ExpectNoError(err, fmt.Sprintf("fail to resize pvc(%s): %v", pvcName, err))
	}
	updatedSize := updatedPvc.Spec.Resources.Requests["storage"]

	By("Wait for resize to complete")
	time.Sleep(30 * time.Second)

	By("checking the resizing result")
	newPvc, err := client.CoreV1().PersistentVolumeClaims(namespace.Name).Get(pvcName, metav1.GetOptions{})
	if err != nil {
		framework.ExpectNoError(err, fmt.Sprintf("fail to get new pvc(%s): %v", pvcName, err))
	}
	newSize := newPvc.Spec.Resources.Requests["storage"]
	if !newSize.Equal(updatedSize) {
		framework.Failf("newSize(%+v) is not equal to updatedSize(%+v)", newSize.String(), updatedSize.String())
	}

	By("checking the resizing PV result")
	newPv, _ := client.CoreV1().PersistentVolumes().Get(newPvc.Spec.VolumeName, metav1.GetOptions{})
	newPvSize := newPv.Spec.Capacity["storage"]
	if !newSize.Equal(newPvSize) {
		By(fmt.Sprintf("newPVCSize(%+v) is not equal to newPVSize(%+v)", newSize.String(), newPvSize.String()))
	}

	By("Validate volume can be attached")
	tpod := NewTestPod(client, namespace, t.Pod.Cmd)

	tpod.SetupVolume(tpvc.persistentVolumeClaim, volume.VolumeMount.NameGenerate+"1", volume.VolumeMount.MountPathGenerate+"1", volume.VolumeMount.ReadOnly)

	By("deploying the pod")
	tpod.Create()
	By("checking that the pods is running")
	tpod.WaitForSuccess()

	defer tpod.Cleanup()

}

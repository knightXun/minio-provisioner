/*
Copyright 2016 The Kubernetes Authors.

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

package volume

import (
	"fmt"
	"github.com/kubernetes-sigs/sig-storage-lib-external-provisioner/controller"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os/exec"
	"strings"
)

const (
	// are we allowed to set this? else make up our own
	annCreatedBy = "kubernetes.io/createdby"
	createdBy    = "minio-dynamic-provisioner"

	// MountOptionAnnotation is the annotation on a PV object that specifies a
	// comma separated list of mount options
	MountOptionAnnotation = "volume.beta.kubernetes.io/mount-options"

	// A PV annotation for the identity of the minioProvisioner that provisioned it
	annProvisionerID = "Provisioner_Id"

	podIPEnv     = "POD_IP"
	serviceEnv   = "SERVICE_NAME"
	namespaceEnv = "POD_NAMESPACE"
	nodeEnv      = "NODE_NAME"

	minioPVName = "volume.kubernetes.io/minio-pv-name"

	minioPVPath = "volume.kubernetes.io/minio-pv-path"

	minioKey    = "volume.kubernetes.io/minio-keys"

	minioURL    = "volume.kubernetrs.io/minio-url"

	provisioner = "volume.kubernetes.io/provisioner"
)

// NewminioProvisioner creates a Provisioner that provisions minio PVs backed by
// the given directory.
func NewMinioProvisioner(client kubernetes.Interface, outOfCluster bool, hostName, minioKey, minioURL, provisioner string) controller.Provisioner {
	return newMinioProvisionerInternal(client, outOfCluster, hostName, minioKey, minioURL, provisioner)
}

func newMinioProvisionerInternal(client kubernetes.Interface, outOfCluster bool, hostName, minioKey, minioURL, provisionerName string) *minioProvisioner {

	provisioner := &minioProvisioner{
		client:         client,
		outOfCluster:   outOfCluster,
		podIPEnv:       podIPEnv,
		serviceEnv:     serviceEnv,
		namespaceEnv:   namespaceEnv,
		nodeEnv:        nodeEnv,
		minioKey:       minioKey,
		minioURL:       minioURL,
		serverHostname: hostName,
		provisioner:    provisionerName,
	}

	return provisioner
}

type minioProvisioner struct {
	// Client, needed for getting a service cluster IP to put as the minio server of
	// provisioned PVs
	client kubernetes.Interface

	serverHostname string
	// Whether the provisioner is running out of cluster and so cannot rely on
	// the existence of any of the pod, service, namespace, node env variables.
	outOfCluster bool

	identity string
	// Environment variables the provisioner pod needs valid values for in order to
	// put a service cluster IP as the server of provisioned minio PVs, passed in
	// via downward API. If serviceEnv is set, namespaceEnv must be too.
	podIPEnv     string
	serviceEnv   string
	namespaceEnv string
	nodeEnv      string
	minioKey     string
	minioURL	 string
	provisioner  string
}

var _ controller.Provisioner = &minioProvisioner{}

// ShouldProvision returns whether provisioning should be attempted for the given
// claim.
func (p *minioProvisioner) ShouldProvision(claim *v1.PersistentVolumeClaim) bool {
	// As long as the export limit has not been reached we're ok to provision
	if claim.Annotations["provisioner"] == p.provisioner {
		return true
	}
	return false
}

// Provision creates a volume i.e. the storage asset and returns a PV object for
// the volume.
func (p *minioProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	if options.PVC.Annotations[minioPVName] == "" {
		return nil, fmt.Errorf("Empty minio-pv-name!!!")
	}

	if options.PVC.Annotations[minioPVPath] == "" {
		 return nil, fmt.Errorf("Empty minio-pv-path!!!")
	}


	volume, err := p.createVolume(options)
	if err != nil {
		return nil, err
	}

	annotations := make(map[string]string)
	annotations[annCreatedBy] = createdBy
	// Only use legacy mount options annotation if StorageClass.MountOptions is empty
	if volume.mountOptions != "" && options.MountOptions == nil {
		annotations[MountOptionAnnotation] = volume.mountOptions
	}
	annotations[annProvisionerID] = string(p.identity)

	pvName := options.PVC.Annotations[minioPVName]
	annotations[minioPVPath] = volume.path
	annotations[minioKey] = p.minioKey
	annotations[minioURL] = p.minioURL

	// example: s3fs -o passwd_file=/root/test.passwd-s3fs -o url=https://192.168.90.133:9000 -o use_path_request_style
	// -o bucket=test /test/sd -o curldbg -o no_check_certificate -o connect_timeout=1 -o retries=1
	fsType := "s3fs"
	readOnly := false
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pvName,
			Labels:      map[string]string{},
			Annotations: annotations,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				CSI: &v1.CSIPersistentVolumeSource{
					Driver:       "s3fs.csi.minio.com",
					ReadOnly:     readOnly,
					FSType:       fsType,
					VolumeHandle: volume.path,
					VolumeAttributes: map[string]string{
						"BucketName":  volume.path,
						"MinioKey":    p.minioKey,
						"MinioURL":    p.minioURL,
					},
				},
			},
		},
	}

	return pv, nil
}

type volume struct {
	server       string
	path         string
	mountOptions string
}

// createVolume creates a volume i.e. the storage asset. It creates a unique
// directory under /export and exports it. Returns the server IP, the path, a
// zero/non-zero supplemental group, the block it added to either the ganesha
// config or /etc/exports, and the exportID
// TODO return values
func (p *minioProvisioner) createVolume(options controller.VolumeOptions) (volume, error) {
	err := p.validateOptions(options)
	if err != nil {
		return volume{}, fmt.Errorf("error validating options for volume: %v", err)
	}

	minioPath := ""

	if options.PVC.Annotations[minioPVPath] != "" {
		minioPath = options.PVC.Annotations[minioPVPath]
	} else {
		minioPath = options.PVName
	}

	minioPath = strings.ReplaceAll(minioPath, "/", "-")
	//err = p.createBucket(bucketName)
	err = p.createBucket(minioPath)
	if err != nil {
		return volume{}, fmt.Errorf("error creating directory for volume: %v", err)
	}

	return volume{
		path:         minioPath,
	}, nil
}

func (p *minioProvisioner) validateOptions(options controller.VolumeOptions) (error) {

	if options.PVC.Spec.Selector != nil {
		return fmt.Errorf("claim.Spec.Selector is not supported")
	}

	return  nil
}

// createBucket creates a minio bucket
func (p *minioProvisioner) createBucket(bucket string) error {
	cmd := exec.Command("/usr/bin/s3cmd", "mb", "s3://"+ bucket,  "--no-check-certificate")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create s3 bucket %v failed with error: %v, output: %s", bucket, err, out)
	}

	return nil
}

package volume

import (
	"fmt"
	"github.com/kubernetes-sigs/sig-storage-lib-external-provisioner/controller"
	"k8s.io/api/core/v1"
	"os/exec"
)

// Delete removes the directory that was created by Provision backing the given
// PV and removes its export from the mini server.
func (p *minioProvisioner) Delete(volume *v1.PersistentVolume) error {
	// Ignore the call if this provisioner was not the one to provision the
	// volume. It doesn't even attempt to delete it, so it's neither a success
	// (nil error) nor failure (any other error)
	provisioned, err := p.provisioned(volume)
	if err != nil {
		return fmt.Errorf("error determining if this provisioner was the one to provision volume %q: %v", volume.Name, err)
	}
	if !provisioned {
		strerr := fmt.Sprintf("this provisioner id %s didn't provision volume %q and so can't delete it; id %s did & can", p.identity, volume.Name, volume.Annotations[annProvisionerID])
		return &controller.IgnoredError{Reason: strerr}
	}

	err = p.deleteBucket(volume)
	if err != nil {
		return fmt.Errorf("error deleting volume's backing path: %v", err)
	}

	return nil
}

func (p *minioProvisioner) provisioned(volume *v1.PersistentVolume) (bool, error) {
	provisionerID, ok := volume.Annotations[annProvisionerID]
	if !ok {
		return false, fmt.Errorf("PV doesn't have an annotation %s", annProvisionerID)
	}

	return provisionerID == string(p.identity), nil
}

func (p *minioProvisioner) deleteBucket(volume *v1.PersistentVolume) error {
	bucketName := volume.Annotations[minioPVPath]
	cmd := exec.Command("/usr/bin/s3cmd", "rb", "s3://"+bucketName,  "--no-check-certificate")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete s3 bucket %v failed with error: %v, output: %s", bucketName, err, out)
	}

	return nil
}

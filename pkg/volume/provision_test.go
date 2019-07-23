package volume

import (
	"os"
	"testing"

	utiltesting "k8s.io/client-go/util/testing"
)

func TestCreateVolume(t *testing.T) {
	tmpDir := utiltesting.MkTmpdirOrDie("minioProvisionTest")
	defer os.RemoveAll(tmpDir)
}

func TestValidateOptions(t *testing.T) {
	tmpDir := utiltesting.MkTmpdirOrDie("minioProvisionTest")
	defer os.RemoveAll(tmpDir)
}

func TestShouldProvision(t *testing.T) {
}

func TestCreateBucket(t *testing.T) {
	tmpDir := utiltesting.MkTmpdirOrDie("minioProvisionTest")
	defer os.RemoveAll(tmpDir)
}


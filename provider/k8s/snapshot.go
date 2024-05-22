package k8s

import (
	"fmt"
)

func (p *Provider) SetDBDeletionProtectionAndCreateSnapShot(app, resource, snapshot string) (string, error) {
	return "", fmt.Errorf("not supported")
}

func (p *Provider) IsDBSnapshotComplete(snapshot string) (bool, error) {
	return false, fmt.Errorf("not supported")
}

func (p *Provider) DeleteDB(resource string) error {
	return fmt.Errorf("not supported")
}

package gitlab

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/xanzy/go-gitlab"
)

// remoteClient is a remote client that stores data in GitLab.
type remoteClient struct {
	client    *gitlab.TerraformService
	projectId string
	stateName string

	lockData *gitlab.LockData
}

func (c *remoteClient) Lock(info *state.LockInfo) (string, error) {
	if c.lockData != nil {
		return "", fmt.Errorf("lock %s should be released before acquiring a new lock", c.lockData.RequestedLockInfo.ID)
	}
	lockData, err := c.client.LockState(c.projectId, c.stateName, &gitlab.LockInfo{
		ID:        info.ID,
		Operation: info.Operation,
		Info:      info.Info,
		Who:       info.Who,
		Version:   info.Version,
		Created:   info.Created,
		Path:      info.Path,
	})
	if err != nil {
		return "", err
	}
	c.lockData = lockData
	return lockData.RequestedLockInfo.ID, nil
}

func (c *remoteClient) Unlock(id string) error {
	if c.lockData == nil {
		return errors.New("cannot unlock - not holding a lock")
	}
	err := c.client.UnlockState(c.lockData)
	if err != nil {
		return err
	}
	c.lockData = nil
	return nil
}

func (c *remoteClient) Get() (*remote.Payload, error) {
	payload, err := c.client.GetState(c.projectId, c.stateName)
	if err != nil {
		return nil, err
	}
	return &remote.Payload{
		MD5:  payload.MD5,
		Data: payload.Data,
	}, nil
}

func (c *remoteClient) Put(data []byte) error {
	if c.lockData == nil {
		return errors.New("a lock must be obtained before putting data")
	}
	return c.client.PutState(c.lockData, data)
}

func (c *remoteClient) Delete() error {
	return c.client.DeleteState(c.projectId, c.stateName)
}

package loader

import (
	"crypto/rsa"
	"encoding/json"
	"os/user"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/instance"
)

type UserLoader interface {
	LoadUser(mattermostUserId string) (user.User, error)
	LoadMattermostUserId(userKey string) (string, error)
}

type userLoader struct {
	userStore user.Store
}

func New(instanceStore instance.Store, currentInstanceStore instance.CurrentInstanceStore,
	rsaPrivateKey *rsa.PrivateKey, authTokenSecret []byte) InstanceLoader {

	return &instanceLoader{
		instanceStore:        instanceStore,
		currentInstanceStore: currentInstanceStore,
		rsaPrivateKey:        rsaPrivateKey,
		authTokenSecret:      authTokenSecret,
	}
}

func (il instanceLoader) Load(key string) (instance.Instance, error) {
	data, err := il.instanceStore.LoadRaw(key)
	if err != nil {
		return nil, err
	}
	inst, err := il.loadInstanceFromJSON(data)
	if err != nil {
		return nil, errors.WithMessage(err, "LoadInstance failed")
	}
	return inst, nil
}

func (il instanceLoader) Current() (instance.Instance, error) {
	data, err := il.currentInstanceStore.LoadRaw()
	if err != nil {
		return nil, err
	}
	inst, err := il.loadInstanceFromJSON(data)
	if err != nil {
		return nil, errors.WithMessage(err, "LoadCurrentInstance failed")
	}
	return inst, nil
}

func (il instanceLoader) loadInstanceFromJSON(data []byte) (instance.Instance, error) {
	// Unmarshal into any of the types just so that we can get the common data
	basic := instance.BasicInstance{}
	err := json.Unmarshal(data, &basic)
	if err != nil {
		return nil, err
	}

	var newInstance instance.Instance
	switch basic.InstanceType {
	case jira_server.Type:
		newInstance, err = jira_server.FromJSON(data, il.rsaPrivateKey)
	case jira_cloud.Type:
		newInstance, err = jira_cloud.FromJSON(data, il.authTokenSecret)
	default:
		return nil, errors.Errorf("Jira instance %s has unsupported type: %s",
			basic.InstanceKey, basic.InstanceType)
	}
	if err != nil {
		return nil, errors.WithMessage(err, "failed to unmarshal")
	}
	return newInstance, nil
}

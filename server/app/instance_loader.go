package app

import (
	"encoding/json"

	"github.com/mattermost/mattermost-plugin-jira/server/instance/jira_server"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-jira/server/instance"
	"github.com/mattermost/mattermost-plugin-jira/server/instance/jira_cloud"
)

func LoadInstance(instanceStore instance.InstanceStore, key string) (instance.Instance, error) {
	be, err := loadInstanceFromRaw(func() ([]byte, error) {
		return instanceStore.LoadRaw(key)
	})
	if err != nil {
		return nil, errors.WithMessage(err, "LoadInstance failed")
	}
	return be, nil
}

func LoadCurrentInstance(currentInstanceStore instance.CurrentInstanceStore) (instance.Instance, error) {
	be, err := loadInstanceFromRaw(currentInstanceStore.LoadCurrentInstanceRaw)
	if err != nil {
		return nil, errors.WithMessage(err, "LoadCurrentInstance failed")
	}
	return be, nil
}

func loadInstanceFromRaw(loaderf func() ([]byte, error)) (instance.Instance, error) {
	data, err := loaderf()
	if err != nil {
		return nil, err
	}

	// Unmarshal into any of the types just so that we can get the common data
	basic := instance.BasicInstance{}
	err = json.Unmarshal(data, &basic)
	if err != nil {
		return nil, err
	}

	var newInstance instance.Instance
	switch basic.InstanceType {
	case jira_server.BackendType:
		newInstance = &jira_server.JiraServerInstance{}
	case jira_cloud.BackendType:
		newInstance = &jira_cloud.JiraCloudInstance{}
	default:
		return nil, errors.Errorf("Jira instance %s has unsupported type: %s",
			basic.InstanceKey, basic.InstanceType)
	}

	err = json.Unmarshal(data, newInstance)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to unmarshal")
	}
	return newInstance, nil
}

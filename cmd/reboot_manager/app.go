package deps

import (
	"errors"
	"fmt"

	"github.com/soerenschneider/sc-agent/internal/config"
	"github.com/soerenschneider/sc-agent/internal/services/components/reboot_manager/group"
)

func BuildGroups(groupUpdates chan *group.Group, conf *config.RebootManagerConfig) ([]*group.Group, error) {
	if conf == nil {
		return nil, errors.New("empty config supplied")
	}

	var groups []*group.Group

	for _, groupConf := range conf.Groups {
		groupConf := groupConf
		group, err := BuildGroup(groupUpdates, &groupConf)
		if err != nil {
			return nil, fmt.Errorf("could not build group '%s': %w", groupConf.Name, err)
		}
		groups = append(groups, group)
	}

	return groups, nil
}

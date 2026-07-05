package main

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/pgaskin/koboutils/v2/kobo"
)

func UpgradeCheck() (string, error) {
	serial, version, id, err := kobo.ParseKoboVersion(Root)
	if err != nil {
		return "", err
	}
	upgrade, err := kobo.CheckUpgrade(id, "kobo", version, serial)
	if err != nil {
		return "", fmt.Errorf("firmware check: %w", err)
	}
	if !upgrade.UpgradeType.IsUpdate() {
		return "", nil
	}

	prompt := promptui.Select{Label: fmt.Sprintf("Update is %s, update?", upgrade.UpgradeType), Items: []string{"yes", "no"}}
	_, s, err := prompt.Run()
	if err != nil {
		return "", err
	}
	proceed := s == "yes"
	if !proceed {
		return "", nil
	}

	return upgrade.UpgradeURL, nil
}

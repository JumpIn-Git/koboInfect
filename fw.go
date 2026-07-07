package main

import (
	"fmt"

	"github.com/charmbracelet/huh"
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

	proceed := true
	if err := huh.NewConfirm().
		Title(fmt.Sprintf("Update is %s, update?", upgrade.UpgradeType)).
		Value(&proceed).
		Run(); err != nil {
		return "", err
	}
	if !proceed {
		return "", nil
	}

	return upgrade.UpgradeURL, nil
}

package main

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/pgaskin/koboutils/v2/kobo"
)

func UpgradeCheck(root string) (string, error) {
	serial, version, id, err := kobo.ParseKoboVersion(root)
	if err != nil {
		return "", err
	}
	upgrade, err := kobo.CheckUpgrade(id, "kobo", version, serial)
	if err != nil {
		return "", err
	}
	if !upgrade.UpgradeType.IsUpdate() {
		return "", nil
	}

	proceed := true
	if err := huh.NewConfirm().
		Title(fmt.Sprintf("Update is %s, update?", upgrade.UpgradeType)).
		Value(&proceed).Run(); err != nil {
		return "", err
	}
	if !proceed {
		return "", nil
	}

	return upgrade.UpgradeURL, nil
}

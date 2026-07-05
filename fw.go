package main

import (
	"fmt"

	"charm.land/huh/v2"
	"github.com/pgaskin/koboutils/v2/kobo"
)

func UpgradeCheck() (string, error) {
	serial, version, id, err := kobo.ParseKoboVersion(Root)
	if err != nil {
		return "", err
	}
	upgrade, err := kobo.CheckUpgrade(id, "kobo", version, serial)
	if err != nil {
		return "", fmt.Errorf("couldn't check for fw: %w", err)
	}
	if !upgrade.UpgradeType.IsUpdate() {
		return "", nil
	}

	var proceed bool
	err = huh.NewConfirm().
		Title(fmt.Sprintf("Update is %s, update?", upgrade.UpgradeType)).
		Value(&proceed).Run()
	if err != nil {
		return "", err
	}
	if !proceed {
		return "", nil
	}

	return upgrade.UpgradeURL, nil
}

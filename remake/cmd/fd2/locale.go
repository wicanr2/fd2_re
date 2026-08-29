package main

import (
	"fmt"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/localization"
)

type localeEntryContract struct {
	variables []string
	sourceID  string
}

var physicalAttackLocaleContract = map[string]localeEntryContract{
	"battle.attack.miss":            {[]string{"%s", "%s"}, "legacy.go.remake.cmd.fd2.main.l5973-c22"},
	"battle.attack.hit":             {[]string{"%s", "%s", "%d"}, "legacy.go.remake.cmd.fd2.main.l5975-c25"},
	"battle.attack.critical_suffix": {[]string{}, "legacy.go.remake.cmd.fd2.main.l5977-c14"},
	"battle.attack.exp_suffix":      {[]string{"%.0f"}, "legacy.go.remake.cmd.fd2.main.l5980-c26"},
}

func loadOfficialLocale(localeID string) (*localization.Catalog, error) {
	root := assetPath("assets/locales")
	catalog, err := localization.LoadOfficial(root, localeID)
	if err != nil {
		return nil, err
	}
	for key, contract := range physicalAttackLocaleContract {
		entry, ok := catalog.Entries[key]
		if !ok {
			return nil, fmt.Errorf("official locale %s is missing %s", localeID, key)
		}
		if entry.SourceStringID != contract.sourceID {
			return nil, fmt.Errorf("official locale %s has invalid provenance for %s", localeID, key)
		}
		if len(entry.Variables) != len(contract.variables) {
			return nil, fmt.Errorf("official locale %s has invalid variables for %s", localeID, key)
		}
		for i := range contract.variables {
			if entry.Variables[i] != contract.variables[i] {
				return nil, fmt.Errorf("official locale %s has invalid variables for %s", localeID, key)
			}
		}
	}
	if filepath.Clean(catalog.Font.Path) != filepath.FromSlash(catalog.Font.Path) {
		return nil, fmt.Errorf("official locale %s has a non-canonical font path", localeID)
	}
	return catalog, nil
}

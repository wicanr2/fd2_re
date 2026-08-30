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

var officialLocaleContract = map[string]localeEntryContract{
	"battle.attack.miss":            {[]string{"%s", "%s"}, "legacy.go.remake.cmd.fd2.main.l5973-c22"},
	"battle.attack.hit":             {[]string{"%s", "%s", "%d"}, "legacy.go.remake.cmd.fd2.main.l5975-c25"},
	"battle.attack.critical_suffix": {[]string{}, "legacy.go.remake.cmd.fd2.main.l5977-c14"},
	"battle.attack.exp_suffix":      {[]string{"%.0f"}, "legacy.go.remake.cmd.fd2.main.l5980-c26"},
	"system.locale.changed":         {[]string{"%s"}, "runtime.settings.locale.changed"},
	"system.audio.changed":          {[]string{"%s"}, "runtime.settings.audio.changed"},
	"save.unsupported":              {[]string{}, "runtime.save.unsupported"},
	"save.postbattle_blocked":       {[]string{}, "runtime.save.postbattle_blocked"},
	"save.saved":                    {[]string{"%d", "%s"}, "runtime.save.saved"},
	"save.none":                     {[]string{}, "runtime.save.none"},
	"save.node_missing":             {[]string{"%s"}, "runtime.save.node_missing"},
	"save.loaded":                   {[]string{"%d", "%s"}, "runtime.save.loaded"},
}

func loadOfficialLocale(localeID string) (*localization.Catalog, error) {
	root := assetPath("assets/locales")
	catalog, err := localization.LoadOfficial(root, localeID)
	if err != nil {
		return nil, err
	}
	if len(catalog.Entries) != len(officialLocaleContract) {
		return nil, fmt.Errorf("official locale %s has %d keys, want exact %d", localeID, len(catalog.Entries), len(officialLocaleContract))
	}
	for key, contract := range officialLocaleContract {
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

// localeMessage formats a required official key. Callers must stop their
// current UI transaction when ok is false; silently falling back to embedded
// Traditional Chinese would create a mixed-language official pack.
func (g *Game) localeMessage(key string, args ...any) (message string, ok bool) {
	if g == nil || g.localeCatalog == nil {
		if g != nil {
			g.loadErr = "locale message: official catalog is unavailable"
		}
		return "", false
	}
	message, err := g.localeCatalog.Format(key, args...)
	if err != nil {
		g.loadErr = "locale message: " + err.Error()
		return "", false
	}
	return message, true
}

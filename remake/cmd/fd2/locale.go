package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/localization"
)

type localeEntryContract struct {
	variables []string
	sourceID  string
}

var officialLocaleContract = map[string]localeEntryContract{
	"battle.attack.miss":                {[]string{"%s", "%s"}, "legacy.go.remake.cmd.fd2.main.l5973-c22"},
	"battle.attack.hit":                 {[]string{"%s", "%s", "%d"}, "legacy.go.remake.cmd.fd2.main.l5975-c25"},
	"battle.attack.critical_suffix":     {[]string{}, "legacy.go.remake.cmd.fd2.main.l5977-c14"},
	"battle.attack.exp_suffix":          {[]string{"%.0f"}, "legacy.go.remake.cmd.fd2.main.l5980-c26"},
	"battle.mp.insufficient":            {[]string{}, "legacy.go.remake.cmd.fd2.main.l5091-c14"},
	"battle.command.unavailable":        {[]string{}, "legacy.go.remake.cmd.fd2.main.l5187-c12"},
	"battle.attack.choose_target":       {[]string{}, "legacy.go.remake.cmd.fd2.main.l5193-c13"},
	"battle.command.native_unavailable": {[]string{}, "legacy.go.remake.cmd.fd2.main.l5201-c14"},
	"battle.spell.sealed":               {[]string{}, "legacy.go.remake.cmd.fd2.main.l5211-c14"},
	"battle.spell.none":                 {[]string{}, "legacy.go.remake.cmd.fd2.main.l5218-c13"},
	"battle.item.choose_slot":           {[]string{}, "legacy.go.remake.cmd.fd2.main.l5225-c13"},
	"system.locale.changed":             {[]string{"%s"}, "runtime.settings.locale.changed"},
	"system.audio.changed":              {[]string{"%s"}, "runtime.settings.audio.changed"},
	"save.unsupported":                  {[]string{}, "runtime.save.unsupported"},
	"save.postbattle_blocked":           {[]string{}, "runtime.save.postbattle_blocked"},
	"save.saved":                        {[]string{"%d", "%s"}, "runtime.save.saved"},
	"save.none":                         {[]string{}, "runtime.save.none"},
	"save.node_missing":                 {[]string{"%s"}, "runtime.save.node_missing"},
	"save.loaded":                       {[]string{"%d", "%s"}, "runtime.save.loaded"},
	"shop.greeting.weapon":              {[]string{}, "FDTXT_000/string_0440"},
	"shop.greeting.item":                {[]string{}, "FDTXT_000/string_0501"},
	"shop.purchase.question.weapon":     {[]string{"%s", "%d"}, "FDTXT_000/string_0439"},
	"shop.purchase.question.item":       {[]string{"%s", "%d"}, "FDTXT_000/string_0502"},
	"shop.purchase.insufficient.weapon": {[]string{}, "FDTXT_000/string_0438"},
	"shop.purchase.insufficient.item":   {[]string{}, "FDTXT_000/string_0504"},
	"shop.purchase.no_recipient.weapon": {[]string{}, "FDTXT_000/string_0437"},
	"shop.purchase.no_recipient.item":   {[]string{}, "FDTXT_000/string_0505"},
	"shop.purchase.equip_question":      {[]string{}, "FDTXT_000/string_0507"},
	"shop.recipient.full":               {[]string{"%s"}, "FDTXT_000/string_0506"},
	"shop.transfer.destination_prompt":  {[]string{}, "FDTXT_000/string_0510"},
	"shop.transfer.empty_source":        {[]string{"%s"}, "FDTXT_000/string_0511"},
	"shop.transfer.source_prompt":       {[]string{}, "FDTXT_000/string_0512"},
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

func loadOfficialLocaleContent(localeID string) (*localization.ContentCatalog, error) {
	return localization.LoadOfficialContent(assetPath("assets/locales"), localeID)
}

func loadOfficialLocaleEntities(localeID string) (*localization.EntityCatalog, error) {
	return localization.LoadOfficialEntities(assetPath("assets/locales"), localeID)
}

func (g *Game) localizedStoryText(line campaign.Line) (string, error) {
	if line.LineID == "" {
		if g != nil && g.localeID == "zh-Hant" && line.Text != "" {
			return line.Text, nil
		}
		return "", errors.New("story line lacks canonical line_id")
	}
	if g == nil || g.localeContent == nil {
		return "", errors.New("official locale content is unavailable")
	}
	return g.localeContent.StoryText(line.LineID)
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

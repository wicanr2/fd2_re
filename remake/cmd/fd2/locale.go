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
	"battle.attack.miss":                  {[]string{"%s", "%s"}, "legacy.go.remake.cmd.fd2.main.l5973-c22"},
	"battle.attack.hit":                   {[]string{"%s", "%s", "%d"}, "legacy.go.remake.cmd.fd2.main.l5975-c25"},
	"battle.attack.critical_suffix":       {[]string{}, "legacy.go.remake.cmd.fd2.main.l5977-c14"},
	"battle.attack.exp_suffix":            {[]string{"%.0f"}, "legacy.go.remake.cmd.fd2.main.l5980-c26"},
	"battle.mp.insufficient":              {[]string{}, "legacy.go.remake.cmd.fd2.main.l5091-c14"},
	"battle.command.unavailable":          {[]string{}, "legacy.go.remake.cmd.fd2.main.l5187-c12"},
	"battle.attack.choose_target":         {[]string{}, "legacy.go.remake.cmd.fd2.main.l5193-c13"},
	"battle.command.native_unavailable":   {[]string{}, "legacy.go.remake.cmd.fd2.main.l5201-c14"},
	"battle.spell.sealed":                 {[]string{}, "legacy.go.remake.cmd.fd2.main.l5211-c14"},
	"battle.spell.none":                   {[]string{}, "legacy.go.remake.cmd.fd2.main.l5218-c13"},
	"battle.item.choose_slot":             {[]string{}, "legacy.go.remake.cmd.fd2.main.l5225-c13"},
	"battle.spell.target_prompt":          {[]string{"%s", "%d"}, "legacy.go.remake.cmd.fd2.main.l5168-c14"},
	"battle.spell.blocked":                {[]string{}, "legacy.go.remake.cmd.fd2.main.l6170-c12"},
	"battle.unit.paralyzed":               {[]string{}, "legacy.go.remake.cmd.fd2.main.l6145-c12"},
	"battle.spell.result_damage":          {[]string{"%s", "%s", "%d", "%d"}, "legacy.go.remake.cmd.fd2.main.l6194-c10"},
	"battle.spell.result_heal":            {[]string{"%s", "%s", "%d", "%d"}, "legacy.go.remake.cmd.fd2.main.l6194-c10"},
	"battle.spell.miss_suffix":            {[]string{"%d"}, "legacy.go.remake.cmd.fd2.main.l6196-c13"},
	"battle.treasure.gold":                {[]string{"%d"}, "legacy.go.remake.cmd.fd2.main.l5396-c13"},
	"battle.treasure.item":                {[]string{"%02X"}, "legacy.go.remake.cmd.fd2.main.l5398-c13"},
	"battle.treasure.inventory_full":      {[]string{}, "legacy.go.remake.cmd.fd2.main.l5402-c12"},
	"battle.reward.item":                  {[]string{"%02X"}, "legacy.go.remake.cmd.fd2.main.l5457-c12"},
	"battle.reward.item_full":             {[]string{"%02X"}, "legacy.go.remake.cmd.fd2.main.l5459-c12"},
	"battle.reward.gold":                  {[]string{"%d"}, "legacy.go.remake.cmd.fd2.main.l5463-c11"},
	"battle.spell.menu_title":             {[]string{"%d"}, "legacy.go.remake.cmd.fd2.main.l8687-c22"},
	"church.transfer.empty":               {[]string{}, "legacy.go.remake.cmd.fd2.native_church_input.l209-c12"},
	"church.transfer.success":             {[]string{"%02X"}, "legacy.go.remake.cmd.fd2.native_church_input.l271-c12"},
	"church.revive.success":               {[]string{"%s", "%d"}, "legacy.go.remake.cmd.fd2.main.l4711-c10"},
	"church.class_change.success":         {[]string{"%s", "%s"}, "legacy.go.remake.cmd.fd2.main.l4674-c10"},
	"church.class_change.confirm_title":   {[]string{}, "legacy.go.remake.cmd.fd2.main.l9090-c13"},
	"church.class_change.empty":           {[]string{}, "legacy.go.remake.cmd.fd2.main.l9094-c24"},
	"church.class_change.target":          {[]string{"%s"}, "legacy.go.remake.cmd.fd2.main.l9100-c35"},
	"common.yes":                          {[]string{}, "legacy.go.remake.cmd.fd2.main.l9101-c36"},
	"common.no":                           {[]string{}, "legacy.go.remake.cmd.fd2.main.l9101-c41"},
	"system.locale.changed":               {[]string{"%s"}, "runtime.settings.locale.changed"},
	"system.audio.changed":                {[]string{"%s"}, "runtime.settings.audio.changed"},
	"save.unsupported":                    {[]string{}, "runtime.save.unsupported"},
	"save.postbattle_blocked":             {[]string{}, "runtime.save.postbattle_blocked"},
	"save.saved":                          {[]string{"%d", "%s"}, "runtime.save.saved"},
	"save.none":                           {[]string{}, "runtime.save.none"},
	"save.node_missing":                   {[]string{"%s"}, "runtime.save.node_missing"},
	"save.loaded":                         {[]string{"%d", "%s"}, "runtime.save.loaded"},
	"shop.greeting.weapon":                {[]string{}, "FDTXT_000/string_0440"},
	"shop.greeting.item":                  {[]string{}, "FDTXT_000/string_0501"},
	"shop.purchase.question.weapon":       {[]string{"%s", "%d"}, "FDTXT_000/string_0439"},
	"shop.purchase.question.item":         {[]string{"%s", "%d"}, "FDTXT_000/string_0502"},
	"shop.purchase.insufficient.weapon":   {[]string{}, "FDTXT_000/string_0438"},
	"shop.purchase.insufficient.item":     {[]string{}, "FDTXT_000/string_0504"},
	"shop.purchase.no_recipient.weapon":   {[]string{}, "FDTXT_000/string_0437"},
	"shop.purchase.no_recipient.item":     {[]string{}, "FDTXT_000/string_0505"},
	"shop.purchase.equip_question":        {[]string{}, "FDTXT_000/string_0507"},
	"shop.recipient.full":                 {[]string{"%s"}, "FDTXT_000/string_0506"},
	"shop.transfer.destination_prompt":    {[]string{}, "FDTXT_000/string_0510"},
	"shop.transfer.empty_source":          {[]string{"%s"}, "FDTXT_000/string_0511"},
	"shop.transfer.source_prompt":         {[]string{}, "FDTXT_000/string_0512"},
	"shop.purchase.success":               {[]string{"%s", "%d"}, "legacy.go.remake.cmd.fd2.main.l4468-c17"},
	"shop.sell.success":                   {[]string{"%02X", "%d"}, "legacy.go.remake.cmd.fd2.main.l4506-c15"},
	"shop.purchase.equip_prompt":          {[]string{}, "legacy.go.remake.cmd.fd2.main.l4555-c15"},
	"shop.recipient.none":                 {[]string{}, "legacy.go.remake.cmd.fd2.main.l4576-c13"},
	"shop.purchase.equip_prompt.title":    {[]string{"%s"}, "legacy.go.remake.cmd.fd2.main.l8864-c24"},
	"shop.purchase.equip_prompt.controls": {[]string{}, "legacy.go.remake.cmd.fd2.main.l8865-c24"},
	"shop.sell.item_title":                {[]string{"%s"}, "legacy.go.remake.cmd.fd2.main.l8874-c24"},
	"shop.item.equipped_label":            {[]string{}, "legacy.go.remake.cmd.fd2.main.l8883-c14"},
	"shop.sell.roster_title":              {[]string{}, "legacy.go.remake.cmd.fd2.main.l8891-c23"},
	"shop.sell.inventory_count":           {[]string{"%s", "%d"}, "legacy.go.remake.cmd.fd2.main.l8898-c24"},
	"shop.purchase.recipient_title":       {[]string{"%s"}, "legacy.go.remake.cmd.fd2.main.l8905-c23"},
	"shop.purchase.recipient_inventory":   {[]string{"%s", "%d"}, "legacy.go.remake.cmd.fd2.main.l8912-c24"},
	"shop.panel.title_controls":           {[]string{"%d"}, "legacy.go.remake.cmd.fd2.main.l8919-c22"},
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

func (g *Game) localizedSpellResult(actor, spell string, hitN, total int, healing bool, missN int) (string, bool) {
	key := "battle.spell.result_damage"
	if healing {
		key = "battle.spell.result_heal"
	}
	message, ok := g.localeMessage(key, actor, spell, hitN, total)
	if !ok {
		return "", false
	}
	if missN == 0 {
		return message, true
	}
	suffix, ok := g.localeMessage("battle.spell.miss_suffix", missN)
	if !ok {
		return "", false
	}
	return message + suffix, true
}

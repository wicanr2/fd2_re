package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/localization"
)

var systemInfoCaptionKeys = [8]string{"map", "turn", "enemy", "friend", "npc", "currency", "victory", "defeat"}

func init() {
	for key, contract := range systemInfoLocaleContract() {
		officialLocaleContract[key] = contract
	}
}

func systemInfoLocaleContract() map[string]localeEntryContract {
	out := make(map[string]localeEntryContract)
	for _, key := range systemInfoCaptionKeys {
		out["battle.info."+key] = localeEntryContract{[]string{}, "FDOTHER_005/system-info/" + key}
	}
	for index := 597; index < 657; index++ {
		out[fmt.Sprintf("battle.info.condition.%04d", index)] = localeEntryContract{[]string{}, fmt.Sprintf("FDTXT_000/string_%04d", index)}
	}
	return out
}

// The index comes from the original producer, including chapter16's party
// branch. Check the frozen words before looking up a translation by that ID.
func (g *Game) localizeSystemInfo(p *modernHDSystemInfo, index int, catalog *localization.Catalog) error {
	if catalog == nil || index < 597 || index > 655 || index%2 != 1 || g.nativeSystemInfoAssets == nil {
		return errors.New("system info locale: incomplete source or catalog")
	}
	var captions [8]string
	var lines [2][]string
	for key, contract := range systemInfoLocaleContract() {
		entry, ok := catalog.Entries[key]
		if !ok || entry.SourceStringID != contract.sourceID || len(entry.Variables) != 0 {
			return fmt.Errorf("system info locale: invalid provenance for %s", key)
		}
	}
	for i, key := range systemInfoCaptionKeys {
		text, err := catalog.Format("battle.info." + key)
		if err != nil || strings.TrimSpace(text) == "" || strings.ContainsAny(text, "\r\n") {
			return fmt.Errorf("system info locale: invalid caption %s: %v", key, err)
		}
		captions[i] = text
	}
	for i := range lines {
		words, err := g.nativeSystemInfoAssets.Numbers.Strings.Words(index + i)
		if err != nil || !reflect.DeepEqual(words, p.input.Lines[i]) {
			return errors.New("system info locale: source words differ")
		}
		text, err := catalog.Format(fmt.Sprintf("battle.info.condition.%04d", index+i))
		if err != nil || strings.TrimSpace(text) == "" || strings.ContainsRune(text, '\r') {
			return fmt.Errorf("system info locale: invalid condition %d: %v", index+i, err)
		}
		lines[i] = strings.Split(text, "\n")
		if len(lines[i]) > 2 {
			return errors.New("system info locale: more than two rows")
		}
		for _, row := range lines[i] {
			if strings.TrimSpace(row) == "" {
				return errors.New("system info locale: empty row")
			}
		}
	}
	if p.captions != captions || !reflect.DeepEqual(p.lines, lines) {
		p.captions, p.lines, p.picture = captions, lines, nil
	}
	return nil
}

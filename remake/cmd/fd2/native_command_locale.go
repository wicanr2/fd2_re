package main

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/localization"
)

const (
	localizedNativeCommandNameWidth = 140
	localizedNativeCommandDrawWidth = 138
	localizedNativeCommandMinScale  = 0.6
)

func localizedSpellBook(entities *localization.EntityCatalog, source []battle.Spell) ([]battle.Spell, error) {
	if entities == nil || len(source) != 36 {
		return nil, errors.New("localized command catalog or spell table is unavailable")
	}
	result := append([]battle.Spell(nil), source...)
	for i := range result {
		if result[i].ID == 31 {
			result[i].Name = ""
			continue
		}
		name, err := entities.CommandName(result[i].ID)
		if err != nil {
			return nil, err
		}
		result[i].Name = name
	}
	return result, nil
}

func (g *Game) localizedNativeCommandLabel(commandID int) (string, float64, error) {
	if g == nil || g.localeEntities == nil || g.font == nil {
		return "", 0, errors.New("localized native command UI is unavailable")
	}
	name, err := g.localeEntities.CommandName(commandID)
	if err != nil {
		return "", 0, err
	}
	scale := 1.0
	for scale >= localizedNativeCommandMinScale && g.font.Width(name, scale) > localizedNativeCommandDrawWidth {
		scale -= 0.02
	}
	if scale < localizedNativeCommandMinScale {
		return "", 0, fmt.Errorf("localized command %d %q exceeds native grid cell", commandID, name)
	}
	return name, scale, nil
}

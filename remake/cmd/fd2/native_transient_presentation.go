package main

import (
	"errors"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
)

// beginNativeTransientPhases preflights every indexed expiry frame before it
// publishes the private raw/typed transaction. Missing original assets keep
// both the countdown and its player-visible feedback unchanged.
func (g *Game) beginNativeTransientPhases(selectors []byte, then func()) error {
	if g == nil || g.nativeClassUIJob != nil || g.transientUI {
		return errors.New("native transient presentation: another indexed owner is active")
	}
	candidate, expired, err := g.buildNativeTransientPhases(selectors...)
	if err != nil {
		return err
	}
	if len(expired) == 0 {
		*g.st = *candidate
		if then != nil {
			then()
		}
		return nil
	}
	if g.nativeClassUI == nil || len(g.nativeMapVGA) != 320*200 {
		return errors.New("native transient presentation: indexed source assets are unavailable")
	}
	datoPath := nativeDATOPath()
	if datoPath == "" {
		return errors.New("native transient presentation: DATO.DAT is unavailable")
	}
	source := append([]byte(nil), g.nativeMapVGA...)
	frames := make([][]byte, 0, len(expired)*11)
	for _, event := range expired {
		unit, counterIndex, err := nativeTransientPresentationEvent(event)
		if err != nil {
			return err
		}
		portraits, err := dato.DecodeResource(filepath.Clean(datoPath), unit.BattleFig)
		if err != nil || len(portraits) == 0 {
			return errors.New("native transient presentation: DATO portrait is unavailable")
		}
		final, err := campaign.ComposeNativeTransientExpiryFrame(
			source, g.nativeClassUI.dialogue, portraits[0],
			g.nativeClassUI.strings, g.nativeClassUI.font, counterIndex,
		)
		if err != nil {
			return err
		}
		opening, err := campaign.NativeClassListOpeningFrames(source, final)
		if err != nil {
			return err
		}
		closing, err := campaign.NativeClassListClosingFrames(source, final)
		if err != nil {
			return err
		}
		frames = append(frames, opening...)
		frames = append(frames, closing...)
	}
	*g.st = *candidate
	g.msg = ""
	g.transientUI = true
	g.nativeClassUIJob = &nativeClassUIJob{
		frames:  frames,
		restore: source,
		after: func() {
			g.transientUI = false
			if then != nil {
				then()
			}
		},
	}
	return nil
}

func nativeTransientPresentationEvent(event battle.NativeTransientExpiry) (*battle.Unit, int, error) {
	if event.Unit == nil || !event.Unit.HasBattleFig {
		return nil, 0, errors.New("native transient presentation: raw DATO selector is unavailable")
	}
	counterIndex := event.Offset - battle.NativeTransientOffset
	if counterIndex < 0 || counterIndex >= battle.NativeTransientCount {
		return nil, 0, errors.New("native transient presentation: raw counter offset is invalid")
	}
	return event.Unit, counterIndex, nil
}

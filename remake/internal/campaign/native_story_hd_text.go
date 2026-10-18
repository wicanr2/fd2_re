package campaign

import "fmt"

// NativeStoryVisibleGlyph describes presentation only. It uses the same typed
// tokens, glyph clock and scroll passes as ComposeNativeStoryDialogueProgressiveFrames.
type NativeStoryVisibleGlyph struct {
	Token   string
	X, Y    int
	GlyphID int // -1 when the typed layout does not bind an original variant.
}

func NativeStoryDialogueVisibleGlyphs(layout *NativeDialogueLayout, page, progress int) ([]NativeStoryVisibleGlyph, error) {
	if err := layout.Validate(); err != nil {
		return nil, err
	}
	if page < 0 || page >= len(layout.Pages) || progress < 0 {
		return nil, fmt.Errorf("campaign: invalid visible dialogue page or progress")
	}
	x, y, _, step, rows, err := NativeStoryDialogueTextGeometry(layout.Control)
	if err != nil {
		return nil, err
	}
	// Previous FFFD pages retain their rows. Their clocks have already elapsed.
	budget := progress
	for p := 0; p < page; p++ {
		clock, err := NativeStoryDialogueGlyphSteps(layout, p)
		if err != nil {
			return nil, err
		}
		budget += len(clock) - 1
	}
	var result []NativeStoryVisibleGlyph
	row := 0
	for p := 0; p <= page; p++ {
		for r, line := range layout.Pages[p] {
			visibleRow := row
			if row >= rows {
				shifts := [...]int{3, 3, 3, 3, 3, 4}
				pass := 0
				for tick := 1; tick <= nativeStoryScrollFrames; tick++ {
					if budget == 0 {
						return result, nil
					}
					budget--
					for pass < len(shifts)*tick/nativeStoryScrollFrames {
						for i := range result {
							result[i].Y -= shifts[pass]
						}
						pass++
					}
				}
				visibleRow = rows - 1
			}
			tokens, err := layout.glyphTokens(p, r, line)
			if err != nil {
				return nil, err
			}
			for column, token := range tokens {
				if budget == 0 {
					return result, nil
				}
				budget--
				id := -1
				if len(layout.GlyphIDs) != 0 {
					id = layout.GlyphIDs[p][r][column]
				}
				result = append(result, NativeStoryVisibleGlyph{Token: token, X: x + column*nativeStoryGlyphStep, Y: y + visibleRow*step, GlyphID: id})
			}
			row++
		}
	}
	if budget != 0 {
		return nil, fmt.Errorf("campaign: dialogue progress exceeds glyph clock")
	}
	return result, nil
}

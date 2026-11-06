package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"math"
)

// Zero is the absent-field default. Explicit JSON values must be approved scales.
type modernHDVisualScale int

func (s *modernHDVisualScale) UnmarshalJSON(raw []byte) error {
	var value int
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return errors.New("modern HD visual scale: null is not a scale")
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	if value != 100 && value != 115 {
		return errors.New("modern HD visual scale: unsupported percentage")
	}
	*s = modernHDVisualScale(value)
	return nil
}

func (s modernHDVisualScale) valid() bool { return s == 0 || s == 100 || s == 115 }

func (s modernHDVisualScale) rect(rect image.Rectangle, foot int) image.Rectangle {
	if s == 0 || s == 100 || rect.Empty() {
		return rect
	}
	w := int(math.RoundToEven(float64(rect.Dx()) * float64(s) / 100))
	h := int(math.RoundToEven(float64(rect.Dy()) * float64(s) / 100))
	x := int(math.RoundToEven(float64(rect.Min.X+rect.Max.X-w) / 2))
	y := int(math.RoundToEven(float64(rect.Min.Y+foot) - float64(foot)*float64(h)/float64(rect.Dy())))
	return image.Rect(x, y, x+w, y+h)
}

func (s *modernStoryPortraitSet) mapSpriteVisualRect(group, frame int, rect image.Rectangle) image.Rectangle {
	if s == nil || s.mapSpriteVisualScale[group] == 0 || s.mapSpriteVisualScale[group] == 100 {
		return rect
	}
	return s.mapSpriteVisualScale[group].rect(rect, s.mapSpriteFeet[group][frame])
}

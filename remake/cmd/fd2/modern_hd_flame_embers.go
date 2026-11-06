package main

import (
	"errors"
	"image"
	"image/draw"
	"math"
)

const modernHDFlameEmberSourceSHA = "8a7d7237c90f3564ec277142c4e65a1639bde1336bea70172fedef2de67054f8"
const modernHDFlameEmberKey = "flame-ember/" + modernHDFlameEmberSourceSHA + "/212,30,226,46/20x24-nearest"

var modernHDFlameEmberCrop = image.Rect(212, 30, 226, 46)

// 只有呈現狀態；讀取原有repeat，不持有或推進另一份時鐘。
type modernHDFlameEmberState struct {
	Tick int
}

func modernHDFlameEmberTick(j *nativeCommand29PresentationJob) (int, bool) {
	if j == nil || j.plan == nil || j.plan.CommandID != 29 || j.hd == nil || j.hd.effectID != 104 ||
		j.preludeFrame < len(j.preludeFrames) || j.transitionFrame >= 0 || j.frame < 14 || j.frame > 16 ||
		j.schedule.TargetStart != 11 || j.schedule.TargetImpactFrame != 14 || len(j.schedule.FrameDelays) != 18 ||
		j.repeat < 0 || j.repeat >= j.schedule.FrameDelays[j.frame] {
		return 0, false
	}
	tick := j.repeat
	for i := j.schedule.TargetStart; i < j.frame; i++ {
		tick += j.schedule.FrameDelays[i]
	}
	return tick, true
}

func modernHDFlameEmberLayers(set *modernStoryPortraitSet, state *modernHDFlameEmberState) ([]modernHDBattleLayer, error) {
	if set == nil || state == nil {
		return nil, nil
	}
	if state.Tick < 6 || state.Tick > 17 {
		return nil, errors.New("modern HD flame embers: invalid native cursor")
	}
	entry := set.battleAnimations[104]
	// 舊主題及其他已登錄美術保持原狀，不從未知構圖猜裁切。
	if entry == nil || len(entry.hashes) <= 14 || entry.hashes[14] != modernHDFlameEmberSourceSHA {
		return nil, nil
	}
	picture, err := modernBattleHDFrame(set.battleAnimations, 104, 14)
	if err != nil {
		return nil, err
	}
	if picture.Bounds() != image.Rect(0, 0, 830, 857) {
		return nil, errors.New("modern HD flame embers: verified source geometry changed")
	}
	if entry.flameEmberSprite == nil {
		crop := image.NewNRGBA(image.Rect(0, 0, modernHDFlameEmberCrop.Dx(), modernHDFlameEmberCrop.Dy()))
		draw.Draw(crop, crop.Bounds(), picture, modernHDFlameEmberCrop.Min, draw.Src)
		// 先做一次明示最近鄰取樣；每格不再上傳或重取樣小火星。
		sprite := image.NewNRGBA(image.Rect(0, 0, 20, 24))
		for y := 0; y < 24; y++ {
			for x := 0; x < 20; x++ {
				sx, sy := (2*x+1)*crop.Bounds().Dx()/40, (2*y+1)*crop.Bounds().Dy()/48
				sprite.SetNRGBA(x, y, crop.NRGBAAt(sx, sy))
			}
		}
		entry.flameEmberSprite = sprite
	}
	layers := make([]modernHDBattleLayer, 22)
	for i := range layers {
		age := float64((state.Tick+i*13)%54) / 54
		x := int(math.Round(32 + float64((i*73)%760) + math.Sin(age*4+float64(i))*14))
		y := int(math.Round(886 - age*float64(660+(i%3)*60)))
		alpha := float32(1 - age)
		layers[i] = modernHDBattleLayer{HD: entry.flameEmberSprite, HDKey: modernHDFlameEmberKey,
			Rect: image.Rect(x, y, x+20, y+24), Fill: -1, Opacity: &alpha}
	}
	return layers, nil
}

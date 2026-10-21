package battlepresent

import "github.com/wicanr2/fd2_re/remake/internal/figani"

// NativeCommandSources 保存同次合成已消費的來源，不重新規劃狀態或亂數。
// 圖層順序固定為 Before、可選 actor、target、After。
type NativeCommandSources struct {
	Base                                                   []byte
	ActorFrame                                             int
	TargetIndex, TargetFrame, TargetOffsetX, TargetOffsetY int
	Before, After                                          []NativeCommandEffectSource
	Phase                                                  string
}

type NativeCommandEffectSource struct{ Frame, X, Y int }

func nativeCommand3SourceLayers(layers []figani.NativeCommand3Layer) []NativeCommandEffectSource {
	out := make([]NativeCommandEffectSource, 0, len(layers))
	for _, layer := range layers {
		out = append(out, NativeCommandEffectSource{Frame: layer.Frame, X: layer.X, Y: layer.Y})
	}
	return out
}

func nativeCommand7SourceLayers(layers []figani.NativeCommand7Layer) []NativeCommandEffectSource {
	out := make([]NativeCommandEffectSource, 0, len(layers))
	for _, layer := range layers {
		out = append(out, NativeCommandEffectSource{Frame: layer.Frame, X: layer.X})
	}
	return out
}

func nativeCommand8SourceLayers(layers []figani.NativeCommand8Layer) []NativeCommandEffectSource {
	out := make([]NativeCommandEffectSource, 0, len(layers))
	for _, layer := range layers {
		out = append(out, NativeCommandEffectSource{Frame: layer.Frame})
	}
	return out
}

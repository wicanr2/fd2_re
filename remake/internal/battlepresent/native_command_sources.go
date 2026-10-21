package battlepresent

import "github.com/wicanr2/fd2_re/remake/internal/figani"

// NativeCommandSources 保存同次合成已消費的來源，不重新規劃狀態或亂數。
// 預設順序為 Before、可選actor、target、After；明示旗標可前置或省略target。
type NativeCommandSources struct {
	Base                                                   []byte
	ActorFrame                                             int
	TargetIndex, TargetFrame, TargetOffsetX, TargetOffsetY int
	Before, After                                          []NativeCommandEffectSource
	Phase                                                  string
	TargetBeforeEffects, OmitTarget                        bool
	PackedTargetShader                                     int
}

type NativeCommandEffectSource struct{ Frame, X, Y int }

func nativeCommand6SourceLayers(layers []figani.NativeCommand6Layer) []NativeCommandEffectSource {
	out := make([]NativeCommandEffectSource, 0, len(layers))
	for _, layer := range layers {
		out = append(out, NativeCommandEffectSource{Frame: layer.Frame, X: layer.X, Y: layer.Y})
	}
	return out
}

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

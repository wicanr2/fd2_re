package fdother

// PendingCode1PresentationKind identifies one observed sub_22E5C operation.
type PendingCode1PresentationKind string

const (
	PendingCode1StopBGM        PendingCode1PresentationKind = "stop_bgm"
	PendingCode1PreparePalette PendingCode1PresentationKind = "palette_prepare"
	PendingCode1ClearScreen    PendingCode1PresentationKind = "clear_indexed_320x200"
	PendingCode1DrawFrame      PendingCode1PresentationKind = "draw_frame"
	PendingCode1FadeIn         PendingCode1PresentationKind = "palette_fade_in"
	PendingCode1WaitTick       PendingCode1PresentationKind = "wait_tick"
	PendingCode1Release        PendingCode1PresentationKind = "release"
)

// PendingCode1PresentationStep keeps the raw sub_22E5C presentation order
// editable without inventing a high-level scene name or chapter binding.
type PendingCode1PresentationStep struct {
	Kind       PendingCode1PresentationKind
	Frame      int
	Count      int
	DurationMS int
}

// NativePendingCode1PresentationPlan 保留 0x22E5C 固定呈現順序；正式敗北
// owner 依 READY 規格接線，其他 pending producer 不自動取得敗北語意。
func NativePendingCode1PresentationPlan() []PendingCode1PresentationStep {
	return []PendingCode1PresentationStep{
		{Kind: PendingCode1StopBGM},
		{Kind: PendingCode1WaitTick, Count: 1},
		{Kind: PendingCode1PreparePalette},
		{Kind: PendingCode1ClearScreen},
		{Kind: PendingCode1DrawFrame, Frame: 0},
		{Kind: PendingCode1FadeIn, Count: 65, DurationMS: 2},
		{Kind: PendingCode1WaitTick, Count: 9},
		{Kind: PendingCode1DrawFrame, Frame: 1},
		{Kind: PendingCode1WaitTick, Count: 36},
		{Kind: PendingCode1Release},
	}
}

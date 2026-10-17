package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// nativeSpawnIntroJob owns sub_32999's twelve observable presentation passes.
// The following 0x1366A call remains the next independent compiled Beat.
type nativeSpawnIntroJob struct {
	schedule []fdother.NativeSpawnIntroStep
	work     []byte
	vga      []byte
	frames   [][]byte
	palette  color.Palette
	sound    []byte
	frame    int
	drawn    bool
	then     func()
}

func cloneStoryUnitPointers(source []battle.Unit) []*battle.Unit {
	values := append([]battle.Unit(nil), source...)
	pointers := make([]*battle.Unit, len(values))
	for i := range values {
		pointers[i] = &values[i]
	}
	return pointers
}

func cloneBattleUnitPointers(source []*battle.Unit) []*battle.Unit {
	clones := make([]*battle.Unit, len(source))
	for i, unit := range source {
		if unit == nil {
			continue
		}
		value := *unit
		clones[i] = &value
	}
	return clones
}

func cloneNativeBattleSpawnState(source *battle.State) (*battle.State, error) {
	if source == nil || source.W <= 0 || source.H <= 0 || len(source.Roster) == 0 ||
		source.NativeMapSelectorCache == nil || source.NativeMapSelectorError != nil {
		return nil, errors.New("native battle spawn lacks a complete runtime roster/selector state")
	}
	candidate := *source
	candidate.Units = cloneBattleUnitPointers(source.Units)
	candidate.Roster = cloneBattleUnitPointers(source.Roster)
	candidate.NativeMapSelectorCache = source.NativeMapSelectorCache.Clone()
	candidate.NativeCompositionEventBytes = append([]byte(nil), source.NativeCompositionEventBytes...)
	return &candidate, nil
}

func (g *Game) bindNativeFutureItemRows(state *battle.State) error {
	if state == nil {
		return errors.New("native future constructor state unavailable")
	}
	rows, err := battle.LoadNativeItemEffectRowPrefix(
		assetPath("assets/data/native_item_effect_rows.json"),
	)
	if err != nil {
		return err
	}
	return state.BindNativeFutureItemRows(rows)
}

// prepareNativeStorySpawn executes the proven future-group placement against
// private clones. The caller commits the returned values only after every
// visual pass has preflighted successfully.
func (g *Game) prepareNativeStorySpawn(group int, rawGate byte) ([]battle.Unit, []battle.Unit, error) {
	if g.m == nil || g.m.W <= 0 || g.m.H <= 0 || len(g.storyRoster) == 0 ||
		len(g.storyCompositionEventBytes) != g.m.W*g.m.H || g.storySpawned == nil || g.storySpawned[group] {
		return nil, nil, fmt.Errorf("native story spawn group %d lacks unmaterialized FDFIELD state", group)
	}
	active := cloneStoryUnitPointers(g.storyActors)
	roster := cloneStoryUnitPointers(g.storyRoster)
	state := &battle.State{
		W: g.m.W, H: g.m.H, Roster: roster,
		NativeCompositionEventBytes: append([]byte(nil), g.storyCompositionEventBytes...),
	}
	if err := g.bindNativeFutureItemRows(state); err != nil {
		return nil, nil, fmt.Errorf("native story spawn item rows: %w", err)
	}
	if err := state.AppendNativeMapSelectorBatch(active); err != nil {
		return nil, nil, fmt.Errorf("native story spawn existing roster: %w", err)
	}
	if n, err := state.AppendGroupWithNativePlacement(group, rawGate); err != nil {
		return nil, nil, err
	} else if n <= 0 {
		return nil, nil, fmt.Errorf("native story spawn group %d appended no units", group)
	}
	resultActors := make([]battle.Unit, len(state.Units))
	for i, unit := range state.Units {
		resultActors[i] = *unit
	}
	resultRoster := make([]battle.Unit, len(state.Roster))
	for i, unit := range state.Roster {
		resultRoster[i] = *unit
	}
	return resultActors, resultRoster, nil
}

func (g *Game) buildNativeSpawnIntroJob(
	oldInput, candidateInput indexedmap.NativeTransitionFrameInput,
	oldCount int,
	then func(),
) (*nativeSpawnIntroJob, error) {
	if g.spawnIntroTransition != nil || g.indexedTransition != nil {
		return nil, errors.New("native spawn-intro transition already active")
	}
	if !nativeSpawnIntroAssetsAvailable(g.nativeMapAssets) || len(g.sfxSpawnIntro) == 0 {
		return nil, errors.New("native spawn-intro visual/audio assets unavailable")
	}
	if oldCount < 0 || oldCount > len(candidateInput.Units) || len(oldInput.Units) != oldCount {
		return nil, errors.New("native spawn-intro old/new roster boundary unavailable")
	}

	work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(work, oldInput); err != nil {
		return nil, fmt.Errorf("native spawn-intro baseline terrain: %w", err)
	}
	if err := indexedmap.RedrawNativeUnitPresentObjects(work, oldInput); err != nil {
		return nil, fmt.Errorf("native spawn-intro baseline objects: %w", err)
	}
	snapshot, err := indexedmap.SeedNativeSpawnIntroSnapshot(work)
	if err != nil {
		return nil, err
	}
	vga := make([]byte, indexedmap.NativeMapVGASize)
	schedule := fdother.NativeSpawnIntroSchedule()
	input := indexedmap.NativeSpawnIntroFrameInput{Frame: candidateInput, OldUnitCount: oldCount}
	frames := make([][]byte, 0, len(schedule))
	for _, step := range schedule {
		if err := indexedmap.ComposeNativeSpawnIntroPass(
			work, vga, snapshot, input,
			g.nativeMapAssets.SpawnIntro[step.LMIEntry], step,
		); err != nil {
			return nil, fmt.Errorf("native spawn-intro preflight pass %d: %w", step.Pass, err)
		}
		frames = append(frames, append([]byte(nil), vga...))
	}
	if len(frames) != fdother.NativeSpawnIntroPassCount {
		return nil, errors.New("native spawn-intro preflight did not produce twelve frames")
	}
	return &nativeSpawnIntroJob{
		schedule: schedule, work: work, vga: append([]byte(nil), frames[0]...),
		frames: frames, palette: append(color.Palette(nil), g.nativeMapAssets.Palette...),
		sound: append([]byte(nil), g.sfxSpawnIntro...), then: then,
	}, nil
}

func (g *Game) startNativeSpawnIntro(group int, rawGate byte, then func()) error {
	oldCount := len(g.storyActors)
	candidateActors, candidateRoster, err := g.prepareNativeStorySpawn(group, rawGate)
	if err != nil {
		return err
	}
	oldInput, err := g.buildNativeIndexedTransitionInputForActors(g.storyActors)
	if err != nil {
		return fmt.Errorf("native spawn-intro baseline input: %w", err)
	}
	candidateInput, err := g.buildNativeIndexedTransitionInputForActors(candidateActors)
	if err != nil {
		return fmt.Errorf("native spawn-intro candidate input: %w", err)
	}
	job, err := g.buildNativeSpawnIntroJob(oldInput, candidateInput, oldCount, then)
	if err != nil {
		return err
	}

	// Commit constructor state only after the full renderer transaction has
	// succeeded. Native mutates the unit array before pass 0, so the blocked
	// presentation owns this already-constructed roster until completion.
	g.storyActors = candidateActors
	g.storyRoster = candidateRoster
	g.storySpawned[group] = true
	g.spawnIntroTransition = job
	return nil
}

type nativeBattleIntroEvidence struct {
	eventID, group, actingResource int
	spawnSource, actingSource      string
}

var nativeBattleIntroEvidenceTable = []nativeBattleIntroEvidence{
	{eventID: 1, group: 4, actingResource: 3, spawnSource: "0x342ce", actingSource: "0x342e7"},
	{eventID: 2, group: 5, actingResource: 4, spawnSource: "0x34336", actingSource: "0x3434f"},
}

// nativeBattleIntroCall admits only the two proven global callers. The raw
// addresses remain authoritative; editable metadata may describe the calls,
// but it cannot silently redefine faithful-mode executable provenance.
func nativeBattleIntroCall(action battle.Action) (battle.NativeSpawnCall, bool, error) {
	introCount := 0
	for _, call := range action.NativeSpawns {
		if call.Via == "spawn_group_with_intro" {
			introCount++
		}
	}
	if introCount == 0 {
		return battle.NativeSpawnCall{}, false, nil
	}
	if action.Type != "spawn_group" || introCount != 1 || len(action.NativeSpawns) != 1 ||
		len(action.Groups) != 1 || action.NativeEventID == nil || action.Camp != "enemy" || action.ActImmediately {
		return battle.NativeSpawnCall{}, false, errors.New("native battle intro action differs from the proven single-call event")
	}
	call := action.NativeSpawns[0]
	if call.RawPlacementGate == nil || *call.RawPlacementGate != 0 || call.FollowingActing == nil ||
		call.Group != action.Groups[0] {
		return battle.NativeSpawnCall{}, false, errors.New("native battle intro call lacks exact gate/group/acting metadata")
	}
	for _, evidence := range nativeBattleIntroEvidenceTable {
		if *action.NativeEventID == evidence.eventID && call.Group == evidence.group &&
			strings.EqualFold(call.Source, evidence.spawnSource) &&
			call.FollowingActing.Resource == evidence.actingResource &&
			strings.EqualFold(call.FollowingActing.Source, evidence.actingSource) {
			return call, true, nil
		}
	}
	return battle.NativeSpawnCall{}, false, errors.New("native battle intro provenance differs from event1/event2 callers")
}

func (g *Game) loadNativeBattleFollowingActing(
	meta *battle.NativeFollowingActing,
	unitCount int,
) ([]campaign.ActingFrame, error) {
	if g.sc == nil || meta == nil || g.sc.NativeActingResources == "" {
		return nil, errors.New("native battle following acting resource set unavailable")
	}
	resources, err := campaign.LoadActingResourceSet(assetPath(g.sc.NativeActingResources))
	if err != nil {
		return nil, fmt.Errorf("native battle acting resources: %w", err)
	}
	frames, ok := resources[meta.Resource]
	if !ok || len(frames) == 0 {
		return nil, fmt.Errorf("native battle acting resource %d unavailable", meta.Resource)
	}
	for frameIndex, frame := range frames {
		if frame.Beats < 0 || frame.Beats > 0x7f || (!frame.Special && frame.Beats == 0) || len(frame.Units) == 0 {
			return nil, fmt.Errorf("native battle acting resource %d frame %d has invalid timing/targets", meta.Resource, frameIndex)
		}
		seen := map[int]bool{}
		for _, unit := range frame.Units {
			if unit.Slot == nil || *unit.Slot < 0 || *unit.Slot >= unitCount || unit.Pose < 0 || unit.Pose > 3 || seen[*unit.Slot] {
				return nil, fmt.Errorf("native battle acting resource %d frame %d has invalid runtime slot/pose", meta.Resource, frameIndex)
			}
			seen[*unit.Slot] = true
		}
	}
	return frames, nil
}

func (g *Game) startNativeBattleFollowingActing(frames []campaign.ActingFrame, then func()) {
	job := &actPoseJob{acting: frames, then: then}
	g.actJob = job
	g.beginActingFrame(job)
}

func (g *Game) startNativeBattleSpawnIntro(
	action battle.Action,
	call battle.NativeSpawnCall,
	then func(),
) error {
	if g.st == nil || call.RawPlacementGate == nil || call.FollowingActing == nil {
		return errors.New("native battle intro state/call unavailable")
	}
	oldCount := len(g.st.Units)
	candidate, err := cloneNativeBattleSpawnState(g.st)
	if err != nil {
		return err
	}
	appended, err := candidate.AppendGroupWithNativePlacement(call.Group, byte(*call.RawPlacementGate))
	if err != nil {
		return err
	}
	if appended <= 0 || len(candidate.Units) != oldCount+appended {
		return errors.New("native battle intro candidate append boundary unavailable")
	}
	// #243 READY：32999沿10EED保留raw camp及+5=0，新敵軍在同phase行動。
	// Camp／Acted已由原生constructor投影，不再套用正規化ActImmediately。
	acting, err := g.loadNativeBattleFollowingActing(call.FollowingActing, len(candidate.Units))
	if err != nil {
		return err
	}
	oldInput, err := g.buildNativeIndexedTransitionInputForState(g.st)
	if err != nil {
		return fmt.Errorf("native battle intro baseline input: %w", err)
	}
	candidateInput, err := g.buildNativeIndexedTransitionInputForState(candidate)
	if err != nil {
		return fmt.Errorf("native battle intro candidate input: %w", err)
	}
	job, err := g.buildNativeSpawnIntroJob(
		oldInput, candidateInput, oldCount,
		func() { g.startNativeBattleFollowingActing(acting, then) },
	)
	if err != nil {
		return err
	}

	// Publish only the fields changed by the proven constructor. Existing unit
	// pointers remain stable; newly appended records and the selector cache come
	// from the fully preflighted private state.
	newUnits := cloneBattleUnitPointers(candidate.Units[oldCount:])
	g.st.Units = append(g.st.Units, newUnits...)
	g.st.Roster = cloneBattleUnitPointers(candidate.Roster)
	g.st.NativeMapSelectorCache = candidate.NativeMapSelectorCache
	g.st.NativeMapSelectorError = candidate.NativeMapSelectorError
	g.st.NativeMapCycleState = candidate.NativeMapCycleState
	g.st.HasNativeMapCycleState = candidate.HasNativeMapCycleState
	g.st.NativeTerrainPhaseState = candidate.NativeTerrainPhaseState
	g.st.HasNativeTerrainPhaseState = candidate.HasNativeTerrainPhaseState
	g.st.NativeTerrainFlipState = candidate.NativeTerrainFlipState
	g.st.NativeUnitPixelShiftState = candidate.NativeUnitPixelShiftState
	g.st.HasNativeMapBinaryTimingState = candidate.HasNativeMapBinaryTimingState
	g.spawnIntroTransition = job
	return nil
}

func (g *Game) stepNativeSpawnIntro() {
	j := g.spawnIntroTransition
	if j == nil || !j.drawn {
		return
	}
	if j.frame+1 < len(j.frames) {
		j.frame++
		copy(j.vga, j.frames[j.frame])
		j.drawn = false
		if j.schedule[j.frame].SoundResource == fdother.NativeSpawnIntroSoundResource {
			g.playRaw(j.sound)
		}
		return
	}
	then := j.then
	g.nativeMapWork = append(g.nativeMapWork[:0], j.work...)
	g.nativeMapVGA = append(g.nativeMapVGA[:0], j.vga...)
	g.spawnIntroTransition = nil
	if then != nil {
		then()
	}
}

func (g *Game) drawNativeSpawnIntro(screen *ebiten.Image) bool {
	j := g.spawnIntroTransition
	if j == nil || len(j.vga) != indexedmap.NativeMapVGASize || len(j.palette) != 256 {
		return false
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), j.palette)
	copy(img.Pix, j.vga)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	j.drawn = true
	return true
}

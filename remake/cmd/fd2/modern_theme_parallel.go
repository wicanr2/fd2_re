package main

import (
	"errors"
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
)

// 主題包載入的分組並行（#317）。下列各類的載入函式只寫自己的欄位（另外可能設 hasHD），也不讀
// 其他角色的結果，所以各在一個 goroutine 依清冊順序載入到私有的暫存集合，全部成功後併回；其餘
// 角色仍在呼叫端依原順序載入。開場、地形、天空之鑰與戰鬥背景載入時只核對檔案與 PNG 標頭、不
// 解碼（#317 後續），剩下的時間多半是讀檔與 SHA-256；戰鬥畫格、HD 地圖人物與設施場景也分組，
// 呼叫端那一條就不再是最長的一條。
//
// 每一條載入流都依清冊順序處理，遇錯即停；回報的錯誤取清冊索引最小的那一個，和逐項依序
// 載入時回報的錯誤相同。任一項失敗整包拒收，不併回任何暫存結果。

// modernThemeParallelRoles 是分組並行載入的角色，值是分組名稱。
var modernThemeParallelRoles = map[string]string{
	modernOpeningRole:         "opening",
	"native_terrain_hd_set":   "terrain",
	"story_animation_hd_set":  "sky-key",
	"battle_surface_hd":       "battle-surface",
	"battle_animation_hd_set": "battle-animation",
	"battle_effect_hd_set":    "battle-effect",
	"map_sprite_hd_set":       "map-sprite-hd",
	"facility_scene_hd":       "facility-scene",
}

func loadModernThemeGroupedAsset(set *modernStoryPortraitSet, asset modernThemeAsset, root string) error {
	switch asset.Role {
	case modernOpeningRole:
		return loadModernOpeningHDAsset(set, asset, root)
	case "native_terrain_hd_set":
		return loadModernTerrainHDAsset(set, asset, root)
	case "story_animation_hd_set":
		return loadModernSkyKeyHDAsset(set, asset, root)
	case "battle_surface_hd":
		return loadModernBattleSurfaceHD(set, asset, root)
	case "battle_animation_hd_set", "battle_effect_hd_set":
		if err := loadModernBattleHDAsset(set, asset, root); err != nil {
			return err
		}
		set.hasHD = true
		return nil
	case "map_sprite_hd_set":
		return loadModernHDMapSpriteAsset(set, asset, root)
	case "facility_scene_hd":
		return loadModernFacilityScene(set, asset, root)
	}
	return errors.New("modern theme catalog: role is not grouped")
}

// mergeModernThemeGroup 把分組的暫存結果併回主集合；各組角色互斥，主集合沒有這些欄位。
func mergeModernThemeGroup(set, scratch *modernStoryPortraitSet) {
	if scratch.openingHD != nil {
		set.openingHD = scratch.openingHD
	}
	if scratch.terrainHD != nil {
		set.terrainHD, set.terrainLazy = scratch.terrainHD, scratch.terrainLazy
	}
	if scratch.skyKeyFD34 != nil {
		set.skyKeyFD34 = scratch.skyKeyFD34
	}
	if scratch.skyKeyANI != nil {
		set.skyKeyANI = scratch.skyKeyANI
	}
	if scratch.battleSurfaces != nil {
		set.battleSurfaces, set.surfaceCache = scratch.battleSurfaces, scratch.surfaceCache
	}
	if scratch.battleAnimations != nil {
		set.battleAnimations = scratch.battleAnimations
	}
	if scratch.battleEffects != nil {
		set.battleEffects = scratch.battleEffects
	}
	if scratch.mapSpritesHD != nil {
		set.mapSpritesHD = scratch.mapSpritesHD
	}
	if scratch.mapSpriteVisualScale != nil {
		set.mapSpriteVisualScale = scratch.mapSpriteVisualScale
	}
	if scratch.facilityScenes != nil {
		set.facilityScenes = scratch.facilityScenes
	}
	set.hasHD = set.hasHD || scratch.hasHD
}

// modernThemeVisualScaleAllowed 是清冊對 visual_scale_percent 的限制，分組與呼叫端共用。
func modernThemeVisualScaleAllowed(asset modernThemeAsset) bool {
	return asset.VisualScalePercent.valid() && (asset.VisualScalePercent == 0 || asset.Role == "map_sprite_hd_set" || asset.Role == "battle_animation_hd_set")
}

// modernThemeLoadWorkersOverride 只給量測測試指定載入流數；0 表示依處理器數決定。
var modernThemeLoadWorkersOverride int

// modernThemeLoadWorkers 是同時執行的載入流數（含呼叫端那一條）：最多用一半的處理器、
// 最多三條，留給同時在跑的遊戲畫面；只有一條時退化成依序載入。三條是以真實 HD 包在 4 處理器
// 限制下量第一次 F2 的結果：四條沒有更快，單幀最長反而變長（#317）。
func modernThemeLoadWorkers() int {
	if modernThemeLoadWorkersOverride > 0 {
		return modernThemeLoadWorkersOverride
	}
	n := runtime.GOMAXPROCS(0) / 2
	if n < 1 {
		n = 1
	}
	if n > 3 {
		n = 3
	}
	return n
}

type modernThemeAssetGroup struct {
	name        string
	indexes     []int
	scratch     *modernStoryPortraitSet
	failedIndex int
	err         error
}

type modernThemeAssetGroups struct {
	groups    []*modernThemeAssetGroup
	wg        sync.WaitGroup
	slots     chan struct{}
	minFailed atomic.Int64 // 目前已知最早失敗的清冊索引
	done      atomic.Int64
	total     int
	progress  func(done, total int)
	notices   *modernThemeNotices
}

// startModernThemeAssetGroups 啟動分組載入。呼叫端的主迴圈佔一個名額，直到 wait 才釋放。
// notices 是主集合的延後檢查訊息；各組的暫存集合共用它，延後檢查拒收時才會提示到玩家。
func startModernThemeAssetGroups(assets []modernThemeAsset, root string, progress func(done, total int), notices *modernThemeNotices) *modernThemeAssetGroups {
	userDataDir() // assetPath 會讀這個快取；先在呼叫端填好，goroutine 只讀
	gs := &modernThemeAssetGroups{total: len(assets), progress: progress, notices: notices, slots: make(chan struct{}, modernThemeLoadWorkers())}
	gs.minFailed.Store(math.MaxInt64)
	byName := map[string]*modernThemeAssetGroup{}
	for index, asset := range assets {
		name, ok := modernThemeParallelRoles[asset.Role]
		if !ok {
			continue
		}
		group := byName[name]
		if group == nil {
			group = &modernThemeAssetGroup{name: name}
			byName[name] = group
			gs.groups = append(gs.groups, group)
		}
		group.indexes = append(group.indexes, index)
	}
	// 較大的組先排，名額有限時總時間較短；回報的錯誤仍依清冊索引決定。
	sort.SliceStable(gs.groups, func(i, j int) bool { return len(gs.groups[i].indexes) > len(gs.groups[j].indexes) })
	gs.slots <- struct{}{}
	for _, group := range gs.groups {
		group := group
		gs.wg.Add(1)
		go func() {
			defer gs.wg.Done()
			gs.slots <- struct{}{}
			defer func() { <-gs.slots }()
			gs.run(group, assets, root)
		}()
	}
	return gs
}

func (gs *modernThemeAssetGroups) run(group *modernThemeAssetGroup, assets []modernThemeAsset, root string) {
	scratch := &modernStoryPortraitSet{notices: gs.notices}
	for _, index := range group.indexes {
		if gs.failedBefore(index) {
			return
		}
		asset := assets[index]
		err := error(nil)
		if !modernThemeVisualScaleAllowed(asset) {
			err = errors.New("modern theme catalog: visual scale requires an HD character asset")
		} else {
			err = loadModernThemeGroupedAsset(scratch, asset, root)
		}
		if err != nil {
			group.failedIndex, group.err = index, err
			gs.fail(index)
			return
		}
		gs.advance()
	}
	group.scratch = scratch
}

// failedBefore 回報清冊中比 index 更早的項目是否已經失敗；是的話後面的錯誤不會被回報，可以停。
func (gs *modernThemeAssetGroups) failedBefore(index int) bool {
	return gs.minFailed.Load() < int64(index)
}

func (gs *modernThemeAssetGroups) fail(index int) {
	for {
		old := gs.minFailed.Load()
		if int64(index) >= old || gs.minFailed.CompareAndSwap(old, int64(index)) {
			return
		}
	}
}

func (gs *modernThemeAssetGroups) advance() {
	n := gs.done.Add(1)
	if gs.progress != nil {
		gs.progress(int(n), gs.total)
	}
}

// wait 釋放主迴圈的名額、等所有組結束，回報清冊索引最小的錯誤；都成功才併回暫存結果。
func (gs *modernThemeAssetGroups) wait(set *modernStoryPortraitSet, mainIndex int, mainErr error) error {
	<-gs.slots
	gs.wg.Wait()
	bestIndex, best := mainIndex, mainErr
	for _, group := range gs.groups {
		if group.err != nil && (best == nil || group.failedIndex < bestIndex) {
			bestIndex, best = group.failedIndex, group.err
		}
	}
	if best != nil {
		return best
	}
	for _, group := range gs.groups {
		if group.scratch == nil {
			return errors.New("modern theme catalog: grouped load stopped without an error")
		}
		mergeModernThemeGroup(set, group.scratch)
	}
	return nil
}

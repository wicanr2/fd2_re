//! Go 來源：`remake/cmd/fd2/native_story_map.go`（182 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 6 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

impl Game {
    /// 第十二章的原生聚焦沿 0x12CEA 重繪，不能只發布六個視圖全域。
    /// 已證實範圍見 fd2_terrain_mode3_review_20261001.json story_frame_continuity。
    ///
    /// Go：`native_story_map.go:13`，`func (g *Game) nativeCh12StoryFocus() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ch12_story_focus(&self) -> bool {
        todo!("native_story_map.go:nativeCh12StoryFocus")
    }

    /// Go：`native_story_map.go:19`，`func (g *Game) nativeCh12StoryWorkHandoff(unitsPath, scenarioPath string) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_ch12_story_work_handoff(&self, units_path: &str, scenario_path: &str) -> bool {
        todo!("native_story_map.go:nativeCh12StoryWorkHandoff")
    }

    /// materializeNativeStoryMapState 建立 LOADCH 場景專用的索引地圖載體。
    /// 場景沿用原始 terrain/unit constructor 與 raw view，但明確不顯示戰鬥 HUD；
    /// 它不會把 cutscene 單位升格成可操作的 battle.State。
    ///
    /// Go：`native_story_map.go:32`，`func (g *Game) materializeNativeStoryMapState(source *battle.State) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn materialize_native_story_map_state(&mut self, source: Option<&battle::State>) -> Result<()> {
        todo!("native_story_map.go:materializeNativeStoryMapState")
    }

    /// appendNativeStoryMapActors 在 SPAWN 後延續同一 first-seen selector cache。
    /// storyActors append 可能搬移底層陣列，所以成功後一併重建全部 unit pointers。
    ///
    /// Go：`native_story_map.go:99`，`func (g *Game) appendNativeStoryMapActors(start int) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn append_native_story_map_actors(&mut self, start: i64) -> Result<()> {
        todo!("native_story_map.go:appendNativeStoryMapActors")
    }

    /// 出場預檢只改候選角色與私有載體。成功後須把兩者一起發布，否則
    /// 原生底圖及HD移動會讀到出場前的角色名冊。
    ///
    /// Go：`native_story_map.go:119`，`func (g *Game) prepareNativeStoryMapSpawnState(actors []battle.Unit, oldCount int) (*battle.State, error)`
    pub(crate) fn prepare_native_story_map_spawn_state(
        &self,
        actors: &[battle::Unit],
        old_count: i64,
    ) -> Result<Option<battle::State>> {
        todo!("native_story_map.go:prepareNativeStoryMapSpawnState")
    }

    /// Go：`native_story_map.go:138`，`func (g *Game) composeNativeStoryMapFrame() error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn compose_native_story_map_frame(&mut self) -> Result<()> {
        todo!("native_story_map.go:composeNativeStoryMapFrame")
    }
}

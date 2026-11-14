//! Go 來源：`remake/cmd/fd2/main.go`（13492 行）。
//!
//! 炎龍騎士團2 重製 — Go/Ebiten 垂直切片(MVP)。
//!
//! 目標:證明「Go/Ebiten 跑得起來,且讀得到我們逆向出的資料」。
//! 本切片:載入一張戰場(tileset PNG + 地圖 JSON)→ 用 hi-res 畫布渲染 →
//!
//!     方向鍵 / WASD / 觸控移動游標,相機跟隨。桌面 / Web(WASM)/ 手機共用。
//!
//! 資產(玩家自備原版後由 tools/ 產生,不隨庫散布):
//!
//!     assets/tileset.png  一張 24×24 圖塊的網格圖(cols 欄)
//!     assets/map.json     {"w","h","tileW","tileH","cols","tiles":[地形索引...]}
//!
//! 建置:見 remake/README.md(docker golang;WASM / 桌面 / 手機)。
//!
//! 骨架：Go 檔的 236 個函式中產生 154 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// hi-res 內部畫布(CJK/觀感原則:拉畫布、別縮字)
pub(crate) const LOGICAL_W: i64 = 640;

pub(crate) const LOGICAL_H: i64 = 400;

/// storyZoom:story 場景(cutscene)世界層放大倍率。原版視窗固定 13×8 格
/// (312×192px @320×200,doc25 0x11eee),remake 戰場自訂 640 寬 FOV 一屏裝下整廳,
/// 走入/運鏡完全失去意義(使用者 2-1);story 場景世界層 2×(48px/格,視野 13.3×8.3 格)
/// 用來保留原版 13×8 格附近的局部取景與長廊運鏡手感；完整鏡頭／HUD
/// 同狀態 E2 仍由 UI 證據矩陣逐場驗收，不能由倍率常數自行宣稱還原。
pub(crate) const STORY_ZOOM: i64 = 2;

/// MapData 對應 assets/map.json(由 tools/export_engine_assets.py 產生)。
///
/// Go：`main.go:67`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct MapData {
    /// Go 標籤：`json:"w"`
    pub(crate) w: i64,
    /// Go 標籤：`json:"h"`
    pub(crate) h: i64,
    /// Go 標籤：`json:"tileW"`
    pub(crate) tile_w: i64,
    /// Go 標籤：`json:"tileH"`
    pub(crate) tile_h: i64,
    /// tileset 每列圖塊數
    ///
    /// Go 標籤：`json:"cols"`
    pub(crate) cols: i64,
    /// Go 標籤：`json:"tiles"`
    pub(crate) tiles: Vec<i64>,
    /// Native fields are optional: legacy PNG-only maps must not be treated as
    /// verified indexed input. They mirror export_engine_assets.py's raw FDFIELD
    /// composition bytes and FDSHAP four-byte control table.
    ///
    /// Go 標籤：`json:"native_composition_event_bytes,omitempty"`
    pub(crate) native_composition_event_bytes: Vec<u8>,
    /// Go 標籤：`json:"native_tile_blit_modes,omitempty"`
    pub(crate) native_tile_blit_modes: Vec<u8>,
    /// Go 標籤：`json:"native_terrain_control,omitempty"`
    pub(crate) native_terrain_control: Vec<u8>,
}

/// Go：`main.go:82`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub struct Game {
    pub(crate) balance: Option<battle::BalanceOptions>,
    pub(crate) balance_ui: BalanceEditor,
    pub(crate) m: Option<MapData>,
    /// all original map HUD resources, nil on any missing/malformed asset
    pub(crate) native_map_assets: Option<NativeMapAssets>,
    /// 本次 loadMap 已原子採用現代圖集
    pub(crate) modern_map_tileset_loaded: bool,
    /// 目前地圖的忠實原版 PNG；F2 只重建圖層，不改戰況
    pub(crate) base_map_tileset: Option<Arc<Picture>>,
    /// 目前地圖 selector；-1 表示尚未載入
    pub(crate) current_map_id: i64,
    /// persistent 456-stride original tactical framebuffer
    pub(crate) native_map_work: Vec<u8>,
    /// persistent 320x200 indexed VGA surface
    pub(crate) native_map_vga: Vec<u8>,
    /// Published with the successful steady snapshot.
    pub(crate) native_map_frame_source: Option<indexedmap::NativeFrameInput<'static>>,
    pub(crate) modern_hd_map_frame: Option<ModernHDMapFrame>,
    pub(crate) modern_hd_map_gpu: Option<ModernHDMapGPU>,
    pub(crate) modern_hd_map_hud_art: Option<ModernHDMapHUDArt>,
    pub(crate) modern_hd_battle_gpu: Option<ModernHDBattleGPU>,
    pub(crate) modern_hd_battle_visible: bool,
    pub(crate) modern_hd_world_visible: bool,
    pub(crate) run_log: RunLogTrace,
    pub(crate) modern_hd_map_composing: bool,
    pub(crate) modern_hd_map_presented_frame: i64,
    pub(crate) modern_hd_map_presented: Option<ModernHDMapFrame>,
    pub(crate) modern_hd_timed_dialogue_visible: bool,
    pub(crate) modern_hd_timed_dialogue_phase: campaign::NativeDialogueFramePhase,
    pub(crate) modern_hd_timed_dialogue_index: i64,
    pub(crate) modern_hd_map_terrain: Vec<u8>,
    pub(crate) modern_hd_map_terrain_assets: Option<NativeMapAssets>,
    pub(crate) modern_hd_map_terrain_state: Option<battle::State>,
    /// current 256xRGB six-bit DAC state for handler palette ramps
    pub(crate) native_map_dac: Vec<u8>,
    /// 2BC9A 的 FIGANI sound banks，獨立於完整 command bank cache。
    pub(crate) native_physical_sound_banks: BTreeMap<i64, BTreeMap<i64, Vec<u8>>>,
    /// exact 0x1f882/0x1f525 indexed DAC presentation
    pub(crate) native_palette_ramp: Option<NativePaletteRampJob>,
    pub(crate) native_town_departure: Option<NativeTownDepartureJob>,
    pub(crate) native_town_load_pending: bool,
    pub(crate) native_defeat: Option<NativeDefeatJob>,
    pub(crate) native_chapter_result: Option<NativeChapterResultJob>,
    pub(crate) native_result_matched_rules: Vec<String>,
    pub(crate) title_needs_new_campaign: bool,
    /// exact 0x35E5A 0..63/hold/62..0 indexed DAC presentation
    pub(crate) native_palette_pulse: Option<NativePalettePulseJob>,
    /// raw ch20 post 0x24336 fixed FDOTHER/ANI/palette sequence
    pub(crate) native_ch20_sky_key: Option<NativeCh20SkyKeyJob>,
    /// raw ch23 staging/latch/timer state shared across both handler loops
    pub(crate) native_ch23_state: Option<NativeCh23AdapterState>,
    /// blocking raw ch23 indexed presentation loop
    pub(crate) native_ch23_loop: Option<NativeCh23LoopJob>,
    /// blocking raw ch22 post 0x2189A ten-pass presentation
    pub(crate) native2189_a: Option<Native2189AJob>,
    /// blocking shared 0x22253 11+6+bridge+10 indexed presentation
    pub(crate) native_unit_present: Option<NativeUnitPresentJob>,
    /// blocking 0x1DB65 13+6+6 indexed presentation
    pub(crate) native_ch28_post_present: Option<NativeCh28PostPresentJob>,
    /// pre-settlement +5, keyed by stable runtime slot
    pub(crate) native_action_death_before: BTreeMap<i64, u8>,
    /// atomic FDFIELD69/FDSHAP46/47/FDOTHER42 tail transaction
    pub(crate) native_ch22_reload: Option<NativeCh22ReloadState>,
    /// process-lifetime 0x4DFCC phase projection (0..15)
    pub(crate) native_fdother_palette_phase: i64,
    /// 最近一次 nativeMapHUDInput 失敗即關閉的原因，只供診斷訊息
    pub(crate) native_map_hud_input_reason: String,
    /// process-lifetime 0x4DFCC unsigned BIOS low-word snapshot
    pub(crate) native_fdother_palette_tick: i64,
    /// exact 0x11DF2(0,255,255) overlay for legacy RGB scenes
    pub(crate) native_full_dac_white: bool,
    /// exact ch07 post 0x11D40(0,255,64)+mode-13h clear
    pub(crate) native_full_dac_black: bool,
    /// gate A save-persistent；anchor process-persistent
    pub(crate) native_map_hud_persistent: battle::NativeMapHUDPersistentState,
    pub(crate) tileset: Option<Arc<Picture>>,
    /// 切好的圖塊
    pub(crate) tiles: Vec<Option<Arc<Picture>>>,
    /// 戰鬥狀態(單位)
    pub(crate) st: Option<battle::State>,
    /// battle-local 18.2065Hz BIOS low-word adapter
    pub(crate) native_map_clock: NativeBIOSClock,
    /// title-lifetime 18.2065Hz timer seed for normal CONTINUE
    pub(crate) native_title_clock: NativeBIOSClock,
    /// 劇本(事件系統,doc 29)
    pub(crate) sc: Option<battle::Scenario>,
    /// 待顯示對話(事件產生,含說話者)
    pub(crate) dialog: Vec<battle::DialogLine>,
    /// 場景背景模式(story 節點指定 Map):鏡頭固定不跟游標,不畫單位/游標/HUD(doc23 §4)
    pub(crate) story_bg: bool,
    /// 原版目前已 materialize 的 scene unit array；index 只在該 load/spawn 時序內有意義
    pub(crate) story_actors: Vec<battle::Unit>,
    /// LOADCH 保留的 FDFIELD records；SPAWN 按 group 順序 append 到 storyActors
    pub(crate) story_roster: Vec<battle::Unit>,
    /// LOADCH 的 immutable FDFIELD composition +2；future-group placement 的原始輸入
    pub(crate) story_composition_event_bytes: Vec<u8>,
    /// 原版 group 已 materialize；防止 handler 重複 SPAWN 時重複 append
    pub(crate) story_spawned: BTreeMap<i64, bool>,
    /// 最近一次 handler LOADCH 的 exact roster source；battle handoff gate
    pub(crate) story_roster_path: String,
    /// 最近一次 handler LOADCH 的 exact party scenario；battle handoff gate
    pub(crate) story_party_scenario: String,
    /// JOIN 建立的永久玩家名冊；key=原版 0..31 charID，不使用 NPC portrait
    pub(crate) party_members: BTreeMap<i64, bool>,
    /// JOIN 首次出現順序；章0 cutscene 的 party runtime slot 以此為準
    pub(crate) party_join_order: Vec<i64>,
    /// 0x11506 戰後同步的跨關角色能力／HP／MP／經驗快照
    pub(crate) party_roster: BTreeMap<i64, battle::Unit>,
    /// preparation 0x318ad 的本戰出擊勾選；不改永久 JOIN 名冊
    pub(crate) party_deploy: BTreeMap<i64, bool>,
    /// preparation UI 角色順序（JOIN chronology）
    pub(crate) prep_ids: Vec<i64>,
    /// preparation UI 游標
    pub(crate) prep_sel: i64,
    /// preparation UI 原版出擊上限（15，末段 19）
    pub(crate) prep_limit: i64,
    /// 已通過前置確認，且流程要求進入原版選人階段
    pub(crate) prep_selecting: bool,
    /// 0x2CCB6→0x30012 的非城鎮四槽保存
    pub(crate) prep_record_slots: bool,
    /// 0x30550 的0..3槽游標
    pub(crate) prep_record_slot_sel: i64,
    /// 選滿或小隊確認後的最終出戰確認階段
    pub(crate) prep_confirm: bool,
    /// 原生必出角色拒收訊息，確認後返回城鎮
    pub(crate) prep_required_missing: Option<i64>,
    /// 0=肯定，1=取消
    pub(crate) prep_confirm_sel: i64,
    /// preparation 0x31e80→0x1297d 的 BIOS 低字來源
    pub(crate) prep_clock: NativeBIOSClock,
    /// 原版 [0x53c0b] 0..3；繪圖時 3 正規化為1
    pub(crate) prep_idle_cycle: i64,
    /// 原版 [0x53c0f] 有號 BIOS 低字 latch
    pub(crate) prep_last_tick: i64,
    /// 固定雜湊截圖專用；一般玩家路徑永不設定
    pub(crate) prep_shot_cycle_frozen: bool,
    /// 0x1956b 前的 town 畫面或 0x2cc04 黑色來源
    pub(crate) prep_prompt_source: Vec<u8>,
    pub(crate) prep_town_source: Option<NativeTownFrameSource>,
    pub(crate) prep_dialogue_hd: Option<ModernHDPreparationDialogue>,
    pub(crate) facility_dialogue_hd: Option<ModernHDFacilityDialogue>,
    pub(crate) shop_comparison_hd: Option<ModernHDShopComparison>,
    pub(crate) facility_item_rows_hd: Option<ModernHDFacilityItemRows>,
    pub(crate) facility_list_roster_hd: Option<ModernFacilityListRoster>,
    /// church service menu cursor (0..3)
    pub(crate) church_sel: i64,
    /// menu / status_* / transfer_* / revive* / class / class_confirm
    pub(crate) church_mode: String,
    /// current church candidate ids
    pub(crate) church_ids: Vec<i64>,
    /// 0x2e6b8 [0x5412f], even six-entry viewport origin
    pub(crate) church_roster_start: i64,
    /// 0x30c22/0x311dc three-row viewport origin
    pub(crate) church_vertical_start: i64,
    /// selected actor passed to 0x17aed
    pub(crate) church_status_id: i64,
    pub(crate) church_status_data: Option<NativeUnitStatusData>,
    /// 0x17eef/0x17fc0 + 0x184c0(actor,-1)
    pub(crate) church_status_panel: Vec<u8>,
    /// 0x17eef/0x17fc0 + 0x1ceed(actor,-1)
    pub(crate) church_command_panel: Vec<u8>,
    /// 0x2df6b even six-entry item viewport origin
    pub(crate) church_item_start: i64,
    /// raw transfer source roster id
    pub(crate) church_transfer_source: i64,
    /// compact source inventory index
    pub(crate) church_transfer_item: i64,
    /// compact source inventory indices
    pub(crate) church_transfer_items: Vec<i64>,
    /// raw destination id used by FDTXT506 FFFC
    pub(crate) church_transfer_dest: i64,
    /// selected 0x30dc3 candidate
    pub(crate) church_revive_id: i64,
    /// level * raw class fee
    pub(crate) church_revive_fee: i64,
    /// selected class-change candidate
    pub(crate) church_class_id: i64,
    pub(crate) church_branches: Vec<campaign::ClassChangeBranch>,
    /// raw 0x2fc85 selector (0..3)
    pub(crate) hotel_sel: i64,
    pub(crate) hotel_route: fdother::NativeHotelServiceRoute,
    pub(crate) hotel_has_route: bool,
    /// title LOAD selector: native 0x30550 slots 0..3
    pub(crate) title_slot_sel: i64,
    pub(crate) class_change_table: campaign::ClassChangeTable,
    pub(crate) class_change_growth: BTreeMap<i64, campaign::ClassChangeGrowth>,
    pub(crate) native_join_constructor: campaign::NativeJoinConstructorTable,
    pub(crate) has_native_join_constructor: bool,
    pub(crate) native_join_bases: campaign::NativeJoinBaseTable,
    pub(crate) has_native_join_bases: bool,
    pub(crate) native_join_item_effect_rows: Vec<u8>,
    /// 原版 [0x53c03]；set_chapter 與無立即數 LOADCH 的 resource chapter
    pub(crate) handler_chapter: i64,
    pub(crate) handler_inherited_map_view: battle::NativeMapViewState,
    pub(crate) has_handler_inherited_map_view: bool,
    /// 場景走位動畫佇列(doc46 §5.3);逐幀推進、完成後移除
    pub(crate) story_walks: Vec<StoryWalkJob>,
    /// story 節點無對白時的自動轉場倒數幀(doc46 行軍蒙太奇,0=不自動)
    pub(crate) story_auto_advance: i64,
    /// story 場景離屏世界層(320×200,放大 storyZoom 倍貼上畫布;2-1 原版取景)
    pub(crate) story_view: Option<Arc<Picture>>,
    /// 本節點:進場走位走完才顯示對白(campaign.Node.WalkFirst)
    pub(crate) walk_first: bool,
    /// 本節點:走位期間鏡頭跟隨走位者(campaign.Node.FollowWalk;beat walk 依 Follow 逐拍設值)
    pub(crate) follow_walk: bool,
    /// 本節點:鏡頭 Y 上限(campaign.Node.CamMaxY;0=不限)
    pub(crate) cam_max_y: f64,
    /// beat「pan」進行中(doc50 §1);storyBG 專用,與 followWalk 互斥
    pub(crate) cam_pan: Option<CamPanJob>,
    /// beat「focus_unit」：依原版 0x12cea 先 X 後 Y 逐格移動游標／鏡頭
    pub(crate) focus_job: Option<FocusUnitJob>,
    /// beat「act」進行中(近似姿態循環,見 actPoseJob 註解)
    pub(crate) act_job: Option<ActPoseJob>,
    /// 60Hz Update 到原版約18.2065Hz來源拍的投影餘數
    pub(crate) act_tick_accumulator: f64,
    /// 目前 cutscene 節點的過場原語序列(doc50 §2)
    pub(crate) beats: Vec<campaign::Beat>,
    /// 目前執行到第幾拍(-1=尚未開始)
    pub(crate) beat_idx: i64,
    /// beat「delay」剩餘幀數(0=非等待中)
    pub(crate) beat_delay: i64,
    /// 戰場事件的阻塞 action 序列；與 campaign BeatRunner 分離
    pub(crate) battle_event: Option<BattleEventRun>,
    /// battle event delay 剩餘幀數
    pub(crate) battle_event_delay: i64,
    /// cutscene 節點載入的章文本(dialog beat 依 Line/Count 取子段)
    pub(crate) camp_lines: Vec<campaign::Line>,
    /// 對話框目前顯示的說話者(dlgNone=無;換人時播縮/展動畫)
    pub(crate) dlg_shown: i64,
    /// 與 dlgShown 同步的上/下框覆蓋(來自 DialogLine.Upper;nil=沿用預設規則)
    pub(crate) dlg_upper: Option<bool>,
    /// 對話框動畫相位:0=常態 1=縮小(換人前收合) 2=展開
    pub(crate) dlg_phase: i64,
    /// 對話框動畫相位內計時(幀)
    pub(crate) dlg_t: i64,
    /// 目前FFFD頁碼；頁內第4個FFFE邏輯列由原版三列窗口捲動，不另造頁
    pub(crate) dlg_page: i64,
    /// 分頁捲動剩餘幀數(0=靜止)
    pub(crate) dlg_scroll_t: i64,
    /// 分頁捲動開始頁碼
    pub(crate) dlg_scroll_from: i64,
    /// caller-specific 0x15F84 stable indexed pages
    pub(crate) native_dialogue_frames: Vec<Vec<u8>>,
    /// 每頁 frame0框／頭像，之後逐字形發布
    pub(crate) native_dialogue_progressive: Vec<Vec<Vec<u8>>>,
    /// 每頁完整文字＋DATO frame3 indexed overlay
    pub(crate) native_dialogue_mouth_open: Vec<Vec<u8>>,
    /// 目前頁已發布的逐字 frame；-1尚未開始
    pub(crate) native_dialogue_progress: i64,
    /// 原版 0x53A10，跨句保留
    pub(crate) native_dialogue_speaking_cycle: i64,
    /// 原版 0x53A14，跨句保留
    pub(crate) native_dialogue_speaking_half: i64,
    pub(crate) native_dialogue_speaking_frame: i64,
    pub(crate) native_dialogue_portraits: Vec<dato::Frame>,
    pub(crate) native_dialogue_layout: Option<campaign::NativeDialogueLayout>,
    pub(crate) native_dialogue_chrome: Option<ModernHDDialogueChrome>,
    pub(crate) native_dialogue_chrome_bounds: BTreeMap<campaign::NativeDialogueFramePhase, Vec<Rectangle>>,
    pub(crate) native_dialogue_glyph_steps: Vec<Vec<bool>>,
    /// caller-specific sub_165AC五階段格網
    pub(crate) native_dialogue_opening: Vec<Vec<u8>>,
    /// caller-specific sub_16B43 snapshot restore＋可選游標尾段
    pub(crate) native_dialogue_closing: Vec<Vec<u8>>,
    pub(crate) native_dialogue_closing_t: i64,
    pub(crate) native_dialogue_closing_live: bool,
    /// 場景淡出/淡入轉場(doc46 §5.2)
    pub(crate) fade: Option<StoryFade>,
    pub(crate) transition_reveal: Option<TransitionRevealJob>,
    pub(crate) indexed_transition: Option<NativeIndexedTransitionJob>,
    pub(crate) native_heal_presentation: Option<NativeCommandHealPresentationJob>,
    pub(crate) native_modifier_presentation: Option<NativeCommandModifierPresentationJob>,
    pub(crate) native_ai_command_modifier: Option<NativeAICommandModifierPresentationJob>,
    /// 0x15311 route being dispatched; owns the winner cell [0x53C27]/[0x53C2B]
    pub(crate) native_ai_action_plan: Option<battle::AIPlan>,
    /// 下一次 enterNode 進城鎮是 hub 子場景返回：保留 [0x5412B] 選擇器
    pub(crate) native_town_hub_return: bool,
    /// 對拍測試用：每個 AI 單位計畫產生後、執行前呼叫（可對齊 RNG）
    pub(crate) ai_plan_observer: Option<Box<dyn FnMut(&mut Game, &battle::AIPlan) + Send>>,
    /// 對拍測試明示事件決策種子；正式執行為nil。
    pub(crate) native_turn_pair_rng_input:
        Option<Box<dyn FnMut(&mut Game, battle::NativeTurnEvent, u16) -> Result<u16> + Send>>,
    /// 對拍測試用：每次升級成長擲骰（0x1E54A）前對齊 RNG
    pub(crate) growth_roll_observer: Option<Box<dyn FnMut(&mut Game, UnitIdx, u16) -> u16 + Send>>,
    pub(crate) native_cmd0_presentation: Option<NativeCommand0PresentationJob>,
    pub(crate) native_cmd1_presentation: Option<NativeCommand1PresentationJob>,
    pub(crate) native_cmd2_presentation: Option<NativeCommand2PresentationJob>,
    pub(crate) native_cmd3_presentation: Option<NativeCommand3PresentationJob>,
    pub(crate) native_cmd5_presentation: Option<NativeCommand5PresentationJob>,
    pub(crate) native_cmd6_presentation: Option<NativeCommand6PresentationJob>,
    pub(crate) native_cmd7_presentation: Option<NativeCommand7PresentationJob>,
    pub(crate) native_cmd8_presentation: Option<NativeCommand8PresentationJob>,
    pub(crate) native_cmd9_player: Option<NativeCommand9PlayerJob>,
    pub(crate) native_cmd9_ai_presentation: Option<NativeCommand9AIPresentationJob>,
    pub(crate) native_cmd1012: Option<NativeCommand1012Job>,
    pub(crate) native_cmd24_presentation: Option<NativeCommand24PresentationJob>,
    pub(crate) native_cmd29_presentation: Option<NativeCommand29PresentationJob>,
    pub(crate) native_cmd32_presentation: Option<NativeCommand32PresentationJob>,
    pub(crate) native_cmd33_presentation: Option<NativeCommand33PresentationJob>,
    pub(crate) native_cmd34_presentation: Option<NativeCommand34PresentationJob>,
    pub(crate) native_cmd35_presentation: Option<NativeCommand35PresentationJob>,
    pub(crate) native_ai_item_presentation: Option<NativeAIItemPresentationJob>,
    pub(crate) spawn_intro_transition: Option<NativeSpawnIntroJob>,
    pub(crate) native_turn_staging: Option<NativeTurnStagingJob>,
    pub(crate) native_field_event61: Option<NativeFieldEvent61Job>,
    /// direct 0x13FD4 indexed/audio owner
    pub(crate) native_ai_idle_recovery: Option<NativeAIIdleRecoveryJob>,
    /// 0x1A30B two-pass indexed owner
    pub(crate) native_end_turn_recovery: Option<NativeEndTurnRecoveryJob>,
    /// FD2_ENDING_PREFIX 或來源約束 campaign ending；缺原始資料時走明示 fallback
    pub(crate) native_ending: Option<NativeEndingPreview>,
    /// 可重播的呈現時鐘；nil時使用正常wall-clock。
    pub(crate) native_ending_now: Option<Box<dyn FnMut(&mut Game) -> GameInstant + Send>>,
    /// 原始素材不足或來源約束終局無法發布時的玩家提示
    pub(crate) ending_notice: String,
    /// 移動動畫(沿路徑逐格走,FDICON 方向幀)
    pub(crate) walk: Option<WalkAnim>,
    /// 劇本節點圖(doc 19;FD2_CAMPAIGN 啟用)
    pub(crate) camp: Option<campaign::Runner>,
    /// choice 節點游標
    pub(crate) camp_sel: i64,
    /// 開頭動畫/主選單(title.go,doc23)
    pub(crate) title_assets: Option<TitleAssets>,
    /// "scroll"→"menu"→""(進遊戲)
    pub(crate) title_phase: String,
    /// 正式標題素材失敗時阻擋輸入，禁止落入預載第一關
    pub(crate) startup_blocked: bool,
    /// 捲動來源列(535→0)
    pub(crate) scroll_y: f64,
    pub(crate) title_sel: i64,
    pub(crate) title_flash: i64,
    pub(crate) title_tick: i64,
    /// 開場 AFM 過場(title.go cutscene phase)
    pub(crate) cut_idx: i64,
    pub(crate) cut_frame: i64,
    pub(crate) cut_tick: i64,
    pub(crate) cut_cur: Vec<Option<Arc<Picture>>>,
    /// radial 指令環(原版 [0x3C57]:↑0=攻擊/←1=法術/→2=物品/↓3=待機)
    pub(crate) ring: bool,
    pub(crate) ring_sel: i64,
    pub(crate) action_overlay_phase: String,
    pub(crate) action_overlay_frame: i64,
    pub(crate) action_overlay_after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) action_overlay_drawn: bool,
    pub(crate) action_overlay_shot_hold: bool,
    /// fallback only: 0上=攻擊 1左=法術 2右=物品 3下=待機
    pub(crate) ring_icons: [Option<Arc<Picture>>; 4],
    /// F1 玩家操作說明；顯示時暫停遊戲輸入
    pub(crate) help_visible: bool,
    /// FDOTHER#2 的完整 78 格；只取自使用者提供的原始資料
    pub(crate) native_action_cells: Vec<Option<Arc<Picture>>>,
    pub(crate) native_ui_palette: ColorPalette,
    pub(crate) native_class_ui: Option<NativeClassUIAssets>,
    pub(crate) native_load_slots_ui: Option<NativeLoadSlotsUIAssets>,
    pub(crate) native_class_ui_job: Option<NativeClassUIJob>,
    pub(crate) native_player_status: Option<NativePlayerStatusState>,
    pub(crate) native_player_focus: Option<battle::Cell>,
    pub(crate) native_player_focus_started: bool,
    /// 0x1A7AB 聚焦在 0x1A30B 裡；閘 B 要等 0x17277／0x135D4 返回後才寫回 1
    pub(crate) native_gate_b_after_focus: bool,
    pub(crate) native_next_player_index: i64,
    pub(crate) transient_ui: bool,
    pub(crate) native_class_ui_clock: NativeBIOSClock,
    pub(crate) native_class_ui_pulse: i64,
    pub(crate) native_class_ui_last_tick: i64,
    pub(crate) native_class_ui_has_tick: bool,
    pub(crate) native_town_ui: Option<NativeTownUIAssets>,
    pub(crate) native_town_ui_clock: NativeBIOSClock,
    pub(crate) native_town_ui_pulse: i64,
    pub(crate) native_town_ui_last_tick: i64,
    pub(crate) native_town_ui_has_tick: bool,
    pub(crate) native_preparation_ui: Option<NativePreparationUIAssets>,
    pub(crate) native_church_ui_job: Option<NativeChurchUIJob>,
    pub(crate) native_church_ui_clock: NativeBIOSClock,
    pub(crate) native_church_ui_pulse: i64,
    pub(crate) native_church_ui_last_tick: i64,
    pub(crate) native_church_ui_has_tick: bool,
    pub(crate) native_church_ui_shot_hold: bool,
    pub(crate) native_church_text_index: i64,
    pub(crate) native_shop_ui: Option<NativeShopUIAssets>,
    pub(crate) native_hotel_ui: Option<NativeHotelUIAssets>,
    /// ""＝舊版面；menu／slots／saved／loadslots／loaded／load_blocked
    pub(crate) native_hotel_mode: String,
    pub(crate) native_hotel_prompt_return: bool,
    pub(crate) native_hotel_slot_sel: i64,
    pub(crate) native_shop_ui_job: Option<NativeClassUIJob>,
    pub(crate) native_shop_ui_clock: NativeBIOSClock,
    pub(crate) native_shop_ui_pulse: i64,
    pub(crate) native_shop_ui_last_tick: i64,
    pub(crate) native_shop_ui_has_tick: bool,
    pub(crate) native_shop_variant: i64,
    pub(crate) native_shop_mode: String,
    pub(crate) native_shop_service_sel: i64,
    pub(crate) native_shop_item_start: i64,
    pub(crate) native_shop_confirm_sel: i64,
    pub(crate) native_shop_recipient_start: i64,
    pub(crate) native_shop_recipient_cycle: i64,
    pub(crate) native_shop_equip_sel: i64,
    pub(crate) native_shop_pending_unit: battle::Unit,
    pub(crate) native_shop_has_pending_unit: bool,
    pub(crate) native_shop_pending_gold: i64,
    pub(crate) native_shop_sell_roster_top: i64,
    pub(crate) native_shop_sell_item_top: i64,
    pub(crate) native_shop_sell_confirm_sel: i64,
    pub(crate) native_shop_sell_item_ids: Vec<i64>,
    pub(crate) native_shop_equip_roster_top: i64,
    pub(crate) native_shop_equip_unit_sel: i64,
    pub(crate) native_shop_transfer_source: i64,
    pub(crate) native_shop_transfer_item: i64,
    pub(crate) native_shop_transfer_items: Vec<i64>,
    pub(crate) native_shop_transfer_dest: i64,
    pub(crate) native_shop_transfer_ids: Vec<i64>,
    pub(crate) native_shop_transfer_sel: i64,
    pub(crate) native_shop_transfer_top: i64,
    pub(crate) native_shop_sell_roster_cycle: i64,
    pub(crate) native_command_labels: BTreeMap<i64, String>,
    pub(crate) native_command_open: bool,
    pub(crate) native_command_sel: i64,
    pub(crate) native_command0_targeting: bool,
    pub(crate) native_command_target_id: i64,
    /// nativeContinueOpeningConfirm 只由已驗證的原版 FD2.SAV current-runtime
    /// 發布點設一次。它讓該 E2 錨點的第一個 Return 直接開 action overlay；
    /// 一經消費，後續空游標確認由已證實的共用 0x117E7 owner 處理。
    pub(crate) native_continue_opening_confirm: bool,
    /// nativeCurrentSavePlain is the checksum-validated plaintext baseline from
    /// the exact FD2.SAV that produced the active CONTINUE battle. Current-save
    /// writes preserve its chapter slots and opaque bytes; authored battles never
    /// receive this carrier.
    pub(crate) native_current_save_plain: Vec<u8>,
    /// nativeChapterSlotPlain／nativeChapterSlotBaseline 是標題四槽 LOAD 或正常 START
    /// 首次保存建立的 FD2.SAV plaintext 與槽快照；後續酒店保存保留其 raw 基底。
    pub(crate) native_chapter_slot_plain: Vec<u8>,
    pub(crate) native_chapter_slot_baseline: Option<fdsave::ChapterSlotSnapshot>,
    /// 最近一次酒店存檔寫回原版槽的結果（對拍收據用）
    pub(crate) native_chapter_slot_save_err: Option<GameError>,
    /// 正式 START 後首次保存；無 LOAD 名冊來源。
    pub(crate) native_chapter_slot_started: bool,
    /// 0x18B84 單位資訊視窗（框／bar／digit／姓名）
    pub(crate) native_map_unit_window_assets: Option<battle::NativeItemPanelDataAssets>,
    /// FDOTHER#2 指令環 indexed 原格（整幀合成用）
    pub(crate) native_action_cells_raw: Vec<fdother::RawCell>,
    /// nativeSystemCursorOverlay 對應共用 0x117E7 在 0x12C0D 回傳 -1 時
    /// 呼叫的 0x16F55 空游標面板。direction0／巢狀戰場資訊、direction2／設定
    /// 與 direction3／END 已有 action owner；巢狀 direction3 離場亦已閉合到
    /// main 清理出口；外層 direction1 亦有無事件全軍移動 owner。巢狀存讀檔與
    /// direction1 尚未接線的90-entry事件效果仍維持失敗即關閉。
    pub(crate) native_system_cursor_overlay: bool,
    pub(crate) native_system_nested_open: bool,
    pub(crate) native_system_options_open: bool,
    pub(crate) native_system_options: Option<fdother::NativeSystemOptions>,
    pub(crate) native_system_info_assets: Option<campaign::NativeSystemInfoAssets>,
    pub(crate) native_system_info_ui: Option<NativeSystemInfoUIState>,
    pub(crate) spell_open: bool,
    pub(crate) spell_sel: i64,
    /// native 0x1b932 eight-slot selector; unsupported effect presentations remain fail-closed
    pub(crate) item_open: bool,
    pub(crate) item_sel: i64,
    pub(crate) item_anim_step: i64,
    pub(crate) item_closing: bool,
    pub(crate) pending_native_death_rewards: Vec<PendingNativeDeathReward>,
    /// 0x1AA1D 物品／金錢成功訊息，行動收尾時逐則播
    pub(crate) pending_native_reward_msgs: Vec<PendingNativeDeathRewardMessage>,
    /// 0x1E292 經驗／升級對話，行動收尾時逐則播
    pub(crate) pending_native_level_ups: Vec<PendingNativeLevelUp>,
    pub(crate) native_level_up_dialogue: Option<NativeLevelUpDialogueState>,
    pub(crate) native_death_reward_ui: Option<NativeDeathRewardUIState>,
    pub(crate) native_death_reward_then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) native_item_targeting: bool,
    pub(crate) native_item_target_id: i64,
    pub(crate) native_item_target_raw_slot: i64,
    pub(crate) native_item_relocating: bool,
    pub(crate) native_command23_relocating: bool,
    pub(crate) native_item_relocation_unit: i64,
    pub(crate) native_movement_cost_rows: Vec<Vec<u8>>,
    /// original 0x627b8: initialized to zero, process-lifetime only
    pub(crate) native_rng_state: u16,
    /// original [0x51A8F]: selector 0 field event noted during the walk, dispatched at action end
    pub(crate) native_field_event_pending: Option<NativeFieldEventPending>,
    pub(crate) native_item_panel_text: Option<NativeItemPanelTextData>,
    pub(crate) native_item_panel: Option<Arc<Picture>>,
    pub(crate) native_item_panel_base: Vec<u8>,
    pub(crate) native_item_panel_record: Vec<u8>,
    pub(crate) native_item_panel_assets: Option<battle::NativeItemPanelDataAssets>,
    pub(crate) native_item_effect_rows: Vec<u8>,
    /// 施法目標選擇中
    pub(crate) cast_sp: Option<battle::Spell>,
    pub(crate) spells: Vec<battle::Spell>,
    pub(crate) native_command_book: Vec<battle::NativeCommandRecord>,
    pub(crate) native_command_palette_flash: Option<battle::NativeCommandPaletteFlashTable>,
    pub(crate) native_command_scene: Option<battle::NativeCommandSceneTable>,
    pub(crate) native_command_resistances: BTreeMap<i64, i64>,
    /// growth byte10 learn_idx -> command rows
    pub(crate) command_learn: BTreeMap<i64, Vec<battle::CommandLearnEntry>>,
    /// raw unit+7 -> growth byte10 learn_idx
    pub(crate) command_learn_selectors: BTreeMap<i64, i64>,
    /// native unit+7 selector -> 0x4E4D1 升級成長列
    pub(crate) native_growth_rows: BTreeMap<i64, battle::GrowthRow>,
    /// BGM(doc12 play_bgm 語意:同曲不重播)
    pub(crate) bgm: Option<AudioVoice>,
    pub(crate) bgm_cur: String,
    pub(crate) native_system_bgm_track: String,
    /// 音源設定 "fm"/"mt32"(settings.go;F2 切換)
    pub(crate) bgm_source: String,
    /// 30份分離OGG完整驗證後的唯一BGM路徑來源
    pub(crate) music_catalog: Option<musiccatalog::Catalog>,
    /// F3:開發除錯 HUD(座標/陣營原文等)
    pub(crate) debug: bool,
    /// FD2_APPROXIMATE=1:可玩近似模式；不宣稱原版 handler 等價
    pub(crate) approximate_mode: bool,
    /// 未綁定戰後節點的近似整理提示，等待玩家確認後才進城鎮／整備
    pub(crate) approximate_postbattle: bool,
    /// FD2_UNIT_LABELS=1:cutscene sprite 左上標 [idx]fig+名+座標(協助回報/對映原版 slot)
    pub(crate) unit_labels: bool,
    /// FD2_CUTSCENE_LOG=1:過場 node/beat/走位逐步 log 到 stderr(協助對原版資料比對)
    pub(crate) cutscene_log: bool,
    /// 回合橫幅文字(PLAYER/ENEMY PHASE)
    pub(crate) banner: String,
    /// 橫幅剩餘幀(60 Hz)
    pub(crate) banner_t: i64,
    /// 橫幅進場時凍結的 320×200 索引畫面(對應 0x1F1CC 開頭的 memcpy(0xA0000))
    pub(crate) banner_frame: Vec<u8>,
    /// SFX PCM（doc36 FDOTHER#31：14個目錄項目／13個非空樣本）
    pub(crate) sfx: BTreeMap<i64, Vec<u8>>,
    /// 保留疊播播放器至自然結束，避免 Play 後立即失去生命週期
    pub(crate) sfx_voices: Vec<SfxVoice>,
    /// 25A96的53EE4同通道；地圖返回後、caller續行前停止
    pub(crate) native_physical_voice: SfxVoice,
    /// 已納入契約的 FDOTHER 指令音效；只讀分離 OGG
    pub(crate) separated_command_sfx: BTreeMap<i64, BTreeMap<i64, Vec<u8>>>,
    /// FDOTHER #77 sub2: sub_1F894 H／P 標題選單移動分支
    pub(crate) sfx_title_move: Vec<u8>,
    /// FDOTHER #77 sub1: sub_1F894 標題選單確認分支
    pub(crate) sfx_title_confirm: Vec<u8>,
    /// FDOTHER #78 sub0: ANI #1 第一幀 companion
    pub(crate) sfx_title_ani1: Vec<u8>,
    /// ANI #1 專用 owner；結束／略過時明確停止
    pub(crate) title_ani1_voice: SfxVoice,
    /// 戰鬥揮擊音(doc36 戰鬥池 #48-64 sub0,七池共用)
    pub(crate) sfx_swing: Vec<u8>,
    /// 命中音(近似:最短最尖池;attack_id→sfx 對照表 doc36 未 RE)
    pub(crate) sfx_impact: Vec<u8>,
    /// 陣亡/重擊音(近似:最長池)
    pub(crate) sfx_death: Vec<u8>,
    /// FDOTHER #88 sub1: ch24 transition SFX
    pub(crate) sfx_transition: Vec<u8>,
    /// FDOTHER #80 selector0: 0x1D6C8 command palette owner
    pub(crate) sfx_command_modifier: Vec<u8>,
    /// FDOTHER #53 sub3: selector32 resource98 actor marker
    pub(crate) sfx_command24_actor: Vec<u8>,
    /// FDOTHER #53 sub2: selector32 resource98 target marker
    pub(crate) sfx_command24_target: Vec<u8>,
    /// FDOTHER #82 sub0: 0x2B659 actor raw +4 marker
    pub(crate) sfx_command0_actor: Vec<u8>,
    /// FDOTHER #82 sub1: 0x26152 seven target markers
    pub(crate) sfx_command0_target: Vec<u8>,
    /// FDOTHER #83 sub0: common 0x2B659 actor marker
    pub(crate) sfx_command2_actor: Vec<u8>,
    /// FDOTHER #83 sub1: 0x26528 mode2/8 marker
    pub(crate) sfx_command2_mode2: Vec<u8>,
    /// FDOTHER #83 sub2: 0x26528 mode5 marker
    pub(crate) sfx_command2_mode5: Vec<u8>,
    /// FDOTHER #83 sub3: 0x26528 mode6 marker
    pub(crate) sfx_command2_mode6: Vec<u8>,
    /// FDOTHER #84 sub0: common actor marker
    pub(crate) sfx_command3_actor: Vec<u8>,
    /// FDOTHER #84 sub1: 0x26795 post-counter3／zero frame-base marker
    pub(crate) sfx_command3_sub1: Vec<u8>,
    /// FDOTHER #84 sub2: 0x26795 pre-counter0／nonzero frame-base marker
    pub(crate) sfx_command3_sub2: Vec<u8>,
    /// FDOTHER #86 sub0: common actor marker
    pub(crate) sfx_command5_actor: Vec<u8>,
    /// FDOTHER #86 sub1: channel0／3 handler markers
    pub(crate) sfx_command5_target: Vec<u8>,
    /// FDOTHER #85 sub0: common actor marker
    pub(crate) sfx_command4_actor: Vec<u8>,
    /// FDOTHER #85 sub1: 0x269D3 six-slot markers
    pub(crate) sfx_command4_target: Vec<u8>,
    /// FDOTHER #87 sub0: common 0x2B659 actor marker
    pub(crate) sfx_command6_actor: Vec<u8>,
    /// FDOTHER #87 sub1: command6 mode5 counter markers
    pub(crate) sfx_command6_target: Vec<u8>,
    /// FDOTHER #87 sub2: command6 mode0 before seven front frames
    pub(crate) sfx_command6_front: Vec<u8>,
    /// FDOTHER #87 sub3: command6 mode6 before seven tail frames
    pub(crate) sfx_command6_tail: Vec<u8>,
    /// FDOTHER #88 sub0: common actor marker
    pub(crate) sfx_command7_actor: Vec<u8>,
    /// FDOTHER #88 sub1: 0x272B8 channel0／1 playback handles
    pub(crate) sfx_command7_target: Vec<u8>,
    /// FDOTHER #90 sub0: common actor marker
    pub(crate) sfx_command8_actor: Vec<u8>,
    /// FDOTHER #90 sub1: 0x274B0 pre-counter0 marker
    pub(crate) sfx_command8_sub1: Vec<u8>,
    /// FDOTHER #90 sub2: 0x274B0 pre-counter4 marker
    pub(crate) sfx_command8_sub2: Vec<u8>,
    /// FDOTHER #80 selector0: 0x1D6C8 palette marker
    pub(crate) sfx_command9_player_palette: Vec<u8>,
    /// FDOTHER #80 selector14: 0x1C4CC frame0
    pub(crate) sfx_command9_player_initial: Vec<u8>,
    /// FDOTHER #80 selector15: 0x1C4CC frame15／19
    pub(crate) sfx_command9_player_repeat: Vec<u8>,
    /// FDOTHER #80 selector2: command11／12 0x2189A prelude
    pub(crate) sfx_command1012_prelude: Vec<u8>,
    /// FDOTHER #80 selector13: 0x21548 frame markers
    pub(crate) sfx_command1012_main: Vec<u8>,
    /// FDOTHER #95 sub0: 0x32999 pass1 raw sample（11025Hz 為既有工具鏈推論）
    pub(crate) sfx_spawn_intro: Vec<u8>,
    /// currently loaded handler resource-table id
    pub(crate) handler_resource: i64,
    /// 游標移動音偵測
    pub(crate) prev_cur_x: i64,
    /// Go 與 `prevCurX` 同列宣告，註解見該欄位。
    pub(crate) prev_cur_y: i64,
    /// AI 回合進行中(逐單位行走動畫)
    pub(crate) ai_busy: bool,
    /// 橫幅播完後第一次 aiStep 要跑 0x13536（全記錄 +5 bit7 清零）
    pub(crate) ai_phase_clear_bit7_pending: bool,
    /// 0x13536 之後、敵方 AI 之前要跑 0x1A813(0) 的 selector 0 回合事件
    pub(crate) ai_phase_selector0_pending: bool,
    /// selector 0 回合事件之後、敵方 AI 之前要跑 sub_1A866(0)
    pub(crate) ai_phase_selector0_sweep_pend: bool,
    /// 橫幅之前先跑 0x1D80B 友軍 AI（pass 0）
    pub(crate) ai_ally_phase_pending: bool,
    /// 每個死亡 transition 的 reward 只執行一次
    pub(crate) death_rewarded: BTreeMap<UnitIdx, bool>,
    /// 本次行動擊倒、帶型態 2／3 死亡效果的單位
    pub(crate) pending_death_programs: Vec<PendingDeathProgram>,
    /// 正在執行的死亡程式的擊殺者（0x1AA1D 的第一個參數）
    pub(crate) death_program_killer: Option<UnitIdx>,
    /// 死亡程式執行中；勝敗判定等它跑完
    pub(crate) death_program_running: bool,
    /// 正在分派死亡效果的單位
    pub(crate) death_program_dead: Option<UnitIdx>,
    /// 施法擲骰(FD2_SEED 可固定,headless 重現)
    pub(crate) rng: Option<battle::GoMathRand>,
    /// 金幣(商店)
    pub(crate) gold: i64,
    /// 隊伍道具(名稱;道具效果待實裝)
    pub(crate) items: Vec<String>,
    /// 商店游標
    pub(crate) shop_sel: i64,
    pub(crate) shop_recipient_sel: i64,
    pub(crate) shop_recipients: Vec<i64>,
    pub(crate) shop_picking: bool,
    pub(crate) shop_pending: campaign::Good,
    pub(crate) shop_equip_prompt: bool,
    pub(crate) shop_equip_unit: i64,
    pub(crate) shop_equip_slot: i64,
    pub(crate) shop_item_types: BTreeMap<i64, i64>,
    pub(crate) shop_equip_types: BTreeMap<i64, Vec<i64>>,
    pub(crate) shop_item_prices: BTreeMap<i64, i64>,
    pub(crate) shop_item_stats: BTreeMap<i64, campaign::ItemStats>,
    /// church 0x30dc3 class fee words
    pub(crate) revive_fee_rates: Vec<i64>,
    /// buy or sell
    pub(crate) shop_mode: String,
    pub(crate) shop_sell_picking: bool,
    pub(crate) shop_sell_unit_sel: i64,
    pub(crate) shop_sell_slot_sel: i64,
    /// DATO 頭像:肖像 id → 4 嘴型幀
    pub(crate) portraits: BTreeMap<i64, Vec<Option<Arc<Picture>>>>,
    /// 嘴型動畫狀態(原版 0x16d00:m0閉/m3開)
    pub(crate) mouth_open: bool,
    /// 閉嘴倒數(原版 rand%30+2 tick)
    pub(crate) mouth_timer: i64,
    /// native 0x16d00 cadence adapter
    pub(crate) mouth_state: dato::MouthState,
    /// 完整頁等待期已取初始閉嘴倒數
    pub(crate) native_dialogue_mouth_ready: bool,
    pub(crate) cur_x: i64,
    pub(crate) cur_y: i64,
    pub(crate) cam_x: f64,
    pub(crate) cam_y: f64,
    pub(crate) load_err: String,
    /// nativeSystemEndTurnConfirm 承接共用 0x16F55 的 END 確認生命週期；
    /// 其餘三個 cell 沒有已證實 owner，仍失敗即關閉。
    pub(crate) native_system_end_turn_confirm: bool,
    pub(crate) native_system_end_turn_delay: i64,
    pub(crate) native_system_end_turn_ui: Option<Box<NativeSystemEndTurnUIState>>,
    pub(crate) native_system_exit_requested: bool,
    pub(crate) native_system_group_march: Option<battle::NativeSystemGroupMarchPlan>,
    pub(crate) native_system_group_march_step: i64,
    /// 截圖鉤子(FD2_SHOT=path 啟用):第 shotFrame 幀存 PNG 後自動退出(有界,供無人值守驗證)
    pub(crate) frame: i64,
    pub(crate) shot_path: String,
    pub(crate) shot_frame: i64,
    /// 逐幀截圖目錄(FD2_SHOT_SERIES):戰鬥演出每幀存 frame_NN.png,演出結束自動退出
    pub(crate) shot_series: String,
    /// 截圖前自動推進到第 N 回合(FD2_SHOT_TURN,驗證增援進場)
    pub(crate) shot_turn: i64,
    /// 截圖時把游標放這(FD2_SHOT_CUR=x,y)
    pub(crate) shot_cur_x: i64,
    pub(crate) shot_cur_y: i64,
    /// 截圖前自動選取游標單位(FD2_SHOT_SELECT=1)
    pub(crate) shot_sel: bool,
    /// screenshot setup also must tolerate skipped exact frames
    pub(crate) shot_setup: bool,
    /// frame scheduling may skip an exact number; capture once at-or-after it
    pub(crate) shot_taken: bool,
    /// 選取狀態
    pub(crate) sel: Option<UnitIdx>,
    pub(crate) reach: BTreeMap<battle::Cell, bool>,
    /// 選取單位當下的原始格(ESC 取消移動時退回,playfix #4)
    pub(crate) sel_orig_x: i64,
    /// Go 與 `selOrigX` 同列宣告，註解見該欄位。
    pub(crate) sel_orig_y: i64,
    /// 已選單位是否移動完(進入攻擊階段)
    pub(crate) moved: bool,
    pub(crate) native_move_plan: Option<battle::NativePlayerMovement>,
    pub(crate) movement_panel_hd: Option<ModernHDMovementPanel>,
    pub(crate) movement_panel_draw: Option<ModernHDMovementPanelDraw>,
    pub(crate) native_move_panel: Option<Arc<Picture>>,
    pub(crate) native_move_panel_x: i64,
    /// 勝負:""/win/lose
    pub(crate) result: String,
    /// 短訊息(攻擊傷害等)
    pub(crate) msg: String,
    /// 地圖單位 sprite(FDICON 待機分鏡):fig index → 幀序列
    pub(crate) sprites: BTreeMap<i64, Vec<Option<Arc<Picture>>>>,
    /// 攻擊全身動畫(FIGANI):fig → 幀序列
    pub(crate) figani: BTreeMap<i64, Vec<Option<Arc<Picture>>>>,
    /// 原始 FIGANI descriptor +6 delay，與 PNG 幀數一一對齊
    pub(crate) figani_delays: BTreeMap<i64, Vec<i64>>,
    /// 進行中的攻擊演出
    pub(crate) atk: Option<AtkAnim>,
    /// 戰鬥背景(BG.DAT,by 戰場;map0=BG_004 森林)
    pub(crate) bg: Option<Arc<Picture>>,
    /// 我方腳下台座(TAI.DAT;0x29164 載 0x28c46,doc35 §3.3)
    pub(crate) tai: Option<Arc<Picture>>,
    /// 狀態欄框素材(FDOTHER#5 LMI1 #22,149×42;含bevel+HP/MP標籤+槽,doc35 §4)
    pub(crate) panel: Option<Arc<Picture>>,
    /// 對話框框素材(FDOTHER#5 LMI1 #21,310×99;orig 下框(5,112)@320)
    pub(crate) dlg_box: Option<Arc<Picture>>,
    /// 對話框內部漸層(比對頭像底色 40,69,138→56,85,154 消接縫色差;lazy 建)
    pub(crate) dlg_grad: Option<Arc<Picture>>,
    /// 狀態欄名字(整數尺寸 face,scale1 銳利)
    pub(crate) font_nm: Option<Font>,
    /// 全螢幕戰鬥狀態欄 FDOTHER#4 16×16 字模
    pub(crate) native_battle_font: Option<fdtxt::Font>,
    /// Unicode→原版 glyph 索引（未知字元失敗即關閉）
    pub(crate) native_battle_glyphs: BTreeMap<String, i64>,
    /// 對拍重播列舉相位變體時凍結 BIOS 取樣；nil 用實際時鐘
    pub(crate) native_map_frozen_now: Option<Box<dyn FnMut(&mut Game) -> GameInstant + Send>>,
    /// 非 nil：0x4EB90 底面相位改用此值（對拍重播依原版收據 [0x539FC] 指定）
    pub(crate) native_chapter_aux_phase_override: Option<i64>,
    /// 對拍重播用：指令環穩態時仍以 0x1741C 最後一張展開幀位移貼圖示（0x179D5 可能在第一次穩態重繪前讀鍵）
    pub(crate) native_action_overlay_open_frame_variant: bool,
    /// 0x18C6D框／bar／digit indexed素材
    pub(crate) native_battle_panel: Option<battle::NativeItemPanelDataAssets>,
    /// 可見值panel快取
    pub(crate) native_battle_values: BTreeMap<NativeBattlePanelValueKey, Option<Arc<Picture>>>,
    /// 狀態欄數字 0-9(LMI1 #31-40 原版 digit cell,白/藍影)
    pub(crate) digits: [Option<Arc<Picture>>; 10],
    /// E1 紅色剪影近似快取；不是 raw DAC 脈衝本身
    pub(crate) red_sil: BTreeMap<Option<Arc<Picture>>, Option<Arc<Picture>>>,
    /// 全螢幕暗化/底板共用(回合橫幅、單位面板)
    pub(crate) dim: Option<Arc<Picture>>,
    /// FIGANI 每幀內嵌絕對螢幕座標 (dx,dy)@320(doc06;動畫走位全靠它)
    pub(crate) fig_meta: BTreeMap<i64, Vec<[i64; 2]>>,
    /// 原版點陣中文字型(doc 08)
    pub(crate) font: Option<Font>,
    /// 玩家選的字級倍率(F7;settings.go fontScales)
    pub(crate) font_scale: f64,
    /// 全域語系設定；不寫入戰役存檔
    pub(crate) locale_id: String,
    /// 已完整驗證的官方語言包
    pub(crate) locale_catalog: Option<localization::Catalog>,
    /// 已完整驗證的全量玩家內容目錄
    pub(crate) locale_content: Option<localization::ContentCatalog>,
    /// 已按遊戲 ID 正規化的實體名稱
    pub(crate) locale_entities: Option<localization::EntityCatalog>,
    /// protect 使用來源姓名，不受顯示語系影響
    pub(crate) source_battle_entities: Option<localization::EntityCatalog>,
    /// 顯式 FD2_THEME 候選；缺 speaker 時整頁拒絕
    pub(crate) modern_story_portraits: Option<ModernStoryPortraitSet>,
    /// 同一程序內已通過預檢的完整素材
    pub(crate) modern_theme_cache: Option<ModernStoryPortraitSet>,
    /// 第一次切到 HD 的背景載入；nil 表示沒有進行中的載入
    pub(crate) modern_theme_loading: Option<ModernThemeLoad>,
    /// 測試替換用；nil 時用 loadPlayerModernTheme
    pub(crate) modern_theme_loader: Option<Box<dyn FnMut(&mut Game) -> Result<Option<ModernStoryPortraitSet>> + Send>>,
    /// 目前對話拍的靜態閉嘴真彩色層
    pub(crate) native_dialogue_modern_portrait: Option<ModernStoryPortraitFrame>,
    /// 本次Draw實際呈現的高解析頭像
    pub(crate) modern_hd_overlay: Option<ModernHDOverlay>,
    pub(crate) modern_hd_scene: Option<ModernFacilitySceneOverlay>,
    pub(crate) preparation_roster_hd: Option<ModernPreparationRoster>,
    pub(crate) facility_roster_hd: Option<ModernFacilityRoster>,
    pub(crate) prep_status_hd: Option<ModernHDPreparationStatus>,
    pub(crate) prep_counts_hd: Option<ModernHDPreparationCounts>,
    pub(crate) prep_cursor_hd: Option<ModernPreparationCursor>,
    pub(crate) prep_native_cursor_unit_hd: Option<ModernPreparationNativeCursorUnit>,
    pub(crate) modern_hd_panel_portraits: Vec<ModernHDOverlay>,
    pub(crate) player_theme_requested: bool,
    pub(crate) player_interface_requested: bool,
    pub(crate) interface_mode: String,
    pub(crate) player_speed: i64,
    pub(crate) player_battle_speed: i64,
    pub(crate) gold_action_frames: BTreeMap<GoldActionKey, Option<Arc<Picture>>>,
    pub(crate) player_hd_quality: HdTextureQuality,
    pub(crate) player_clock: PlayerGameClock,
    pub(crate) player_speed_requested: bool,
    pub(crate) player_font_requested: bool,
    pub(crate) player_fullscreen_requested: bool,
    pub(crate) player_poster_cycle_requested: bool,
    pub(crate) player_viewport_left: i64,
    pub(crate) player_overlay_block: Rectangle,
    pub(crate) native_map_presented: bool,
    pub(crate) native_interface_hud: Option<ModernHDMapHUDPanel>,
    pub(crate) native_interface_hud_visible: bool,
    pub(crate) player_locale_requested: bool,
    pub(crate) player_toolbar_touch_blocked: bool,
    pub(crate) player_viewport_top: i64,
    pub(crate) native_command30_saved_cursor: Option<battle::Cell>,
    pub(crate) modern_hd_text: Option<ModernHDTextFrame>,
    pub(crate) native_dialogue_arrow_ticks: i64,
    /// 四槽 LOAD 的已驗證戰間狀態；未知 raw bytes 僅保存、不猜接
    pub(crate) native_chapter_restore: Option<campaign::NativeChapterSlotRestorePlan>,
    /// LOADCH 後原版六個視圖全域的場景專用載體；不冒充 battle.State
    pub(crate) story_native_map_view: battle::NativeMapViewState,
    /// 僅在已證實的 LOADCH 視圖重設與 pan 步進後有效
    pub(crate) has_story_native_map_view: bool,
    /// LOADCH 的完整 renderer source；只在 caller 要求原生對話時 materialize
    pub(crate) story_native_map_source: Option<battle::State>,
    /// LOADCH 場景的 indexed terrain/unit timing 載體；不含戰鬥規則或可見 HUD
    pub(crate) story_native_map_state: Option<battle::State>,
    /// actionOverlayBlink 保存 sub_17898 直接讀寫的 [0x53c13]／[0x53c17]：
    /// 穩態重繪時選中格每四個 BIOS tick 在 base 與 base+1 之間閃爍。
    pub(crate) action_overlay_blink: fdother::ActionOverlaySelectionBlink,
    pub(crate) native_confirmation_mouth: dato::ConfirmationMouthState,
    pub(crate) native_confirmation_mouth_owner: Option<OwnerToken>,
    pub(crate) native_confirmation_mouth_rng: Option<battle::GoMathRand>,
    /// Rust 新增（沒有 Go 對應）：本邏輯幀的輸入。取代 Go 直接查詢 `inpututil.IsKeyJustPressed`、
    /// `ebiten.IsKeyPressed`；驅動端（無畫面重播或 `fd2-app`）在每次 `update` 前設定。
    pub(crate) tick_input: TickInput,
    /// Rust 新增（沒有 Go 對應）：虛擬時鐘目前的時間。取代 Go `gameNow()` 的牆鐘乘遊戲速度；
    /// 驅動端每個邏輯幀前進固定步長。
    pub(crate) virtual_now: GameInstant,
}

/// atkAnim 是重製端 E1 全螢幕戰鬥演出；土台、角色、斬擊弧與血條使用
/// 已接資源與幾何，但命中 DAC、逐幀調色盤及完整原版 owner 尚未達 E2。
///
/// Go：`main.go:655`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct AtkAnim {
    /// 攻方(右土台)/ 守方(左)FIGANI
    pub(crate) atk_fig: i64,
    /// Go 與 `atkFig` 同列宣告，註解見該欄位。
    pub(crate) def_fig: i64,
    /// 名字(資訊條)
    pub(crate) atk_name: String,
    /// Go 與 `atkName` 同列宣告，註解見該欄位。
    pub(crate) def_name: String,
    pub(crate) atk_hp: i64,
    pub(crate) atk_max: i64,
    pub(crate) atk_mp: i64,
    pub(crate) atk_max_mp: i64,
    pub(crate) def_mp: i64,
    pub(crate) def_max_mp: i64,
    pub(crate) atk_lv: i64,
    pub(crate) def_lv: i64,
    /// 守方攻擊前/後 HP(impact 抽乾動畫)
    pub(crate) def_hp0: i64,
    /// Go 與 `defHP0` 同列宣告，註解見該欄位。
    pub(crate) def_hp1: i64,
    pub(crate) def_max: i64,
    pub(crate) timer: i64,
    pub(crate) total: i64,
    /// accumMillis 是還沒換成顯示 tick 的毫秒。一個顯示 tick 是「一個 BIOS
    /// tick ÷ fpt」，不是一個 60 Hz 畫格；見 Update 裡的推進段。
    pub(crate) accum_millis: f64,
    /// 播放速度(tick/幀;FD2_BATTLE_FPT 可調)
    pub(crate) fpt: i64,
    /// 攻方是否我方(狀態欄按陣營:我方欄右上/敵方欄左下)
    pub(crate) atk_own: bool,
    /// 攻擊格地形索引(戰鬥背景 = 戰場地形,跟 FDFIELD 戰場資料有關)
    pub(crate) terrain: i64,
    /// 已證實 FIGANI 幀延遲；不承載命中／傷害語意
    pub(crate) figani_timeline: Option<figani::DisplayScheduler>,
    /// 尚未接線；缺 raw provenance 時保持 nil
    pub(crate) native_impact_raw: Option<NativeImpactDACInput>,
    /// 正常物理入口的 BG／TAI；反擊不重選
    pub(crate) native_scene: Option<NativePhysicalScene>,
    /// 目前已呈現的 FIGANI 幀
    pub(crate) frame_index: i64,
    /// 幀本體的精確延遲總長，尾段停格另計
    pub(crate) body_ticks: i64,
    /// counter 是反擊那一段。原版一次攻擊的兩次結算共用一段演出（`sub_29164` 只
    /// 滑入一次，`sub_2939D` 跑兩次，第二次攻守對調），所以這裡不另起一段演出，
    /// 而是在主攻段跑完時就地換上第二段的資源與資訊條。nil 表示沒有反擊。
    pub(crate) counter: Option<AtkStage>,
    /// 原版 action handler 完成後才進 selector1；不得在演出前提交
    pub(crate) after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// atkStage 是一段交鋒要換上的東西。欄位刻意只有「會變的那些」——資訊條的名字、
/// 等級與 MP 由 atkAnim 自己對調，不必再存一份。
///
/// Go：`main.go:685`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct AtkStage {
    /// 這一段的攻方動作資源與守方待機資源
    pub(crate) atk_fig: i64,
    /// Go 與 `atkFig` 同列宣告，註解見該欄位。
    pub(crate) def_fig: i64,
    pub(crate) timeline: Option<figani::DisplayScheduler>,
    pub(crate) body_ticks: i64,
    pub(crate) total: i64,
    /// 這一段守方（＝上一段的攻方）的 HP 變化
    pub(crate) def_hp0: i64,
    /// Go 與 `defHP0` 同列宣告，註解見該欄位。
    pub(crate) def_hp1: i64,
    pub(crate) def_max: i64,
}

/// nativeImpactDACInput 只保存原版 0x2939d 命中分支仍可回查的 raw 條件。
/// rawOutput20/rawOutput1C 對應 0x29f72 輸出暫存的 stack local 位移；它們不是
/// 暴擊、狀態或其他高階語意名稱。正式 renderer 尚未取得這組來源，故指標為 nil。
///
/// Go：`main.go:696`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeImpactDACInput {
    pub(crate) frame_flag: u8,
    pub(crate) damage_step_complete: bool,
    pub(crate) raw_output20: bool,
    pub(crate) raw_output1_c: bool,
}

/// storyWalkJob 場景走位動畫(doc46 §5.3):cutscene 固定路徑位移,非玩家可控,重用
/// battle.Unit.OffX/OffY/Dir 插值(同行軍/移動動畫的畫法),完成時呼叫 then(可為 nil)。
///
/// Go：`main.go:710`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct StoryWalkJob {
    /// g.storyActors 索引
    pub(crate) actor: i64,
    pub(crate) from_x: i64,
    pub(crate) from_y: i64,
    pub(crate) to_x: i64,
    pub(crate) to_y: i64,
    pub(crate) t: i64,
    pub(crate) frames: i64,
    /// 走完後面向(-1=保留走位末向;>=0=設定,如進場走位面向 actor 目標 dir)
    pub(crate) final_dir: i64,
    pub(crate) scroll_follow: bool,
    /// 0x13185 專用：超過原版 screen-row 安全帶後與 actor 同速捲圖
    pub(crate) from_cam_y: f64,
    /// actor 可先上移而不捲圖的格數（screenY-1）
    pub(crate) scroll_free: i64,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// storyFade 場景淡出/淡入轉場(doc46 §5.2):story 節點換場時不硬切。out=true 淡出(變黑)、
/// false 淡入(轉亮);走完(不分 out/in)觸發 then(可為 nil;beat fade 靠它接回下一拍)。
///
/// Go：`main.go:724`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct StoryFade {
    pub(crate) out: bool,
    pub(crate) t: i64,
    pub(crate) total: i64,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// transitionRevealJob owns native 0x24B4D's two proven row-shifted indexed
/// viewports. The address-specific constructor validates and prepares every
/// buffer before this blocking job can become visible.
///
/// Go：`main.go:734`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct TransitionRevealJob {
    pub(crate) hd_source: Option<ModernHDRowShakeSource>,
    pub(crate) hd_frames: Vec<ModernHDMapFrame>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) work: Vec<u8>,
    pub(crate) frames: [Vec<u8>; 2],
    pub(crate) palette: ColorPalette,
    pub(crate) index: i64,
    pub(crate) remaining: i64,
    pub(crate) ticks: i64,
    pub(crate) delay: i64,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) rollback: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// storyFadeFrames 淡出/淡入各自幀數(60fps;doc46 要求 0.5–1s,先做快版 0.6s,實測後可調)。
pub(crate) const STORY_FADE_FRAMES: i64 = 36;

/// camPanJob beat「pan」原語。tileStep 精確重現 0x135dd：先 X 後 Y、每 tick
/// 移一個 tile 並 redraw；舊 authored scenes 保留 frames 線性內插相容模式。
///
/// Go：`main.go:755`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct CamPanJob {
    pub(crate) from_x: f64,
    pub(crate) from_y: f64,
    pub(crate) to_x: f64,
    pub(crate) to_y: f64,
    pub(crate) t: i64,
    pub(crate) frames: i64,
    pub(crate) tile_step: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// battleEventRun preserves the authored order of on-turn battle actions.
/// It deliberately does not reuse campaign beats: finishing a battle event
/// must finish the turn, never advance the campaign node.
///
/// Go：`main.go:766`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct BattleEventRun {
    pub(crate) actions: Vec<battle::Action>,
    pub(crate) index: i64,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// focusUnitJob 保留 0x12cea 的阻塞移動目標。原版每輪只移動一格，X 到位後才移 Y；
/// 游標接近 13×8 視窗邊緣時才推進 map origin，並非直接把目標置中。
///
/// Go：`main.go:774`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct FocusUnitJob {
    pub(crate) target_x: i64,
    pub(crate) target_y: i64,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) native_view: bool,
    /// 0x12CEA 開頭的 0x11CAC(0) 已經做過
    pub(crate) started: bool,
    pub(crate) unit_present: Option<NativeUnitPresentJob>,
}

/// actPoseJob 承接 beat「act」。acting 非空時按原版 0x1366a 規則播放：正常 frame
/// (bit7=0)的每一 Beat 都走一格、每格 7 個內插 tick；special frame(bit7=1)原地停留。
/// poses/frames 則保留給尚未轉錄的舊場景作原地姿態相容。
///
/// Go：`main.go:785`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct ActPoseJob {
    pub(crate) actor: i64,
    pub(crate) poses: Vec<i64>,
    pub(crate) frames: i64,
    pub(crate) t: i64,
    pub(crate) idx: i64,
    pub(crate) acting: Vec<campaign::ActingFrame>,
    pub(crate) frame: i64,
    pub(crate) beat: i64,
    pub(crate) tick: i64,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
}

/// 原版 acting 的格線動作以 BIOS 約 18.2065 Hz 的來源拍等待；Ebiten 預設
/// 60 Hz Update 只負責跨平台投影，不可把一次 Update 當成一次原版來源拍。
/// 這是硬體規格近似，不宣稱 DOS PIT wall-clock 逐週期一致。
///
/// Go 原式：`(1193182.0 / 65536.0) / 60.0`
pub(crate) const ACTING_SOURCE_TICKS_PER_UPDATE: f64 = 0.3034418741861979;

impl Game {
    /// startFadeTransition 淡出 storyFadeFrames 幀 → 執行 action(通常是 Advance+enterNode 或
    /// 換擺位)→ 淡入 storyFadeFrames 幀。
    ///
    /// Go：`main.go:804`，`func (g *Game) startFadeTransition(action func())`
    /// Go 的 func 參數 `action` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_fade_transition(&mut self, action: Option<Box<dyn FnMut(&mut Game) + Send>>) {
        todo!("main.go:startFadeTransition")
    }

    /// advanceStoryNode 對白播完離開 story 節點:若節點設 ExitWalk(s),先讓對應 actor(可多人,
    /// 使用者回饋 #A:草地小徑幕結尾索爾+亞雷斯一起走離)全部走完一段路(doc46 §5.3)再淡出換場;
    /// 否則直接淡出換場。多個 exit walk 共用同一個「全部走完才轉場」計數器,不是走完第一個就轉場。
    ///
    /// Go：`main.go:814`，`func (g *Game) advanceStoryNode(n *campaign.Node)`
    pub(crate) fn advance_story_node(&mut self, n: &campaign::Node) {
        todo!("main.go:advanceStoryNode")
    }

    /// stepStoryWalks 逐幀推進場景走位動畫佇列,完成的 job 更新最終座標、呼叫 then 後移除。
    /// 走位沿格線(先長軸後短軸,不斜切)——原版單位走位以戰場格為單位軸向移動
    /// (使用者回饋 2026-07-04 #4:「走路沒照著格子走」=舊版起訖點直線內插會斜切格線)。
    ///
    /// Go：`main.go:859`，`func (g *Game) stepStoryWalks()`
    pub(crate) fn step_story_walks(&mut self) {
        todo!("main.go:stepStoryWalks")
    }

    /// stepFade 逐幀推進場景淡出/淡入(storyFade,doc46 §5.2):走完(不分 out/in)呼叫 then;
    /// g.fade 為空時內部直接返回(同 stepStoryWalks/stepActJob/stepCamPan 慣例)。
    ///
    /// Go：`main.go:952`，`func (g *Game) stepFade()`
    pub(crate) fn step_fade(&mut self) {
        todo!("main.go:stepFade")
    }

    /// Go：`main.go:966`，`func (g *Game) stepTransitionReveal()`
    pub(crate) fn step_transition_reveal(&mut self) {
        todo!("main.go:stepTransitionReveal")
    }

    /// syncStoryNativeMapPanView 發布 0x135DD（劇情 pan）走完之後的六個視圖全域。
    /// 0x135DD 的迴圈對每一格同時寫鏡頭與絕對游標：X 先走完再走 Y，
    /// 每格一次 0x11CAC(0) 重繪，可見游標整段不寫。
    ///
    ///     0x13606 dec [0x53AB1] / 0x1360C dec [0x53AA9]   ; 游標與鏡頭同減
    ///     0x13614 inc [0x53AB1] / 0x1361A inc [0x53AA9]   ; 同加
    ///     0x1363F/0x13645、0x1364D/0x13653                ; Y 軸同一對
    ///
    /// 因此游標的位移量就是鏡頭的位移量，不是由 camera + visible 反推——後者只有
    /// 在恆等式成立時才碰巧一致；游標的位移要照 0x135DD 自己的寫法算。
    /// 收據：docs/data/ui-traces/fd2-story-pan-cursor-20260909.json。
    ///
    /// Go：`main.go:1009`，`func (g *Game) syncStoryNativeMapPanView() bool`
    pub(crate) fn sync_story_native_map_pan_view(&mut self) -> bool {
        todo!("main.go:syncStoryNativeMapPanView")
    }

    /// syncStoryNativeMapScrollView 發布 0x13185（劇情走行捲動）整段走完之後的視圖。
    ///
    /// 0x13185 家族一格寫一次：絕對游標一定跟著走行單位動一格，另外二選一——單位
    /// 還在安全帶內（`unitY - camY >= 2`，或鏡頭已到邊界）就動可見游標，否則捲鏡頭
    /// 一格。三者的關係不是 `cursor = camera + visible` 反推得出來的：ch00_pre 的
    /// 15 格捲動在原版走完之後是 camera_y 34→20、cursor_y 34→19、visible_y 0→-1
    /// （收據 docs/data/ui-traces/fd2-story-pan-cursor-20260909.json，抓幀邊界在
    /// 0x11CAC，整段捲動一次都沒進去，所以只取得端點），反推會少算第一格那次
    /// 可見游標位移，把游標停在 20。
    ///
    /// 因此這裡逐格套用同一條規則（battle.AdvanceNativeMapWalkStepViewState），
    /// 再核對逐格算出來的鏡頭與插值走完的鏡頭一致；不一致就失敗即關閉，不從兩個
    /// 答案裡挑一個。
    ///
    /// Go：`main.go:1032`，`func (g *Game) syncStoryNativeMapScrollView(w *storyWalkJob) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn sync_story_native_map_scroll_view(&mut self, w: Option<&StoryWalkJob>) -> bool {
        todo!("main.go:syncStoryNativeMapScrollView")
    }
}

/// Go：`main.go:1068`，`func sgnInt(v int) int`
pub(crate) fn sgn_int(v: i64) -> i64 {
    todo!("main.go:sgnInt")
}

impl Game {
    /// publishStoryNativeMapView 是劇情／戰鬥事件共用的視圖發布邊界：取出目前追蹤
    /// 的六個全域，交給 advance 依各自的原版家族算出新值，再原子地發布。
    ///
    /// Go：`main.go:1080`，`func (g *Game) publishStoryNativeMapView( advance func(battle.NativeMapViewState) (battle.NativeMapViewState, bool), ) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn publish_story_native_map_view(
        &mut self,
        advance: &mut dyn FnMut(battle::NativeMapViewState) -> (battle::NativeMapViewState, bool),
    ) -> bool {
        todo!("main.go:publishStoryNativeMapView")
    }

    /// ch28HandlerNativeMapViewContinuity is deliberately caller-specific. Raw
    /// ch28 pre/post share the six native view globals across the remake's
    /// artificial cutscene/battle node boundary. Battle turn-event staging owns a
    /// separate private snapshot and must not be mutated by this bridge.
    ///
    /// Go：`main.go:1132`，`func (g *Game) ch28HandlerNativeMapViewContinuity() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn ch28_handler_native_map_view_continuity(&self) -> bool {
        todo!("main.go:ch28HandlerNativeMapViewContinuity")
    }

    /// syncStoryNativeMapFocusView mirrors the six globals after one 0x12CEA
    /// focus step. Unlike 0x135DD, focus updates absolute and visible cursor while
    /// moving the camera only when the native safe band is crossed.
    ///
    /// Go：`main.go:1146`，`func (g *Game) syncStoryNativeMapFocusView(visibleX, visibleY int) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn sync_story_native_map_focus_view(&mut self, visible_x: i64, visible_y: i64) -> bool {
        todo!("main.go:syncStoryNativeMapFocusView")
    }

    /// stepCamPan 逐幀推進 beat「pan」鏡頭位移；原版模式逐 tile、X-first，
    /// 相容模式線性內插。走完清除 job 並接下一拍。
    ///
    /// Go：`main.go:1188`，`func (g *Game) stepCamPan()`
    pub(crate) fn step_cam_pan(&mut self) {
        todo!("main.go:stepCamPan")
    }
}

/// nativeMapFocusVisibleSeed 是「完全沒有已追蹤可見游標」時的重製端後備值：
/// 把 cursor - camera 夾回 13×8 視窗。原版永遠有一份可見游標可以沿用，這裡只
/// 是不讓直接進場的 renderer 因為缺少那份狀態而產生視窗外的座標。
///
/// Go：`main.go:1248`，`func nativeMapFocusVisibleSeed(x, y int) (int, int)`
pub(crate) fn native_map_focus_visible_seed(x: i64, y: i64) -> (i64, i64) {
    todo!("main.go:nativeMapFocusVisibleSeed")
}

impl Game {
    /// stepFocusUnit 逐格重現 0x12cea 與 0x11b48/0x11b9b/0x11bfa/0x11c59。
    /// 原版視窗游標的安全帶為 X=2..10、Y=2..5；超出安全帶後，能捲圖時保持
    /// screen cursor 不動並移動 map origin，碰地圖邊界時才讓 screen cursor 靠邊。
    ///
    /// Go：`main.go:1264`，`func (g *Game) stepFocusUnit()`
    pub(crate) fn step_focus_unit(&mut self) {
        todo!("main.go:stepFocusUnit")
    }

    /// finishActJob 清除 acting job 後接下一個 beat。
    ///
    /// Go：`main.go:1382`，`func (g *Game) finishActJob(j *actPoseJob)`
    pub(crate) fn finish_act_job(&mut self, j: &ActPoseJob) {
        todo!("main.go:finishActJob")
    }

    /// actingActor resolves decoded acting against the original FDFIELD roster
    /// slot first.  Fig is deliberately only a legacy authored-scene fallback:
    /// using it for decoded frames would move the first matching guard when the
    /// original bytecode targeted a different same-Fig guard.
    ///
    /// Go：`main.go:1393`，`func (g *Game) actingActor(target campaign.ActingUnit) *battle.Unit`
    pub(crate) fn acting_actor(&self, target: campaign::ActingUnit) -> Option<UnitIdx> {
        todo!("main.go:actingActor")
    }
}

/// nativeRuntimeRecordCapacity 是原版 runtime 記錄表的配置筆數（[0x53A45] 之後 96×0x50
/// bytes）；[0x53BEB] 是已物化筆數，兩者不同。
pub(crate) const NATIVE_RUNTIME_RECORD_CAPACITY: i64 = 96;

impl Game {
    /// Go：`main.go:1420`，`func (g *Game) handlerUnitCount() int`
    pub(crate) fn handler_unit_count(&self) -> i64 {
        todo!("main.go:handlerUnitCount")
    }

    /// handlerRuntimeSlotCount preserves LOADCH's raw FDFIELD count before later
    /// SPAWN beats append records to the scene actor array.  A runtime_context beat
    /// therefore validates the original load context, not only the currently
    /// visible/materialized actors.
    ///
    /// Go：`main.go:1431`，`func (g *Game) handlerRuntimeSlotCount() int`
    pub(crate) fn handler_runtime_slot_count(&self) -> i64 {
        todo!("main.go:handlerRuntimeSlotCount")
    }

    /// Go：`main.go:1441`，`func (g *Game) handlerUnitAt(slot int) *battle.Unit`
    pub(crate) fn handler_unit_at(&self, slot: i64) -> Option<UnitIdx> {
        todo!("main.go:handlerUnitAt")
    }

    /// beginActingFrame 設定目前 frame 的姿態。呼叫端保證 j.frame 有效。
    ///
    /// Go：`main.go:1458`，`func (g *Game) beginActingFrame(j *actPoseJob)`
    pub(crate) fn begin_acting_frame(&self, j: &mut ActPoseJob) {
        todo!("main.go:beginActingFrame")
    }
}

/// Go：`main.go:1467`，`func actingDelta(pose int) (int, int)`
pub(crate) fn acting_delta(pose: i64) -> (i64, i64) {
    todo!("main.go:actingDelta")
}

impl Game {
    /// Go：`main.go:1482`，`func (g *Game) nextActingFrame(j *actPoseJob)`
    pub(crate) fn next_acting_frame(&mut self, j: &mut ActPoseJob) {
        todo!("main.go:nextActingFrame")
    }

    /// stepOriginalActing 精確承接已破解的 acting frame 行為(doc50 §1.2)。
    ///
    /// Go：`main.go:1492`，`func (g *Game) stepOriginalActing(j *actPoseJob)`
    pub(crate) fn step_original_acting(&mut self, j: &mut ActPoseJob) {
        todo!("main.go:stepOriginalActing")
    }

    /// stepActJob 逐幀推進原版 acting frame 或舊版姿態近似。
    ///
    /// Go：`main.go:1560`，`func (g *Game) stepActJob()`
    pub(crate) fn step_act_job(&mut self) {
        todo!("main.go:stepActJob")
    }

    /// Go：`main.go:1581`，`func (g *Game) stepActJobAtUpdateRate()`
    pub(crate) fn step_act_job_at_update_rate(&mut self) {
        todo!("main.go:stepActJobAtUpdateRate")
    }

    /// beatAdvance 進下一拍;序列跑完則走節點收尾流程。
    ///
    /// Go：`main.go:1601`，`func (g *Game) beatAdvance()`
    pub(crate) fn beat_advance(&mut self) {
        todo!("main.go:beatAdvance")
    }
}

/// beatStart 依原語種類啟動目前這一拍(狀態掛到 g.camPan/g.storyWalks/g.dialog/g.actJob/
/// g.fade/g.beatDelay,交給 Update 既有機制逐幀推進)。找不到對應角色 / 資料缺漏時直接跳拍
/// 並記到 loadErr,不讓整個過場卡死(誠實 stub,勝過假裝完成)。
/// figName cutscene sprite 標號用:fig id → 角色名(協助使用者回報 + 對映原版 slot)。
///
/// Go：`main.go:1816`，`func figName(fig int) string`
pub(crate) fn fig_name(fig: i64) -> String {
    todo!("main.go:figName")
}

impl Game {
    /// Go：`main.go:1836`，`func (g *Game) beatStart(b campaign.Beat)`
    pub(crate) fn beat_start(&mut self, b: &campaign::Beat) {
        todo!("main.go:beatStart")
    }

    /// Go：`main.go:2625`，`func (g *Game) resolveCampaignDialogLine(line campaign.Line, upperOverride *bool, nativeOverride *campaign.NativeDialogueLayout) (battle.DialogLine, error)`
    pub(crate) fn resolve_campaign_dialog_line(
        &mut self,
        line: &campaign::Line,
        upper_override: Option<&mut bool>,
        native_override: &mut campaign::NativeDialogueLayout,
    ) -> Result<battle::DialogLine> {
        todo!("main.go:resolveCampaignDialogLine")
    }

    /// resolveNativeStoryDialogueSpeaker 重播 sub_15F84 的 DATO selector owner。
    /// FFED/FFEC 的 operand 是第一張 runtime array slot；FFEF/FFEE 則先以
    /// raw +8 查找 active record，命中後取 raw +7，未命中保留 direct-DATO operand。
    ///
    /// Go：`main.go:2704`，`func (g *Game) resolveNativeStoryDialogueSpeaker(layout *campaign.NativeDialogueLayout) (int, error)`
    pub(crate) fn resolve_native_story_dialogue_speaker(&self, layout: &campaign::NativeDialogueLayout) -> Result<i64> {
        todo!("main.go:resolveNativeStoryDialogueSpeaker")
    }

    /// resolveNativeStoryDialogueMotionTarget 重播 sub_15F84 的 var_20 writer。
    /// FFED/FFEC 是 caller 常數；FFEF/FFEE 只在 sub_12C60 的第一張0x50-byte
    /// scene-unit array 找到 raw +8，且 sub_3453E 證實的 raw +5 bit0 為0時命中。
    ///
    /// Go：`main.go:2746`，`func (g *Game) resolveNativeStoryDialogueMotionTarget(layout *campaign.NativeDialogueLayout) (int, error)`
    pub(crate) fn resolve_native_story_dialogue_motion_target(
        &self,
        layout: &campaign::NativeDialogueLayout,
    ) -> Result<i64> {
        todo!("main.go:resolveNativeStoryDialogueMotionTarget")
    }

    /// Go：`main.go:2981`，`func (g *Game) materializeStoryGroup(group int) int`
    pub(crate) fn materialize_story_group(&mut self, group: i64) -> i64 {
        todo!("main.go:materializeStoryGroup")
    }

    /// loadCHPartyOrder projects the permanent JOIN chronology through the current
    /// preparation selection.  The original LOADCH constructs the deployed party
    /// first, so a late-game cutscene must not silently resurrect every joined
    /// character merely because partyJoinOrder is longer than the selected roster.
    /// With no JOIN history this remains a direct/debug binding's authored order.
    ///
    /// Go：`main.go:3021`，`func (g *Game) loadCHPartyOrder(state *campaign.LoadCHState) []int`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn load_ch_party_order(&self, state: Option<&campaign::LoadCHState>) -> Vec<i64> {
        todo!("main.go:loadCHPartyOrder")
    }

    /// Go：`main.go:3041`，`func (g *Game) nativeRestoredPartyUnitsForLoadCH( order []int, deploy []battle.Cell, scenario *battle.Scenario, ) ([]*battle.Unit, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_restored_party_units_for_load_ch(
        &self,
        order: &[i64],
        deploy: &[battle::Cell],
        scenario: Option<&battle::Scenario>,
    ) -> Result<Vec<UnitIdx>> {
        todo!("main.go:nativeRestoredPartyUnitsForLoadCH")
    }

    /// applyLoadCH is the remake adapter for original 0x205da/0x1088d.  The
    /// original operation selects FDTXT chapter+1 and the three FDFIELD resources
    /// for the same chapter in one call; it is not merely a camera/map command.
    /// The binding therefore provides all three editable counterparts.  Validate
    /// roster and text before replacing the rendered map so a malformed asset does
    /// not leave a half-applied chapter transition behind.
    ///
    /// Go：`main.go:3099`，`func (g *Game) applyLoadCH(state *campaign.LoadCHState) error`
    pub(crate) fn apply_load_ch(&mut self, state: Option<&mut campaign::LoadCHState>) -> Result<()> {
        todo!("main.go:applyLoadCH")
    }
}

/// Go：`main.go:3305`，`func equalIntOrder(a, b []int) bool`
pub(crate) fn equal_int_order(a: &[i64], b: &[i64]) -> bool {
    todo!("main.go:equalIntOrder")
}

/// dlgPageCount 該句對白的FFFD總頁數；FFFE只形成頁內邏輯列與三列窗口捲動。
///
/// Go：`main.go:3356`，`func dlgPageCount(dl battle.DialogLine) int`
pub(crate) fn dlg_page_count(dl: &battle::DialogLine) -> i64 {
    todo!("main.go:dlgPageCount")
}

impl Game {
    /// dlgAdvance 處理 Enter:目前句還有下一頁就翻頁(回 false,不換句);翻完(或本就單頁)就 pop
    /// 到下一句、頁碼歸零(回 true=已換句)。修長對白被截斷丟棄的 bug(使用者回饋 2026-07-05)。
    ///
    /// Go：`main.go:3372`，`func (g *Game) dlgAdvance() bool`
    pub(crate) fn dlg_advance(&mut self) -> bool {
        todo!("main.go:dlgAdvance")
    }
}

/// ── 對話框切換動畫(使用者回饋 2026-07-04 #3:換人說話時框先收合再展開)──
///
/// dlgShown 無框哨兵
///
/// Go 原式：`-999`
pub(crate) const DLG_NONE: i64 = -999;

/// 縮/展各自幀數(~0.12s@60fps)
pub(crate) const DLG_ANIM_FRAMES: i64 = 7;

/// 分頁文字往上捲動幀數(~0.17s@60fps)
pub(crate) const DLG_SCROLL_FRAMES: i64 = 10;

impl Game {
    /// stepDlgAnim 逐幀推進對話框縮/展動畫:首句直接展開;說話者變更先縮(phase1)再換人展開(phase2)。
    ///
    /// Go：`main.go:3417`，`func (g *Game) stepDlgAnim()`
    pub(crate) fn step_dlg_anim(&mut self) {
        todo!("main.go:stepDlgAnim")
    }

    /// terrainAt 回傳某格的地形索引(戰鬥背景用;越界回 -1)。
    ///
    /// Go：`main.go:3447`，`func (g *Game) terrainAt(x, y int) int`
    pub(crate) fn terrain_at(&self, x: i64, y: i64) -> i64 {
        todo!("main.go:terrainAt")
    }

    /// enterNode 進入 camp 目前節點:story→掛對白、battle→重開戰場、inventory_gate→依角色物品欄自動分支、
    /// event→套旗標直通、town/preparation/choice/ending→等輸入。
    ///
    /// Go：`main.go:3458`，`func (g *Game) enterNode()`
    pub(crate) fn enter_node(&mut self) {
        todo!("main.go:enterNode")
    }

    /// Go：`main.go:3749`，`func (g *Game) materializeNativeMapRuntime(n *campaign.Node) bool`
    pub(crate) fn materialize_native_map_runtime(&mut self, n: Option<&campaign::Node>) -> bool {
        todo!("main.go:materializeNativeMapRuntime")
    }

    /// applyFieldOwnAttackRanges 讓 FDFIELD 直接登場的我方記錄（raw +6==2、不在持久名冊裡，
    /// 例如第八章 group 0 的洛娜）和 LOAD 進來的名冊用同一條射程規則：地圖單位檔沒有射程
    /// 欄位，沒有這一步就一律是 1，裝槍的單位在距離 2 的敵人前面攻擊不可選（原版 r1 seq 94
    /// 可以攻擊）。已經有射程的單位（名冊、劇本手寫值）不動。
    ///
    /// Go：`main.go:3837`，`func (g *Game) applyFieldOwnAttackRanges()`
    pub(crate) fn apply_field_own_attack_ranges(&mut self) {
        todo!("main.go:applyFieldOwnAttackRanges")
    }

    /// captureNativeMapHUDPersistence runs before a campaign node can clear or
    /// replace the previous battle state. The original carries gate A and anchor
    /// across nodes, but not the transient gate B.
    ///
    /// Go：`main.go:3855`，`func (g *Game) captureNativeMapHUDPersistence()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn capture_native_map_hud_persistence(&mut self) {
        todo!("main.go:captureNativeMapHUDPersistence")
    }

    /// resetBattle 重開一場戰鬥(campaign battle 節點;敗北重試也走這裡)。
    ///
    /// Go：`main.go:3863`，`func (g *Game) resetBattle(unitsPath, scnPath string)`
    pub(crate) fn reset_battle(&mut self, units_path: &str, scn_path: &str) {
        todo!("main.go:resetBattle")
    }
}

/// pruneHandlerMaterializedRoster removes only FDFIELD rows already copied by
/// the immediately preceding LOADCH. Persistent party constructors have no
/// NativePositionRecord and are intentionally ignored. Matching the immutable
/// raw position/key provenance rather than current X/Y keeps staged movement
/// from duplicating an initial group at the battle handoff.
///
/// Go：`main.go:4005`，`func pruneHandlerMaterializedRoster(actors, roster []battle.Unit) []battle.Unit`
pub(crate) fn prune_handler_materialized_roster(actors: &[battle::Unit], roster: &[battle::Unit]) -> Vec<battle::Unit> {
    todo!("main.go:pruneHandlerMaterializedRoster")
}

impl Game {
    /// bindNativeCommandBook gives each freshly loaded battle the immutable raw
    /// ABI table.  It never derives records from the legacy SpellBook.
    ///
    /// Go：`main.go:4039`，`func (g *Game) bindNativeCommandBook(st *battle.State)`
    pub(crate) fn bind_native_command_book(&self, st: Option<&mut battle::State>) {
        todo!("main.go:bindNativeCommandBook")
    }

    /// bindNativeMovementCostRows 將有機會進入 AI runner 的每個 battle state 綁定
    /// 版本化 0x4e555 表。缺少或格式錯誤時，在 mode 2 窄切片嘗試以正規化移動成本
    /// 取代原始成本前先停止。
    ///
    /// Go：`main.go:4049`，`func (g *Game) bindNativeMovementCostRows(st *battle.State) error`
    pub(crate) fn bind_native_movement_cost_rows(&self, st: Option<&mut battle::State>) -> Result<()> {
        todo!("main.go:bindNativeMovementCostRows")
    }

    /// Go：`main.go:4062`，`func (g *Game) bindNativeCommandResistances(st *battle.State)`
    pub(crate) fn bind_native_command_resistances(&self, st: Option<&mut battle::State>) {
        todo!("main.go:bindNativeCommandResistances")
    }

    /// bindCommandLearn makes every newly loaded battle state use the same
    /// explicit editable export. A missing table leaves the state fail-closed;
    /// it never falls back to the legacy normalized Spells list.
    ///
    /// Go：`main.go:4075`，`func (g *Game) bindCommandLearn(st *battle.State)`
    pub(crate) fn bind_command_learn(&self, st: Option<&mut battle::State>) {
        todo!("main.go:bindCommandLearn")
    }

    /// applyPersistentParty overlays the post-battle roster snapshot on freshly
    /// materialized player units while preserving this scenario's deployment,
    /// camp/group and on-field state. Original 0x11506 matches on charID; remake
    /// player Fig is the same stable 0..31 identity used by JOIN.
    ///
    /// Go：`main.go:4096`，`func (g *Game) applyPersistentParty(st *battle.State)`
    pub(crate) fn apply_persistent_party(&self, st: Option<&mut battle::State>) {
        todo!("main.go:applyPersistentParty")
    }

    /// Go：`main.go:4110`，`func (g *Game) initializeEquipmentBases(st *battle.State)`
    pub(crate) fn initialize_equipment_bases(&self, st: Option<&mut battle::State>) {
        todo!("main.go:initializeEquipmentBases")
    }

    /// seedPersistentPartyFromLoadCH materializes the remake counterpart of the
    /// persistent records already created by preceding native JOIN calls. Existing
    /// records always win, preserving battle/shop/class-change progress. This is a
    /// normal-campaign bridge, not FD2.SAV byte compatibility.
    ///
    /// Go：`main.go:4125`，`func (g *Game) seedPersistentPartyFromLoadCH( order []int, units []*battle.Unit, ) error`
    pub(crate) fn seed_persistent_party_from_load_ch(&mut self, order: &[i64], units: &[UnitIdx]) -> Result<()> {
        todo!("main.go:seedPersistentPartyFromLoadCH")
    }

    /// Go：`main.go:4158`，`func (g *Game) materializeNativeJoinPersistentUnit(id int, base battle.Unit) (battle.Unit, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn materialize_native_join_persistent_unit(
        &mut self,
        id: i64,
        base: &battle::Unit,
    ) -> Result<battle::Unit> {
        todo!("main.go:materializeNativeJoinPersistentUnit")
    }

    /// nativeJoinResidualRecord 回 sub_112A5 這次要寫的那一格在寫入前的內容：持續名冊是
    /// 0x10010 從 LOAD 的槽整批還原的，count 之後的格子就是槽裡的 bytes；沒有 LOAD 基底
    /// 時，重製端使用零基底；這不代表原版新開局記憶體全零。首章正常 START
    /// 收據已觀察到非零殘值，保存限制見 fd2_ch01_flow_20261017.json。
    /// 索引 = 基底筆數 + 基底之後已加入的人數。
    ///
    /// Go：`main.go:4189`，`func (g *Game) nativeJoinResidualRecord() fdsave.PersistentRecord`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_join_residual_record(&self) -> fdsave::PersistentRecord {
        todo!("main.go:nativeJoinResidualRecord")
    }
}

/// Go：`main.go:4217`，`func applyPersistentStats(dst, src *battle.Unit)`
pub(crate) fn apply_persistent_stats(dst: Option<&mut battle::Unit>, src: Option<&battle::Unit>) {
    todo!("main.go:applyPersistentStats")
}

impl Game {
    /// syncPartyFromBattle is the remake projection of original 0x11506. The EXE
    /// matches persistent records by raw +0x08 identity before copying its full
    /// 0x50-byte runtime record. When optional raw identities are present, this
    /// projection uses that exact key; records with an unknown raw key are skipped
    /// (fail closed). Records without raw identity retain the legacy Fig projection.
    /// It clears transient state/path bytes, restores
    /// active survivors to full HP and restores everyone's MP. Defeated/inactive
    /// members retain their zero HP. The projection snapshots JOIN member 0 as
    /// compatibility behavior; it is not byte-identical proof of native 0x11506.
    ///
    /// Go：`main.go:4374`，`func (g *Game) syncPartyFromBattle() error`
    pub(crate) fn sync_party_from_battle(&mut self) -> Result<()> {
        todo!("main.go:syncPartyFromBattle")
    }

    /// syncPartyFromBattleRecords 回傳實際發布到持續隊伍的紀錄（record）數。一般戰後
    /// 呼叫維持既有相容行為；終局來源約束邊界另要求至少一筆，避免所有原始身分
    /// （raw identity）都不相符時帶著舊名冊靜默進入角色蒙太奇。
    ///
    /// Go：`main.go:4382`，`func (g *Game) syncPartyFromBattleRecords() (int, error)`
    pub(crate) fn sync_party_from_battle_records(&mut self) -> Result<i64> {
        todo!("main.go:syncPartyFromBattleRecords")
    }

    /// Go：`main.go:4510`，`func (g *Game) applyScenarioPartyJoins()`
    pub(crate) fn apply_scenario_party_joins(&mut self) {
        todo!("main.go:applyScenarioPartyJoins")
    }
}

/// filterScenarioParty applies the campaign's JOIN membership to a freshly
/// loaded battle scenario.  A nil/empty membership intentionally means a
/// direct chapter/debug start, so the authored scenario party remains usable.
///
/// Go：`main.go:4590`，`func filterScenarioParty(sc *battle.Scenario, members map[int]bool)`
pub(crate) fn filter_scenario_party(sc: Option<&mut battle::Scenario>, members: &BTreeMap<i64, bool>) {
    todo!("main.go:filterScenarioParty")
}

impl Game {
    /// battlePartyMembers returns the temporary roster selected by the original
    /// preparation screen, falling back to the permanent JOIN roster for direct
    /// starts and campaigns that have not reached a preparation node yet.
    ///
    /// Go：`main.go:4608`，`func (g *Game) battlePartyMembers() map[int]bool`
    pub(crate) fn battle_party_members(&self) -> BTreeMap<i64, bool> {
        todo!("main.go:battlePartyMembers")
    }

    /// Go：`main.go:4628`，`func (g *Game) setupPreparation(n *campaign.Node)`
    pub(crate) fn setup_preparation(&mut self, n: Option<&campaign::Node>) {
        todo!("main.go:setupPreparation")
    }

    /// Go：`main.go:4807`，`func (g *Game) setupChurch()`
    pub(crate) fn setup_church(&mut self) {
        todo!("main.go:setupChurch")
    }
}

/// reorderScenarioParty applies the original JOIN chronology before either a
/// battle or handler cutscene constructs its runtime unit array. Chapter 0
/// proves the order 0,9,4,30 rather than the authored battle-UI order 0,4,9,30,
/// and later acting/post handlers address those construction slots.
///
/// 部署格不跟著角色走：0x1088D 讓第 i 筆 runtime 記錄站 FDFIELD 出場位置的第 i 格，
/// 所以 DeployCells 維持劇本（FDFIELD）順序，只有隊員順序改成 JOIN 順序。第六章
/// 收據（parity-ch06 r3）：slot 1 悠妮站第 2 格 (7,23)、slot 2 亞雷斯站第 3 格 (10,25)，
/// 而劇本裡亞雷斯排在悠妮前面。
///
/// Go：`main.go:4938`，`func reorderScenarioParty(sc *battle.Scenario, joinOrder []int) error`
pub(crate) fn reorder_scenario_party(sc: Option<&mut battle::Scenario>, join_order: &[i64]) -> Result<()> {
    todo!("main.go:reorderScenarioParty")
}

impl Game {
    /// focusOnParty 開局/戰鬥重開後把游標(=鏡頭中心)移到我方主角隊部署格的重心。
    /// 原鏡頭預設停在 (0,0),不對準的話玩家開局完全看不到主角隊(playfix #3)。
    /// 現有影片證據支持第一關開場含多幀人物走位，但確切起點、路徑與時序尚未閉合；
    /// 本函式只負責戰鬥控制權交接時把鏡頭對準已物化的我方隊伍，不宣稱重現該段演出。
    ///
    /// Go：`main.go:4967`，`func (g *Game) focusOnParty()`
    pub(crate) fn focus_on_party(&mut self) {
        todo!("main.go:focusOnParty")
    }

    /// campInput 處理 campaign 節點的輸入。回傳 true = 已攔截(擋掉戰場一般輸入)。
    ///
    /// Go：`main.go:5044`，`func (g *Game) campInput() bool`
    pub(crate) fn camp_input(&mut self) -> bool {
        todo!("main.go:campInput")
    }

    /// confirmBattleResult consumes the already displayed battle result at the
    /// explicit Enter boundary.  Keeping this seam separate prevents tests and
    /// future input adapters from bypassing the same postbattle/campaign handoff
    /// used by the player-facing control path.
    ///
    /// Go：`main.go:5413`，`func (g *Game) confirmBattleResult() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn confirm_battle_result(&mut self) -> bool {
        todo!("main.go:confirmBattleResult")
    }

    /// leaveChurch is the editable campaign boundary for the church menu's Escape
    /// action. It never assumes a specific town or service return target.
    ///
    /// Go：`main.go:5501`，`func (g *Game) leaveChurch()`
    pub(crate) fn leave_church(&mut self) {
        todo!("main.go:leaveChurch")
    }

    /// leaveShop is the campaign boundary for the shop's Escape/leave action. It
    /// deliberately delegates the next node to editable campaign data instead of
    /// assuming every shop returns to the same town.
    ///
    /// Go：`main.go:5550`，`func (g *Game) leaveShop()`
    pub(crate) fn leave_shop(&mut self) {
        todo!("main.go:leaveShop")
    }

    /// leaveHotel returns through the authored campaign edge. It never assumes a
    /// specific town or converts an unresolved raw hotel callee into gameplay.
    ///
    /// Go：`main.go:5588`，`func (g *Game) leaveHotel()`
    pub(crate) fn leave_hotel(&mut self) {
        todo!("main.go:leaveHotel")
    }

    /// ringInput 暫定四向命令選單 + 法術選單輸入。回傳 true = 已攔截。
    /// 方向配對已由 Docker Capstone 0x18d8c switch 釘死：↑0攻擊/←1法術/→2物品/↓3待機。
    ///
    /// Go：`main.go:5614`，`func (g *Game) ringInput() bool`
    pub(crate) fn ring_input(&mut self) -> bool {
        todo!("main.go:ringInput")
    }

    /// Go：`main.go:6201`，`func (g *Game) resetNativeTargetField() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn reset_native_target_field(&mut self) -> bool {
        todo!("main.go:resetNativeTargetField")
    }

    /// finishSelectedWait 對應原版行動選單第四項「下／休息」。未移動時走
    /// 0x13fd4：current HP != max HP 且 raw +0x25/+0x26 都為零才增加
    /// floor(max HP/5)，不保證最少回復 1。其後 0x190ac 檢查當格寶物；
    /// 不是踩上格子立即開箱，也不是道具指令。
    ///
    /// Go：`main.go:6239`，`func (g *Game) finishSelectedWait()`
    pub(crate) fn finish_selected_wait(&mut self) {
        todo!("main.go:finishSelectedWait")
    }

    /// finishSuccessfulUnitAction 對應 0x18890 外層在 action handler 成功返回後
    /// 才呼叫 selector1 的共同提交點。呼叫者必須先完成實際 mutation；取消、目標
    /// 不合法及 executor 錯誤不得抵達此處。after 保留各動作自己的介面清理。
    ///
    /// Go：`main.go:6288`，`func (g *Game) finishSuccessfulUnitAction(actor *battle.Unit, after func())`
    /// Go 的 func 參數 `after` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn finish_successful_unit_action(
        &mut self,
        actor: Option<UnitIdx>,
        after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) {
        todo!("main.go:finishSuccessfulUnitAction")
    }

    /// resetAllNativeMapPoses 是 0x134E4：對 [0x53BEB] 筆記錄逐筆 `+3 = 0`，再 delay(0x14) 毫秒。
    ///
    /// Go：`main.go:6369`，`func (g *Game) resetAllNativeMapPoses()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn reset_all_native_map_poses(&mut self) {
        todo!("main.go:resetAllNativeMapPoses")
    }

    /// autoEndPlayerPhase 是玩家控制器 0x117E7 在每個單位行動收尾（0x1E292 升級與
    /// 章節函式表之後、0x1198A 格子事件分派之前、0x11985）呼叫的 0x13565：掃全部 record，只要還有一筆
    /// `(+5 & 0x81)==0`、`+6==2`、`+0x26==0` 的單位就什麼都不做；一筆都沒有就直接
    /// 走 0x1A30B（與系統選單 END／YES 同一條路）換手。第四章原版收據
    /// （docs/data/ui-traces/parity-ch04.json）第 3 回合最後一個單位待機之後沒有送 END
    /// 就進了敵方回合，就是這一條。只有玩家自己的行動會觸發；AI 行動不在那個控制器裡。
    ///
    /// Go：`main.go:6386`，`func (g *Game) autoEndPlayerPhase(actor *battle.Unit) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn auto_end_player_phase(&mut self, actor: Option<UnitIdx>) -> bool {
        todo!("main.go:autoEndPlayerPhase")
    }

    /// awardDeathReward 在擊倒當下處理死亡效果：型態 0／1 直接給（只給原版陣營 2 的
    /// 擊殺者，見 grantNativeDeathReward），型態 2／3 登記成死亡程式，等行動收尾依記錄
    /// 順序執行。每個死亡 transition 只處理一次。
    ///
    /// Go：`main.go:6410`，`func (g *Game) awardDeathReward(dead, killer *battle.Unit)`
    pub(crate) fn award_death_reward(&mut self, dead: Option<UnitIdx>, killer: UnitIdx) {
        todo!("main.go:awardDeathReward")
    }
}

/// walkAnim 沿路徑逐格行走(玩家/AI 移動;FDICON 方向走動幀 + OffX/OffY 內插)。
///
/// Go：`main.go:6428`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct WalkAnim {
    pub(crate) u: Option<UnitIdx>,
    /// 含起點
    pub(crate) path: Vec<battle::Cell>,
    /// 目前段:path[seg] → path[seg+1]
    pub(crate) seg: i64,
    /// 原版 unit+4:每格 1..6，第7 tick提交目的格並直接接上下一段
    pub(crate) tick: i64,
    /// 走完回呼(nil=玩家預設:開指令環)
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    /// followView：走行步進家族搬視圖。玩家自己的走行（then 為 nil）一定跟；敵方 AI
    /// 走的是同一支 0x13488（0x14EC4），所以也跟；劇情走位另有 owner。
    pub(crate) follow_view: bool,
    /// nil 保留既有 selector0；全軍移動明確帶 selector1。
    pub(crate) native_event_selector: Option<u8>,
    /// 全軍移動的 selector1 事件已在確認前預演；walk 到達指定格時暫停，
    /// 由既有 event61／75 owner 完成後才續行。
    pub(crate) native_group_march_events: Vec<battle::NativeSystemGroupMarchEvent>,
    pub(crate) native_group_march_event: i64,
    pub(crate) native_group_march_paused: bool,
}

impl Game {
    /// Go：`main.go:6446`，`func (g *Game) stepBattleWalk()`
    pub(crate) fn step_battle_walk(&mut self) {
        todo!("main.go:stepBattleWalk")
    }
}

/// nativeMapGridMotionFrames 是原版每格發佈的 `+4` 動作幀數。dosgolem 收據
/// （docs/data/ui-traces/fd2-map-walk-pose-20260909.json，10 萬指令粒度）量到
/// 每格 `+4` 走 1..6，跨格那一幀 raw 座標換成新格、`+4` 同時重設為 1，中間
/// **沒有** `+4 = 0`；只有整段抵達才是 pose 0／`+4` 0。24 px 的格子配六幀等於
/// 每幀 4 px。
pub(crate) const NATIVE_MAP_GRID_MOTION_FRAMES: i64 = 6;

impl Game {
    /// stepBattleWalkSegment 推進一拍。回傳 true 表示這一拍只做了跨格提交、還要
    /// 在同一幀接著發佈下一段的第一拍（原版沒有跨格的 `+4 = 0` 幀）。
    ///
    /// Go：`main.go:6487`，`func (g *Game) stepBattleWalkSegment(w *walkAnim, finish func(pose int)) bool`
    pub(crate) fn step_battle_walk_segment(&mut self, w: &mut WalkAnim, finish: &mut dyn FnMut(i64)) -> bool {
        todo!("main.go:stepBattleWalkSegment")
    }
}

/// nativeMapRestPose 是原版單位靜止時的 raw +3 值。移動抵達、等待指令期間都是
/// 這個值；行走途中才是移動方向。收據見 docs/knowledge-base/97-map-walk-pose-20260909.md。
pub(crate) const NATIVE_MAP_REST_POSE: i64 = 0;

pub(crate) const SFX_CURSOR: i64 = 0;

pub(crate) const SFX_CONFIRM: i64 = 12;

impl Game {
    /// loadMap 載入一張戰場(dir 下的 map.json + tileset.png,並切圖塊)。
    /// dir 例:"assets"(map0 舊結構)或 "assets/maps/map3"(全 33 圖匯出結構)。
    ///
    /// Go：`main.go:6587`，`func (g *Game) loadMap(dir string) error`
    pub(crate) fn load_map(&mut self, dir: &str) -> Result<()> {
        todo!("main.go:loadMap")
    }
}

/// battleFPT 戰鬥演出播放速度(tick/幀):環境變數 FD2_BATTLE_FPT 可調(調慢=數字大),預設 3。
///
/// Go：`main.go:6664`，`func battleFPT() int`
pub(crate) fn battle_fpt() -> i64 {
    todo!("main.go:battleFPT")
}

impl Game {
    /// newAtkAnim 建立全螢幕戰鬥演出(所有角色通用):攻方=攻擊動作(組×3+1)、守方=待機(組×3),
    /// 幀長由 FIGANI descriptor +6 與明確的 fpt 顯示倍率決定；位置/走位由幀內嵌 (dx,dy)
    /// 資料驅動(doc06)。缺少一一配對的延遲表時返回 nil，呼叫端必須明確記錄呈現缺口，
    /// 不得宣稱畫面已播放。
    ///
    /// Go：`main.go:6675`，`func (g *Game) newAtkAnim(atkGroup, defGroup int, atkName, defName string, atkHP, atkMax, atkLV, atkMP, atkMaxMP, defLV, defMP, defMaxMP, defHP0, defHP1, defMax, terrain int, atkOwn bool) *atkAnim`
    pub(crate) fn new_atk_anim(
        &self,
        atk_group: i64,
        def_group: i64,
        atk_name: &str,
        def_name: &str,
        atk_hp: i64,
        atk_max: i64,
        atk_lv: i64,
        atk_mp: i64,
        atk_max_mp: i64,
        def_lv: i64,
        def_mp: i64,
        def_max_mp: i64,
        def_hp0: i64,
        def_hp1: i64,
        def_max: i64,
        terrain: i64,
        atk_own: bool,
    ) -> Option<AtkAnim> {
        todo!("main.go:newAtkAnim")
    }

    /// newCounterStage 建反擊那一段：攻方換成原守方的攻擊動作、守方換成原攻方的待機。
    /// 資源或延遲表缺一不可——缺了就回 nil，呼叫端維持只有主攻的演出，而不是用猜的
    /// 幀數硬演一段。defHP0／defHP1 是這一段守方（也就是原攻方）反擊前後的 HP。
    ///
    /// Go：`main.go:6715`，`func (g *Game) newCounterStage(counterAtkGroup, counterDefGroup, defHP0, defHP1, defMax int) *atkStage`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn new_counter_stage(
        &self,
        counter_atk_group: i64,
        counter_def_group: i64,
        def_hp0: i64,
        def_hp1: i64,
        def_max: i64,
    ) -> Option<AtkStage> {
        todo!("main.go:newCounterStage")
    }

    /// attachCounterPresentation 把反擊接成同一段演出的第二半，並把第一段顯示的攻方
    /// HP 修回反擊之前的值——結算在演出之前就完成了，actor.HP 這時已經是被反擊之後的。
    ///
    /// 反擊的素材缺件時只接不上第二段，主攻那一段照常演；傷害本身已經結算，不會因為
    /// 演不出來就不算。
    ///
    /// Go：`main.go:6742`，`func (g *Game) attachCounterPresentation(a *atkAnim, actor, target *battle.Unit, result battle.AttackResult)`
    pub(crate) fn attach_counter_presentation(
        &self,
        a: Option<&mut AtkAnim>,
        actor: Option<UnitIdx>,
        target: Option<UnitIdx>,
        result: &battle::AttackResult,
    ) {
        todo!("main.go:attachCounterPresentation")
    }

    /// nativeAttackPresentationAvailable 執行 newAtkAnim 所需的完整資產與排程檢查，
    /// 但不消耗戰鬥亂數，也不修改 HP；敵方 AI 以此作為失敗即關閉的交易預檢。
    ///
    /// Go：`main.go:6753`，`func (g *Game) nativeAttackPresentationAvailable(atkGroup, defGroup int) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_attack_presentation_available(&self, atk_group: i64, def_group: i64) -> bool {
        todo!("main.go:nativeAttackPresentationAvailable")
    }
}

/// attackPresentationTickMillis 是一個顯示 tick 的長度。原版 `sub_2939D` 對每一格
/// 重複 `cell+6` 次 `sub_17AA9(1)`，每次一個 BIOS tick；fpt 是重製端在一個原版延遲
/// 單位裡插幾個顯示 tick 的解析度，所以一個顯示 tick 就是 BIOS tick 除以 fpt。
///
/// Go：`main.go:6777`，`func attackPresentationTickMillis(fpt int) float64`
pub(crate) fn attack_presentation_tick_millis(fpt: i64) -> f64 {
    todo!("main.go:attackPresentationTickMillis")
}

/// attackPresentationTicks 把累積的毫秒換成這一畫格要推進幾個顯示 tick，並回傳
/// 還沒換掉的餘數。餘數要留著——丟掉的話 60 Hz 與 BIOS tick 的不整除會讓整段
/// 演出愈跑愈短。
///
/// Go：`main.go:6787`，`func attackPresentationTicks(accumMillis float64, fpt int) (ticks int, rest float64)`
pub(crate) fn attack_presentation_ticks(accum_millis: f64, fpt: i64) -> (i64, f64) {
    todo!("main.go:attackPresentationTicks")
}

/// errAttackPresentationYield 不是失敗：它表示這一幀要在演出收尾之後立刻交還，
/// 因為接手的事件流程（field event 61 或 battleEvent）已經接管。
///
/// Go：`main.go:6799`
/// Go `errors.New` 的哨兵錯誤；`errors.Is` 以 [`GameError::is`] 判斷。
pub(crate) static ERR_ATTACK_PRESENTATION_YIELD: GameError =
    GameError::sentinel("attack presentation yielded to an event");

impl Game {
    /// stepAttackPresentationTick 推進一個**顯示 tick**（不是一個畫格）：走一步
    /// FIGANI 時間軸、判三段音效、時間到就收掉演出。
    ///
    /// Go：`main.go:6803`，`func (g *Game) stepAttackPresentationTick() error`
    pub(crate) fn step_attack_presentation_tick(&mut self) -> Result<()> {
        todo!("main.go:stepAttackPresentationTick")
    }

    /// Go：`main.go:6918`，`func (g *Game) finishAttackPresentation()`
    pub(crate) fn finish_attack_presentation(&mut self) {
        todo!("main.go:finishAttackPresentation")
    }

    /// Go：`main.go:6962`，`func (g *Game) completeAttackPresentation()`
    pub(crate) fn complete_attack_presentation(&mut self) {
        todo!("main.go:completeAttackPresentation")
    }
}

/// figaniIndex maps the battle FIGANI visual selector to its resource trio.
/// Native ABI keeps this separate from the map FDICON selector: 0x127e0 uses
/// unit+2 ×12 for map sprites, while 0x287b5..0x2884c uses unit+7 ×3 for
/// FIGANI. Current exported records only pass the shared visual id where that
/// equality is evidenced; this helper must not be read as an ABI field alias.
///
/// Go：`main.go:6978`，`func figaniIndex(fig int) int`
pub(crate) fn figani_index(fig: i64) -> i64 {
    todo!("main.go:figaniIndex")
}

/// loadSprites 載入 assets/sprites/fig_NNN_fMM.png,按 fig index 分組成幀序列。
///
/// Go：`main.go:6981`，`func loadSprites() map[int][]*ebiten.Image`
pub(crate) fn load_sprites() -> BTreeMap<i64, Vec<Option<Arc<Picture>>>> {
    todo!("main.go:loadSprites")
}

/// loadPortraits 載入 assets/portraits/DATO_NNN_mM.png,按肖像 id 分組成 4 嘴型幀。
///
/// Go：`main.go:7017`，`func loadPortraits() map[int][]*ebiten.Image`
pub(crate) fn load_portraits() -> BTreeMap<i64, Vec<Option<Arc<Picture>>>> {
    todo!("main.go:loadPortraits")
}

/// loadFIGANI 載入 assets/figani/FIGANI_NNN_fNN.png,按 fig id 分組成攻擊全身分鏡。
///
/// Go：`main.go:7053`，`func loadFIGANI() map[int][]*ebiten.Image`
pub(crate) fn load_figani() -> BTreeMap<i64, Vec<Option<Arc<Picture>>>> {
    todo!("main.go:loadFIGANI")
}

/// loadFIGANIDelays loads descriptor +6 values exported from the fixed
/// player-provided FIGANI.DAT. It is deliberately separate from the PNG
/// presentation assets: a missing, malformed, or mismatched schedule must not
/// be replaced with a guessed frame count or delay.
///
/// Go：`main.go:7092`，`func loadFIGANIDelays() (map[int][]int, error)`
pub(crate) fn load_figani_delays() -> Result<BTreeMap<i64, Vec<i64>>> {
    todo!("main.go:loadFIGANIDelays")
}

/// loadStoryScriptWithIdentityAt joins the legacy runtime script (numeric
/// speaker/control provenance) with the editor-canonical story (stable line_id).
/// Array positions are accepted only after every line's source text and legacy
/// speaker agree, so a reordered or stale document fails closed.
///
/// Go：`main.go:7164`，`func loadStoryScriptWithIdentityAt(path, scene string, sceneIndex *int) ([]campaign.Line, error)`
pub(crate) fn load_story_script_with_identity_at(
    path: &str,
    scene: &str,
    scene_index: Option<&i64>,
) -> Result<Vec<campaign::Line>> {
    todo!("main.go:loadStoryScriptWithIdentityAt")
}

/// handlerStoryPath converts a StoryIndexMap script path (relative to
/// assets/story) into the normal asset lookup path.  Existing authored beats
/// may already carry assets/... or an absolute path, which stays untouched.
///
/// Go：`main.go:7240`，`func handlerStoryPath(script string) string`
pub(crate) fn handler_story_path(script: &str) -> String {
    todo!("main.go:handlerStoryPath")
}

/// loadFigMeta 載入 FIGANI 每幀內嵌絕對螢幕座標 (dx,dy)@320(assets/figani/meta.json;doc06:
/// 幀標頭 +0/+2,動畫的走位/伸擊/突刺全靠逐幀 (dx,dy) 變化,引擎不需錨點/位移計算)。
///
/// Go：`main.go:7249`，`func loadFigMeta() map[int][][2]int`
pub(crate) fn load_fig_meta() -> BTreeMap<i64, Vec<[i64; 2]>> {
    todo!("main.go:loadFigMeta")
}

/// Go：`main.go:7293`，`func absInt(v int) int`
pub(crate) fn abs_int(v: i64) -> i64 {
    todo!("main.go:absInt")
}

impl Game {
    /// resolvePhysicalAttackFull 回傳完整結算，含守方反擊拿到的經驗與升級。
    ///
    /// Go：`main.go:7315`，`func (g *Game) resolvePhysicalAttackFull(actor, target *battle.Unit) (battle.NativePhysicalAttackResult, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn resolve_physical_attack_full(
        &mut self,
        actor: Option<UnitIdx>,
        target: Option<UnitIdx>,
    ) -> Result<battle::NativePhysicalAttackResult> {
        todo!("main.go:resolvePhysicalAttackFull")
    }

    /// publishPhysicalAttackMessage 只在 F3 診斷模式顯示物理攻擊的結果字串。
    ///
    /// 原版沒有這一行。`battle.attack.hit`／`miss`／`critical_suffix`／`exp_suffix`
    /// 四筆的 `source_string_id` 都指向重製端自己的 Go 原始碼，而原版分離出的
    /// FDTXT 全庫沒有任何「造成 N 傷害」型的戰鬥結果模板（僅有的「傷害／造成」
    /// 字樣都在劇情對白裡）。原版的傷害數字出現在全螢幕戰鬥演出**之內**，不是
    /// 戰鬥結束後留在地圖底部的一行字。
    ///
    /// 字串本身保留：它同時是語言包完整性的前置檢查，缺條目要在改動 HP 之前就
    /// 失敗。這裡只擋顯示。
    ///
    /// Go：`main.go:7360`，`func (g *Game) publishPhysicalAttackMessage(message string)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn publish_physical_attack_message(&mut self, message: &str) {
        todo!("main.go:publishPhysicalAttackMessage")
    }
}

/// Go：`main.go:7367`，`func playerPhysicalAttackMessage(catalog *localization.Catalog, actor, target *battle.Unit, result battle.AttackResult) (string, error)`
pub(crate) fn player_physical_attack_message(
    catalog: Option<&localization::Catalog>,
    actor: Option<&battle::Unit>,
    target: Option<&battle::Unit>,
    result: &battle::AttackResult,
) -> Result<String> {
    todo!("main.go:playerPhysicalAttackMessage")
}

/// dirToward 從 (ax,ay) 朝 (tx,ty) 的方向:0下 1左 2上 3右(FDICON 方向幀)。
///
/// Go：`main.go:7407`，`func dirToward(ax, ay, tx, ty int) int`
pub(crate) fn dir_toward(ax: i64, ay: i64, tx: i64, ty: i64) -> i64 {
    todo!("main.go:dirToward")
}

impl Game {
    /// confirm 處理 Enter/Space:選取我方單位顯示移動範圍,或移動到可達格 / 原地待機。
    ///
    /// Go：`main.go:7422`，`func (g *Game) confirm()`
    pub(crate) fn confirm(&mut self) {
        todo!("main.go:confirm")
    }

    /// checkResult 檢查勝負(失敗條件:索爾死;勝利:敵全滅,doc28 第1章)。
    ///
    /// Go：`main.go:8308`，`func (g *Game) checkResult()`
    pub(crate) fn check_result(&mut self) {
        todo!("main.go:checkResult")
    }

    /// Go：`main.go:8370`，`func (g *Game) Update() error`
    pub(crate) fn update(&mut self) -> Result<()> {
        todo!("main.go:Update")
    }

    /// Go：`main.go:9179`，`func (g *Game) syncNativeMapView() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn sync_native_map_view(&mut self) -> bool {
        todo!("main.go:syncNativeMapView")
    }

    /// nativeCursorStepHUD 在一格游標步之後照 0x11C59 家族的重繪規則更新 HUD anchor：
    /// 沒重繪（只動可見游標且 [0x51A83]==0）就不評估 0x1AD2A。
    ///
    /// Go：`main.go:9192`，`func (g *Game) nativeCursorStepHUD(before, after battle.NativeMapViewState)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_cursor_step_hud(
        &mut self,
        before: battle::NativeMapViewState,
        after: battle::NativeMapViewState,
    ) {
        todo!("main.go:nativeCursorStepHUD")
    }

    /// redrawNativeMapHUD 是一次原版 0x11CAC 重繪裡 0x1ACF3→0x1AD2A 的 HUD anchor 評估：閘 A／閘 B
    /// 由 AdvanceNativeMapHUDAnchor 判，可見游標取當下視圖。重製端每幀 Draw 的整幀重組不是原版的
    /// 重繪（走行步進 0x12EAA 家族就不呼叫 0x11CAC），所以 anchor 只在對應原版重繪點的地方經這裡
    /// 評估；對照表見 56 §#40 重繪入口。
    ///
    /// Go：`main.go:9212`，`func (g *Game) redrawNativeMapHUD()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn redraw_native_map_hud(&mut self) {
        todo!("main.go:redrawNativeMapHUD")
    }

    /// Go：`main.go:9220`，`func (g *Game) moveMapCursor(dx, dy int)`
    pub(crate) fn move_map_cursor(&mut self, dx: i64, dy: i64) {
        todo!("main.go:moveMapCursor")
    }

    /// focusNativeMapCursorOnUnit 重現 0x12D7B→0x12CEA：讀單位記錄的 x/y，再把游標
    /// 先 X 後 Y 逐格移過去。逐格走的是四個鍵盤處理器，所以可見游標與鏡頭跟著同一套
    /// 安全帶規則，三組全域不會脫節。
    ///
    /// Go：`main.go:9236`，`func (g *Game) focusNativeMapCursorOnUnit(u *battle.Unit)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn focus_native_map_cursor_on_unit(&mut self, u: Option<UnitIdx>) {
        todo!("main.go:focusNativeMapCursorOnUnit")
    }

    /// aiFocusCursor 是敵方回合裡的 0x12D7B／0x12CEA：游標先 X 後 Y 逐格走到 (x,y)，
    /// 可見游標與鏡頭照鍵盤處理器的安全帶規則跟著動。第四章 r10 收據（eip-trace
    /// 0x12CEA）的順序：走路的單位先聚焦自己（0x13BA5／0x14203）再走（0x13488 跟視圖）；
    /// 攻擊前 0x1548E 再聚焦自己（0x154AD）與目標（0x154DE）；指令／物品路徑聚焦自己後
    /// 把游標移到目標格（0x153C2／0x1515B）。原版每一步有沒有等 tick 只影響時間，
    /// 不影響最後的視圖狀態。
    ///
    /// Go：`main.go:9253`，`func (g *Game) aiFocusCursor(x, y int)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn ai_focus_cursor(&mut self, x: i64, y: i64) {
        todo!("main.go:aiFocusCursor")
    }

    /// followNativeMapCursorStep 讓視圖跟著走行的單位走一格。判準是單位自己的
    /// 相對列／行，不是已存的可見游標值；寫入端清單與安全帶規則見
    /// docs/data/ida/fd2_visible_cursor_writers_ida.txt。
    ///
    /// Go：`main.go:9270`，`func (g *Game) followNativeMapCursorStep(fromX, fromY, dx, dy int)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn follow_native_map_cursor_step(&mut self, from_x: i64, from_y: i64, dx: i64, dy: i64) {
        todo!("main.go:followNativeMapCursorStep")
    }

    /// positionScreenshotCursor drives the same recovered cursor/camera state
    /// machine as interactive input. Directly assigning curX/curY leaves the
    /// production indexed frame on stale NativeMapViewState and produces a false
    /// visual oracle.
    ///
    /// Go：`main.go:9287`，`func (g *Game) positionScreenshotCursor(x, y int) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn position_screenshot_cursor(&mut self, x: i64, y: i64) -> bool {
        todo!("main.go:positionScreenshotCursor")
    }
}

/// keyRepeat 方向鍵按住持續觸發(playfix #1):首次按下立即動一格,按住 12 tick 後
/// 每 5 tick 再動一格(可玩性手感參數,原版掃描碼節奏未逐格量測,見 doc13 §游標)。
///
/// Go：`main.go:9312`，`func keyRepeat(k ebiten.Key) bool`
pub(crate) fn key_repeat(k: Key) -> bool {
    todo!("main.go:keyRepeat")
}

/// Go：`main.go:9326`，`func clamp(v *float64, lo, hi float64)`
pub(crate) fn clamp(v: &mut f64, lo: f64, hi: f64) {
    todo!("main.go:clamp")
}

impl Game {
    /// Go：`main.go:9355`，`func (g *Game) Draw(screen *ebiten.Image)`
    pub(crate) fn draw(&mut self, screen: &mut Picture) {
        todo!("main.go:Draw")
    }

    /// Go：`main.go:10305`，`func (g *Game) usesModernBattleActionIcons() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn uses_modern_battle_action_icons(&mut self) -> bool {
        todo!("main.go:usesModernBattleActionIcons")
    }

    /// Go：`main.go:10339`，`func (g *Game) actionOverlayAvailability() [4]int`
    pub(crate) fn action_overlay_availability(&mut self) -> [i64; 4] {
        todo!("main.go:actionOverlayAvailability")
    }
}

/// hasNativeEquippedWeapon is the recovered 0x1b83d inventory precondition.
/// Inventory/Equipped are the remake's compact projection of the native eight
/// slots; a missing equipped entry is deliberately not treated as a weapon.
///
/// Go：`main.go:10413`，`func hasNativeEquippedWeapon(unit *battle.Unit) bool`
pub(crate) fn has_native_equipped_weapon(unit: Option<&battle::Unit>) -> bool {
    todo!("main.go:hasNativeEquippedWeapon")
}

/// nativeCommandAllowedAfterMove 是 0x18a5f..0x18a70 的三個 raw +7 常數；其餘
/// 單位移動之後指令格一律不可選。
///
/// Go：`main.go:10431`，`func nativeCommandAllowedAfterMove(battleFig int) bool`
pub(crate) fn native_command_allowed_after_move(battle_fig: i64) -> bool {
    todo!("main.go:nativeCommandAllowedAfterMove")
}

/// hasNativeCommand uses only the exact 0x1c269 bit inventory.  It must not
/// infer native command availability from the normalized editable Spells list.
///
/// Go：`main.go:10442`，`func hasNativeCommand(unit *battle.Unit) bool`
pub(crate) fn has_native_command(unit: Option<&battle::Unit>) -> bool {
    todo!("main.go:hasNativeCommand")
}

/// nativeCommandActionBlocked is the raw action-menu gate recovered from
/// 0x18d8c: after 0x1c269 finds a command, unit+0x27 nonzero disables the
/// command direction. Command 22 is a verified writer of this duration byte,
/// but its gameplay name and exhaustive producer set remain intentionally
/// unknown; NativeTransient[5] is the fail-closed storage of that byte.
///
/// Go：`main.go:10459`，`func nativeCommandActionBlocked(unit *battle.Unit) bool`
pub(crate) fn native_command_action_blocked(unit: Option<&battle::Unit>) -> bool {
    todo!("main.go:nativeCommandActionBlocked")
}

/// nativeActionSelectable mirrors the final 0x177fc availability check after
/// a direction is chosen: only a zero disabled-word may dispatch an action.
///
/// Go：`main.go:10465`，`func nativeActionSelectable(availability [4]int, direction int) bool`
pub(crate) fn native_action_selectable(availability: [i64; 4], direction: i64) -> bool {
    todo!("main.go:nativeActionSelectable")
}

impl Game {
    /// nativeActionOverlayState keeps each original 0x1741c caller's two raw
    /// tables separate. The shared empty-cursor system owner is not a battle-unit
    /// wrapper: it uses 0x16f55's [7,5,6,4]/[0,0,0,0] tables and therefore must not
    /// borrow the normalized selected-unit availability approximation.
    ///
    /// Go：`main.go:10516`，`func (g *Game) nativeActionOverlayState() fdother.ActionOverlayState`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_action_overlay_state(&mut self) -> fdother::ActionOverlayState {
        todo!("main.go:nativeActionOverlayState")
    }

    /// drawBattleScene 全螢幕戰鬥演出(對照原版 orig_05:守方左面右/攻方右土台/斬擊弧/血條/命中閃紅抽血)。
    ///
    /// Go：`main.go:11231`，`func (g *Game) drawBattleScene(screen *ebiten.Image)`
    pub(crate) fn draw_battle_scene(&mut self, screen: &mut Picture) {
        todo!("main.go:drawBattleScene")
    }
}

pub(crate) const NATIVE_ACTION_OVERLAY_CELL_COUNT: i64 = 78;

/// loadNativeCommandLabels reads the editable export of FDTXT_000 command
/// labels. It is optional because the export is player-provided original text;
/// absence deliberately leaves the normalized presentation names intact.
///
/// Go：`main.go:11595`，`func loadNativeCommandLabels() map[int]string`
pub(crate) fn load_native_command_labels() -> BTreeMap<i64, String> {
    todo!("main.go:loadNativeCommandLabels")
}

/// Go：`main.go:11619`，`func loadGame() *Game`
pub(crate) fn load_game() -> Option<Game> {
    todo!("main.go:loadGame")
}

pub(crate) const PHASE_BANNER_ENEMY_HOLD_MILLIS: i64 = 20;

pub(crate) const PHASE_BANNER_PLAYER_HOLD_MILLIS: i64 = 150;

pub(crate) const PHASE_BANNER_PLAYER_TEXT: &str = "PLAYER PHASE";

/// phaseBannerHoldMillis 回傳中段停留的毫秒數。未知字樣時採敵方回合的 20 毫秒，
/// 這是兩者中較短的一個，不會憑空拉長演出。
///
/// Go：`main.go:12132`，`func phaseBannerHoldMillis(banner string) float64`
pub(crate) fn phase_banner_hold_millis(banner: &str) -> f64 {
    todo!("main.go:phaseBannerHoldMillis")
}

/// phaseBannerTotalMillis 是整段橫幅的原版長度：字樣滑入 7 步、馬賽克 33 步、
/// 滑出 5 步，各一個 BIOS tick，加上中段停留。
///
/// Go：`main.go:12141`，`func phaseBannerTotalMillis(banner string) float64`
pub(crate) fn phase_banner_total_millis(banner: &str) -> f64 {
    todo!("main.go:phaseBannerTotalMillis")
}

/// phaseBannerFrames 把原版長度換成 60 Hz 的重製端幀數(無條件進位，寧可多留一
/// 幀也不提前收掉最後一步)。
///
/// Go：`main.go:12175`，`func phaseBannerFrames(banner string) int`
pub(crate) fn phase_banner_frames(banner: &str) -> i64 {
    todo!("main.go:phaseBannerFrames")
}

impl Game {
    /// Go：`main.go:12254`，`func (g *Game) showBanner(s string)`
    pub(crate) fn show_banner(&mut self, s: &str) {
        todo!("main.go:showBanner")
    }

    /// drawNativeMapHUD presents the proven indexed 0x1acf3 panel/terrain/AP/DP
    /// subpasses at the native 320x200 surface. It returns false on any missing or
    /// unverified raw input, preserving the legacy approximation without partial
    /// drawing. Raw gates/anchor and optional unit/HP must all come from the
    /// materialized battle runtime; this function never hardcodes native globals.
    ///
    /// Go：`main.go:12356`，`func (g *Game) nativeMapHUDInput() (indexedmap.NativeMapHUDInput, bool)`
    pub(crate) fn native_map_hud_input(&mut self) -> (indexedmap::NativeMapHUDInput, bool) {
        todo!("main.go:nativeMapHUDInput")
    }

    /// Go：`main.go:12527`，`func (g *Game) composeNativeMapFrame() error`
    pub(crate) fn compose_native_map_frame(&mut self) -> Result<()> {
        todo!("main.go:composeNativeMapFrame")
    }

    /// composeNativeMapFrameForActionBackground 的 closedAction 投影只供 0x190AC
    /// 取寶提示底圖；依 17643 的背景恢復與 seq3820 的同狀態畫面，保留原始
    /// selector，省略互動 range／selection。契約見 SDD #53，不能套用其他對話。
    ///
    /// Go：`main.go:12534`，`func (g *Game) composeNativeMapFrameForActionBackground(closedAction bool) error`
    pub(crate) fn compose_native_map_frame_for_action_background(&mut self, closed_action: bool) -> Result<()> {
        todo!("main.go:composeNativeMapFrameForActionBackground")
    }

    /// 正常物理返回呼叫11CAC(1)，只略過4DFCC；其他地圖層及時鐘照常。
    /// 契約見physical_return_palette_spec，不用closedAction省略range／selection。
    ///
    /// Go：`main.go:12540`，`func (g *Game) composeNativeMapFrameForPhysicalReturn() error`
    pub(crate) fn compose_native_map_frame_for_physical_return(&mut self) -> Result<()> {
        todo!("main.go:composeNativeMapFrameForPhysicalReturn")
    }

    /// Go：`main.go:12544`，`func (g *Game) composeNativeMapFrameForCaller(closedAction, updateSteadyPalette bool) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn compose_native_map_frame_for_caller(
        &mut self,
        closed_action: bool,
        update_steady_palette: bool,
    ) -> Result<()> {
        todo!("main.go:composeNativeMapFrameForCaller")
    }

    /// Go：`main.go:12570`，`func (g *Game) composeNativeMapFrameAtWithPaletteCycle(now time.Time, closedAction, updateSteadyPalette bool) error`
    pub(crate) fn compose_native_map_frame_at_with_palette_cycle(
        &mut self,
        now: GameInstant,
        closed_action: bool,
        update_steady_palette: bool,
    ) -> Result<()> {
        todo!("main.go:composeNativeMapFrameAtWithPaletteCycle")
    }

    /// endTurn 是 0x1A30B 的入口（系統選單 END／YES、全員行動完的 0x13565、全軍移動收尾
    /// 都走這裡）。原版順序：selector 1 回合事件（0x1A813(1)）→ selector 1 暫時狀態
    /// → 友軍 AI → selector 0 回合事件 → selector 0 暫時狀態 → 敵方 AI → 回合數加一。
    /// 可編輯的回合事件依 NativeTurnPhaseSelector 插在對應位置；沒有 selector 的手寫
    /// 事件仍留在敵方回合之後（finishTurn）。
    ///
    /// Go：`main.go:12766`，`func (g *Game) endTurn()`
    pub(crate) fn end_turn(&mut self) {
        todo!("main.go:endTurn")
    }

    /// Go：`main.go:12793`，`func (g *Game) endTurnAfterSelector1Events()`
    pub(crate) fn end_turn_after_selector1_events(&mut self) {
        todo!("main.go:endTurnAfterSelector1Events")
    }

    /// nativeTransientSweepAvailable：sub_1A866 的掃描只在名冊每筆都有 raw +5／+6 出處時
    /// 跑（原版名冊或 saved runtime 投影）；沒有 raw 出處的舊可編輯劇本沒有 +0x22..+0x27，
    /// 跳過。第六章 r4 收據：敵方法師的強化 +0x22 每回合 −1、歸零後才會再被 0x1598A
    /// 的指令 17 評分選中，不遞減就永遠不再施放。
    ///
    /// Go：`main.go:12818`，`func (g *Game) nativeTransientSweepAvailable() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_transient_sweep_available(&self) -> bool {
        todo!("main.go:nativeTransientSweepAvailable")
    }

    /// beginEnemyPhaseAfterTurnEvents 進敵方回合。0x1A30B 的順序是橫幅（0x1A3A2）→
    /// 0x13536 清已行動 → 0x1A813(0) 的 selector 0 回合事件 → 0x1D8BA 敵方 AI，所以
    /// selector 0 的事件（第五章第 3 回合的 event 14：AI 模式改寫＋對白）要等橫幅播完才跑，
    /// 由 aiStep 在清 bit7 之後接手；不跑 AI 的截圖模式直接在這裡跑完再收回合。
    ///
    /// Go：`main.go:12837`，`func (g *Game) beginEnemyPhaseAfterTurnEvents()`
    pub(crate) fn begin_enemy_phase_after_turn_events(&mut self) {
        todo!("main.go:beginEnemyPhaseAfterTurnEvents")
    }

    /// runEditableTurnEvents 觸發 on_turn_end 裡 phase selector 相符的事件，跑完再 then。
    ///
    /// Go：`main.go:12846`，`func (g *Game) runEditableTurnEvents(selector int, then func())`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn run_editable_turn_events(&mut self, selector: i64, then: Option<Box<dyn FnOnce(&mut Game) + Send>>) {
        todo!("main.go:runEditableTurnEvents")
    }

    /// Go：`main.go:12866`，`func (g *Game) beginEnemyPhase()`
    pub(crate) fn begin_enemy_phase(&mut self) {
        todo!("main.go:beginEnemyPhase")
    }

    /// startBattleEvent runs ordered scenario actions without borrowing campaign's
    /// BeatRunner. Immediate state actions execute in-order; pan/delay/dialogue block
    /// until their visual/input boundary finishes, then resume at the next action.
    ///
    /// Go：`main.go:12883`，`func (g *Game) startBattleEvent(actions []battle.Action, then func())`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn start_battle_event(
        &mut self,
        actions: &mut [battle::Action],
        then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    ) {
        todo!("main.go:startBattleEvent")
    }

    /// Go：`main.go:12900`，`func (g *Game) advanceBattleEvent()`
    pub(crate) fn advance_battle_event(&mut self) {
        todo!("main.go:advanceBattleEvent")
    }

    /// Go：`main.go:13040`，`func (g *Game) stepBattleEventDelay()`
    pub(crate) fn step_battle_event_delay(&mut self) {
        todo!("main.go:stepBattleEventDelay")
    }

    /// finishTurn starts the ordered on_turn_end event. Turn/status bookkeeping is
    /// deferred until the complete visual sequence finishes.
    ///
    /// Go：`main.go:13052`，`func (g *Game) finishTurn()`
    pub(crate) fn finish_turn(&mut self) {
        todo!("main.go:finishTurn")
    }

    /// Go：`main.go:13070`，`func (g *Game) completeTurn()`
    pub(crate) fn complete_turn(&mut self) {
        todo!("main.go:completeTurn")
    }

    /// Go：`main.go:13097`，`func (g *Game) completeTurnPlayerPhase()`
    pub(crate) fn complete_turn_player_phase(&mut self) {
        todo!("main.go:completeTurnPlayerPhase")
    }

    /// Go：`main.go:13110`，`func (g *Game) finishNativeTransientPlayerPhase()`
    pub(crate) fn finish_native_transient_player_phase(&mut self) {
        todo!("main.go:finishNativeTransientPlayerPhase")
    }

    /// Go：`main.go:13119`，`func (g *Game) finishNativeTransientPlayerPhaseInput()`
    pub(crate) fn finish_native_transient_player_phase_input(&mut self) {
        todo!("main.go:finishNativeTransientPlayerPhaseInput")
    }

    /// restoreNativeDisplayGateB 是 0x1A30B 返回後的 0x17277／0x135D4 閘 B 寫回；輸入迴圈接著的
    /// 0x11CAC 重繪經 0x1ACF3 依當下可見游標評估 anchor。
    ///
    /// Go：`main.go:13140`，`func (g *Game) restoreNativeDisplayGateB()`
    pub(crate) fn restore_native_display_gate_b(&mut self) {
        todo!("main.go:restoreNativeDisplayGateB")
    }

    /// aiStep AI 回合驅動:一次取一個單位的行動計畫,播行走動畫→到位攻擊(全螢幕演出)。
    /// 全單位動完 → finishTurn。
    ///
    /// Go：`main.go:13151`，`func (g *Game) aiStep()`
    pub(crate) fn ai_step(&mut self) {
        todo!("main.go:aiStep")
    }
}

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
///
/// R4 A 組（共用變更需求）：`loadMap` 要讀 `map.json`，加上 `Deserialize`。缺欄位與 `null` 照 Go
/// `json.Unmarshal` 留零值；`[]byte` 欄位是 base64（`fd2_rules::battle::go_bytes`）。Go 的鍵比對不分
/// 大小寫，Rust 只認標籤上的寫法（正式匯出檔與標籤相同）。
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Deserialize)]
#[serde(default)]
pub(crate) struct MapData {
    /// Go 標籤：`json:"w"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) w: i64,
    /// Go 標籤：`json:"h"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) h: i64,
    /// Go 標籤：`json:"tileW"`
    #[serde(rename = "tileW", deserialize_with = "battle::null_as_default")]
    pub(crate) tile_w: i64,
    /// Go 標籤：`json:"tileH"`
    #[serde(rename = "tileH", deserialize_with = "battle::null_as_default")]
    pub(crate) tile_h: i64,
    /// tileset 每列圖塊數
    ///
    /// Go 標籤：`json:"cols"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) cols: i64,
    /// Go 標籤：`json:"tiles"`
    #[serde(deserialize_with = "battle::null_as_default")]
    pub(crate) tiles: Vec<i64>,
    /// Native fields are optional: legacy PNG-only maps must not be treated as
    /// verified indexed input. They mirror export_engine_assets.py's raw FDFIELD
    /// composition bytes and FDSHAP four-byte control table.
    ///
    /// Go 標籤：`json:"native_composition_event_bytes,omitempty"`
    #[serde(with = "battle::go_bytes")]
    pub(crate) native_composition_event_bytes: Vec<u8>,
    /// Go 標籤：`json:"native_tile_blit_modes,omitempty"`
    #[serde(with = "battle::go_bytes")]
    pub(crate) native_tile_blit_modes: Vec<u8>,
    /// Go 標籤：`json:"native_terrain_control,omitempty"`
    #[serde(with = "battle::go_bytes")]
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
    pub(crate) modern_hd_separate_world: bool,
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
    pub(crate) title_load_slots_fallback: bool,
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
    /// 敵方指令／道具 owner 成功收尾後的接續（Go `executeNativeAIActionWithContinuation` 的 `after`）；
    /// 只有 mode 11 在分派前設定，各 owner 在成功邊界取走。
    pub(crate) native_ai_action_after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
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
    pub(crate) native_dialogue_hd_masks: Option<ModernStoryDialogueMasks>,
    pub(crate) native_dialogue_mask_motion: (i64, i64, i64),
    pub(crate) native_dialogue_chrome: Option<ModernHDDialogueChrome>,
    pub(crate) window_dialogue_border: Option<(Arc<Picture>, Rectangle)>,
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
    pub(crate) growth_roll_observer: Option<battle::Callback<dyn Fn(&battle::Unit, u16) -> u16 + Send + Sync>>,
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
    ///
    /// R4 A 組（共用變更需求）：Go 介面值可為 nil，Rust 用 `Option`。
    pub(crate) native_physical_voice: Option<SfxVoice>,
    /// 已納入契約的 FDOTHER 指令音效；只讀分離 OGG
    pub(crate) separated_command_sfx: BTreeMap<i64, BTreeMap<i64, Vec<u8>>>,
    /// FDOTHER #77 sub2: sub_1F894 H／P 標題選單移動分支
    pub(crate) sfx_title_move: Vec<u8>,
    /// FDOTHER #77 sub1: sub_1F894 標題選單確認分支
    pub(crate) sfx_title_confirm: Vec<u8>,
    /// FDOTHER #78 sub0: ANI #1 第一幀 companion
    pub(crate) sfx_title_ani1: Vec<u8>,
    /// ANI #1 專用 owner；結束／略過時明確停止
    ///
    /// R4 A 組（共用變更需求）：Go 介面值可為 nil，Rust 用 `Option`。
    pub(crate) title_ani1_voice: Option<SfxVoice>,
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
    /// 系統確認框建好、等命令框收合時暫存的狀態（`begin_native_system_confirmation`）。
    pub(crate) native_system_confirmation_pending: Option<Box<NativeSystemEndTurnUIState>>,
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
    pub(crate) modern_opening: Option<super::modern_opening::ModernOpening>,
    pub(crate) modern_opening_view: Option<Arc<Picture>>,
    /// F11 的標題選單畫面，三種選取與一個全未選狀態。
    pub(crate) hd_title_menu: Option<[Arc<Picture>; 4]>,
    /// 切回原版時留存已通過預檢的 HD 美術。
    pub(crate) modern_theme_cache: Option<ModernThemeArt>,
    /// 第一次切到 HD 的背景載入；nil 表示沒有進行中的載入
    pub(crate) modern_theme_loading: Option<ModernThemeLoad>,
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
    pub(crate) toolbar_ticks: i64,
    pub(crate) chapter_guide_open: bool,
    pub(crate) chapter_guide_detail: bool,
    pub(crate) chapter_guide_page: usize,
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
    pub(crate) posters: PlayerPosters,
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
    /// R4 B 組新增（共用變更需求，沒有 Go 欄位）：故事對白嘴型的亂數。Go `stepDialogueMouth` 呼叫套件層
    /// `rand.Intn(30)`（全域來源，每次啟動不同種子）；Rust 不用全域狀態，改由這個欄位以固定種子
    /// `STORY_MOUTH_RNG_SEED` 惰性建立（KNOWN_DIFFERENCES `cosmetic-mouth-rng-fixed-seed`）。
    pub(crate) story_mouth_rng: Option<battle::GoMathRand>,
    /// R5 階段 2 r5d 新增（沒有 Go 欄位）：Go `math/rand` 的套件層全域來源。只有畫格擷取打開時由驅動端設成
    /// `GoMathRand::new(4)`（Go 黃金在測試開頭 `rand.Seed(4)`）；這時故事對白嘴型的 `rand.Intn(30)` 與確認框嘴型的
    /// `rand.New(rand.NewSource(rand.Int63()))` 依 Go 的取用順序共用它（見 [`Game::story_mouth_intn30`]、
    /// [`Game::sample_native_confirmation_mouth_random`]）。`None`（預設）時維持 R4 的兩個固定種子
    /// （KNOWN_DIFFERENCES `cosmetic-mouth-rng-fixed-seed`）。
    pub(crate) go_global_rand: Option<battle::GoMathRand>,
    /// Rust 新增（沒有 Go 對應）：本邏輯幀的輸入。取代 Go 直接查詢 `inpututil.IsKeyJustPressed`、
    /// `ebiten.IsKeyPressed`；驅動端（無畫面重播或 `fd2-app`）在每次 `update` 前設定。
    pub(crate) tick_input: TickInput,
    /// Rust 新增（沒有 Go 對應）：虛擬時鐘目前的時間。取代 Go `gameNow()` 的牆鐘乘遊戲速度；
    /// 驅動端每個邏輯幀前進固定步長。
    pub(crate) virtual_now: GameInstant,
    /// R4 A 組新增（共用變更需求，沒有 Go 對應）：Go `Update` 在 Alt+Enter 或工具列請求時直接呼叫
    /// `ebiten.SetFullscreen(!ebiten.IsFullscreen())`；`fd2-game` 不碰視窗，改設這個旗標，外殼讀取後清除。
    pub(crate) fullscreen_toggle_requested: bool,
    /// R5 階段 4（r5i）：Go `g.input`（`main.go` 新增欄位，`player_input.go`）。正式執行（[`PlayerRunner`]）時建立；
    /// 測試與重播是 `None`，或在 F12 開面板時才由 `open_controls_panel` 建立（同 Go）。
    pub(crate) player_input: Option<PlayerInput>,
    /// R5 階段 4（r5i）：Go `g.pointer`：這一次 Update 的滑鼠指向（`sample_input` 寫入）。
    pub(crate) pointer: PointerTick,
    /// R5 階段 4（r5i）：Go `g.controls`：F12 按鍵設定面板（`player_controls_panel.rs`）。
    pub(crate) controls: ControlsPanel,
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
    /// Go 的 `action` 只呼叫一次，Rust 收 `FnOnce`（骨架是 `FnMut`；簽名變更見 `PORTING.md`）。
    pub(crate) fn start_fade_transition(&mut self, action: Option<Box<dyn FnOnce(&mut Game) + Send>>) {
        self.fade = Some(StoryFade {
            out: true,
            total: STORY_FADE_FRAMES,
            then: Some(Box::new(move |g: &mut Game| {
                // Go 直接呼叫 action()；nil 會 panic，Rust 的 None 視為空動作。
                if let Some(action) = action {
                    action(g);
                }
                g.fade = Some(StoryFade { out: false, total: STORY_FADE_FRAMES, ..Default::default() });
            })),
            ..Default::default()
        });
    }

    /// advanceStoryNode 對白播完離開 story 節點:若節點設 ExitWalk(s),先讓對應 actor(可多人,
    /// 使用者回饋 #A:草地小徑幕結尾索爾+亞雷斯一起走離)全部走完一段路(doc46 §5.3)再淡出換場;
    /// 否則直接淡出換場。多個 exit walk 共用同一個「全部走完才轉場」計數器,不是走完第一個就轉場。
    ///
    /// Go：`main.go:814`，`func (g *Game) advanceStoryNode(n *campaign.Node)`
    pub(crate) fn advance_story_node(&mut self, n: &campaign::Node) {
        fn do_advance(g: &mut Game) {
            g.start_fade_transition(Some(Box::new(|g: &mut Game| {
                // Go `g.camp.Advance("")`：camp 為 nil 時 Go 會 panic；Rust 略過。
                if let Some(camp) = g.camp.as_mut() {
                    camp.advance("");
                }
                g.enter_node();
            })));
        }
        let mut walks: Vec<campaign::ActorWalk> = Vec::new();
        if let Some(ew) = n.exit_walk.as_ref() {
            walks.push(*ew);
        }
        walks.extend(n.exit_walks.iter().cloned());
        if walks.is_empty() {
            do_advance(self);
            return;
        }
        // Go 以閉包捕捉的區域變數 `remaining` 計數；多個 then 共用同一個計數器。
        let remaining = Arc::new(std::sync::atomic::AtomicI64::new(walks.len() as i64));
        for ew in &walks {
            let Some(i) = self.story_actors.iter().position(|u| u.fig == ew.fig) else {
                continue;
            };
            let u = &self.story_actors[i];
            // 預設保留走位末段方向;ew.Dir 指定則覆蓋(如索爾走到亞雷斯旁定住面右)
            let final_dir = ew.dir.unwrap_or(-1);
            let remaining = remaining.clone();
            let job = StoryWalkJob {
                actor: i as i64,
                from_x: u.x,
                from_y: u.y,
                to_x: ew.to_x,
                to_y: ew.to_y,
                frames: ew.frames,
                final_dir,
                then: Some(Box::new(move |g: &mut Game| {
                    if remaining.fetch_sub(1, std::sync::atomic::Ordering::Relaxed) - 1 == 0 {
                        do_advance(g);
                    }
                })),
                ..Default::default()
            };
            self.story_walks.push(job);
        }
    }

    /// stepStoryWalks 逐幀推進場景走位動畫佇列,完成的 job 更新最終座標、呼叫 then 後移除。
    /// 走位沿格線(先長軸後短軸,不斜切)——原版單位走位以戰場格為單位軸向移動
    /// (使用者回饋 2026-07-04 #4:「走路沒照著格子走」=舊版起訖點直線內插會斜切格線)。
    ///
    /// Go：`main.go:859`，`func (g *Game) stepStoryWalks()`
    ///
    /// Go `main.go:877` 以 `live := g.storyWalks[:0]` 原地壓縮，迴圈結束時 `g.storyWalks = live`：
    /// 完成走位的 `then`（例如 `beatAdvance` 接著開始下一個 walk 拍）在迴圈中 append 進
    /// `g.storyWalks` 的新工作會被這一行覆蓋掉。Rust 照 Go 的實際結果：先把佇列取出，迴圈中
    /// 延續推入的工作在結尾同樣被 `live` 取代（教訓 `rust-port-go-quirk-follow-actual`）。
    /// 迴圈中延續看到的 `story_walks` 是空的（Go 看到原長度）；目前沒有延續讀它。
    pub(crate) fn step_story_walks(&mut self) {
        if self.story_walks.is_empty() {
            return;
        }
        fn sgn(v: i64) -> f64 {
            match v.cmp(&0) {
                std::cmp::Ordering::Greater => 1.0,
                std::cmp::Ordering::Less => -1.0,
                std::cmp::Ordering::Equal => 0.0,
            }
        }
        let walks = std::mem::take(&mut self.story_walks);
        let mut live = Vec::with_capacity(walks.len());
        for mut w in walks {
            w.t += 1;
            let mut frac = w.t as f64 / w.frames as f64;
            if frac > 1.0 {
                frac = 1.0;
            }
            let (tile_w, tile_h, map_h) = {
                let m = self.m.as_ref().expect("Go：g.m 為 nil 時解參考 panic");
                (m.tile_w, m.tile_h, m.h)
            };
            let (dx, dy) = (w.to_x - w.from_x, w.to_y - w.from_y);
            let (adx, ady) = (dx.abs(), dy.abs());
            let mut total = adx + ady; // 曼哈頓距離(格)
            if total == 0 {
                total = 1;
            }
            let dist = frac * total as f64; // 已走格數(含小數)
            let (cx, cy, pose);
            if adx >= ady {
                // 先走長軸,再走短軸(格線行走)
                if dist <= adx as f64 {
                    (cx, cy) = (dist * sgn(dx), 0.0);
                    pose = dir_toward(0, 0, dx, 0);
                } else {
                    (cx, cy) = (dx as f64, (dist - adx as f64) * sgn(dy));
                    pose = dir_toward(0, 0, 0, dy);
                }
            } else if dist <= ady as f64 {
                (cx, cy) = (0.0, dist * sgn(dy));
                pose = dir_toward(0, 0, 0, dy);
            } else {
                (cx, cy) = ((dist - ady as f64) * sgn(dx), dy as f64);
                pose = dir_toward(0, 0, dx, 0);
            }
            {
                let u = &mut self.story_actors[w.actor as usize];
                u.set_map_pose(pose);
                // 掛在終點格,Off 為「當前位置-終點」(同 walkAnim 慣例)
                (u.x, u.y) = (w.to_x, w.to_y);
                u.off_x = (w.from_x as f64 + cx - w.to_x as f64) * tile_w as f64;
                u.off_y = (w.from_y as f64 + cy - w.to_y as f64) * tile_h as f64;
            }
            if w.scroll_follow {
                let mut scroll_dist = dist - w.scroll_free as f64;
                if scroll_dist < 0.0 {
                    scroll_dist = 0.0;
                }
                self.cam_y = w.from_cam_y - scroll_dist * tile_h as f64;
                clamp(&mut self.cam_y, 0.0, (map_h * tile_h - LOGICAL_H / STORY_ZOOM) as f64);
            }
            if w.t >= w.frames {
                {
                    let u = &mut self.story_actors[w.actor as usize];
                    (u.off_x, u.off_y) = (0.0, 0.0);
                }
                // 0x13185 與後續 0x15F84 共用同一組相機全域。scroll_step
                // 完成後必須先發布最後一格 view，再讓 dialog 預建 indexed
                // 背景；否則畫面會錯用 STEP 前的 LOADCH/PAN snapshot。
                if w.scroll_follow && !self.sync_story_native_map_scroll_view(Some(&w)) {
                    self.story_walks = Vec::new();
                    return;
                }
                let u = &mut self.story_actors[w.actor as usize];
                if w.final_dir >= 0 {
                    // 走完面向目標(如 Ares 走到索爾旁面向他),不停在走位末段的短軸方向
                    u.set_map_pose(w.final_dir);
                }
                // 完成的故事行走必須同步原生 +0/+1/+3/+4；不能依靠下一次
                // 繪圖快取重建補座標，因為重新建構同時會錯誤清掉既有方向。
                let (x, y, dir) = (u.x, u.y, u.dir);
                u.set_map_placement(x, y, dir);
                if self.cutscene_log {
                    // FD2_CUTSCENE_LOG:印走位完成(誰、從哪到哪、末向),對原版走位比對
                    let u = &self.story_actors[w.actor as usize];
                    eprintln!(
                        "[cutscene] walk done: {} ({},{})->({},{}) dir={}",
                        fig_name(u.fig),
                        w.from_x,
                        w.from_y,
                        w.to_x,
                        w.to_y,
                        u.dir
                    );
                }
                if let Some(then) = w.then.take() {
                    then(self);
                }
                continue; // 不放回 live,即移除
            }
            live.push(w);
        }
        self.story_walks = live;
    }

    /// stepFade 逐幀推進場景淡出/淡入(storyFade,doc46 §5.2):走完(不分 out/in)呼叫 then;
    /// g.fade 為空時內部直接返回(同 stepStoryWalks/stepActJob/stepCamPan 慣例)。
    ///
    /// Go：`main.go:952`，`func (g *Game) stepFade()`
    pub(crate) fn step_fade(&mut self) {
        let Some(fade) = self.fade.as_mut() else {
            return;
        };
        fade.t += 1;
        if fade.t >= fade.total {
            let cb = self.fade.take().and_then(|f| f.then);
            if let Some(cb) = cb {
                cb(self);
            }
        }
    }

    /// Go：`main.go:966`，`func (g *Game) stepTransitionReveal()`
    ///
    /// `drawn` 由呈現回報設定（Go 在 `Draw` 裡設，見 `PORTING.md`「Draw 中改狀態的位置」）。
    pub(crate) fn step_transition_reveal(&mut self) {
        let Some(job) = self.transition_reveal.as_mut() else {
            return;
        };
        if !job.drawn {
            return;
        }
        if job.ticks > 0 {
            job.ticks -= 1;
            return;
        }
        job.remaining -= 1;
        if job.remaining <= 0 {
            let mut job = self.transition_reveal.take().unwrap();
            job.rollback = None;
            if let Some(then) = job.then.take() {
                then(self);
            }
            return;
        }
        job.index ^= 1;
        let frame = job.frames[job.index as usize].clone();
        job.drawn = false;
        // The current tick already presents one frame; wait only the remaining
        // delay ticks so a 20ms native delay maps to one 60Hz update, not two.
        job.ticks = (job.delay - 1).max(0);
        self.native_map_vga = frame;
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
        // Go 的閉包在 publish 檢查完 g.m 之後才讀 g.camX／g.m.TileW；這裡先取值，g.m 為 nil 時
        // publish 會在呼叫閉包之前失敗，取到的 0 不會被用到。
        let (tile_w, tile_h) = self.m.as_ref().map_or((0, 0), |m| (m.tile_w, m.tile_h));
        let (cam_x, cam_y) = (self.cam_x as i64, self.cam_y as i64);
        self.publish_story_native_map_view(&mut |mut view: battle::NativeMapViewState| {
            let (camera_x, camera_y) = (cam_x / tile_w, cam_y / tile_h);
            view.cursor_x += camera_x - view.camera_x;
            view.cursor_y += camera_y - view.camera_y;
            (view.camera_x, view.camera_y) = (camera_x, camera_y);
            (view, true)
        })
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
        let Some(w) = w else {
            return true;
        };
        let (dx, dy) = (w.to_x - w.from_x, w.to_y - w.from_y);
        if (dx != 0 && dy != 0) || (dx == 0 && dy == 0) {
            self.load_err = format!(
                "native story map view: 走行捲動只支援單軸整段，({},{})→({},{})",
                w.from_x, w.from_y, w.to_x, w.to_y
            );
            return false;
        }
        let (step_x, step_y) = (sgn_int(dx), sgn_int(dy));
        let (map_w, map_h, tile_w, tile_h) = self.m.as_ref().map_or((0, 0, 0, 0), |m| (m.w, m.h, m.tile_w, m.tile_h));
        let (cam_x, cam_y) = (self.cam_x as i64, self.cam_y as i64);
        let (from_x, from_y, to_x, to_y) = (w.from_x, w.from_y, w.to_x, w.to_y);
        // Go 的閉包直接寫 g.loadErr；Rust 先記在區域變數，publish 回來後寫回。
        let mut err: Option<String> = None;
        let ok = self.publish_story_native_map_view(&mut |mut view: battle::NativeMapViewState| {
            let (mut x, mut y) = (from_x, from_y);
            while x != to_x || y != to_y {
                let (next, ok) =
                    battle::advance_native_map_walk_step_view_state(view, map_w, map_h, x, y, step_x, step_y);
                if !ok {
                    err = Some(format!("native story map view: 走行捲動第 ({x},{y}) 格的視圖更新被拒絕"));
                    return (view, false);
                }
                view = next;
                (x, y) = (x + step_x, y + step_y);
            }
            let (camera_x, camera_y) = (cam_x / tile_w, cam_y / tile_h);
            if view.camera_x != camera_x || view.camera_y != camera_y {
                err = Some(format!(
                    "native story map view: 逐格鏡頭 ({},{}) 與插值鏡頭 ({camera_x},{camera_y}) 不一致",
                    view.camera_x, view.camera_y
                ));
                return (view, false);
            }
            (view, true)
        });
        if let Some(err) = err {
            self.load_err = err;
        }
        ok
    }
}

/// Go：`main.go:1068`，`func sgnInt(v int) int`
pub(crate) fn sgn_int(v: i64) -> i64 {
    v.signum()
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
        let ch28_continuity = self.ch28_handler_native_map_view_continuity();
        let owned_battle_event = self.battle_event.is_some();
        let has_battle_view =
            (ch28_continuity || owned_battle_event) && self.st.as_ref().is_some_and(|st| st.has_native_map_view_state);
        if !self.has_story_native_map_view && !has_battle_view {
            return true;
        }
        let Some((map_w, map_h)) = self.tile_aligned_story_map(true) else {
            self.load_err = String::from("native story map view: pan camera is not tile-aligned");
            return false;
        };
        let view = if self.has_story_native_map_view {
            self.story_native_map_view
        } else {
            self.st.as_ref().unwrap().native_map_view_state
        };
        let (view, ok) = advance(view);
        if !ok {
            return false;
        }
        if self.materialize_story_native_map_view(map_w, map_h, view).is_none() {
            return false;
        }
        // The original uses the same absolute cursor globals throughout the
        // handler. Keep the generic renderer cursor aligned with that typed carrier
        // so a following 0x12D7B focus starts from the post-pan position.
        if ch28_continuity || owned_battle_event || self.camp.as_ref().is_some_and(|c| c.node_id() == "story_ch24") {
            (self.cur_x, self.cur_y) = (view.cursor_x, view.cursor_y);
        }
        true
    }

    /// Rust 新增：`publishStoryNativeMapView`／`syncStoryNativeMapFocusView` 共用的 `g.m` 與鏡頭整格
    /// 檢查（Go `main.go:1092`、`main.go:1157`）。`require_size` 對應 publish 多檢查的 `W`／`H` 大於 0。
    /// 通過時回傳 `(W, H)`。
    fn tile_aligned_story_map(&self, require_size: bool) -> Option<(i64, i64)> {
        let m = self.m.as_ref()?;
        if m.tile_w <= 0 || m.tile_h <= 0 || (require_size && (m.w <= 0 || m.h <= 0)) {
            return None;
        }
        let (cx, cy) = (self.cam_x as i64, self.cam_y as i64);
        if cx % m.tile_w != 0 || cy % m.tile_h != 0 || self.cam_x != cx as f64 || self.cam_y != cy as f64 {
            return None;
        }
        Some((m.w, m.h))
    }

    /// Rust 新增：Go `carrier := &battle.State{W, H}; carrier.MaterializeNativeMapViewState(view)` 後依
    /// `hasStoryNativeMapView` 發布到劇情或戰場的視圖（`main.go:1107`、`main.go:1171`，兩處相同）。
    fn materialize_story_native_map_view(
        &mut self,
        map_w: i64,
        map_h: i64,
        view: battle::NativeMapViewState,
    ) -> Option<battle::NativeMapViewState> {
        let mut carrier = battle::State::default();
        (carrier.w, carrier.h) = (map_w, map_h);
        if let Err(err) = carrier.materialize_native_map_view_state(view) {
            self.load_err = format!("native story map view: {}", err.message());
            return None;
        }
        let published = carrier.native_map_view_state;
        if self.has_story_native_map_view {
            self.story_native_map_view = published;
            let ch23 = self.camp.as_ref().is_some_and(|c| c.node_id() == "postbattle_ch23_persist");
            if let Some(st) = self.st.as_mut().filter(|st| ch23 && st.has_native_map_view_state) {
                st.native_map_view_state = published;
            }
        } else {
            // Go 直接寫 g.st（has_battle_view 或 focus 的 hasBattleView 已保證非 nil）。
            self.st.as_mut().expect("Go：g.st 為 nil 時解參考 panic").native_map_view_state = published;
        }
        Some(published)
    }

    /// ch28HandlerNativeMapViewContinuity is deliberately caller-specific. Raw
    /// ch28 pre/post share the six native view globals across the remake's
    /// artificial cutscene/battle node boundary. Battle turn-event staging owns a
    /// separate private snapshot and must not be mutated by this bridge.
    ///
    /// Go：`main.go:1132`，`func (g *Game) ch28HandlerNativeMapViewContinuity() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn ch28_handler_native_map_view_continuity(&self) -> bool {
        let Some(camp) = self.camp.as_ref() else {
            return self.handler_chapter == 28; // narrow unit fixture
        };
        let id = camp.node_id();
        id == "story_ch29" || id == "postbattle_ch29_persist"
    }

    /// syncStoryNativeMapFocusView mirrors the six globals after one 0x12CEA
    /// focus step. Unlike 0x135DD, focus updates absolute and visible cursor while
    /// moving the camera only when the native safe band is crossed.
    ///
    /// Go：`main.go:1146`，`func (g *Game) syncStoryNativeMapFocusView(visibleX, visibleY int) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn sync_story_native_map_focus_view(&mut self, visible_x: i64, visible_y: i64) -> bool {
        let owned_by_native_wrapper = self.focus_job.as_ref().is_some_and(|j| j.native_view);
        if !self.ch28_handler_native_map_view_continuity() && !owned_by_native_wrapper {
            return true;
        }
        let has_battle_view = self.st.as_ref().is_some_and(|st| st.has_native_map_view_state);
        if !self.has_story_native_map_view && !has_battle_view {
            return true;
        }
        let Some((map_w, map_h)) = self.tile_aligned_story_map(false) else {
            self.load_err = String::from("native story map view: focus camera is not tile-aligned");
            return false;
        };
        let m = self.m.as_ref().unwrap();
        let (origin_x, origin_y) = (self.cam_x as i64 / m.tile_w, self.cam_y as i64 / m.tile_h);
        let view = battle::NativeMapViewState {
            camera_x: origin_x,
            camera_y: origin_y,
            cursor_x: self.cur_x,
            cursor_y: self.cur_y,
            visible_cursor_x: visible_x,
            visible_cursor_y: visible_y,
        };
        self.materialize_story_native_map_view(map_w, map_h, view).is_some()
    }

    /// stepCamPan 逐幀推進 beat「pan」鏡頭位移；原版模式逐 tile、X-first，
    /// 相容模式線性內插。走完清除 job 並接下一拍。
    ///
    /// Go：`main.go:1188`，`func (g *Game) stepCamPan()`
    pub(crate) fn step_cam_pan(&mut self) {
        let Some(mut j) = self.cam_pan.take() else {
            return;
        };
        if j.tile_step {
            fn step(mut current: f64, target: f64, tile: i64) -> f64 {
                if current < target {
                    current += tile as f64;
                    if current > target {
                        current = target;
                    }
                } else if current > target {
                    current -= tile as f64;
                    if current < target {
                        current = target;
                    }
                }
                current
            }
            let (tile_w, tile_h) = {
                let m = self.m.as_ref().expect("Go：g.m 為 nil 時解參考 panic");
                (m.tile_w, m.tile_h)
            };
            if self.cam_x != j.to_x {
                self.cam_x = step(self.cam_x, j.to_x, tile_w);
            } else if self.cam_y != j.to_y {
                self.cam_y = step(self.cam_y, j.to_y, tile_h);
            }
            if !self.sync_story_native_map_pan_view() {
                // Go：g.camPan = nil（已取出）
                return;
            }
            if self.cam_x == j.to_x && self.cam_y == j.to_y {
                if let Some(then) = j.then.take() {
                    then(self);
                }
                return;
            }
            self.cam_pan = Some(j);
            return;
        }
        j.t += 1;
        let mut frac = j.t as f64 / j.frames as f64;
        if frac > 1.0 {
            frac = 1.0;
        }
        self.cam_x = j.from_x + (j.to_x - j.from_x) * frac;
        self.cam_y = j.from_y + (j.to_y - j.from_y) * frac;
        if j.t >= j.frames {
            // 相容插值仍須在整格終點交接同一組原生視圖；後續對話聚焦
            // 不可混用新鏡頭與插值前的視窗游標。中途的非整格不發布。
            if !self.sync_story_native_map_pan_view() {
                return;
            }
            if let Some(then) = j.then.take() {
                then(self);
            }
            return;
        }
        self.cam_pan = Some(j);
    }
}

/// nativeMapFocusVisibleSeed 是「完全沒有已追蹤可見游標」時的重製端後備值：
/// 把 cursor - camera 夾回 13×8 視窗。原版永遠有一份可見游標可以沿用，這裡只
/// 是不讓直接進場的 renderer 因為缺少那份狀態而產生視窗外的座標。
///
/// Go：`main.go:1248`，`func nativeMapFocusVisibleSeed(x, y int) (int, int)`
pub(crate) fn native_map_focus_visible_seed(x: i64, y: i64) -> (i64, i64) {
    let clamp = |v: i64, limit: i64| -> i64 {
        if v < 0 {
            return 0;
        }
        if v >= limit {
            return limit - 1;
        }
        v
    };
    (clamp(x, 13), clamp(y, 8))
}

impl Game {
    /// stepFocusUnit 逐格重現 0x12cea 與 0x11b48/0x11b9b/0x11bfa/0x11c59。
    /// 原版視窗游標的安全帶為 X=2..10、Y=2..5；超出安全帶後，能捲圖時保持
    /// screen cursor 不動並移動 map origin，碰地圖邊界時才讓 screen cursor 靠邊。
    ///
    /// Go：`main.go:1264`，`func (g *Game) stepFocusUnit()`
    ///
    /// R4 B 組：job 留在 `self.focus_job` 裡逐步改寫，不先取出。Go 的 `syncStoryNativeMapFocusView`
    /// 在步進途中讀 `g.focusJob.nativeView`（`main.go:1150`），取出會讓它看到空的 job。
    pub(crate) fn step_focus_unit(&mut self) {
        let Some(j) = self.focus_job.as_ref() else {
            return;
        };
        let (tile_w, tile_h, map_w, map_h) = match self.m.as_ref() {
            Some(m) if m.tile_w > 0 && m.tile_h > 0 => (m.tile_w, m.tile_h, m.w, m.h),
            _ => return,
        };
        let (target_x, target_y, native_view, started) = (j.target_x, j.target_y, j.native_view, j.started);
        // 戰場視圖上的聚焦（0x13FD4 原地回復的 0x12D7B 等）與 AI 聚焦是同一支 0x12CEA：開頭重繪
        // 一次，逐步照游標鍵處理器的重繪規則；故事視圖另有自己的 HUD，不動戰場的 anchor。
        let battle_view = native_view
            && !self.has_story_native_map_view
            && self.st.as_ref().is_some_and(|st| st.has_native_map_view_state);
        let story_view = native_view && self.native_ch12_story_focus();
        if !started {
            if let Some(j) = self.focus_job.as_mut() {
                j.started = true;
            }
            if battle_view {
                let hook = battle::NATIVE_FOCUS_TRACE_HOOK.read().unwrap_or_else(|e| e.into_inner()).clone();
                if let Some(hook) = hook {
                    hook(target_x, target_y, self.st.as_ref().unwrap().native_map_view_state);
                }
                self.redraw_native_map_hud();
            }
            if story_view && let Err(err) = self.compose_native_story_map_frame() {
                self.load_err = format!("native story focus: {err}");
                self.focus_job = None;
                return;
            }
        }
        let mut step_before = battle::NativeMapViewState::default();
        if battle_view {
            step_before = self.st.as_ref().unwrap().native_map_view_state;
        } else if story_view {
            step_before = self.story_native_map_view;
        }
        fn finish(g: &mut Game) {
            let then = g.focus_job.take().and_then(|mut j| j.then.take());
            if let Some(then) = then {
                then(g);
                // 有明確 continuation 的 native wrapper 自行接管下一階段。
            } else if g.camp.is_some() {
                // 一般 focus beat 由 campaign 擁有者接下一拍；direct-entry
                // renderer／證據測試不會因此偽造戰役進度。
                g.beat_advance();
            }
        }
        let (mut origin_x, mut origin_y) = (self.cam_x as i64 / tile_w, self.cam_y as i64 / tile_h);
        // 可見游標一律沿用已追蹤的值。原版沒有任何一處由 cursor - camera 重算它
        // （寫入端清單見 docs/data/ida/fd2_visible_cursor_writers_ida.txt），鏡頭
        // 一旦捲過游標，減法就會給出負值，而那是視窗外的無效格。
        let (mut screen_x, mut screen_y) =
            native_map_focus_visible_seed(self.cur_x.wrapping_sub(origin_x), self.cur_y.wrapping_sub(origin_y));
        if self.has_story_native_map_view {
            (screen_x, screen_y) =
                (self.story_native_map_view.visible_cursor_x, self.story_native_map_view.visible_cursor_y);
        } else if let Some(st) = self.st.as_ref().filter(|st| st.has_native_map_view_state) {
            (screen_x, screen_y) =
                (st.native_map_view_state.visible_cursor_x, st.native_map_view_state.visible_cursor_y);
        }
        if self.cur_x == target_x && self.cur_y == target_y {
            if !self.sync_story_native_map_focus_view(screen_x, screen_y) {
                self.focus_job = None;
                return;
            }
            finish(self);
            return;
        }
        let (mut max_origin_x, mut max_origin_y) = (map_w - 13, map_h - 8);
        if max_origin_x < 0 {
            max_origin_x = 0;
        }
        if max_origin_y < 0 {
            max_origin_y = 0;
        }
        if self.cur_x > target_x {
            self.cur_x -= 1;
            if screen_x < 2 && origin_x > 0 {
                origin_x -= 1;
            } else {
                screen_x -= 1;
            }
        } else if self.cur_x < target_x {
            self.cur_x += 1;
            if screen_x > 10 && origin_x < max_origin_x {
                origin_x += 1;
            } else {
                screen_x += 1;
            }
        } else if self.cur_y > target_y {
            self.cur_y -= 1;
            if screen_y < 2 && origin_y > 0 {
                origin_y -= 1;
            } else {
                screen_y -= 1;
            }
        } else if self.cur_y < target_y {
            self.cur_y += 1;
            if screen_y > 5 && origin_y < max_origin_y {
                origin_y += 1;
            } else {
                screen_y += 1;
            }
        }
        (self.cam_x, self.cam_y) = ((origin_x * tile_w) as f64, (origin_y * tile_h) as f64);
        if !self.sync_story_native_map_focus_view(screen_x, screen_y) {
            self.focus_job = None;
            return;
        }
        if battle_view {
            let after = self.st.as_ref().unwrap().native_map_view_state;
            self.native_cursor_step_hud(step_before, after);
        }
        if story_view
            && self
                .story_native_map_state
                .as_ref()
                .is_some_and(|s| s.native_map_cursor_step_redraws(step_before, self.story_native_map_view))
            && let Err(err) = self.compose_native_story_map_frame()
        {
            self.load_err = format!("native story focus step: {err}");
            self.focus_job = None;
            return;
        }
        if self.cur_x == target_x && self.cur_y == target_y {
            finish(self);
        }
    }

    /// finishActJob 清除 acting job 後接下一個 beat。
    ///
    /// Go：`main.go:1382`，`func (g *Game) finishActJob(j *actPoseJob)`
    ///
    /// R4 B 組：`j` 改收 `&mut`，延續要 `take()` 出來才能以 `&mut self` 呼叫。呼叫端先把 job 從
    /// `self.act_job` 取出（Go 的 `j` 與 `g.actJob` 是同一個指標）。
    pub(crate) fn finish_act_job(&mut self, j: &mut ActPoseJob) {
        self.act_job = None;
        if let Some(then) = j.then.take() {
            then(self);
        }
    }

    /// actingActor resolves decoded acting against the original FDFIELD roster
    /// slot first.  Fig is deliberately only a legacy authored-scene fallback:
    /// using it for decoded frames would move the first matching guard when the
    /// original bytecode targeted a different same-Fig guard.
    ///
    /// Go：`main.go:1393`，`func (g *Game) actingActor(target campaign.ActingUnit) *battle.Unit`
    ///
    /// R4 B 組：Go 回傳的 `*battle.Unit` 可能指向 `g.st.Units` 或 `g.storyActors`，`UnitIdx` 只能指前者，
    /// 改回傳 [`HandlerUnitRef`]；`target` 改收借用。
    pub(crate) fn acting_actor(&self, target: &campaign::ActingUnit) -> Option<HandlerUnitRef> {
        if let Some(slot) = target.slot {
            if let Some(st) = self.st.as_ref() {
                if slot >= 0 && (slot as usize) < st.units.len() {
                    return Some(HandlerUnitRef::Battle(slot as UnitIdx));
                }
                return None;
            }
            if slot >= 0 && (slot as usize) < self.story_actors.len() {
                return Some(HandlerUnitRef::Story(slot as usize));
            }
            return None;
        }
        let fig = target.fig;
        self.story_actors.iter().position(|a| a.fig == fig).map(HandlerUnitRef::Story)
    }

    /// R4 B 組新增：取 [`HandlerUnitRef`] 指的單位（Go 直接解參考 `*battle.Unit`）。
    pub(crate) fn handler_unit(&self, r: HandlerUnitRef) -> &battle::Unit {
        match r {
            HandlerUnitRef::Battle(i) => &self.st.as_ref().expect("HandlerUnitRef::Battle 需要 st").units[i],
            HandlerUnitRef::Story(i) => &self.story_actors[i],
        }
    }

    /// R4 B 組新增：[`Game::handler_unit`] 的可寫版本。
    pub(crate) fn handler_unit_mut(&mut self, r: HandlerUnitRef) -> &mut battle::Unit {
        match r {
            HandlerUnitRef::Battle(i) => &mut self.st.as_mut().expect("HandlerUnitRef::Battle 需要 st").units[i],
            HandlerUnitRef::Story(i) => &mut self.story_actors[i],
        }
    }
}

/// R4 B 組新增：處理器（handler）的單位參照。Go 的 `actingActor`／`handlerUnitAt` 回傳
/// `*battle.Unit`，戰場上指向 `g.st.Units[slot]`，劇情場景指向 `&g.storyActors[slot]`；Rust 以
/// 表別加索引表示，取用經 [`Game::handler_unit`]／[`Game::handler_unit_mut`]。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) enum HandlerUnitRef {
    /// `st.units[i]`
    Battle(UnitIdx),
    /// `story_actors[i]`
    Story(usize),
}

/// nativeRuntimeRecordCapacity 是原版 runtime 記錄表的配置筆數（[0x53A45] 之後 96×0x50
/// bytes）；[0x53BEB] 是已物化筆數，兩者不同。
pub(crate) const NATIVE_RUNTIME_RECORD_CAPACITY: i64 = 96;

impl Game {
    /// 把道具放進第一個未滿 8 格的我方單位；全員背包都滿時原版靜默放棄獎勵。
    ///
    /// Go：`main.go:4268`，`func (g *Game) grantItemToParty(itemID int) bool`
    pub(crate) fn grant_item_to_party(&mut self, item_id: i64) -> bool {
        let Some(st) = self.st.as_mut() else {
            return false;
        };
        for unit in st.units.iter_mut() {
            if unit.camp != battle::OWN || unit.inventory.len() >= 8 {
                continue;
            }
            if unit.add_inventory_item(item_id, false) {
                return true;
            }
        }
        false
    }

    /// 戰間節點用的持有判斷：先以原始記錄搜尋執行期 0..15 格，失敗再看正規化背包，最後退回持續名冊。
    /// 不宣稱與原版 0x24B14 逐位元組相同（說明見 Go 註解）。
    ///
    /// Go：`main.go:4291`，`func (g *Game) partyHasItemID(itemID int) bool`
    pub(crate) fn party_has_item_id(&self, item_id: i64) -> bool {
        if let Some(st) = self.st.as_ref() {
            if (0..=0xff).contains(&item_id)
                && let Ok(records) = battle::native_inventory_records(&st.units, 16)
            {
                return battle::find_native_inventory_item(&records, item_id as u8).is_ok_and(|(unit, _)| unit >= 0);
            }
            if st.units.iter().take(16).any(|unit| unit.inventory.contains(&item_id)) {
                return true;
            }
        }
        self.party_roster.values().any(|unit| unit.inventory.contains(&item_id))
    }

    /// 照原版 ch20_post 的巢狀迴圈：每組（道具,執行期格）最多算一次，總數恰好等於 `required_matches` 才合成；
    /// 成功時依道具再依格的順序各移除第一份，再以一般給道具路徑發獎勵。
    ///
    /// Go：`main.go:4330`，`func (g *Game) applyInventoryRecipe(n *campaign.Node) (bool, error)`
    pub(crate) fn apply_inventory_recipe(&mut self, n: &campaign::Node) -> std::result::Result<bool, String> {
        let Some(reward) =
            n.reward_item_id.filter(|_| !n.item_ids.is_empty() && n.slot_count > 0 && n.required_matches > 0)
        else {
            return Err(String::from("invalid recipe data"));
        };
        let slots = n.slot_count as usize;
        if self.st.as_ref().is_none_or(|st| st.units.len() < slots) {
            return Err(format!("runtime slots={}, want at least {}", self.handler_unit_count(), n.slot_count));
        }
        let st = self.st.as_mut().unwrap();
        let find = |unit: &battle::Unit, item: i64| unit.inventory.iter().position(|&held| held == item);
        let matches = n
            .item_ids
            .iter()
            .map(|&item| st.units[..slots].iter().filter(|unit| find(unit, item).is_some()).count() as i64)
            .sum::<i64>();
        if matches != n.required_matches {
            return Ok(false);
        }
        for &item in &n.item_ids {
            for unit in st.units[..slots].iter_mut() {
                if let Some(index) = find(unit, item) {
                    unit.remove_inventory_index(index as i64);
                }
            }
        }
        self.grant_item_to_party(reward); // 原版只在全員背包都滿時靜默放棄獎勵
        Ok(true)
    }

    /// Go：`main.go:1420`，`func (g *Game) handlerUnitCount() int`
    pub(crate) fn handler_unit_count(&self) -> i64 {
        if let Some(st) = self.st.as_ref() {
            return st.units.len() as i64;
        }
        self.story_actors.len() as i64
    }

    /// handlerRuntimeSlotCount preserves LOADCH's raw FDFIELD count before later
    /// SPAWN beats append records to the scene actor array.  A runtime_context beat
    /// therefore validates the original load context, not only the currently
    /// visible/materialized actors.
    ///
    /// Go：`main.go:1431`，`func (g *Game) handlerRuntimeSlotCount() int`
    pub(crate) fn handler_runtime_slot_count(&self) -> i64 {
        if let Some(st) = self.st.as_ref() {
            return st.units.len() as i64;
        }
        // Go 比較 `g.storyRoster != nil`；Rust 的 Vec 不分 nil 與空（KNOWN_DIFFERENCES `story-roster-nil-vs-empty`）。
        if !self.story_roster.is_empty() {
            return self.story_roster.len() as i64;
        }
        self.story_actors.len() as i64
    }

    /// Go：`main.go:1441`，`func (g *Game) handlerUnitAt(slot int) *battle.Unit`
    ///
    /// R4 B 組：回傳 [`HandlerUnitRef`]（理由同 `acting_actor`）。
    pub(crate) fn handler_unit_at(&self, slot: i64) -> Option<HandlerUnitRef> {
        if slot < 0 {
            return None;
        }
        if let Some(st) = self.st.as_ref() {
            if (slot as usize) < st.units.len() {
                return Some(HandlerUnitRef::Battle(slot as UnitIdx));
            }
            return None;
        }
        if (slot as usize) < self.story_actors.len() {
            return Some(HandlerUnitRef::Story(slot as usize));
        }
        None
    }

    /// beginActingFrame 設定目前 frame 的姿態。呼叫端保證 j.frame 有效。
    ///
    /// Go：`main.go:1458`，`func (g *Game) beginActingFrame(j *actPoseJob)`
    ///
    /// R4 B 組：接收者改 `&mut self`（Go 經單位指標改寫姿態）。
    pub(crate) fn begin_acting_frame(&mut self, j: &mut ActPoseJob) {
        (j.beat, j.tick) = (0, 0);
        for au in &j.acting[j.frame as usize].units {
            if let Some(u) = self.acting_actor(au) {
                self.handler_unit_mut(u).set_map_pose(au.pose);
            }
        }
    }
}

/// Go：`main.go:1467`，`func actingDelta(pose int) (int, int)`
pub(crate) fn acting_delta(pose: i64) -> (i64, i64) {
    match pose {
        0 => (0, 1),
        1 => (-1, 0),
        2 => (0, -1),
        3 => (1, 0),
        _ => (0, 0),
    }
}

impl Game {
    /// Go：`main.go:1482`，`func (g *Game) nextActingFrame(j *actPoseJob)`
    pub(crate) fn next_acting_frame(&mut self, j: &mut ActPoseJob) {
        j.frame += 1;
        if j.frame >= j.acting.len() as i64 {
            self.finish_act_job(j);
            return;
        }
        self.begin_acting_frame(j);
    }

    /// stepOriginalActing 精確承接已破解的 acting frame 行為(doc50 §1.2)。
    ///
    /// Go：`main.go:1492`，`func (g *Game) stepOriginalActing(j *actPoseJob)`
    pub(crate) fn step_original_acting(&mut self, j: &mut ActPoseJob) {
        let f = j.acting[j.frame as usize].clone();
        if f.special && f.beats == 0 {
            // Original bit7=1/low7=0 is not an empty terminator. 0x136d9..
            // 0x137c5 performs a terrain/unit composite bracketed by delay(1) and
            // delay(2), then redraws. Ebiten redraws continuously, so retain its
            // three-tick hold before advancing to preserve handler timing.
            // The native loop writes pose (+3) before every redraw even for this
            // zero-duration special frame.  Re-apply it here so a concurrent
            // actor update cannot leave the frame with a stale direction.
            for au in &f.units {
                if let Some(u) = self.acting_actor(au) {
                    self.handler_unit_mut(u).set_map_pose(au.pose);
                }
            }
            j.tick += 1;
            if j.tick >= 3 {
                self.next_acting_frame(j);
            }
            return;
        }
        if f.beats <= 0 {
            self.next_acting_frame(j);
            return;
        }
        if f.special {
            // bit7=1：原地姿態，beat 為顯示節奏
            j.tick += 1;
            if j.tick >= f.beats {
                self.next_acting_frame(j);
            }
            return;
        }

        // bit7=0：原版對每一格跑 tick=1..6，再在第 7 tick 寫入 X/Y。
        j.tick += 1;
        let frac = j.tick as f64 / 7.0;
        let (tile_w, tile_h) = {
            let m = self.m.as_ref().expect("Go：g.m 為 nil 時解參考 panic");
            (m.tile_w, m.tile_h)
        };
        for au in &f.units {
            if let Some(r) = self.acting_actor(au) {
                let (dx, dy) = acting_delta(au.pose);
                let u = self.handler_unit_mut(r);
                u.off_x = dx as f64 * tile_w as f64 * frac;
                u.off_y = dy as f64 * tile_h as f64 * frac;
                if j.tick < 7 {
                    u.set_native_map_grid_motion(au.pose, j.tick);
                }
            }
        }
        if j.tick < 7 {
            return;
        }
        for au in &f.units {
            if let Some(r) = self.acting_actor(au) {
                let (dx, dy) = acting_delta(au.pose);
                let u = self.handler_unit_mut(r);
                let (x, y) = (u.x + dx, u.y + dy);
                if !u.finish_native_map_grid_step(au.pose, x, y) {
                    (u.x, u.y) = (x, y);
                    (u.off_x, u.off_y) = (0.0, 0.0);
                    u.set_map_pose(au.pose);
                }
            }
        }
        j.tick = 0;
        j.beat += 1;
        if j.beat >= f.beats {
            self.next_acting_frame(j);
        }
    }

    /// stepActJob 逐幀推進原版 acting frame 或舊版姿態近似。
    ///
    /// Go：`main.go:1560`，`func (g *Game) stepActJob()`
    ///
    /// R4 B 組：job 先從 `self.act_job` 取出再推進（`step_original_acting` 等收 `&mut ActPoseJob`）；
    /// 沒有收尾就放回。收尾時 `finish_act_job` 已呼叫延續，延續可能裝上新的 job，所以不放回。
    pub(crate) fn step_act_job(&mut self) {
        let Some(mut j) = self.act_job.take() else {
            return;
        };
        if !j.acting.is_empty() {
            self.step_original_acting(&mut j);
            if j.frame < j.acting.len() as i64 {
                self.act_job = Some(j);
            }
            return;
        }
        let pose = j.poses[j.idx as usize];
        self.story_actors[j.actor as usize].set_map_pose(pose);
        j.t += 1;
        if j.t >= j.frames {
            j.t = 0;
            j.idx += 1;
            if j.idx >= j.poses.len() as i64 {
                self.finish_act_job(&mut j);
                return;
            }
        }
        self.act_job = Some(j);
    }

    /// Go：`main.go:1581`，`func (g *Game) stepActJobAtUpdateRate()`
    pub(crate) fn step_act_job_at_update_rate(&mut self) {
        if self.act_job.is_none() {
            self.act_tick_accumulator = 0.0;
            return;
        }
        self.act_tick_accumulator += ACTING_SOURCE_TICKS_PER_UPDATE;
        if self.act_tick_accumulator < 1.0 {
            return;
        }
        self.act_tick_accumulator -= 1.0;
        self.step_act_job();
    }

    /// beatAdvance 進下一拍;序列跑完則走節點收尾流程。
    ///
    /// Go：`main.go:1601`，`func (g *Game) beatAdvance()`
    pub(crate) fn beat_advance(&mut self) {
        self.beat_idx += 1;
        if self.beat_idx >= self.beats.len() as i64 {
            self.follow_walk = false; // 走位跟焦只在 walk 拍內有效,收尾(ExitWalk/淡出)一律鏡頭固定
            if let Some(n) = self.camp.as_ref().and_then(|c| c.node().cloned()) {
                self.advance_story_node(&n);
            }
            return;
        }
        let b = self.beats[self.beat_idx as usize].clone();
        self.beat_start(&b);
    }
}

/// beatStart 依原語種類啟動目前這一拍(狀態掛到 g.camPan/g.storyWalks/g.dialog/g.actJob/
/// g.fade/g.beatDelay,交給 Update 既有機制逐幀推進)。找不到對應角色 / 資料缺漏時直接跳拍
/// 並記到 loadErr,不讓整個過場卡死(誠實 stub,勝過假裝完成)。
/// figName cutscene sprite 標號用:fig id → 角色名(協助使用者回報 + 對映原版 slot)。
///
/// Go：`main.go:1816`，`func figName(fig int) string`
pub(crate) fn fig_name(fig: i64) -> String {
    match fig {
        0 => "索爾",
        4 => "亞雷斯",
        9 => "悠妮",
        30 => "蓋亞",
        48 => "國王",
        66 => "王后",
        68 | 69 => "守衛",
        _ => return format!("fig{fig}"),
    }
    .to_string()
}

impl Game {
    /// Go：`main.go:1836`，`func (g *Game) beatStart(b campaign.Beat)`
    ///
    /// R4 B 組：`then: g.beatAdvance` 一律寫成 [`beat_advance_then`]。呼叫第一章沒有執行到的 `r8` 函式的
    /// 分支（ch20 天空之鑰、ch22／ch23／ch28 演出、`if`、物品給予等）先照 Go 做完 payload 檢查，呼叫處停在
    /// `todo!("R8: …")`。Go `%v` 印 `*int` 是記憶體位址，Rust 印值（KNOWN_DIFFERENCES `go-int-pointer-format`）。
    pub(crate) fn beat_start(&mut self, b: &campaign::Beat) {
        if self.cutscene_log {
            // FD2_CUTSCENE_LOG:印每一拍(op+參數),對原版 handler beat 序列比對
            eprintln!(
                "[cutscene] beat op={} source={} fig={} x={} y={} frames={} line={} count={} script={} scene={} scene_index={} loadch={:?}",
                b.op,
                b.source,
                b.fig,
                b.x,
                b.y,
                b.frames,
                b.line,
                b.count,
                b.script,
                b.scene,
                go_int_ptr(b.scene_index),
                b.load_ch
            );
        }
        if b.op != "walk" && b.op != "scroll_step" {
            // 兩種格線走位拍才啟用跟焦
            self.follow_walk = false;
        }
        let find_actor =
            |g: &Game, fig: i64| -> i64 { g.story_actors.iter().position(|a| a.fig == fig).map_or(-1, |i| i as i64) };
        match b.op.as_str() {
            "runtime_context" => {
                let Some(rc) = b.runtime_context.as_ref().filter(|rc| rc.minimum_slot_count() > 0) else {
                    self.load_err = String::from("beat runtime_context:缺少有效 slot_count/slot_counts");
                    return;
                };
                if !rc.accepts_slot_count(self.handler_runtime_slot_count()) {
                    // Compiler metadata is emitted before LOADCH, while the previous
                    // battle state may still be installed (or the scene array may be
                    // empty). Validate the declared LOADCH record count here instead of
                    // comparing unrelated prior-battle slots; applyLoadCH then validates
                    // the actual roster bytes before replacing the scene state.
                    let next = self.beats.get((self.beat_idx + 1) as usize);
                    let next_load_ch = next
                        .filter(|n| self.beat_idx + 1 < self.beats.len() as i64 && n.op == "loadch")
                        .and_then(|n| n.load_ch.as_ref());
                    if !next_load_ch.is_some_and(|l| rc.accepts_slot_count(l.slot_count)) {
                        self.load_err = format!(
                            "beat runtime_context: runtime slots={}, want exact {} or one of {}",
                            self.handler_runtime_slot_count(),
                            rc.slot_count,
                            go_int_slice(&rc.slot_counts)
                        );
                        return;
                    }
                }
                if rc.story_viewport {
                    self.story_bg = true;
                    if self.has_handler_inherited_map_view {
                        self.story_native_map_view = self.handler_inherited_map_view;
                        self.has_story_native_map_view = true;
                        if let Some(m) = self.m.as_ref().filter(|m| m.tile_w > 0 && m.tile_h > 0) {
                            self.cam_x = (self.story_native_map_view.camera_x * m.tile_w) as f64;
                            self.cam_y = (self.story_native_map_view.camera_y * m.tile_h) as f64;
                            (self.cur_x, self.cur_y) =
                                (self.story_native_map_view.cursor_x, self.story_native_map_view.cursor_y);
                        }
                    }
                }
                self.beat_advance();
            }
            "unit_present" => {
                // Legacy metadata described only the six-frame 0x22547 tail and did
                // not identify a proven caller.  The separate native_unit_present op
                // owns the recovered 0x25535 battle-state path; keep this legacy shape
                // closed instead of silently treating it as that caller.
                if b.unit_present.is_none() {
                    self.load_err = String::from("beat unit_present:缺少 placement payload");
                    return;
                }
                self.load_err = String::from("beat unit_present: legacy payload未綁定已證實的0x22253 caller");
            }
            "native_unit_present" => {
                if !b.native_unit_present.as_ref().is_some_and(|p| p.last_runtime_slot) || b.source != "0x25535" {
                    self.load_err = String::from("beat native_unit_present:缺少原版 0x25535 payload");
                    return;
                }
                let spec = b.native_unit_present.unwrap();
                let then: Box<dyn FnOnce(&mut Game) + Send> = Box::new(|g: &mut Game| g.beat_advance());
                if let Err(err) = self.start_native_unit_present(&spec, Some(then)) {
                    self.load_err = format!("beat native_unit_present: {err}");
                }
            }
            "native_ch28_post_present" => {
                if b.native_ch28_post_present.as_ref().is_none_or(|p| !p.is_recovered_contract())
                    || b.source != "0x254c0"
                {
                    self.load_err = String::from("beat native_ch28_post_present:缺少原版 0x254c0 payload");
                    return;
                }
                let then: Box<dyn FnOnce(&mut Game) + Send> = Box::new(|g: &mut Game| g.beat_advance());
                if let Err(err) = self.start_native_ch28_post_present(Some(then)) {
                    self.load_err = format!("beat native_ch28_post_present: {err}");
                }
            }
            "indexed_transition" => {
                if b.indexed_transition.is_none() {
                    self.load_err = String::from("beat indexed_transition:缺少 transition payload");
                    return;
                }
                let spec = b.indexed_transition.clone().unwrap();
                if let Err(err) =
                    self.start_native_indexed_transition(&spec, &b.source, Box::new(|g: &mut Game| g.beat_advance()))
                {
                    self.load_err = format!("beat indexed_transition: {}", err.message());
                }
            }
            "native_ch20_sky_key_sequence" => {
                if b.source != "0x242c9" || b.native_ch20_sky_key.as_ref().is_none_or(|s| !s.is_recovered_contract()) {
                    self.load_err = String::from("beat native_ch20_sky_key_sequence:缺少原版 0x242c9 payload");
                    return;
                }
                todo!("R8: native_ch20_sky_key.go:startNativeCh20SkyKeySequence（第一章沒有執行到）")
            }
            "native_palette_fade_out" => {
                if !b.native_palette_fade.as_ref().is_some_and(|f| f.start == 0 && f.end == 63 && f.delay_ms == 2) {
                    self.load_err = String::from("beat native_palette_fade_out:缺少原版 64-step DAC payload");
                    return;
                }
                if let Err(err) = self.start_native_palette_ramp(0, 63, 2, beat_advance_then()) {
                    self.load_err = format!("beat native_palette_fade_out: {err}");
                }
            }
            "native_palette_fade_in" => {
                if !b.native_palette_fade_in.as_ref().is_some_and(|f| f.start == 64 && f.end == 0 && f.delay_ms == 2) {
                    self.load_err = String::from("beat native_palette_fade_in:缺少原版 65-step DAC payload");
                    return;
                }
                if let Err(err) = self.start_native_palette_ramp(64, 0, 2, beat_advance_then()) {
                    self.load_err = format!("beat native_palette_fade_in: {err}");
                }
            }
            "native_palette_pulse" => {
                let valid = b.native_palette_pulse.as_ref().is_some_and(|p| {
                    p.rise_start == 0
                        && p.rise_end == 63
                        && p.rise_delay_ms == 8
                        && p.hold_ms == 400
                        && p.fall_start == 62
                        && p.fall_end == 0
                        && p.fall_delay_ms == 8
                });
                if !valid {
                    self.load_err = String::from("beat native_palette_pulse:缺少原版 DAC pulse payload");
                    return;
                }
                let pulse = b.native_palette_pulse.unwrap();
                if let Err(err) = self.start_native_palette_pulse(&pulse, Box::new(|g: &mut Game| g.beat_advance())) {
                    self.load_err = format!("beat native_palette_pulse: {}", err.message());
                }
            }
            "native_palette_blackout" => {
                let Some(blackout) = b.native_palette_blackout.as_ref().filter(|bo| {
                    bo.start == 0
                        && bo.end == 255
                        && bo.delta == 64
                        && bo.clear_bytes == 0xFA00
                        && b.source == "0x23599"
                }) else {
                    self.load_err = String::from("beat native_palette_blackout:缺少原版全色盤與畫面清除 payload");
                    return;
                };
                // Every baseline DAC component is at most 63, so the exact native
                // subtraction clamps all 256 entries to zero.  The adjacent memset
                // clears the complete 320x200 framebuffer.  A full-surface black cover
                // is therefore exact for this one call site without inventing an
                // indexed-palette approximation for other 0x11d40 callers.
                if !self.native_map_vga.is_empty() {
                    if (self.native_map_vga.len() as i64) < blackout.clear_bytes {
                        self.load_err = format!(
                            "beat native_palette_blackout: indexed framebuffer={}, want at least {}",
                            self.native_map_vga.len(),
                            blackout.clear_bytes
                        );
                        return;
                    }
                    self.native_map_vga[..blackout.clear_bytes as usize].fill(0);
                }
                self.native_full_dac_black = true;
                self.beat_advance();
            }
            "native_staging_present" => {
                if !b.native_staging_present.as_ref().is_some_and(|p| p.slot >= 0 && p.x >= 0 && p.y >= 0) {
                    self.load_err = String::from("beat native_staging_present:缺少原版 wrapper ABI payload");
                    return;
                }
                let spec = b.native_staging_present.unwrap();
                if let Err(err) = self.start_native_staging_present(&spec) {
                    self.load_err = format!("beat native_staging_present: {err}");
                }
            }
            "native_ch23_loop" => {
                if b.native_ch23_loop.is_none() {
                    self.load_err = String::from("beat native_ch23_loop:缺少原始兩段 loop payload");
                    return;
                }
                let lp = b.native_ch23_loop.clone().unwrap();
                if let Err(err) = self.start_native_ch23_loop(&lp, Box::new(|g: &mut Game| g.beat_advance())) {
                    self.load_err = format!("beat native_ch23_loop: {}", err.message());
                }
            }
            "native_2189a_loop" => {
                if b.native2189_a_loop.is_none() {
                    self.load_err = String::from("beat native_2189a_loop:缺少原始十次 loop payload");
                    return;
                }
                let lp = b.native2189_a_loop.clone().unwrap();
                if let Err(err) = self.start_native_2189a(&lp, Box::new(|g: &mut Game| g.beat_advance())) {
                    self.load_err = format!("beat native_2189a_loop: {}", err.message());
                }
            }
            "layout_units" => {
                let Some(layout) = b.layout.as_ref().filter(|l| !l.units.is_empty()) else {
                    self.load_err = String::from("beat layout_units:缺少可編輯的 runtime layout");
                    return;
                };
                // 0x247B4 calls 0x233C6: 0x23465..0x23493 resets all six
                // view globals, not only the pixel camera. Both postbattle carriers
                // represent those same globals (Issue #140).
                let mut layout_view: Option<battle::NativeMapViewState> = None;
                if b.source == "0x247b4" || b.source == "0x257b4" {
                    let Some(m) = self.m.as_ref().filter(|m| {
                        m.tile_w > 0 && m.tile_h > 0 && layout.cam_x % m.tile_w == 0 && layout.cam_y % m.tile_h == 0
                    }) else {
                        self.load_err = String::from("beat layout_units: native reset camera is unavailable");
                        return;
                    };
                    let (x, y) = (layout.cam_x / m.tile_w, layout.cam_y / m.tile_h);
                    let view = battle::NativeMapViewState {
                        camera_x: x,
                        camera_y: y,
                        cursor_x: x,
                        cursor_y: y,
                        ..Default::default()
                    };
                    let mut carrier = battle::State::default();
                    (carrier.w, carrier.h) = (m.w, m.h);
                    if let Err(err) = carrier.materialize_native_map_view_state(view) {
                        self.load_err = format!("beat layout_units: {err}");
                        return;
                    }
                    layout_view = Some(view);
                }
                for placement in &layout.units {
                    if self.handler_unit_at(placement.slot).is_none() || placement.pose < 0 || placement.pose > 3 {
                        self.load_err =
                            format!("beat layout_units: slot{}/pose{} unavailable", placement.slot, placement.pose);
                        return;
                    }
                }
                for placement in &layout.units {
                    let unit = self.handler_unit_at(placement.slot).unwrap();
                    self.handler_unit_mut(unit).set_map_placement(placement.x, placement.y, placement.pose);
                }
                (self.cam_x, self.cam_y) = (layout.cam_x as f64, layout.cam_y as f64);
                if let Some(view) = layout_view {
                    (self.cur_x, self.cur_y) = (view.cursor_x, view.cursor_y);
                    (self.story_native_map_view, self.has_story_native_map_view) = (view, true);
                    if let Some(st) = self.st.as_mut() {
                        (st.native_map_view_state, st.has_native_map_view_state) = (view, true);
                        st.materialize_native_map_range_mode(0);
                    }
                }
                self.beat_advance();
            }
            "direct_record_patch" => {
                if (b.source != "0x2362d" && b.source != "0x23ec4" && b.source != "0x33346" && b.source != "0x25116")
                    || b.direct_record_patch.is_none()
                {
                    self.load_err = String::from("beat direct_record_patch:缺少原版來源或 sparse payload");
                    return;
                }
                let patch = b.direct_record_patch.clone().expect("前面已核對 payload");
                if let Err(err) = self.apply_handler_direct_record_patch(&patch) {
                    self.load_err = format!("beat direct_record_patch: {}", err.message());
                    return;
                }
                self.beat_advance();
            }
            "if" => match self.eval_story_condition(b.condition.as_ref()) {
                Ok(matched) => {
                    let arm = if matched { &b.then } else { &b.r#else };
                    let insert_at = (self.beat_idx + 1) as usize;
                    self.beats.splice(insert_at..insert_at, arm.iter().cloned());
                    self.beat_advance();
                }
                Err(err) => self.load_err = format!("beat if: {}", err.message()),
            },
            "loadch" => {
                if b.load_ch.is_none() {
                    self.load_err = String::from("beat loadch:缺少完整狀態映射");
                    return; // compiler should prevent this; never turn it into a no-op.
                }
                if let Err(err) = self.apply_load_ch(b.load_ch.as_ref()) {
                    self.load_err = format!("beat loadch: {err}");
                    return; // fail closed rather than continuing on the old map/roster.
                }
                if self.native_town_load_pending {
                    let source = self.story_native_map_source.clone();
                    if let Err(err) = self.materialize_native_story_map_state(source.as_ref()) {
                        self.load_err = format!("town LOADCH state: {err}");
                        return;
                    }
                    if let Err(err) = self.compose_native_story_map_frame() {
                        self.load_err = format!("town LOADCH frame: {err}");
                        return;
                    }
                    if let Err(err) = self.start_native_palette_ramp(64, 0, 2, beat_advance_then()) {
                        self.load_err = format!("town LOADCH fade-in: {err}");
                        return;
                    }
                    self.native_town_load_pending = false;
                    return;
                }
                self.beat_advance();
            }
            "ending_handoff" => {
                let node_ok = self.camp.as_ref().and_then(|c| c.node()).is_some_and(|n| !n.next.is_empty());
                if b.source != "0x25970" || !node_ok || self.beat_idx != self.beats.len() as i64 - 1 {
                    self.load_err = String::from("ending handoff: recovered final edge unavailable");
                    return;
                }
                // 25970直接呼叫2BCE5，不額外排入一般故事節點的淡出。
                self.camp.as_mut().unwrap().advance("");
                self.enter_node();
                if self.camp.as_ref().and_then(|c| c.node()).is_none_or(|n| n.r#type != "ending") {
                    self.load_err = String::from("ending handoff: final edge is not an ending");
                }
            }
            "pan" => {
                let mut frames = b.frames;
                if frames == 0 {
                    frames = 30;
                }
                self.cam_pan = Some(CamPanJob {
                    from_x: self.cam_x,
                    from_y: self.cam_y,
                    to_x: b.x as f64,
                    to_y: b.y as f64,
                    frames,
                    tile_step: b.tile_step,
                    then: beat_advance_then(),
                    ..Default::default()
                });
            }
            "walk" => {
                let idx = find_actor(self, b.fig);
                if idx < 0 {
                    self.load_err = format!("beat walk:找不到 fig={}", b.fig);
                    self.beat_advance();
                    return;
                }
                let u = &self.story_actors[idx as usize];
                let (mut from_x, mut from_y) = (b.from_x, b.from_y);
                if from_x == 0 && from_y == 0 {
                    // 未指定起點:沿用角色目前座標(接續上一拍)
                    (from_x, from_y) = (u.x, u.y);
                }
                let mut frames = b.frames;
                if frames == 0 {
                    frames = 60;
                }
                self.follow_walk = b.follow;
                // beat walk 面向:預設保留走位末向(如索爾往上走完仍面上);b.Dir 指定則走完面向它
                let bdir = b.dir.unwrap_or(-1);
                self.story_walks.push(StoryWalkJob {
                    actor: idx,
                    from_x,
                    from_y,
                    to_x: b.x,
                    to_y: b.y,
                    frames,
                    final_dir: bdir,
                    then: beat_advance_then(),
                    ..Default::default()
                });
            }
            "scroll_step" => {
                let Some(idx) = b.slot.filter(|&s| s >= 0 && (s as usize) < self.story_actors.len() && b.steps > 0)
                else {
                    self.load_err = format!(
                        "beat scroll_step: runtime slot {}/steps={} unavailable (materialized={})",
                        go_int_ptr(b.slot),
                        b.steps,
                        self.story_actors.len()
                    );
                    return;
                };
                let (ux, uy) = (self.story_actors[idx as usize].x, self.story_actors[idx as usize].y);
                let mut frames = b.frames;
                if frames <= 0 {
                    frames = b.steps * 7;
                }
                // 0x13185 does not center on the actor. It lets the actor reach screen
                // row 1, then scrolls one tile (smoothly over the same seven ticks) per
                // further upward step. Preserve the current map-origin boundary.
                self.follow_walk = false;
                let origin_y = self.cam_y as i64 / self.m.as_ref().expect("Go：g.m 為 nil 時解參考 panic").tile_h;
                let mut free = uy - origin_y - 1;
                if free < 0 {
                    free = 0;
                }
                if origin_y == 0 {
                    free = b.steps;
                }
                self.story_walks.push(StoryWalkJob {
                    actor: idx,
                    from_x: ux,
                    from_y: uy,
                    to_x: ux,
                    to_y: uy - b.steps,
                    frames,
                    final_dir: 2,
                    scroll_follow: true,
                    from_cam_y: self.cam_y,
                    scroll_free: free,
                    then: beat_advance_then(),
                    ..Default::default()
                });
            }
            "dialog" => {
                let mut lines = self.camp_lines.clone();
                if !b.script.is_empty() {
                    // A compiled handler carries an explicit editable-story context.  Do
                    // not fall back to the enclosing Node's lines here: that could play a
                    // valid index from the wrong FDTXT/loadch segment.
                    match load_story_script_with_identity_at(
                        &handler_story_path(&b.script),
                        &b.scene,
                        b.scene_index.as_ref(),
                    ) {
                        Ok(l) => lines = l,
                        Err(err) => {
                            self.load_err = format!(
                                "beat dialog:無法載入 script={} scene={} scene_index={}: {err}",
                                battle::go_quote(&b.script),
                                battle::go_quote(&b.scene),
                                go_int_ptr(b.scene_index)
                            );
                            self.beat_advance();
                            return;
                        }
                    }
                }
                let mut n = b.count;
                if n <= 0 {
                    n = 1;
                }
                if !b.native_dialogues.is_empty() && b.native_dialogues.len() as i64 != n {
                    self.load_err =
                        format!("beat dialog: native_dialogues={} want count={}", b.native_dialogues.len(), n);
                    return;
                }
                if b.native_dialogue.is_some() && !b.native_dialogues.is_empty() {
                    self.load_err = String::from("beat dialog: native_dialogue 與 native_dialogues 不可混用");
                    return;
                }
                let mut end = b.line + n;
                if end > lines.len() as i64 {
                    end = lines.len() as i64;
                }
                self.dialog = Vec::new();
                self.dlg_page = 0; // 新對白從第一頁起
                let mut i = end - 1;
                while i >= b.line && i >= 0 {
                    // 反序堆疊(同 enterNode story 分支慣例)
                    let ln = &lines[i as usize];
                    let native = if b.native_dialogues.is_empty() {
                        b.native_dialogue.as_ref()
                    } else {
                        Some(&b.native_dialogues[(i - b.line) as usize])
                    };
                    match self.resolve_campaign_dialog_line(ln, b.upper, native) {
                        Ok(dialog_line) => self.dialog.push(dialog_line),
                        Err(err) => {
                            self.dialog = Vec::new();
                            self.load_err = format!("beat dialog:{err}");
                            return;
                        }
                    }
                    i -= 1;
                }
                if self.dialog.is_empty() {
                    // line/count 對不到資料:跳拍避免卡死
                    self.load_err =
                        format!("beat dialog:line={} count={} 對不到 script lines(len={})", b.line, n, lines.len());
                    self.beat_advance();
                    return;
                }
                if let Err(err) = self.start_native_dialogue_frames() {
                    self.dialog = Vec::new();
                    self.load_err = format!("beat dialog:{err}");
                }
            }
            "act" => {
                if !b.acting.is_empty() {
                    // Decoded acting refers to the current materialized unit array. A slot
                    // beyond the materialized count but inside the native 96-record
                    // allocation is what the original also hits: 0x1366A writes the pose
                    // into a record nothing draws. 第六章 ch05_post 的 ACTING 27 指 slot 34，
                    // 原版當時只有 34 筆（r3 seq 1508–1528，記錄 33 的 +3 全程是 0），畫面沒有
                    // 任何單位動——重製端同樣略過那一筆，不當成錯誤。超出 96 筆才是資料錯。
                    for frame in &b.acting {
                        for target in &frame.units {
                            if let Some(slot) = target.slot {
                                if !(0..NATIVE_RUNTIME_RECORD_CAPACITY).contains(&slot) {
                                    self.load_err = format!(
                                        "beat act {}: original runtime slot {} outside the {}-record allocation",
                                        b.source, slot, NATIVE_RUNTIME_RECORD_CAPACITY
                                    );
                                    return;
                                }
                                if slot >= self.handler_unit_count() && self.cutscene_log {
                                    eprintln!(
                                        "[cutscene] act {} slot {} beyond materialized {}: original writes an unmaterialized record, no visible effect",
                                        b.source,
                                        slot,
                                        self.handler_unit_count()
                                    );
                                }
                            }
                        }
                    }
                    let mut job =
                        ActPoseJob { acting: b.acting.clone(), then: beat_advance_then(), ..Default::default() };
                    self.begin_acting_frame(&mut job);
                    self.act_job = Some(job);
                    return;
                }
                let idx = find_actor(self, b.fig);
                if idx < 0 || b.poses.is_empty() {
                    self.beat_advance();
                    return;
                }
                let mut frames = b.pose_frames;
                if frames == 0 {
                    frames = 30;
                }
                self.act_job = Some(ActPoseJob {
                    actor: idx,
                    poses: b.poses.clone(),
                    frames,
                    then: beat_advance_then(),
                    ..Default::default()
                });
            }
            "spawn" => {
                // Original 0x10b4e scans the FDFIELD records for this group and calls
                // the unit constructor once per match.  That constructor writes at the
                // current unit_count and increments it: this is append, not toggling an
                // already slot-stable full roster.  Keep the legacy activation fallback
                // for authored scene beats that have no LOADCH roster.
                let materialized;
                if let Some(st) = self.st.as_mut() {
                    if let Some(gate) = b.raw_placement_gate {
                        match st.append_group_with_native_placement(b.group, gate as u8) {
                            Ok(n) => materialized = n,
                            Err(err) => {
                                self.load_err = format!("beat spawn {}: {err}", b.source);
                                return;
                            }
                        }
                    } else {
                        materialized = st.append_group(b.group);
                    }
                } else {
                    materialized = self.materialize_story_group(b.group);
                }
                if materialized == 0 && (!b.source.is_empty() || b.count > 0) {
                    self.load_err = format!("beat spawn {}: group {} unavailable in runtime roster", b.source, b.group);
                    return;
                }
                if b.count > 0 && materialized != b.count {
                    self.load_err = format!(
                        "beat spawn {}: group {} materialized={} want={}",
                        b.source, b.group, materialized, b.count
                    );
                    return;
                }
                self.beat_advance();
            }
            "spawn_intro" => {
                if !b.source.is_empty() {
                    let Some(gate) = b.raw_placement_gate else {
                        self.load_err = format!("beat spawn_intro {}: raw placement gate unavailable", b.source);
                        return;
                    };
                    if let Err(err) = self.start_native_spawn_intro(b.group, gate as u8, beat_advance_then()) {
                        self.load_err = format!("beat spawn_intro {}: {err}", b.source);
                    }
                    return;
                }
                if let (Some(st), Some(gate)) = (self.st.as_mut(), b.raw_placement_gate) {
                    if let Err(err) = st.append_group_with_native_placement(b.group, gate as u8) {
                        self.load_err = format!("beat spawn_intro {}: {err}", b.source);
                        return;
                    }
                } else {
                    self.materialize_story_group(b.group);
                }
                let mut frames = b.frames;
                if frames <= 0 {
                    frames = 12;
                }
                self.beat_delay = frames;
            }
            "deactivate_unit" => {
                let Some(r) = b.slot.and_then(|s| self.handler_unit_at(s)) else {
                    self.load_err = format!(
                        "beat deactivate_unit: runtime slot {} unavailable (materialized={})",
                        go_int_ptr(b.slot),
                        self.handler_unit_count()
                    );
                    return;
                };
                let u = self.handler_unit_mut(r);
                u.on_field = false;
                if u.has_native_record_byte5 {
                    // 0x32975 overwrites the complete byte with 1; do not OR only bit0.
                    u.native_record_byte5 = 1;
                }
                self.beat_advance();
            }
            "reactivate_nonzero_hp" => {
                let mut known_loop = b.source == "0x33cea" && b.dir.is_none();
                if b.source == "0x336ed" {
                    known_loop = b.count == 16 && b.dir == Some(2);
                }
                if !known_loop || b.count <= 0 {
                    self.load_err = String::from("beat reactivate_nonzero_hp:缺少已證實的 counted-loop 來源");
                    return;
                }
                // Preflight the whole raw loop before changing any record.  The native
                // +0x40 word is the already-closed current-HP field represented by Unit.HP;
                // byte +5 remains explicit provenance and is never inferred from HP.
                for slot in 0..b.count {
                    let ok = self.handler_unit_at(slot).is_some_and(|r| {
                        let u = self.handler_unit(r);
                        u.camp == battle::OWN && u.has_native_record_byte5
                    });
                    if !ok {
                        self.load_err =
                            format!("beat reactivate_nonzero_hp: slot{slot} lacks player/raw byte+5 provenance");
                        return;
                    }
                }
                for slot in 0..b.count {
                    let r = self.handler_unit_at(slot).unwrap();
                    let u = self.handler_unit_mut(r);
                    if u.hp != 0 {
                        u.on_field = true;
                        u.native_record_byte5 = 0;
                        if let Some(dir) = b.dir {
                            u.set_map_pose(dir);
                        }
                    }
                }
                self.beat_advance();
            }
            "reset_pose" => {
                if let Some(st) = self.st.as_mut() {
                    for unit in &mut st.units {
                        unit.set_map_pose(0);
                    }
                } else {
                    for a in &mut self.story_actors {
                        a.set_map_pose(0);
                    }
                }
                self.beat_delay = 1; // original 20ms at a 60Hz remake clock
            }
            "redraw" => {
                if (b.source == "0x236ee" || b.source == "0x23f69")
                    && let Err(err) = self.compose_native_map_frame()
                {
                    self.load_err = format!("beat redraw: native {}: {err}", b.source);
                    return;
                }
                // Ebiten presents the current state once after this Update.  Blocking
                // one frame preserves the standalone original 0x11cac(0) boundary.
                self.beat_delay = 1;
            }
            "focus_unit" => {
                let Some(slot) = b.slot.filter(|&s| s >= 0 && (s as usize) < self.story_actors.len()) else {
                    self.load_err = format!(
                        "beat focus_unit: runtime slot {} unavailable (materialized={})",
                        go_int_ptr(b.slot),
                        self.story_actors.len()
                    );
                    return;
                };
                let u = &self.story_actors[slot as usize];
                self.focus_job = Some(FocusUnitJob { target_x: u.x, target_y: u.y, ..Default::default() });
                if self.camp.as_ref().is_some_and(|c| c.node_id() == "story_ch24") {
                    // 0x33142 的尾端聚焦延續對白留下的六個全域，不能改回章節常數。
                    if !self.has_story_native_map_view {
                        self.load_err = String::from("beat focus_unit:原生視圖交接缺失");
                        self.focus_job = None;
                        return;
                    }
                    (self.cur_x, self.cur_y) =
                        (self.story_native_map_view.cursor_x, self.story_native_map_view.cursor_y);
                    self.focus_job.as_mut().unwrap().native_view = true;
                }
            }
            "join" => {
                if !campaign::joinable_character_id(b.char_id) {
                    self.load_err = format!("beat join:非法 player char_id={}", b.char_id);
                    return;
                }
                if !b.source.is_empty() && !self.party_roster.contains_key(&b.char_id) {
                    let mut base: Option<battle::Unit> = None;
                    if let Some(st) = self.st.as_ref() {
                        let mut matches = 0;
                        for unit in &st.units {
                            if unit.has_native_record_byte8 && unit.native_record_byte8 as i64 == b.char_id {
                                base = Some(unit.clone());
                                matches += 1;
                            }
                        }
                        if matches > 1 {
                            self.load_err =
                                format!("beat join:char_id={} 有 {} 筆 raw +8 runtime records", b.char_id, matches);
                            return;
                        }
                    }
                    if base.is_none() {
                        let mut matches = 0;
                        for unit in &self.story_actors {
                            if unit.has_native_record_byte8 && unit.native_record_byte8 as i64 == b.char_id {
                                base = Some(unit.clone());
                                matches += 1;
                            }
                        }
                        if matches > 1 {
                            self.load_err =
                                format!("beat join:char_id={} 有 {} 筆 raw +8 story records", b.char_id, matches);
                            return;
                        }
                    }
                    let base = match base {
                        Some(base) => base,
                        None => {
                            // Native sub_112A5 appends a new persistent record from its
                            // character-data helpers; it does not require the character to
                            // already occupy the current field array.  Only an explicit,
                            // evidence-labelled base roster may bridge that boundary.
                            if !self.has_native_join_bases {
                                let join_path = asset_path("assets/data/native_join_base_units.json");
                                match campaign::load_native_join_base_table(&join_path) {
                                    Ok(table) => {
                                        self.native_join_bases = table;
                                        self.has_native_join_bases = true;
                                    }
                                    Err(err) => {
                                        self.load_err = format!(
                                            "beat join:char_id={} 缺少 raw +8 runtime record 且 base table 無法載入: {err}",
                                            b.char_id
                                        );
                                        return;
                                    }
                                }
                            }
                            match self.native_join_bases.load_base_unit(b.char_id) {
                                Ok(fallback) => fallback,
                                Err(err) => {
                                    self.load_err = format!("beat join: {err}");
                                    return;
                                }
                            }
                        }
                    };
                    let materialized = match self.materialize_native_join_persistent_unit(b.char_id, &base) {
                        Ok(u) => u,
                        Err(err) => {
                            self.load_err = format!("beat join: {err}");
                            return;
                        }
                    };
                    self.party_roster.insert(b.char_id, clone_native_shop_unit(&materialized));
                }
                if !self.party_members.get(&b.char_id).copied().unwrap_or(false) {
                    self.party_members.insert(b.char_id, true);
                    self.party_join_order.push(b.char_id);
                }
                self.beat_advance();
            }
            "sync_party" => {
                if let Err(err) = self.sync_party_from_battle() {
                    self.load_err = format!("beat sync_party: {err}");
                    return;
                }
                self.beat_advance();
            }
            "clear_native_record_bit7" => {
                let Some(st) = self.st.as_mut().filter(|st| !st.units.is_empty()) else {
                    self.load_err = String::from("beat clear_native_record_bit7:缺少 runtime battle records");
                    return;
                };
                if let Some(i) = st.units.iter().position(|u| !u.has_native_record_byte5) {
                    self.load_err = format!("beat clear_native_record_bit7: slot{i} lacks raw byte+5 provenance");
                    return;
                }
                for unit in &mut st.units {
                    unit.native_record_byte5 &= 0x7f;
                }
                self.beat_advance();
            }
            "reset_persistent_roster_state" => {
                self.reset_persistent_roster_state();
                self.beat_advance();
            }
            "set_chapter" => {
                let Some(chapter) = b.chapter.filter(|&c| c >= 0) else {
                    self.load_err = String::from("beat set_chapter:缺少有效章節");
                    return;
                };
                self.handler_chapter = chapter;
                self.beat_advance();
            }
            "grant_item" => {
                let Some(item) = b.item_id.filter(|id| (0..=0xff).contains(id)) else {
                    self.load_err = String::from("beat grant_item:缺少有效 item_id");
                    return;
                };
                let Some(state) = self.st.as_mut() else {
                    self.load_err = String::from("beat grant_item:缺少 runtime battle state");
                    return;
                };
                for unit in state.units.iter_mut().filter(|unit| unit.camp == battle::OWN && unit.inventory.len() < 8) {
                    if unit.add_inventory_item(item, false) {
                        break;
                    }
                }
                self.beat_advance();
            }
            "bgm" => {
                self.play_bgm(&b.track);
                self.beat_advance();
            }
            "bgm_stop" => {
                self.stop_bgm();
                self.beat_advance();
            }
            "fade" => {
                let mut frames = b.frames;
                if frames == 0 {
                    frames = STORY_FADE_FRAMES;
                }
                self.fade =
                    Some(StoryFade { out: b.out, total: frames, then: beat_advance_then(), ..Default::default() });
            }
            "transition_reveal" => {
                if b.reveal_frames <= 0 {
                    self.load_err = String::from("beat transition_reveal:缺少正確 frame count");
                    return;
                }
                if let Err(err) = self.start_native_24b4d(
                    b.reveal_frames,
                    b.reveal_delay_ms,
                    Box::new(|g: &mut Game| g.beat_advance()),
                ) {
                    self.load_err = format!("beat transition_reveal: {}", err.message());
                }
            }
            "load_res" => {
                let Some(resource_id) = b.resource_id.filter(|&id| id >= 0) else {
                    self.load_err = String::from("beat load_res:缺少 resource_id");
                    return;
                };
                if b.source == "0x24a4b" || b.source == "0x24a65" || b.source == "0x24a7f" {
                    if let Err(err) = self.stage_native_ch22_resource(b) {
                        self.native_ch22_reload = None;
                        self.load_err = format!("beat load_res: {}", err.message());
                        return;
                    }
                } else {
                    self.handler_resource = resource_id;
                }
                self.beat_advance();
            }
            "native_ch22_reset_grid" => {
                if let Err(err) = self.reset_native_ch22_reload_grid() {
                    self.native_ch22_reload = None;
                    self.load_err = format!("beat native_ch22_reset_grid: {}", err.message());
                    return;
                }
                self.beat_advance();
            }
            "native_ch22_prepare_aux" => {
                if let Err(err) = self.prepare_native_ch22_aux() {
                    self.native_ch22_reload = None;
                    self.load_err = format!("beat native_ch22_prepare_aux: {}", err.message());
                    return;
                }
                self.beat_advance();
            }
            "play_sfx" => {
                if b.resource_id != Some(88) || !(b.sfx_index == Some(1) || b.sfx_index == Some(-1)) {
                    self.load_err = format!(
                        "beat play_sfx:未映射 resource={} index={}",
                        go_int_ptr(b.resource_id),
                        go_int_ptr(b.sfx_index)
                    );
                    return;
                }
                if self.handler_resource != 88 {
                    self.load_err = format!("beat play_sfx:resource handle={} want 88", self.handler_resource);
                    return;
                }
                if b.sfx_index == Some(1) {
                    let sfx = self.sfx_transition.clone();
                    self.play_raw(&sfx);
                }
                self.beat_advance();
            }
            "release_res" => {
                if b.resource_id != Some(self.handler_resource) {
                    self.load_err = format!(
                        "beat release_res:resource={} handle={}",
                        go_int_ptr(b.resource_id),
                        self.handler_resource
                    );
                    return;
                }
                self.handler_resource = 0;
                self.beat_advance();
            }
            "palette_update" => {
                if b.palette_start == 0
                    && b.palette_end == 255
                    && b.palette_delta >= 0
                    && b.palette_delta <= 63
                    && native_map_assets_available(self.native_map_assets.as_ref())
                {
                    if let Err(err) =
                        self.start_native_palette_update(b.palette_delta, Box::new(|g: &mut Game| g.beat_advance()))
                    {
                        self.load_err = format!("beat palette_update: {}", err.message());
                    }
                    return;
                }
                // Full range delta=255 is independent of source RGB: every six-bit
                // component saturates to white. This exact operation can cover legacy
                // scene presentation without inventing a general RGB approximation.
                // Legacy scenes without an indexed frame retain their white/restore path.
                if b.palette_start == 0 && b.palette_end == 255 && b.palette_delta == 255 {
                    self.native_full_dac_white = true;
                } else if b.palette_delta == 0 {
                    self.native_full_dac_white = false;
                } else {
                    self.load_err = format!(
                        "beat palette_update: range {}..{} delta {} requires indexed palette renderer",
                        b.palette_start, b.palette_end, b.palette_delta
                    );
                    return;
                }
                self.beat_advance();
            }
            "delay" => {
                let mut frames = b.frames;
                if frames == 0 && b.ms > 0 {
                    frames = b.ms * 60 / 1000;
                }
                if frames <= 0 {
                    frames = 1;
                }
                self.beat_delay = frames;
            }
            _ => {
                self.load_err = format!("beat:未知原語 {}", b.op);
                self.beat_advance();
            }
        }
    }

    /// Go：`main.go:2625`，`func (g *Game) resolveCampaignDialogLine(line campaign.Line, upperOverride *bool, nativeOverride *campaign.NativeDialogueLayout) (battle.DialogLine, error)`
    ///
    /// R4 B 組：`upperOverride *bool` 改收值（Go 只讀、再存進 `DialogLine.Upper`），`nativeOverride` 改收
    /// `Option<&_>`（Go 可傳 nil）。R6 的非繁中斷行只替換顯示頁面。
    pub(crate) fn resolve_campaign_dialog_line(
        &mut self,
        line: &campaign::Line,
        upper_override: Option<bool>,
        native_override: Option<&campaign::NativeDialogueLayout>,
    ) -> Result<battle::DialogLine> {
        let text = self.localized_story_text(line)?;
        let mut speaker = line.speaker;
        if let Some(slot) = line.speaker_slot {
            let unit = if let Some(st) = self.st.as_ref() {
                if slot < 0 || slot as usize >= st.units.len() {
                    return Err(game_err!("speaker_slot {} unavailable (battle units={})", slot, st.units.len()));
                }
                &st.units[slot as usize]
            } else {
                if slot < 0 || slot as usize >= self.story_actors.len() {
                    return Err(game_err!(
                        "speaker_slot {} unavailable (cutscene units={})",
                        slot,
                        self.story_actors.len()
                    ));
                }
                &self.story_actors[slot as usize]
            };
            speaker = unit.portrait;
        }
        let mut upper = line.upper;
        if upper_override.is_some() {
            upper = upper_override;
        }
        let mut native: Option<battle::NativeDialogueLayout> = None;
        if let Some(source) = native_override {
            let localized;
            let layout = if self.locale_id != "zh-Hant" {
                localized = localized_native_dialogue_layout(&text, source, self.font.as_mut())?;
                &localized
            } else {
                source
            };
            let resolved_speaker = self.resolve_native_story_dialogue_speaker(layout)?;
            let motion_target_y = self.resolve_native_story_dialogue_motion_target(layout)?;
            speaker = resolved_speaker;
            native = Some(battle::NativeDialogueLayout {
                source_dat: layout.source_dat.clone(),
                string_index: layout.string_index,
                utterance: layout.utterance,
                control: layout.control.clone(),
                operand: layout.operand,
                pages: layout.pages.clone(),
                glyph_ids: layout.glyph_ids.clone(),
                glyph_pages: layout.glyph_pages.clone(),
                motion_target_y,
                has_motion_target_y: true,
            });
        }
        Ok(battle::DialogLine { speaker, text, upper, native_dialogue: native })
    }

    /// resolveNativeStoryDialogueSpeaker 重播 sub_15F84 的 DATO selector owner。
    /// FFED/FFEC 的 operand 是第一張 runtime array slot；FFEF/FFEE 則先以
    /// raw +8 查找 active record，命中後取 raw +7，未命中保留 direct-DATO operand。
    ///
    /// Go：`main.go:2704`，`func (g *Game) resolveNativeStoryDialogueSpeaker(layout *campaign.NativeDialogueLayout) (int, error)`
    pub(crate) fn resolve_native_story_dialogue_speaker(&self, layout: &campaign::NativeDialogueLayout) -> Result<i64> {
        let units: &[battle::Unit] = match self.st.as_ref() {
            Some(st) => &st.units,
            None => &self.story_actors,
        };
        match layout.control.as_str() {
            "FFED" | "FFEC" => {
                let op = layout.operand;
                if op < 0 || op as usize >= units.len() || !units[op as usize].has_battle_fig {
                    return Err(game_err!(
                        "native story dialogue: runtime speaker slot {} lacks raw +7 provenance",
                        layout.operand
                    ));
                }
                return Ok(units[op as usize].battle_fig);
            }
            "FFEF" | "FFEE" => {}
            _ => {
                return Err(game_err!(
                    "native story dialogue: unsupported speaker control {}",
                    battle::go_quote(&layout.control)
                ));
            }
        }
        for (index, unit) in units.iter().enumerate() {
            if !unit.has_native_record_byte5 || (!unit.has_native_record_byte8 && !unit.has_native_identity) {
                return Err(game_err!(
                    "native story dialogue: scene unit {} lacks speaker raw +8/+5 provenance",
                    index
                ));
            }
            let mut raw_identity = unit.native_identity;
            if unit.has_native_record_byte8 {
                raw_identity = unit.native_record_byte8 as i64;
            }
            if raw_identity == layout.operand && unit.native_record_byte5 & 1 == 0 {
                if !unit.has_battle_fig {
                    return Err(game_err!(
                        "native story dialogue: matched identity {} lacks raw +7 provenance",
                        layout.operand
                    ));
                }
                return Ok(unit.battle_fig);
            }
        }
        Ok(layout.operand)
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
        match layout.control.as_str() {
            "FFED" => return Ok(2),
            "FFEC" => return Ok(112),
            "FFEF" | "FFEE" => {}
            _ => {
                return Err(game_err!(
                    "native story dialogue: unsupported motion control {}",
                    battle::go_quote(&layout.control)
                ));
            }
        }
        let units: &[battle::Unit] = match self.st.as_ref() {
            Some(st) => &st.units,
            None => &self.story_actors,
        };
        for (index, unit) in units.iter().enumerate() {
            if !unit.has_native_record_byte5 || (!unit.has_native_record_byte8 && !unit.has_native_identity) {
                return Err(game_err!(
                    "native story dialogue: scene unit {} lacks 0x12C60 raw +8/+5 provenance",
                    index
                ));
            }
            let mut raw_identity = unit.native_identity;
            if unit.has_native_record_byte8 {
                raw_identity = unit.native_record_byte8 as i64;
            }
            if raw_identity == layout.operand && unit.native_record_byte5 & 1 == 0 {
                if layout.control == "FFEF" {
                    return Ok(2);
                }
                return Ok(112);
            }
        }
        Ok(0)
    }

    /// Go：`main.go:2981`，`func (g *Game) materializeStoryGroup(group int) int`
    pub(crate) fn materialize_story_group(&mut self, group: i64) -> i64 {
        // Go 比較 `g.storyRoster != nil`（KNOWN_DIFFERENCES `story-roster-nil-vs-empty`）。
        if !self.story_roster.is_empty() {
            if self.story_spawned.get(&group).copied().unwrap_or(false) {
                return 0;
            }
            let mut materialized = 0;
            let start = self.story_actors.len() as i64;
            for actor in &self.story_roster {
                if actor.group == group {
                    let mut actor = actor.clone();
                    actor.on_field = true;
                    self.story_actors.push(actor);
                    materialized += 1;
                }
            }
            if materialized > 0 {
                if self.story_native_map_state.is_some()
                    && let Err(err) = self.append_native_story_map_actors(start)
                {
                    self.load_err = format!("story SPAWN native map state: {err}");
                    return 0;
                }
                self.story_spawned.insert(group, true);
            }
            return materialized;
        }
        let mut materialized = 0;
        for a in &mut self.story_actors {
            if a.group == group {
                a.on_field = true;
                materialized += 1;
            }
        }
        materialized
    }
}

/// R4 B 組新增：Go 以方法值 `g.beatAdvance` 當延續傳給演出工作；Rust 每次建立一個新的閉包。
fn beat_advance_then() -> Option<Box<dyn FnOnce(&mut Game) + Send>> {
    Some(Box::new(|g: &mut Game| g.beat_advance()))
}

/// R4 B 組新增：Go 錯誤訊息裡 `%v` 印 `*int`。Go 印 nil 為 `<nil>`、非 nil 為記憶體位址；Rust 印值
/// （KNOWN_DIFFERENCES `go-int-pointer-format`）。
fn go_int_ptr(v: Option<i64>) -> String {
    match v {
        None => String::from("<nil>"),
        Some(v) => v.to_string(),
    }
}

impl Game {
    /// loadCHPartyOrder projects the permanent JOIN chronology through the current
    /// preparation selection.  The original LOADCH constructs the deployed party
    /// first, so a late-game cutscene must not silently resurrect every joined
    /// character merely because partyJoinOrder is longer than the selected roster.
    /// With no JOIN history this remains a direct/debug binding's authored order.
    ///
    /// Go：`main.go:3021`，`func (g *Game) loadCHPartyOrder(state *campaign.LoadCHState) []int`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn load_ch_party_order(&self, state: Option<&campaign::LoadCHState>) -> Vec<i64> {
        if self.party_join_order.is_empty() {
            return state.map(|s| s.party_order.clone()).unwrap_or_default();
        }
        if state.is_some_and(|s| s.all_persistent_party) {
            return self.party_join_order.clone();
        }
        let members = self.battle_party_members();
        self.party_join_order.iter().copied().filter(|id| members.get(id).copied().unwrap_or(false)).collect()
    }

    /// Go：`main.go:3041`，`func (g *Game) nativeRestoredPartyUnitsForLoadCH( order []int, deploy []battle.Cell, scenario *battle.Scenario, ) ([]*battle.Unit, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_restored_party_units_for_load_ch(
        &self,
        order: &[i64],
        deploy: &[battle::Cell],
        scenario: Option<&battle::Scenario>,
    ) -> Result<Option<Vec<battle::Unit>>> {
        // Go 回傳新建的 `[]*battle.Unit`（不在 g.st 裡），Rust 回傳擁有的單位；`None` 對應 Go 的
        // nil 結果（沒有原版章節還原計畫），與「長度 0」區分（簽名變更見 `PORTING.md`）。
        if self.native_chapter_restore.is_none() {
            return Ok(None);
        }
        if order.is_empty() || deploy.len() < order.len() {
            return Err(game_err!("原版 LOADCH 部署格／順序長度不符：deploy={} order={}", deploy.len(), order.len()));
        }
        let mut templates: BTreeMap<i64, battle::Unit> = BTreeMap::new();
        if let Some(scenario) = scenario {
            for unit in scenario.party_units(&[]) {
                templates.insert(unit.fig, unit);
            }
        }
        let mut units = Vec::with_capacity(order.len());
        for (index, &id) in order.iter().enumerate() {
            let source = match self.party_roster.get(&id) {
                Some(source)
                    if source.has_native_identity
                        && source.native_identity == id
                        && source.has_map_selector_key
                        && source.has_native_record_death_effect =>
                {
                    source
                }
                _ => return Err(game_err!("原版 LOADCH 隊伍 slot {index} 缺 persistent identity {id} 來源")),
            };
            let mut unit = match templates.get(&id) {
                Some(template) => {
                    let mut unit = template.clone();
                    apply_persistent_stats(Some(&mut unit), Some(source));
                    unit
                }
                None => clone_native_shop_unit(source),
            };
            unit.camp = battle::OWN;
            // 0x1088D 的80-byte持續record copy後，0x10AB1只將+31寫FF。
            unit.native_record_death_effect = source.native_record_death_effect;
            unit.native_record_death_effect[0] = 0xff;
            unit.has_native_record_death_effect = true;
            (unit.death_effect, unit.death_reward) = (None, None);
            unit.fig = id;
            unit.battle_fig = unit.map_selector_key;
            unit.has_battle_fig = true;
            (unit.x, unit.y) = (deploy[index].x, deploy[index].y);
            (unit.group, unit.on_field, unit.acted) = (0, true, false);
            units.push(unit);
        }
        Ok(Some(units))
    }

    /// applyLoadCH is the remake adapter for original 0x205da/0x1088d.  The
    /// original operation selects FDTXT chapter+1 and the three FDFIELD resources
    /// for the same chapter in one call; it is not merely a camera/map command.
    /// The binding therefore provides all three editable counterparts.  Validate
    /// roster and text before replacing the rendered map so a malformed asset does
    /// not leave a half-applied chapter transition behind.
    ///
    /// Go：`main.go:3099`，`func (g *Game) applyLoadCH(state *campaign.LoadCHState) error`
    ///
    /// Go 不改寫 `state`，Rust 收唯讀借用（簽名變更見 `PORTING.md`）。Go 以 `&battle.State{Units: party}`
    /// 暫時包住 `[]*Unit` 交給 `initializeEquipmentBases`／`applyPersistentParty` 原地改寫；Rust 把 `party`
    /// 移進暫時的 `State` 再移回，結果相同。
    pub(crate) fn apply_load_ch(&mut self, state: Option<&campaign::LoadCHState>) -> Result<()> {
        let state = match state {
            Some(s)
                if s.chapter >= 0
                    && !s.map.is_empty()
                    && !s.roster.is_empty()
                    && s.slot_count > 0
                    && !s.script.is_empty() =>
            {
                s
            }
            _ => return Err(GameError::new("incomplete map/roster/story state")),
        };
        let mut roster = battle::load(&asset_path(&state.roster))
            .map_err(|e| GameError::from(e).wrap(format!("roster {}: ", battle::go_quote(&state.roster))))?;
        // 原版 LOADCH 建構的 runtime record 已經走過 0x10C50＋0x1B750；這裡綁物品表
        // 讓 FDFIELD 單位的 AP／DP／HIT／EV／HP／MP 由建構器補齊（見 BindNativeFutureItemRows）。
        self.bind_native_future_item_rows(Some(&mut roster)).map_err(|e| {
            e.wrap(format!("roster {} native constructor item rows: ", battle::go_quote(&state.roster)))
        })?;
        let mut source_slots = state.slot_count;
        if let Some(source_slot_count) = state.source_slot_count {
            if !state.all_persistent_party
                || state.chapter != 30
                || state.slot_count != 31
                || source_slot_count != 1
                || roster.own_deploy.len() != 31
            {
                return Err(GameError::new("LOADCH source/deployment capacity lacks selector30 provenance"));
            }
            source_slots = source_slot_count;
        }
        if roster.units.len() as i64 != source_slots {
            return Err(game_err!(
                "roster {} has {} slots, binding declares {source_slots}",
                battle::go_quote(&state.roster),
                roster.units.len()
            ));
        }
        let lines = load_story_script_with_identity_at(&state.script, "", None)
            .map_err(|e| e.wrap(format!("story script {}: ", battle::go_quote(&state.script))))?;
        self.load_map(&state.map).map_err(|e| e.wrap(format!("map {}: ", battle::go_quote(&state.map))))?;
        let (map_w, map_h, tile_w, tile_h, composition) = {
            let m = self.m.as_ref().expect("loadMap 成功後 g.m 不是 nil");
            (m.w, m.h, m.tile_w, m.tile_h, m.native_composition_event_bytes.clone())
        };
        // 0x205da/0x1088d resets camera, absolute cursor and visible cursor to
        // zero after loading the field. Keep this raw view in a scene-only carrier;
        // putting it in g.st would make later story SPAWN beats use the battle-state
        // append path and invent runtime records that LOADCH has not materialized.
        let mut load_ch_view = battle::NativeMapViewState::default();
        let mut view_carrier = battle::State::default();
        (view_carrier.w, view_carrier.h) = (map_w, map_h);
        if let Some(view) = state.post_load_view {
            if !state.all_persistent_party || view.validate(map_w, map_h).is_err() {
                return Err(GameError::new("LOADCH caller view lacks the selector30 contract"));
            }
            load_ch_view = battle::NativeMapViewState {
                camera_x: view.camera_x,
                camera_y: view.camera_y,
                cursor_x: view.cursor_x,
                cursor_y: view.cursor_y,
                visible_cursor_x: view.visible_cursor_x,
                visible_cursor_y: view.visible_cursor_y,
            };
        }
        view_carrier.materialize_native_map_view_state(load_ch_view).map_err(|e| {
            GameError::from(e).wrap(format!("map {} native view reset: ", battle::go_quote(&state.map)))
        })?;
        let mut party: Vec<battle::Unit> = Vec::new();
        if state.all_persistent_party
            && (state.chapter != 30
                || state.slot_count != 31
                || source_slots != 1
                || roster.own_deploy.len() != 31
                || self.party_join_order.is_empty())
        {
            return Err(GameError::new("LOADCH all-party source is not selector30/31 records"));
        }
        if !state.party_scenario.is_empty() {
            let quoted = battle::go_quote(&state.party_scenario);
            let mut scenario = battle::load_scenario(&asset_path(&state.party_scenario))
                .map_err(|e| GameError::from(e).wrap(format!("party scenario {quoted}: ")))?;
            // A normal campaign reaches this LOADCH after JOIN established permanent
            // membership. Direct scene/debug starts have no membership history and
            // use the evidence-backed PartyOrder stored in the editable binding.
            let party_order = self.load_ch_party_order(Some(state));
            // A binding's PartyOrder describes the normal permanent chronology.  Once
            // preparation has an active selection, the projected deployed order is the
            // native runtime source and may intentionally be shorter than that field.
            if !self.party_join_order.is_empty()
                && self.party_deploy.is_empty()
                && !state.party_order.is_empty()
                && !equal_int_order(&party_order, &state.party_order)
            {
                return Err(game_err!(
                    "party JOIN chronology {} differs from binding {}",
                    go_int_slice(&party_order),
                    go_int_slice(&state.party_order)
                ));
            }
            match self
                .native_restored_party_units_for_load_ch(&party_order, &roster.own_deploy, Some(&scenario))
                .map_err(|e| e.wrap(format!("party scenario {quoted}: ")))?
            {
                Some(restored) => party = restored,
                None => {
                    let mut members = self.battle_party_members();
                    if state.all_persistent_party {
                        members = party_order.iter().map(|&id| (id, true)).collect();
                    }
                    filter_scenario_party(Some(&mut scenario), &members);
                    reorder_scenario_party(Some(&mut scenario), &party_order)
                        .map_err(|e| e.wrap(format!("party scenario {quoted}: ")))?;
                    party = scenario.party_units(&roster.own_deploy);
                }
            }
            // Native JOIN has already created persistent records before a normal
            // campaign reaches this LOADCH. The remake JOIN beat records membership
            // and chronology, so seed only records that are still absent from the
            // typed roster. Direct/debug LOADCH replay has no JOIN history and must
            // not silently manufacture persistent campaign state.
            if !self.party_join_order.is_empty() && self.native_chapter_restore.is_none() {
                let mut carrier = battle::State::default();
                carrier.units = std::mem::take(&mut party);
                self.initialize_equipment_bases(Some(&mut carrier));
                party = carrier.units;
                self.seed_persistent_party_from_load_ch(&party_order, &party)
                    .map_err(|e| e.wrap(format!("party scenario {quoted} persistence: ")))?;
            }
            if !self.party_roster.is_empty() {
                let mut carrier = battle::State::default();
                carrier.units = std::mem::take(&mut party);
                self.apply_persistent_party(Some(&mut carrier));
                party = carrier.units;
            }
        }

        // A handler cutscene uses the loaded FDFIELD records as a source, not as a
        // pre-built unit array. Original 0x10b4e appends matching group records to
        // the current array. Persistent party members are constructed first when a
        // binding supplies PartyScenario; LOADCH then materializes group 0 and later
        // SPAWN calls append groups in FDFIELD order. This preserves the actual
        // runtime slot identity for every evidence-backed load/spawn sequence.
        let mut actors: Vec<battle::Unit> = Vec::with_capacity(party.len() + roster.units.len());
        let mut source_roster: Vec<battle::Unit> = Vec::with_capacity(roster.units.len());
        // Go 的 prospective 是指向每個已建構 actor 副本的指標（不含全隊模式補的空 record）。
        let mut prospective: Vec<battle::Unit> = Vec::with_capacity(party.len() + roster.units.len());
        for unit in &party {
            let mut actor = unit.clone();
            // 0x10A8A..0x10AAD writes deployment coordinates, zero pose/motion
            // and raw camp 2 after copying the persistent record.
            actor
                .materialize_native_map_presentation()
                .map_err(|e| GameError::from(e).wrap("LOADCH party native presentation: "))?;
            prospective.push(actor.clone());
            actors.push(actor);
        }
        if state.all_persistent_party {
            while actors.len() < 31 {
                // 1088D對未用deployment record清零，僅寫raw+5=1。
                let mut empty = battle::Unit::default();
                (empty.group, empty.native_record_byte5) = (255, 1);
                empty.has_native_record_byte5 = true;
                empty.has_native_record_byte6 = true;
                empty.has_native_record_byte8 = true;
                empty.has_native_map_presentation = true;
                empty.has_map_selector_key = true;
                empty.has_map_selector_slot = true;
                empty.has_battle_fig = true;
                empty.has_native_record_race = true;
                empty.has_native_record_class = true;
                actors.push(empty);
            }
        }
        // Go 的 roster.Units 元素可為 nil（`State.Load` 不會產生）；Rust 的 `Vec<Unit>` 沒有空洞。
        for u in &roster.units {
            let mut actor = u.clone();
            (actor.off_x, actor.off_y) = (0.0, 0.0);
            actor.on_field = false;
            source_roster.push(actor.clone());
            if actor.group == 0 {
                // 0x10B3C calls 0x10B4E(0) with [0x53AFA]=0. Each
                // constructor sees the party and all preceding group0 records.
                // Evidence and READY contract: fd2_ch20_initial_placement_20261003.json.
                let row = source_roster.len() - 1;
                if !actor.has_native_position_record {
                    return Err(game_err!("LOADCH group0 row {row} lacks native position record"));
                }
                let cell = battle::native_future_group_placement(
                    map_w,
                    map_h,
                    &composition,
                    &prospective,
                    actor.native_position_record,
                    0,
                )
                .map_err(|e| GameError::from(e).wrap(format!("LOADCH group0 row {row} placement: ")))?;
                if !actor.set_map_placement(cell.x, cell.y, 0) {
                    return Err(game_err!("LOADCH group0 row {row} invalid placement"));
                }
                actor
                    .materialize_native_map_presentation()
                    .map_err(|e| GameError::from(e).wrap(format!("LOADCH group0 row {row} presentation: ")))?;
                actor.on_field = true;
                prospective.push(actor.clone());
                actors.push(actor);
            }
        }
        (self.st, self.sel, self.sc) = (None, None, None);
        (self.story_actors, self.story_roster) = (actors, source_roster);
        self.story_spawned = BTreeMap::from([(0, true)]);
        // LOADCH has constructed these runtime records before any hard-coded
        // 0x22253 coordinate writer can consume them. Scenario/FDFIELD JSON keeps
        // the raw coordinates but is not itself a live runtime record, so publish
        // the verified +0/+1/+3/+4 presentation carrier now. Persistent actors that
        // already carry it retain their exact pose/motion state.
        for (i, actor) in self.story_actors.iter_mut().enumerate() {
            if actor.has_native_map_presentation {
                continue;
            }
            actor
                .materialize_native_map_presentation()
                .map_err(|e| GameError::from(e).wrap(format!("story actor {i} native presentation: ")))?;
        }
        self.story_composition_event_bytes = composition;
        self.story_roster_path = state.roster.clone();
        self.story_party_scenario = state.party_scenario.clone();
        self.story_walks = Vec::new();
        self.story_bg = true;
        (self.walk_first, self.follow_walk) = (false, false);
        self.cam_max_y = state.cam_max_y as f64;
        (self.cam_x, self.cam_y) = (state.cam_x as f64, state.cam_y as f64);
        // 0x205DA/0x1088D resets the absolute cursor globals together with the
        // scene-only native view.  Keeping the generic renderer cursor from the
        // preceding battle makes the first 0x12CEA focus start from stale bounds.
        (self.cur_x, self.cur_y) = (0, 0);
        if state.post_load_view.is_some() {
            (self.cam_x, self.cam_y) =
                ((load_ch_view.camera_x * tile_w) as f64, (load_ch_view.camera_y * tile_h) as f64);
            (self.cur_x, self.cur_y) = (load_ch_view.cursor_x, load_ch_view.cursor_y);
        }
        self.story_native_map_view = load_ch_view;
        self.has_story_native_map_view = true;
        self.story_native_map_source = Some(roster);
        self.story_native_map_state = None;
        self.camp_lines = lines;
        Ok(())
    }
}

/// Go：`main.go:3305`，`func equalIntOrder(a, b []int) bool`
pub(crate) fn equal_int_order(a: &[i64], b: &[i64]) -> bool {
    a == b
}

/// dlgPageCount 該句對白的FFFD總頁數；FFFE只形成頁內邏輯列與三列窗口捲動。
///
/// Go：`main.go:3356`，`func dlgPageCount(dl battle.DialogLine) int`
///
/// 舊式對白與繪製端共用三列分頁。
pub(crate) fn dlg_page_count(dl: &battle::DialogLine) -> i64 {
    if let Some(native) = dl.native_dialogue.as_ref() {
        if !native.pages.is_empty() {
            return native.pages.len() as i64;
        }
        return 1;
    }
    legacy_dialogue_lines(dl).len().div_ceil(3) as i64
}

impl Game {
    /// dlgAdvance 處理 Enter:目前句還有下一頁就翻頁(回 false,不換句);翻完(或本就單頁)就 pop
    /// 到下一句、頁碼歸零(回 true=已換句)。修長對白被截斷丟棄的 bug(使用者回饋 2026-07-05)。
    ///
    /// Go：`main.go:3372`，`func (g *Game) dlgAdvance() bool`
    pub(crate) fn dlg_advance(&mut self) -> bool {
        if self.dlg_scroll_t > 0 {
            // 捲動尚未完成時，Enter 不得跳過下一頁
            return false;
        }
        if self.dialog.last().is_some_and(|d| d.native_dialogue.is_some()) {
            let page = self.dlg_page;
            if page < 0
                || page >= self.native_dialogue_progressive.len() as i64
                || self.native_dialogue_progressive[page as usize].is_empty()
            {
                // 已宣告原生 caller 卻缺正式幀時，不得退回可玩但未證實的換頁路徑。
                return false;
            }
            if self.native_dialogue_progress < self.native_dialogue_progressive[page as usize].len() as i64 - 1 {
                // 原版 0x15F84 尚在逐字寫入時不會回到外層輸入 owner。
                return false;
            }
        }
        if let Some(last) = self.dialog.last()
            && self.dlg_page + 1 < dlg_page_count(last)
        {
            let native = last.native_dialogue.is_some();
            self.dlg_scroll_from = self.dlg_page;
            self.dlg_page += 1;
            self.native_dialogue_progress = -1;
            self.native_dialogue_speaking_frame = 0;
            self.reset_native_story_dialogue_mouth();
            self.dlg_scroll_t = DLG_SCROLL_FRAMES;
            if self.locale_id == "zh-Hant" && native {
                // 原生後頁本身已含三行窗口捲動，不再先插入空頁翻頁延遲。
                self.dlg_scroll_t = 0;
            }
            return false;
        }
        self.dialog.pop();
        self.dlg_page = 0;
        self.dlg_scroll_t = 0;
        self.dlg_scroll_from = 0;
        self.native_dialogue_progress = -1;
        self.reset_native_story_dialogue_mouth();
        true
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
        let show = !self.dialog.is_empty() && !(self.story_bg && self.walk_first && !self.story_walks.is_empty());
        if !show {
            (self.dlg_shown, self.dlg_phase, self.dlg_t) = (DLG_NONE, 0, 0);
            return;
        }
        let last = self.dialog.last().unwrap();
        let (cur, upper) = (last.speaker, last.upper);
        match self.dlg_phase {
            0 => {
                if self.dlg_shown == DLG_NONE {
                    // 首句:直接展開
                    (self.dlg_shown, self.dlg_phase, self.dlg_t) = (cur, 2, 0);
                    self.dlg_upper = upper;
                } else if cur != self.dlg_shown {
                    // 換人:先收合
                    (self.dlg_phase, self.dlg_t) = (1, 0);
                }
            }
            1 => {
                self.dlg_t += 1;
                if self.dlg_t >= DLG_ANIM_FRAMES {
                    (self.dlg_shown, self.dlg_phase, self.dlg_t) = (cur, 2, 0);
                    self.dlg_upper = upper;
                }
            }
            2 => {
                self.dlg_t += 1;
                if self.dlg_t >= DLG_ANIM_FRAMES {
                    (self.dlg_phase, self.dlg_t) = (0, 0);
                }
            }
            _ => {}
        }
    }

    /// terrainAt 回傳某格的地形索引(戰鬥背景用;越界回 -1)。
    ///
    /// Go：`main.go:3447`，`func (g *Game) terrainAt(x, y int) int`
    pub(crate) fn terrain_at(&self, x: i64, y: i64) -> i64 {
        let Some(m) = self.m.as_ref() else {
            return -1;
        };
        if x < 0 || y < 0 || x >= m.w || y >= m.h {
            return -1;
        }
        m.tiles[(y * m.w + x) as usize]
    }

    /// enterNode 進入 camp 目前節點:story→掛對白、battle→重開戰場、inventory_gate→依角色物品欄自動分支、
    /// event→套旗標直通、town/preparation/choice/ending→等輸入。
    ///
    /// Go：`main.go:3458`，`func (g *Game) enterNode()`
    ///
    /// Go 的 `n := g.camp.Node()` 是指向戰役圖的指標；Rust 複製一份節點，避免與 `&mut self` 同時借用。
    /// 第一章沒有的節點型別（物品閘門、合成、原生結局）呼叫的 `r8` 函式停在 `todo!("R8: …")`。
    pub(crate) fn enter_node(&mut self) {
        /// Go：`main.go:7137`，`func defaultChapterStoryScript(nodeID string) string`（`r8`；enterNode
        /// 的 story 節點需要它，隨呼叫端補移植）。
        fn default_chapter_story_script(node_id: &str) -> String {
            if !node_id.starts_with("story_ch") || node_id.len() != "story_ch00".len() {
                return String::new();
            }
            match node_id["story_ch".len()..].parse::<i64>() {
                Ok(chapter) if (1..=33).contains(&chapter) => format!("assets/story/ch{chapter:02}.json"),
                _ => String::new(),
            }
        }
        if self.camp.is_none() {
            return;
        }
        self.capture_native_map_hud_persistence();
        // 第27章同一原生handler拆成可編輯節點後，故事視圖仍是目前共享globals。
        // READY契約見fd2_ch27_flow_20261016.json node_view_handoff。
        if self.story_bg && self.has_story_native_map_view {
            let view = self.story_native_map_view;
            if let Some(st) = self.st.as_mut() {
                (st.native_map_view_state, st.has_native_map_view_state) = (view, true);
            }
        }
        // 原版故事 renderer 與戰場共用 [0x53AB9]/[0x53ABD] 等視圖全域；在戰果
        // handler 清除 battle.State 前保存該 raw carrier，只有明示 story_viewport
        // 的 runtime_context 才能領回，避免一般故事節點繼承陳舊視圖。
        self.has_handler_inherited_map_view = self.st.as_ref().is_some_and(|st| st.has_native_map_view_state);
        self.handler_inherited_map_view = if self.has_handler_inherited_map_view {
            self.st.as_ref().unwrap().native_map_view_state
        } else {
            battle::NativeMapViewState::default()
        };
        let node_id = self.camp.as_ref().unwrap().node_id();
        let node = self.camp.as_ref().unwrap().node().cloned();
        if node_id == "battle_ch24"
            && self.has_story_native_map_view
            && node
                .as_ref()
                .is_some_and(|n| self.story_roster_path == n.units && self.story_party_scenario == n.scenario)
        {
            self.handler_inherited_map_view = self.story_native_map_view;
            self.has_handler_inherited_map_view = true;
        }
        self.reset_action_overlay_lifecycle();
        self.native_ending = None;
        self.ending_notice.clear();
        let Some(n) = node else {
            return; // 流程結束(game over)
        };
        let stage_handoff = node_id == "battle_ch24"
            && self.native_map_assets.as_ref().is_some_and(|a| a.map_index == 23)
            && self.has_story_native_map_view
            && self.story_roster_path == n.units
            && self.story_party_scenario == n.scenario;
        // Go 以 nil 切片表示「沒有交接」；Rust 用 Option。
        let mut story_handoff: Option<(Vec<u8>, Vec<u8>)> = None;
        if self.native_ch12_story_work_handoff(&n.units, &n.scenario) {
            story_handoff = Some((self.native_map_work.clone(), self.native_map_vga.clone()));
        }
        // campaign 結局尚無已證實的場景→曲目對映；空白 ending BGM 先停止前一場景，
        // 避免戰鬥音樂漏到終局頁。若資料明確填入 BGM，仍保留可編輯的曲目入口，
        // 待證據閉合後即可使用。
        if n.r#type == "ending" && n.bgm.is_empty() {
            self.stop_bgm();
        } else {
            self.play_bgm(&n.bgm);
        }
        self.story_bg = false; // 預設離開場景背景模式;story+Map 節點下面再開回
        self.story_walks = Vec::new();
        self.story_auto_advance = 0;
        (self.walk_first, self.follow_walk, self.cam_max_y) = (false, false, 0.0);
        self.story_native_map_view = battle::NativeMapViewState::default();
        self.has_story_native_map_view = false;
        self.story_native_map_source = None;
        self.story_native_map_state = None;
        self.cam_pan = None;
        self.focus_job = None;
        self.act_job = None;
        self.beats = Vec::new();
        self.beat_idx = -1;
        self.beat_delay = 0;
        self.transition_reveal = None;
        self.indexed_transition = None;
        self.native_ch20_sky_key = None;
        if node_id != "postbattle_ch24_persist" && !stage_handoff {
            self.native_ch23_state = None;
        }
        self.native_ch23_loop = None;
        self.native2189_a = None;
        self.native_unit_present = None;
        self.native_ch28_post_present = None;
        self.native_ch22_reload = None;
        self.spawn_intro_transition = None;
        self.native_turn_staging = None;
        self.native_full_dac_white = false;
        self.native_full_dac_black = false;
        self.handler_resource = 0;
        (self.battle_event, self.battle_event_delay) = (None, 0);
        (self.native_chapter_result, self.native_result_matched_rules) = (None, Vec::new());
        (self.dlg_shown, self.dlg_phase, self.dlg_t) = (DLG_NONE, 0, 0);
        self.dlg_upper = None;
        (self.dlg_scroll_t, self.dlg_scroll_from) = (0, 0);
        match n.r#type.as_str() {
            "story" | "cutscene" => {
                // cutscene(doc50):同一套場景設置,進行中改由 Beats 驅動(見下)
                self.dialog = Vec::new();
                self.dlg_page = 0; // 新對白從第一頁起
                let mut lines = n.lines.clone();
                let mut script = n.script.clone();
                if script.is_empty() && n.r#type == "story" {
                    script = default_chapter_story_script(&node_id);
                }
                if !script.is_empty() {
                    // 正式劇情必須與 editor canonical line_id 完整對齊
                    match load_story_script_with_identity_at(&script, &n.scene, None) {
                        Ok(ls) => lines = ls,
                        Err(err) => {
                            self.load_err = format!("story script: {}", err.message());
                            return;
                        }
                    }
                }
                // cutscene dialog beat 依 Line/Count 取子段;story 節點也存一份備用
                self.camp_lines = lines.clone();
                if n.r#type == "story" {
                    for line in lines.iter().rev() {
                        // 反序堆疊:顯示取末端,Enter 逐句 pop
                        match self.localized_story_text(line) {
                            Ok(text) => self.dialog.push(battle::DialogLine {
                                speaker: line.speaker,
                                text,
                                ..Default::default()
                            }),
                            Err(err) => {
                                self.dialog = Vec::new();
                                self.load_err = format!("story locale: {}", err.message());
                                return;
                            }
                        }
                    }
                }
                if !n.map.is_empty() {
                    // 場景背景圖(doc23 §4:序幕王城/草地= FDFIELD map32 複合場景,非戰場地圖疊對白)
                    if let Err(err) = self.load_map(&n.map) {
                        self.load_err = format!("map: {}", err.message());
                    }
                    (self.st, self.sel) = (None, None); // 清殘留單位/選取(避免上一戰場畫面疊在新背景上)
                    self.story_bg = true;
                    (self.walk_first, self.follow_walk) = (n.walk_first, n.follow_walk);
                    self.cam_max_y = n.cam_max_y as f64;
                    (self.cam_x, self.cam_y) = (n.cam_x as f64, n.cam_y as f64);
                    self.story_actors = Vec::new();
                    (self.story_roster, self.story_spawned) = (Vec::new(), BTreeMap::new());
                    (self.story_roster_path, self.story_party_scenario) = (String::new(), String::new());
                    for a in &n.actors {
                        // cutscene 靜態擺位(國王/王后/主角等),無 AI/戰鬥邏輯
                        let mut u = battle::Unit::default();
                        (u.fig, u.x, u.y, u.dir) = (a.fig, a.x, a.y, a.dir);
                        u.on_field = true;
                        self.story_actors.push(u);
                        if a.walk_frames > 0 && (a.from_x != a.x || a.from_y != a.y) {
                            // 進場走位(doc46 §5.3)
                            let idx = self.story_actors.len() as i64 - 1;
                            self.story_walks.push(StoryWalkJob {
                                actor: idx,
                                from_x: a.from_x,
                                from_y: a.from_y,
                                to_x: a.x,
                                to_y: a.y,
                                frames: a.walk_frames,
                                final_dir: a.dir, // 進場走完面向 actor 宣告的 dir(如 Ares 面向索爾)
                                ..Default::default()
                            });
                        }
                    }
                } else {
                    self.story_actors = Vec::new();
                    (self.story_roster, self.story_spawned) = (Vec::new(), BTreeMap::new());
                    (self.story_roster_path, self.story_party_scenario) = (String::new(), String::new());
                }
                if self.cutscene_log {
                    // FD2_CUTSCENE_LOG:進場印節點 + 每個 actor(idx/名/座標/dir)
                    let cur = self.camp.as_ref().unwrap().cur.clone();
                    eprintln!(
                        "[cutscene] === node {} map={} cam=({},{}) ===",
                        battle::go_quote(&cur),
                        n.map,
                        n.cam_x,
                        n.cam_y
                    );
                    for (i, a) in self.story_actors.iter().enumerate() {
                        eprintln!("[cutscene]   actor[{i}] {} ({},{}) dir={}", fig_name(a.fig), a.x, a.y, a.dir);
                    }
                }
                if n.r#type == "cutscene" {
                    self.beats = n.beats.clone();
                    if !n.handler_binding.is_empty() {
                        match campaign::compile_handler_binding(&asset_path(&n.handler_binding)) {
                            Ok((beats, issues)) if issues.is_empty() => self.beats = beats,
                            result => {
                                let (err, issues) = match result {
                                    Ok((_, issues)) => (String::from("<nil>"), issues.len()),
                                    Err(err) => (err.message().to_string(), 0),
                                };
                                self.load_err = format!(
                                    "handler binding {} unresolved: {err} issues={issues}",
                                    battle::go_quote(&n.handler_binding)
                                );
                                return; // fail closed: never replace authored beats with a partial handler
                            }
                        }
                    }
                    match campaign::expand_native_dialogue_groups(&self.beats) {
                        Ok(expanded) => self.beats = expanded,
                        Err(err) => {
                            self.load_err =
                                format!("cutscene {} 原生對話群組無效: {}", battle::go_quote(&node_id), err.message());
                            return;
                        }
                    }
                    // An unbound postbattle node must never be treated as an empty
                    // interlude: doing so would silently skip persistent sync/rewards and
                    // jump straight to town. Keep the authored graph cursor in place until
                    // an active handler binding is supplied.
                    if self.beats.is_empty() && n.handler_binding.is_empty() && node_id.starts_with("postbattle_") {
                        if self.approximate_mode {
                            // 此節點的原版 handler 仍未知。明確啟用的近似模式只保留玩家可見的
                            // 戰役邊界，不捏造 JOIN／獎勵語意：只同步已物化的戰場隊伍，等待
                            // Enter 後才沿 authored Next 邊進入城鎮／整備。
                            if let Err(err) = self.sync_party_from_battle() {
                                self.load_err = format!(
                                    "approximate postbattle {}: party sync unavailable: {}",
                                    battle::go_quote(&node_id),
                                    err.message()
                                );
                                return;
                            }
                            self.approximate_postbattle = true;
                            // 這是未綁定原版 handler 時才允許的後備流程，只維持 authored
                            // 戰役邊界；不得用來宣稱該章戰後 handler 或一般玩家路徑已達 E2。
                            self.msg = String::from("戰後整理（近似模式；原版 handler 尚未證實）按 Enter 繼續");
                            return;
                        }
                        self.load_err =
                            format!("postbattle node {} has no active handler binding", battle::go_quote(&node_id));
                        self.msg = String::from("戰後 handler 尚未接線，流程已停止");
                        return;
                    }
                    self.beat_advance(); // beatIdx -1 → 0,啟動第一拍(doc50 BeatRunner)
                } else if lines.is_empty() && n.auto_advance > 0 {
                    // 無對白節點(行軍蒙太奇):進場後自動倒數轉場
                    self.story_auto_advance = n.auto_advance;
                }
            }
            "battle" => {
                if !n.map.is_empty() {
                    // 指定戰場(assets/maps/mapN;全 33 圖已匯出)
                    if let Err(err) = self.load_map(&n.map) {
                        self.load_err = format!("map: {}", err.message());
                    }
                }
                self.reset_battle(&n.units, &n.scenario);
                self.apply_field_own_attack_ranges();
                if !self.materialize_native_map_runtime(Some(&n)) {
                    return;
                }
                if let Some((work, vga)) = story_handoff {
                    (self.native_map_work, self.native_map_vga) = (work, vga);
                }
            }
            "inventory_gate" => {
                let Some(item_id) = n.item_id else {
                    // Load 已拒絕；保留 runtime fail-closed 防線給手工測試 Campaign。
                    self.load_err = String::from("inventory_gate: missing item_id");
                    return;
                };
                let outcome = if self.party_has_item_id(item_id) { "present" } else { "missing" };
                self.camp.as_mut().unwrap().advance(outcome);
                self.enter_node();
            }
            "inventory_recipe" => {
                let crafted = match self.apply_inventory_recipe(&n) {
                    Ok(crafted) => crafted,
                    Err(err) => {
                        self.load_err = format!("inventory_recipe: {err}");
                        return;
                    }
                };
                self.camp.as_mut().unwrap().advance(if crafted { "crafted" } else { "insufficient" });
                self.enter_node();
            }
            "event" => {
                self.camp.as_mut().unwrap().advance("");
                self.enter_node();
            }
            "choice" | "town" => {
                // 戰間 hub 不可殘留上一戰的單位或勝利對白
                (self.dialog, self.st, self.sel) = (Vec::new(), None, None);
                // 城鎮 hub 從自己的子場景（酒店／商店／教會／出口提示答 NO）回來時，
                // 原版在 0x2CAD7 內重複 hub 迴圈、不重設 [0x5412B]：第四章原版收據
                // （docs/data/ui-traces/parity-ch04.json）出口答 NO 之後 left 落在道具店、
                // 教會之後 left 落在酒店。只有從外面（戰後劇情／讀檔）進城鎮才歸零。
                if n.r#type != "town" || !self.native_town_hub_return {
                    self.camp_sel = 0;
                }
                self.native_town_hub_return = false;
                if n.r#type == "town" {
                    self.reset_native_town_ui_pulse();
                }
            }
            "preparation" | "church" => {
                (self.dialog, self.st, self.sel) = (Vec::new(), None, None);
                // 節點邊界 UI；preparation 可在此安全 F5 存檔，Enter 才進下一章 pre handler。
                if n.r#type == "preparation" {
                    self.setup_preparation(Some(&n));
                } else {
                    self.setup_church();
                }
            }
            "hotel" => {
                (self.dialog, self.st, self.sel) = (Vec::new(), None, None);
                self.hotel_sel = 0;
                self.hotel_route = fdother::NativeHotelServiceRoute::default();
                self.hotel_has_route = false;
                self.native_hotel_mode.clear();
                self.setup_native_hotel();
            }
            "shop" => {
                (self.dialog, self.st, self.sel) = (Vec::new(), None, None);
                self.shop_sel = 0;
                self.shop_picking = false;
                self.shop_equip_prompt = false;
                self.shop_mode = String::from("buy");
                self.shop_sell_picking = false;
                self.shop_sell_unit_sel = 0;
                self.shop_sell_slot_sel = 0;
                self.shop_recipient_sel = 0;
                self.shop_recipients = Vec::new();
                self.native_shop_ui_job = None;
                self.native_shop_variant = 0;
                self.native_shop_mode.clear();
                self.native_shop_service_sel = 0;
                self.native_shop_item_start = 0;
                self.native_shop_equip_roster_top = 0;
                self.native_shop_equip_unit_sel = 0;
                self.native_shop_transfer_source = -1;
                self.native_shop_transfer_item = -1;
                self.native_shop_transfer_items = Vec::new();
                self.native_shop_transfer_dest = -1;
                self.native_shop_transfer_ids = Vec::new();
                self.native_shop_transfer_sel = 0;
                self.native_shop_transfer_top = 0;
                self.clear_native_item_panel();
                self.setup_native_shop();
            }
            "ending" => {
                (self.dialog, self.st, self.sel) = (Vec::new(), None, None);
                if let Some(prefix) = n.native_ending_prefix.clone()
                    && let Err(err) = self.start_campaign_native_ending(&prefix)
                {
                    // 原始結局資源是玩家自備且不隨專案散布；載入失敗時保留可編輯結語，不發布部分原生演出。
                    self.ending_notice = format!("原始結局素材不足，顯示可編輯結語：{err}");
                }
            }
            _ => {}
        }
    }

    /// Go：`main.go:3749`，`func (g *Game) materializeNativeMapRuntime(n *campaign.Node) bool`
    pub(crate) fn materialize_native_map_runtime(&mut self, n: Option<&campaign::Node>) -> bool {
        let Some(n) = n else {
            return true;
        };
        if n.native_map_view.is_none() && n.native_map_hud.is_none() && n.native_map_hud_inherited.is_none() {
            return true;
        }
        let Some((st_w, st_h)) = self.st.as_ref().map(|st| (st.w, st.h)) else {
            self.load_err = String::from("native map runtime: battle state is unavailable");
            return false;
        };
        let Some(view) = n.native_map_view else {
            self.load_err = String::from("native map runtime: HUD lacks a sourced view");
            return false;
        };
        // Preflight into a narrow candidate so an invalid optional HUD cannot leave
        // a published view/range half-state. A sourced view may stand alone: ch27's
        // handler closes the six view globals, while gate A remains inherited from
        // a separate persistent option and must not be fabricated.
        let mut candidate = battle::State::default();
        (candidate.w, candidate.h) = (st_w, st_h);
        // 節點常數走它自己的入口契約（進場即繪的靜止視圖），不是執行期狀態的
        // 那一套；兩者的差別寫在 campaign.NativeMapViewConfig.Validate。
        if let Err(err) = view.validate(st_w, st_h) {
            self.load_err = format!("native map runtime view: {}", err.message());
            return false;
        }
        let mut entry_view = battle::NativeMapViewState {
            camera_x: view.camera_x,
            camera_y: view.camera_y,
            cursor_x: view.cursor_x,
            cursor_y: view.cursor_y,
            visible_cursor_x: view.visible_cursor_x,
            visible_cursor_y: view.visible_cursor_y,
        };
        if self.camp.as_ref().is_some_and(|c| c.node_id() == "battle_ch24") && self.has_handler_inherited_map_view {
            entry_view = self.handler_inherited_map_view;
        }
        if let Err(err) = candidate.materialize_native_map_view_state(entry_view) {
            self.load_err = format!("native map runtime view: {}", err.message());
            return false;
        }
        match view.range_mode {
            Some(mode) if (mode == 0 || mode == 1) && candidate.materialize_native_map_range_mode(mode) => {}
            _ => {
                self.load_err = String::from("native map campaign selector is not a verified entry value");
                return false;
            }
        }
        if let Some(hud) = n.native_map_hud.as_ref() {
            if !(0..=0xff).contains(&hud.display_gate_a)
                || !(0..=0xff).contains(&hud.display_gate_b)
                || !candidate.materialize_native_map_hud_state(
                    hud.display_gate_a as u8,
                    hud.display_gate_b as u8,
                    hud.anchor_x,
                )
            {
                self.load_err = String::from("native map runtime HUD is outside raw bounds");
                return false;
            }
        } else if let Some(inherited) = n.native_map_hud_inherited.as_ref() {
            let mut persistent_source = self.native_map_hud_persistent;
            // A wholly uninitialized Game still represents a fresh original process,
            // whose data image seeds gate A and anchor to 1. A partial state is not a
            // fresh process and remains rejected rather than filling one missing field.
            if !persistent_source.has_display_gate_a && !persistent_source.has_anchor_x {
                persistent_source = battle::initial_native_map_hud_persistent_state();
            }
            // Go `byte(inherited.DisplayGateB)`：截斷成 8 位元。
            let (hud, ok) = persistent_source.materialize_runtime(inherited.display_gate_b as u8);
            if !ok || !candidate.materialize_native_map_hud_state(hud.display_gate_a, hud.display_gate_b, hud.anchor_x)
            {
                self.load_err = String::from("native map runtime HUD persistent state is incomplete");
                return false;
            }
        }
        let mut persistent_candidate = self.native_map_hud_persistent;
        if candidate.has_native_map_hud_state
            && !persistent_candidate.capture_native_map_hud(candidate.native_map_hud_state)
        {
            self.load_err = String::from("native map runtime HUD persistent capture failed");
            return false;
        }
        let st = self.st.as_mut().unwrap();
        st.native_map_view_state = candidate.native_map_view_state;
        st.has_native_map_view_state = true;
        st.native_map_range_mode = candidate.native_map_range_mode;
        st.has_native_map_range_mode_state = true;
        st.native_map_hud_state = candidate.native_map_hud_state;
        st.has_native_map_hud_state = candidate.has_native_map_hud_state;
        self.native_map_hud_persistent = persistent_candidate;
        // 戰前 handler 把游標聚焦到記錄 0 之後，第一次重繪就走 0x1ACF3→0x1AD2A：兩個閘門都開
        // 時 anchor 依可見游標決定。第七章 battle_start（可見游標 (2,7)）的原版 checkpoint
        // 小窗已經在右側；只靠繼承的 anchor 1 會畫在左側（r3 seq 60 差 4617 px）。
        st.advance_native_map_hud_anchor(entry_view.visible_cursor_x, entry_view.visible_cursor_y);
        let hud_state = st.native_map_hud_state;
        self.native_map_hud_persistent.capture_native_map_hud(hud_state);
        self.sync_native_map_view();
        true
    }

    /// applyFieldOwnAttackRanges 讓 FDFIELD 直接登場的我方記錄（raw +6==2、不在持久名冊裡，
    /// 例如第八章 group 0 的洛娜）和 LOAD 進來的名冊用同一條射程規則：地圖單位檔沒有射程
    /// 欄位，沒有這一步就一律是 1，裝槍的單位在距離 2 的敵人前面攻擊不可選（原版 r1 seq 94
    /// 可以攻擊）。已經有射程的單位（名冊、劇本手寫值）不動。
    ///
    /// Go：`main.go:3837`，`func (g *Game) applyFieldOwnAttackRanges()`
    pub(crate) fn apply_field_own_attack_ranges(&mut self) {
        let Some(st) = self.st.as_mut() else {
            return;
        };
        let stats = &self.shop_item_stats;
        for u in st.units.iter_mut().chain(st.roster.iter_mut()) {
            if !u.has_native_record_byte6 || u.native_record_byte6 != 2 || u.atk_min != 0 || u.atk_max != 0 {
                continue;
            }
            campaign::apply_equipped_attack_range(Some(u), stats);
        }
    }

    /// captureNativeMapHUDPersistence runs before a campaign node can clear or
    /// replace the previous battle state. The original carries gate A and anchor
    /// across nodes, but not the transient gate B.
    ///
    /// Go：`main.go:3855`，`func (g *Game) captureNativeMapHUDPersistence()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn capture_native_map_hud_persistence(&mut self) {
        let Some(st) = self.st.as_ref().filter(|st| st.has_native_map_hud_state) else {
            return;
        };
        self.native_map_hud_persistent.capture_native_map_hud(st.native_map_hud_state);
    }

    /// resetBattle 重開一場戰鬥(campaign battle 節點;敗北重試也走這裡)。
    ///
    /// Go：`main.go:3863`，`func (g *Game) resetBattle(unitsPath, scnPath string)`
    pub(crate) fn reset_battle(&mut self, units_path: &str, scn_path: &str) {
        let continue_native_ch23_stage = self.native_map_assets.as_ref().is_some_and(|a| a.map_index == 23)
            && self.native_ch23_state.is_some()
            && !self.story_actors.is_empty()
            && self.story_roster_path == units_path
            && self.story_party_scenario == scn_path;
        self.cancel_native_command32_presentation();
        self.cancel_native_command33_presentation();
        self.reset_action_overlay_lifecycle();
        self.native_continue_opening_confirm = false;
        self.native_current_save_plain = Vec::new();
        self.native_system_cursor_overlay = false;
        self.native_ch20_sky_key = None;
        if !continue_native_ch23_stage {
            self.native_ch23_state = None;
            self.native_map_clock.reset();
        }
        self.native_ch23_loop = None;
        self.native2189_a = None;
        self.native_unit_present = None;
        self.native_ch28_post_present = None;
        self.native_ch22_reload = None;
        self.indexed_transition = None;
        self.spawn_intro_transition = None;
        self.native_turn_staging = None;
        self.native_full_dac_white = false;
        self.native_full_dac_black = false;
        (self.native_map_work, self.native_map_vga) = (Vec::new(), Vec::new());
        let units_path = if units_path.is_empty() { "assets/map0_units.json" } else { units_path };
        // A verified pre-handler LOADCH owns the same runtime array used by the
        // following battle. Preserve it only when both editable source paths match
        // exactly; direct/debug starts and unrelated scene maps still rebuild.
        let mut adopt_handler_state = !self.story_actors.is_empty()
            && self.story_roster_path == units_path
            && self.story_party_scenario == scn_path;
        if self.cutscene_log {
            eprintln!(
                "[cutscene] battle adopt={adopt_handler_state} actors={} roster={}/{} scenario={}/{}",
                self.story_actors.len(),
                battle::go_quote(&self.story_roster_path),
                battle::go_quote(units_path),
                battle::go_quote(&self.story_party_scenario),
                battle::go_quote(scn_path)
            );
        }
        let handler_actors = std::mem::take(&mut self.story_actors);
        let handler_roster = std::mem::take(&mut self.story_roster);
        self.story_spawned = BTreeMap::new();
        self.story_composition_event_bytes = Vec::new();
        (self.story_roster_path, self.story_party_scenario) = (String::new(), String::new());
        if let Ok(mut st) = battle::load(&asset_path(units_path)) {
            // Go 先把 g.st 指向新 state 再綁定；綁定失敗時 g.st 已是新 state。
            let bound = self
                .bind_native_future_item_rows(Some(&mut st))
                .map_err(|e| format!("native future constructor item rows: {}", e.message()))
                .and_then(|()| {
                    self.bind_native_movement_cost_rows(Some(&mut st))
                        .map_err(|e| format!("native movement rows: {}", e.message()))
                });
            if let Err(err) = bound {
                self.st = Some(st);
                self.load_err = err;
                return;
            }
            self.bind_command_learn(Some(&mut st));
            self.bind_native_command_book(Some(&mut st));
            self.bind_native_command_resistances(Some(&mut st));
            self.st = Some(st);
        }
        (self.result, self.sel, self.reach, self.moved) = (String::new(), None, BTreeMap::new(), false);
        (self.atk, self.walk, self.dialog, self.msg) = (None, None, Vec::new(), String::new());
        // scenario 空 = 無劇本(FDFIELD roster 全員照 units.json 登場;不 fallback ch01——
        // ch01 的 initial_groups/party/deploy 是 map0 專屬,錯配到他章會讓單位消失。每章 stub 見 worklist)
        self.sc = None;
        if self.st.is_none() || scn_path.is_empty() {
            return;
        }
        let Ok(mut sc) = battle::load_scenario(&asset_path(scn_path)) else {
            return;
        };
        // Scenario owns chapter-specific combat statistics, but its party list
        // is filtered by the permanent membership established by JOIN.  A
        // direct chapter/debug start has no JOIN history and therefore keeps
        // the authored scenario party intact.
        if !(adopt_handler_state && self.native_chapter_restore.is_some()) {
            filter_scenario_party(Some(&mut sc), &self.battle_party_members());
            let mut party_order = self.party_join_order.clone();
            if !self.party_deploy.is_empty() {
                party_order = self.load_ch_party_order(None);
            }
            if let Err(err) = reorder_scenario_party(Some(&mut sc), &party_order) {
                self.load_err = format!("scenario party order: {}", err.message());
                return;
            }
        }
        if adopt_handler_state && !sc.runtime_append_groups {
            // Matching asset paths alone do not prove that the following battle
            // consumes the handler's partial runtime array.  Without the
            // explicit runtime-append contract, rebuild the authored battle
            // state and keep the boundary fail-closed rather than calling
            // AdoptHandlerBattleState on an incompatible scenario.
            adopt_handler_state = false;
            if continue_native_ch23_stage {
                self.native_ch23_state = None;
                self.native_map_clock.reset();
            }
        }
        // Go 的 g.sc 與 sc 是同一個指標：先發布 g.sc，後續 SetupChecked／AdoptHandlerBattleState
        // 的改寫都落在 g.sc 上。Rust 先在區域變數改完再放回 self.sc（中間只有 bindNativeDeathPrograms
        // 讀 g.sc，所以先放回再取出）。
        self.sc = Some(sc);
        self.bind_native_death_programs();
        let mut sc = self.sc.take().unwrap();
        let mut st = self.st.take().unwrap();
        let outcome: std::result::Result<(), String> = if adopt_handler_state {
            // 只有 runtime-append scenario 可用 handler 前置陣列取代 authored state。
            // 若在讀到 scenario 合約前就取代，後續 fallback 即使關閉採用旗標，
            // handler 單位仍會殘留，普通 spawn_party 便會重複加入玩家隊伍。
            let remaining_roster = prune_handler_materialized_roster(&handler_actors, &handler_roster);
            st.units = Vec::new();
            st.roster = remaining_roster;
            st.native_map_selector_cache = None;
            st.native_map_selector_error = None;
            st.append_native_map_selector_batch(handler_actors)
                .map_err(|e| format!("handler battle roster: {}", e.message()))
                .and_then(|()| {
                    sc.adopt_handler_battle_state(Some(&mut st))
                        .map_err(|e| format!("handler battle scenario: {}", e.message()))
                })
        } else {
            match sc.setup_checked(Some(&mut st)) {
                Ok(dialogue) => {
                    self.dialog.extend(dialogue);
                    Ok(())
                }
                Err(e) => Err(format!("scenario setup: {}", e.message())),
            }
        };
        self.sc = Some(sc);
        self.st = Some(st);
        if let Err(err) = outcome {
            self.load_err = err;
            return;
        }
        let mut st = self.st.take().unwrap();
        self.initialize_equipment_bases(Some(&mut st));
        self.st = Some(st);
        self.apply_scenario_party_joins();
        let mut st = self.st.take().unwrap();
        self.apply_persistent_party(Some(&mut st));
        self.st = Some(st);
        self.focus_on_party();
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
    // Go 的 handlerRosterOriginKey{group, x, y, rawKey}（`main.go:3995`，型別不在骨架）以 tuple 表示。
    fn key(u: &battle::Unit) -> (i64, u16, u16, u16) {
        let p = &u.native_position_record;
        (u.group, p.x_word, p.y_word, p.raw_key)
    }
    let mut want: BTreeMap<(i64, u16, u16, u16), i64> = BTreeMap::new();
    for actor in actors.iter().filter(|a| a.has_native_position_record) {
        *want.entry(key(actor)).or_insert(0) += 1;
    }
    let mut remaining = Vec::with_capacity(roster.len());
    for row in roster {
        if row.has_native_position_record
            && let Some(n) = want.get_mut(&key(row)).filter(|n| **n > 0)
        {
            *n -= 1;
            continue;
        }
        remaining.push(row.clone());
    }
    // A SPAWN beat consumes its FDFIELD source rows before this boundary, so a
    // materialized actor may legitimately have no row left to remove. Exact
    // provenance matches are removed when present; unmatched rows stay pending.
    remaining
}

impl Game {
    /// bindNativeCommandBook gives each freshly loaded battle the immutable raw
    /// ABI table.  It never derives records from the legacy SpellBook.
    ///
    /// Go：`main.go:4039`，`func (g *Game) bindNativeCommandBook(st *battle.State)`
    pub(crate) fn bind_native_command_book(&self, st: Option<&mut battle::State>) {
        if let Some(st) = st {
            st.native_command_book = self.native_command_book.clone();
        }
    }

    /// bindNativeMovementCostRows 將有機會進入 AI runner 的每個 battle state 綁定
    /// 版本化 0x4e555 表。缺少或格式錯誤時，在 mode 2 窄切片嘗試以正規化移動成本
    /// 取代原始成本前先停止。
    ///
    /// Go：`main.go:4049`，`func (g *Game) bindNativeMovementCostRows(st *battle.State) error`
    pub(crate) fn bind_native_movement_cost_rows(&self, st: Option<&mut battle::State>) -> Result<()> {
        let Some(st) = st else {
            return Err(GameError::new("native movement rows: nil battle state"));
        };
        let rows = battle::load_native_movement_cost_rows(&asset_path("assets/data/native_movement_cost_rows.json"))?;
        Ok(st.bind_native_movement_cost_rows(&rows)?)
    }

    /// Go：`main.go:4062`，`func (g *Game) bindNativeCommandResistances(st *battle.State)`
    pub(crate) fn bind_native_command_resistances(&self, st: Option<&mut battle::State>) {
        let Some(st) = st else {
            return;
        };
        if self.native_command_resistances.is_empty() {
            return;
        }
        st.native_command_resistances = self.native_command_resistances.clone();
    }

    /// bindCommandLearn makes every newly loaded battle state use the same
    /// explicit editable export. A missing table leaves the state fail-closed;
    /// it never falls back to the legacy normalized Spells list.
    ///
    /// Go：`main.go:4075`，`func (g *Game) bindCommandLearn(st *battle.State)`
    ///
    /// Go 以 nil map 表示「沒有載入」；Rust 的 `Game` 欄位是非 `Option` 的 map，空 map 視同 nil
    /// （正式匯出檔不會是空表）。`growthRollObserver` 是測試觀察者，與 `State::native_growth_roll_observer`
    /// 同型（R4 階段 3），照 Go 交給戰場狀態。
    pub(crate) fn bind_command_learn(&self, st: Option<&mut battle::State>) {
        let Some(st) = st else {
            return;
        };
        st.balance = self.balance.clone();
        if !self.command_learn.is_empty() && !self.command_learn_selectors.is_empty() {
            st.command_learn = Some(self.command_learn.clone());
            st.command_learn_selectors = Some(self.command_learn_selectors.clone());
        }
        if !self.native_growth_rows.is_empty() {
            st.native_growth_rows = Some(self.native_growth_rows.clone());
        }
        // R4 階段 3：`growth_roll_observer` 改成與 `State::native_growth_roll_observer` 同型，照 Go 直接交給戰場狀態。
        st.native_growth_roll_observer = self.growth_roll_observer.clone();
    }

    /// applyPersistentParty overlays the post-battle roster snapshot on freshly
    /// materialized player units while preserving this scenario's deployment,
    /// camp/group and on-field state. Original 0x11506 matches on charID; remake
    /// player Fig is the same stable 0..31 identity used by JOIN.
    ///
    /// Go：`main.go:4096`，`func (g *Game) applyPersistentParty(st *battle.State)`
    pub(crate) fn apply_persistent_party(&self, st: Option<&mut battle::State>) {
        let Some(st) = st else {
            return;
        };
        if self.party_roster.is_empty() {
            return;
        }
        for dst in st.units.iter_mut().filter(|u| u.camp == battle::OWN) {
            if let Some(src) = self.party_roster.get(&dst.fig) {
                apply_persistent_stats(Some(dst), Some(src));
            }
        }
    }

    /// Go：`main.go:4110`，`func (g *Game) initializeEquipmentBases(st *battle.State)`
    /// Go `g.shopItemStats == nil` 以空 map 判斷（`loadGame` 載入失敗時留 nil）。
    pub(crate) fn initialize_equipment_bases(&self, st: Option<&mut battle::State>) {
        let Some(st) = st else {
            return;
        };
        if self.shop_item_stats.is_empty() {
            return;
        }
        for u in st.units.iter_mut().filter(|u| u.camp == battle::OWN) {
            campaign::initialize_equipment_base(Some(u), &self.shop_item_stats);
        }
    }

    /// seedPersistentPartyFromLoadCH materializes the remake counterpart of the
    /// persistent records already created by preceding native JOIN calls. Existing
    /// records always win, preserving battle/shop/class-change progress. This is a
    /// normal-campaign bridge, not FD2.SAV byte compatibility.
    ///
    /// Go：`main.go:4125`，`func (g *Game) seedPersistentPartyFromLoadCH( order []int, units []*battle.Unit, ) error`
    /// Go 的 `units` 是 LOADCH 剛建好、尚未進 `g.st` 的單位，Rust 收單位本體（簽名變更見 `PORTING.md`）。
    pub(crate) fn seed_persistent_party_from_load_ch(&mut self, order: &[i64], units: &[battle::Unit]) -> Result<()> {
        if order.is_empty() || units.len() != order.len() {
            return Err(game_err!("LOADCH party/order length mismatch: units={} order={}", units.len(), order.len()));
        }
        for (i, &id) in order.iter().enumerate() {
            let unit = &units[i];
            if unit.fig != id || !self.party_members.get(&id).copied().unwrap_or(false) {
                return Err(game_err!("LOADCH party slot {i} does not match joined identity {id}"));
            }
            if self.party_roster.contains_key(&id) {
                continue;
            }
            let materialized = self.materialize_native_join_persistent_unit(id, unit)?;
            self.party_roster.insert(id, clone_native_shop_unit(&materialized));
        }
        Ok(())
    }

    /// Go：`main.go:4158`，`func (g *Game) materializeNativeJoinPersistentUnit(id int, base battle.Unit) (battle.Unit, error)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn materialize_native_join_persistent_unit(
        &mut self,
        id: i64,
        base: &battle::Unit,
    ) -> Result<battle::Unit> {
        if !self.has_native_join_constructor {
            let table =
                campaign::load_native_join_constructor_table(&asset_path("assets/data/native_join_constructor.json"))
                    .map_err(|e| GameError::from(e).wrap("native JOIN constructor table: "))?;
            self.native_join_constructor = table;
            self.has_native_join_constructor = true;
        }
        if self.native_join_item_effect_rows.is_empty() {
            let rows =
                battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json"))
                    .map_err(|e| GameError::from(e).wrap("native JOIN item effect rows: "))?;
            self.native_join_item_effect_rows = rows;
        }
        Ok(self.native_join_constructor.materialize_persistent_unit_on(
            self.native_join_residual_record(),
            id,
            base.clone(),
            &self.native_join_item_effect_rows,
        )?)
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
        let Some(baseline) = self.native_chapter_slot_baseline.as_ref() else {
            return fdsave::PersistentRecord::default();
        };
        let count = baseline.verified.roster_count as i64;
        if count < 0 || count > baseline.records.len() as i64 {
            return fdsave::PersistentRecord::default();
        }
        let mut known: BTreeSet<i64> = BTreeSet::new();
        for record in &baseline.records[..count as usize] {
            known.insert(record.raw[8] as i64);
        }
        let mut index = count;
        for id in &self.party_join_order {
            let Some(unit) = self.party_roster.get(id) else {
                continue;
            };
            if !unit.has_native_identity || known.contains(&unit.native_identity) {
                continue;
            }
            known.insert(unit.native_identity);
            index += 1;
        }
        if index >= baseline.records.len() as i64 {
            return fdsave::PersistentRecord::default();
        }
        baseline.records[index as usize]
    }
}

/// Go：`main.go:4217`，`func applyPersistentStats(dst, src *battle.Unit)`
pub(crate) fn apply_persistent_stats(dst: Option<&mut battle::Unit>, src: Option<&battle::Unit>) {
    let (Some(dst), Some(src)) = (dst, src) else {
        return;
    };
    (dst.name, dst.cls_name, dst.class_id, dst.lv) = (src.name.clone(), src.cls_name.clone(), src.class_id, src.lv);
    (dst.hp, dst.max_hp, dst.mp, dst.max_mp) = (src.hp, src.max_hp, src.mp, src.max_mp);
    (dst.ap, dst.dp, dst.dx) = (src.ap, src.dp, src.dx);
    (dst.hit, dst.ev, dst.crit_pct, dst.mv) = (src.hit, src.ev, src.crit_pct, src.mv);
    (dst.atk_min, dst.atk_max) = (src.atk_min, src.atk_max);
    (dst.base_ap, dst.base_dp, dst.base_hit, dst.base_ev, dst.base_mv) =
        (src.base_ap, src.base_dp, src.base_hit, src.base_ev, src.base_mv);
    (dst.base_atk_min, dst.base_atk_max, dst.equipment_base_set) =
        (src.base_atk_min, src.base_atk_max, src.equipment_base_set);
    // Portrait 與 Fig 是目的端 constructor 的穩定 authored 身分／呈現欄位。
    // 原版 persistent record 沒有已證實的 portrait selector；它只在 +8 保存身分、
    // +7 保存可變 presentation selector，因此此處不可覆蓋兩者。
    if src.has_battle_fig {
        (dst.battle_fig, dst.has_battle_fig) = (src.battle_fig, true);
    }
    (dst.map_selector_key, dst.has_map_selector_key) = (src.map_selector_key, src.has_map_selector_key);
    (dst.exp, dst.exp_per_level) = (src.exp, src.exp_per_level);
    dst.spells = src.spells.clone();
    dst.native_command_mask = src.native_command_mask;
    (dst.native_identity, dst.has_native_identity) = (src.native_identity, src.has_native_identity);
    (dst.native_record_race, dst.has_native_record_race) = (src.native_record_race, src.has_native_record_race);
    (dst.native_record_class, dst.has_native_record_class) = (src.native_record_class, src.has_native_record_class);
    dst.native_transient = src.native_transient;
    (dst.native_record_byte5, dst.has_native_record_byte5) = (src.native_record_byte5, src.has_native_record_byte5);
    (dst.native_record_byte6, dst.has_native_record_byte6) = (src.native_record_byte6, src.has_native_record_byte6);
    (dst.native_record_byte8, dst.has_native_record_byte8) = (src.native_record_byte8, src.has_native_record_byte8);
    (dst.native_record_byte34, dst.has_native_record_byte34) = (src.native_record_byte34, src.has_native_record_byte34);
    (dst.native_record_byte35, dst.has_native_record_byte35) = (src.native_record_byte35, src.has_native_record_byte35);
    (dst.native_record_byte36, dst.has_native_record_byte36) = (src.native_record_byte36, src.has_native_record_byte36);
    // +0x42 is a raw persistent word used by ch15_post; preserve it only when
    // the source carries explicit provenance, never derive it from normalized HP.
    (dst.native_record_word42, dst.has_native_record_word42) = (src.native_record_word42, src.has_native_record_word42);
    (dst.native_record_word46, dst.has_native_record_word46) = (src.native_record_word46, src.has_native_record_word46);
    // Go `append(dst.X[:0], src.X...)`：內容等於 src；nil／空切片的差別 Rust 沒有。
    dst.inventory = src.inventory.clone();
    dst.equipped = src.equipped.clone();
    dst.inventory_slots = src.inventory_slots.clone();
    dst.native_inventory_flags = src.native_inventory_flags.clone();
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
        self.sync_party_from_battle_records().map(|_| ())
    }

    /// syncPartyFromBattleRecords 回傳實際發布到持續隊伍的紀錄（record）數。一般戰後
    /// 呼叫維持既有相容行為；終局來源約束邊界另要求至少一筆，避免所有原始身分
    /// （raw identity）都不相符時帶著舊名冊靜默進入角色蒙太奇。
    ///
    /// Go：`main.go:4382`，`func (g *Game) syncPartyFromBattleRecords() (int, error)`
    ///
    /// Go 以 `range g.partyRoster`（map，順序隨機）找第一個 raw identity 相符的名冊項；Rust 依 id 排序。
    /// 名冊裡的 identity 唯一時結果相同（KNOWN_DIFFERENCES `party-roster-map-order`）。
    pub(crate) fn sync_party_from_battle_records(&mut self) -> Result<i64> {
        let Some(st) = self.st.as_ref() else {
            return Err(GameError::new("缺少已完成的戰場狀態"));
        };
        let mut candidate = self.party_roster.clone();
        let mut item_rows: Option<Vec<u8>> = None;
        let mut synced = 0;
        for (index, current) in st.units.iter().enumerate() {
            let mut id = current.fig;
            let (mut raw_identity, mut has_raw_identity) = (current.native_identity, current.has_native_identity);
            if !has_raw_identity && current.has_native_record_byte8 {
                (raw_identity, has_raw_identity) = (current.native_record_byte8 as i64, true);
            }
            if has_raw_identity {
                let matched = self
                    .party_roster
                    .iter()
                    .find(|(_, roster)| roster.has_native_identity && roster.native_identity == raw_identity);
                match matched {
                    Some((&roster_id, _)) => id = roster_id,
                    None => continue,
                }
            }
            if !self.party_members.is_empty() {
                if !self.party_members.get(&id).copied().unwrap_or(false) {
                    continue;
                }
            } else if current.camp != battle::OWN {
                continue;
            }
            let mut snapshot = current.clone();
            if has_raw_identity {
                (snapshot.native_identity, snapshot.has_native_identity) = (raw_identity, true);
            }
            // 0x11586／0x1158E：清六個 raw 暫態 bytes，+5 僅保留 inactive bit0。
            snapshot.native_transient = [0; battle::NATIVE_TRANSIENT_COUNT as usize];
            if snapshot.has_native_record_byte5 {
                snapshot.native_record_byte5 &= 1;
            }
            if snapshot.max_mp < snapshot.mp {
                snapshot.max_mp = snapshot.mp;
            }
            if snapshot.has_native_record_byte5 {
                if snapshot.native_record_byte5 & 1 == 0 {
                    snapshot.hp = snapshot.max_hp;
                }
            } else if snapshot.on_field && snapshot.alive() {
                snapshot.hp = snapshot.max_hp;
            }
            snapshot.mp = snapshot.max_mp;
            snapshot.acted = false;
            (snapshot.off_x, snapshot.off_y) = (0.0, 0.0);
            // 0x11506 整筆抄回之後，JOIN 建構留下的殘值就被場上記錄取代。
            snapshot.native_join_persistent_pending = false;
            // 0x11506 把 runtime 紀錄整筆抄回持續槽，+2／+3／+4 照抄當下的值：+2 是
            // 0x11019 的 FDICON 選擇器快取槽（第五章戰前事件登場的角色 10 是第 14 個
            // 鍵，存檔裡 +2=13，不是 runtime 索引 41）；+3 是戰後 layout（0x233c6）寫下的
            // 朝向，沒有 layout 的章在行動收尾（0x134E4）與死亡演出結尾（0x1DB65 第 12 格）
            // 都已歸零。第四章 r9 與第五章 r5 的酒店存檔收據各證實一種形狀。沒有原生
            // 快取槽出處的舊資料才退回 runtime 索引。
            if !snapshot.has_map_selector_slot {
                (snapshot.map_selector_slot, snapshot.has_map_selector_slot) = (index as i64, true);
            }
            (snapshot.buff_ap_pct, snapshot.buff_dp_pct) = (0, 0);
            (snapshot.buff_hit, snapshot.buff_ev, snapshot.buff_turns) = (0, 0, 0);
            (snapshot.sealed, snapshot.seal_turns) = (false, 0);
            (snapshot.poisoned, snapshot.poison_turns) = (false, 0);
            (snapshot.paralyzed, snapshot.paralyze_turns) = (false, 0);
            // 0x115AC → sub_1145A：持續副本清暫態後，從基礎值與裝備重算。
            // 沒有原生出處的舊相容紀錄保留既有投影；正式原生紀錄缺欄位則拒收。
            if st.has_native_runtime_unit_projection
                || snapshot.equipment_base_set
                || snapshot.has_native_record_race
                || snapshot.has_native_record_class
            {
                let mut record = campaign::native_shop_equipment_record_for_unit(Some(&snapshot))
                    .map_err(|e| GameError::from(e).wrap(format!("sync_party record {index}: ")))?;
                if item_rows.is_none() {
                    item_rows = Some(
                        battle::load_native_item_effect_row_prefix(&asset_path(
                            "assets/data/native_item_effect_rows.json",
                        ))
                        .map_err(|e| GameError::from(e).wrap("sync_party item rows: "))?,
                    );
                }
                battle::apply_native_equipment_recalc(&mut record, item_rows.as_ref().unwrap())
                    .map_err(|e| GameError::from(e).wrap(format!("sync_party record {index} equipment: ")))?;
                let word = |at: usize| i16::from_le_bytes([record[at], record[at + 1]]) as i64;
                snapshot.ap = word(0x48);
                snapshot.dp = word(0x4a);
                snapshot.hit = word(0x4c);
                snapshot.ev = word(0x4e);
            }
            candidate.insert(id, snapshot);
            synced += 1;
        }
        self.party_roster = candidate;
        Ok(synced)
    }

    /// Go：`main.go:4510`，`func (g *Game) applyScenarioPartyJoins()`
    pub(crate) fn apply_scenario_party_joins(&mut self) {
        let Some(sc) = self.sc.as_mut() else {
            return;
        };
        for id in sc.take_party_joins() {
            if !campaign::joinable_character_id(id) {
                self.load_err = format!("scenario join_party:非法 player char_id={id}");
                continue;
            }
            if self.party_roster.contains_key(&id) {
                if !self.party_members.get(&id).copied().unwrap_or(false) {
                    self.party_members.insert(id, true);
                    self.party_join_order.push(id);
                }
                continue;
            }
            let Some(st) = self.st.as_ref() else {
                self.load_err = String::from("scenario join_party:缺少戰場來源");
                return;
            };
            let mut joined: Option<battle::Unit> = None;
            for unit in &st.units {
                // Native JOIN establishes permanent membership independently of the
                // actor's current camp colour. Many recruits are still Ally here.
                if unit.fig == id {
                    if joined.is_some() {
                        self.load_err = format!("scenario join_party:角色{id}有多筆場上記錄");
                        joined = None;
                        break;
                    }
                    joined = Some(unit.clone());
                }
            }
            if joined.is_none() && self.load_err.is_empty() {
                // 原版 JOIN 先建永久記錄，場上登場可以是後續動作。
                // 待登場名冊只提供原始身分相符的來源，不提前發布場上單位。
                for unit in &st.roster {
                    if !unit.has_native_record_byte8 || unit.native_record_byte8 as i64 != id {
                        continue;
                    }
                    if joined.is_some() {
                        self.load_err = format!("scenario join_party:角色{id}有多筆待登場來源");
                        return;
                    }
                    joined = Some(unit.clone());
                }
            }
            let Some(joined) = joined else {
                if self.load_err.is_empty() {
                    self.load_err = format!("scenario join_party:找不到角色{id}的我方記錄");
                }
                continue;
            };
            let base = clone_native_shop_unit(&joined);
            let mut carrier = battle::State::default();
            carrier.units = vec![base];
            self.initialize_equipment_bases(Some(&mut carrier));
            let base = carrier.units.pop().unwrap();
            let materialized = match self.materialize_native_join_persistent_unit(id, &base) {
                Ok(materialized) => materialized,
                Err(err) => {
                    self.load_err = format!("scenario join_party:角色{id} persistent record: {}", err.message());
                    continue;
                }
            };
            self.party_roster.insert(id, clone_native_shop_unit(&materialized));
            if !self.party_members.get(&id).copied().unwrap_or(false) {
                self.party_members.insert(id, true);
                self.party_join_order.push(id);
            }
        }
    }
}

/// filterScenarioParty applies the campaign's JOIN membership to a freshly
/// loaded battle scenario.  A nil/empty membership intentionally means a
/// direct chapter/debug start, so the authored scenario party remains usable.
///
/// Go：`main.go:4590`，`func filterScenarioParty(sc *battle.Scenario, members map[int]bool)`
pub(crate) fn filter_scenario_party(sc: Option<&mut battle::Scenario>, members: &BTreeMap<i64, bool>) {
    let Some(sc) = sc else {
        return;
    };
    if members.is_empty() {
        return;
    }
    // 部署格跟槽位不跟角色：0x1088D 把出戰的持續記錄依序建成 runtime 記錄，第 i 筆
    // 站 FDFIELD 出場位置的第 i 格。這裡只把沒出戰的人拿掉，格子清單原封不動。
    sc.party.retain(|member| members.get(&member.fig).copied().unwrap_or(false));
}

impl Game {
    /// battlePartyMembers returns the temporary roster selected by the original
    /// preparation screen, falling back to the permanent JOIN roster for direct
    /// starts and campaigns that have not reached a preparation node yet.
    ///
    /// Go：`main.go:4608`，`func (g *Game) battlePartyMembers() map[int]bool`
    pub(crate) fn battle_party_members(&self) -> BTreeMap<i64, bool> {
        if !self.party_deploy.is_empty() {
            // sub_320FC keeps persistent record zero fixed and reorders only
            // selection-table entry i -> persistent record i+1. partyDeploy stores
            // those selectable flags, so add the fixed record without mutating the
            // editable selection set itself.
            let mut members: BTreeMap<i64, bool> =
                self.party_deploy.iter().filter(|(_, selected)| **selected).map(|(&id, _)| (id, true)).collect();
            if let Some(&first) = self.party_join_order.first() {
                members.insert(first, true);
            }
            return members;
        }
        self.party_members.clone()
    }

    /// Go：`main.go:4628`，`func (g *Game) setupPreparation(n *campaign.Node)`
    ///
    /// Go `for id := range g.partyRoster` 以隨機順序補上不在 JOIN 順序裡的成員；Rust 依 ID 排序
    /// （KNOWN_DIFFERENCES `preparation-roster-map-order`）。
    pub(crate) fn setup_preparation(&mut self, n: Option<&campaign::Node>) {
        // 0x318AD exposes [0x53BFB]-1 flags. sub_320FC maps flag i to
        // persistent record i+1 and always leaves record zero in slot zero.
        // The fixed leader therefore must not consume one of the 15/19 choices.
        self.prep_ids.clear();
        if self.party_join_order.len() > 1 {
            self.prep_ids.extend_from_slice(&self.party_join_order[1..]);
        }
        let mut seen: BTreeSet<i64> = BTreeSet::new();
        if let Some(&first) = self.party_join_order.first() {
            seen.insert(first);
        }
        seen.extend(self.prep_ids.iter().copied());
        for &id in self.party_roster.keys() {
            if seen.insert(id) {
                self.prep_ids.push(id);
            }
        }
        self.prep_sel = 0;
        self.prep_selecting = false;
        self.prep_record_slots = false;
        self.prep_record_slot_sel = 0;
        self.prep_confirm = false;
        self.prep_confirm_sel = 0;
        self.prep_required_missing = None;
        self.native_class_ui_job = None;
        self.reset_native_class_ui_pulse();
        self.prep_limit = 15;
        if let Some(n) = n
            && n.party_limit > 0
        {
            self.prep_limit = n.party_limit;
        }
        // Direct EXE evidence: 0x318ad uses 0x13 instead of 0x0f after the
        // late-game chapter threshold. Keep this fallback editable via PartyLimit.
        if self.prep_limit == 15
            && let Some(camp) = self.camp.as_ref()
            && (camp.cur.ends_with("28") || camp.cur.ends_with("29") || camp.cur.ends_with("30"))
        {
            self.prep_limit = 19;
        }
        // 0x318c7 calls memset(flags, 0, 30) on every entry. A previous battle's
        // deployment and the remake save projection therefore cannot preselect
        // this native selection pass.
        self.party_deploy = BTreeMap::new();
        if n.is_none_or(|n| n.cancel.is_empty()) {
            // 0x2cc04 clears the complete VGA target before the standalone
            // FDTXT 0x19a record prompt.
            self.prep_prompt_source = vec![0u8; 320 * 200];
            self.prep_town_source = None;
        } else if self.prep_prompt_source.len() != 320 * 200 {
            // A town-backed prompt must retain its actual town frame; never
            // substitute a black or modern background for missing provenance.
            self.prep_prompt_source = Vec::new();
        }
        self.begin_native_preparation_prompt_opening();
    }

    /// Go：`main.go:4807`，`func (g *Game) setupChurch()`
    pub(crate) fn setup_church(&mut self) {
        self.native_class_ui_job = None;
        self.native_church_ui_job = None;
        self.church_sel = 0;
        self.church_mode = String::from("menu");
        self.church_ids = Vec::new();
        self.church_roster_start = 0;
        self.church_vertical_start = 0;
        self.church_transfer_source = -1;
        self.church_transfer_item = -1;
        self.church_transfer_items = Vec::new();
        self.church_transfer_dest = -1;
        self.church_item_start = 0;
        self.church_revive_id = -1;
        self.church_revive_fee = 0;
        self.church_class_id = -1;
        self.church_status_id = -1;
        self.church_status_panel = Vec::new();
        self.church_command_panel = Vec::new();
        self.church_status_data = None;
        self.church_branches = Vec::new();
        self.native_church_text_index = 585;
        self.begin_native_church_menu_opening();
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
    let Some(sc) = sc else {
        return Ok(());
    };
    if join_order.is_empty() {
        return Ok(());
    }
    if !sc.deploy_cells.is_empty() && sc.deploy_cells.len() < sc.party.len() {
        return Err(game_err!(
            "JOIN reordering requires complete deploy cells, got {} for {} party members",
            sc.deploy_cells.len(),
            sc.party.len()
        ));
    }
    // Go `byID[member.Fig] = member`：同一 Fig 重複時後者覆蓋。
    let mut by_id: BTreeMap<i64, battle::PartyMember> = BTreeMap::new();
    for member in &sc.party {
        by_id.insert(member.fig, member.clone());
    }
    let mut ordered = Vec::with_capacity(sc.party.len());
    for id in join_order {
        if let Some(member) = by_id.remove(id) {
            ordered.push(member);
        }
    }
    if ordered.len() != sc.party.len() {
        return Err(game_err!("JOIN order covers {} of {} scenario party members", ordered.len(), sc.party.len()));
    }
    sc.party = ordered;
    Ok(())
}

impl Game {
    /// focusOnParty 開局/戰鬥重開後把游標(=鏡頭中心)移到我方主角隊部署格的重心。
    /// 原鏡頭預設停在 (0,0),不對準的話玩家開局完全看不到主角隊(playfix #3)。
    /// 現有影片證據支持第一關開場含多幀人物走位，但確切起點、路徑與時序尚未閉合；
    /// 本函式只負責戰鬥控制權交接時把鏡頭對準已物化的我方隊伍，不宣稱重現該段演出。
    ///
    /// Go：`main.go:4967`，`func (g *Game) focusOnParty()`
    pub(crate) fn focus_on_party(&mut self) {
        let Some(st) = self.st.as_ref() else {
            return;
        };
        let (mut n, mut sx, mut sy) = (0, 0, 0);
        for u in st.units.iter().filter(|u| u.camp == battle::OWN && u.on_field) {
            sx += u.x;
            sy += u.y;
            n += 1;
        }
        if n > 0 {
            (self.cur_x, self.cur_y) = (sx / n, sy / n);
        }
    }

    /// campInput 處理 campaign 節點的輸入。回傳 true = 已攔截(擋掉戰場一般輸入)。
    ///
    /// Go：`main.go:5044`，`func (g *Game) campInput() bool`
    ///
    /// 輸入讀 `self.tick_input`（Go 的 `inpututil` 查詢）。Go 傳給 `handleNativeStoryInput` 的是 campaign
    /// 節點指標；Rust 傳節點複本（B 組的函式只讀它）。
    pub(crate) fn camp_input(&mut self) -> bool {
        if self.camp.is_none() {
            return false;
        }
        let input = self.tick_input.clone();
        let just = |k: Key| input.is_just_pressed(k);
        let enter = just(Key::Enter) || just(Key::Space);
        let Some(mut n) = self.camp.as_ref().unwrap().node().cloned() else {
            return true; // game over:鎖定
        };
        match n.r#type.as_str() {
            "story" | "cutscene" => {
                self.handle_native_story_input(Some(&mut n), NativeStoryInput { enter, escape: just(Key::Escape) })
            }
            "choice" | "town" => {
                let native_town = n.r#type == "town" && n.native_town_variant.is_some();
                if native_town {
                    let (scan_code, ok) = pressed_native_town_secret_scan(&input);
                    if ok && self.reveal_native_town_secret(scan_code) {
                        return true;
                    }
                    if just(Key::ArrowRight) {
                        self.move_native_town_selection(-1);
                    }
                    if just(Key::ArrowLeft) {
                        self.move_native_town_selection(1);
                    }
                    // Go `main.go:5083`（r5i）：`*n.NativeTownVariant`。
                    self.apply_pointer_to_native_town(n.native_town_variant.unwrap_or_default());
                    if enter {
                        let sel = self.camp_sel;
                        if self.camp.as_mut().unwrap().confirm_native_town_secret(sel) {
                            self.enter_node();
                            return true;
                        }
                        if (0..5).contains(&self.camp_sel) {
                            self.capture_native_town_caller();
                            let outcome = format!("opt{}", self.camp_sel);
                            self.camp.as_mut().unwrap().advance(&outcome);
                            self.enter_node();
                        }
                    }
                    return true;
                }
                if just(Key::ArrowUp) {
                    self.step_campaign_menu(campaign::MENU_UP);
                }
                if just(Key::ArrowDown) {
                    self.step_campaign_menu(campaign::MENU_DOWN);
                }
                self.apply_pointer_to_choice_menu(); // Go `main.go:5103`（r5i）
                let (mut selected, mut confirm) = self.step_campaign_menu(campaign::MENU_TICK);
                if enter {
                    (selected, confirm) = self.step_campaign_menu(campaign::MENU_CONFIRM);
                }
                if confirm {
                    self.camp.as_mut().unwrap().advance(&format!("opt{selected}"));
                    self.enter_node();
                }
                true
            }
            "preparation" | "church" => {
                if n.r#type == "preparation" {
                    return self.handle_native_preparation_input(NativePreparationInput {
                        left: just(Key::ArrowLeft),
                        right: just(Key::ArrowRight),
                        up: just(Key::ArrowUp),
                        down: just(Key::ArrowDown),
                        enter,
                        escape: just(Key::Escape),
                    });
                }
                self.handle_native_church_input(&input, enter);
                true
            }
            "shop" => {
                if !self.native_shop_mode.is_empty() {
                    return self.handle_native_shop_input(&input, enter);
                }
                if just(Key::Escape) {
                    self.leave_shop();
                }
                true
            }
            "hotel" => {
                if !self.native_hotel_mode.is_empty() {
                    return self.handle_native_hotel_input(current_native_hotel_input(&input, enter));
                }
                if just(Key::ArrowUp) && self.hotel_sel > 0 {
                    self.hotel_sel -= 1;
                }
                if just(Key::ArrowDown) && self.hotel_sel < 3 {
                    self.hotel_sel += 1;
                }
                if just(Key::Escape) {
                    self.leave_hotel();
                    return true;
                }
                if enter {
                    self.apply_hotel_service_selection(self.hotel_sel as u8);
                }
                true
            }
            "ending" => true,
            "battle" => {
                if !self.result.is_empty() && enter {
                    // 勝敗後 Enter → 依結果轉場(敗北可走敗北路線)
                    return self.confirm_battle_result();
                }
                false // 戰鬥照常
            }
            _ => false,
        }
    }

    /// stepCampaignMenu is the runtime seam for deterministic choice/town input
    /// traces. It owns only cursor state; the runner still owns editable option
    /// visibility and transition targets.
    ///
    /// Go：`main.go:5602`，`func (g *Game) stepCampaignMenu(event campaign.MenuEvent) (selected int, confirm bool)`
    /// （`r8`，E 組隨 `campInput` 補移植）
    pub(crate) fn step_campaign_menu(&mut self, event: campaign::MenuEvent) -> (i64, bool) {
        let Some(camp) = self.camp.as_ref() else {
            return (0, false);
        };
        let mut menu = campaign::MenuState { selection: self.camp_sel, count: camp.visible().len() as i64 };
        let (selected, confirm) = menu.step(event);
        self.camp_sel = menu.selection;
        (selected, confirm)
    }

    /// confirmBattleResult consumes the already displayed battle result at the
    /// explicit Enter boundary.  Keeping this seam separate prevents tests and
    /// future input adapters from bypassing the same postbattle/campaign handoff
    /// used by the player-facing control path.
    ///
    /// Go：`main.go:5413`，`func (g *Game) confirmBattleResult() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn confirm_battle_result(&mut self) -> bool {
        if self.camp.is_none() || self.result.is_empty() {
            return false;
        }
        let current = self.camp.as_ref().unwrap().node().cloned();
        if self.result == "lose" && current.as_ref().is_some_and(|n| n.native_defeat_return_title.is_some()) {
            return false; // 原生敗北自動返回標題，Enter 不得走相容 on_lose。
        }
        let outcome = self.result.clone();
        // 第30戰的可編輯 ending edge明確要求保存最後隊伍，供來源約束 E1 的角色
        // 終局回顧使用；這是 remake 資料邊界，不冒稱原版 FD2.SAV ABI。
        if outcome == "win" && current.as_ref().is_some_and(|n| n.ending_party_snapshot_on_win) {
            let result = match self.sync_party_from_battle_records() {
                Ok(0) => Err(GameError::new("完成的戰場沒有任何持續隊伍身分符合")),
                other => other,
            };
            if let Err(err) = result {
                self.load_err = format!("終局隊伍同步失敗: {}", err.message());
                self.msg = String::from("最終戰結果尚未保存，流程已停止");
                return false;
            }
        }
        // An empty next id is a valid terminal/game-over result; consume the same
        // Enter boundary and let enterNode observe the ended campaign.
        self.camp.as_mut().unwrap().advance(&outcome);
        // The result belongs to the battle node only.  Postbattle/cutscene and town
        // nodes must not inherit a stale win/lose overlay while they persist the
        // party and expose the next hub.
        self.result.clear();
        self.enter_node();
        true
    }

    /// Rust 新增：Go `g.camp == nil || g.camp.Node() == nil || g.camp.Node().Type != kind`。
    fn current_node_type_is(&self, kind: &str) -> bool {
        self.camp.as_ref().and_then(|c| c.node()).is_some_and(|n| n.r#type == kind)
    }

    /// leaveChurch is the editable campaign boundary for the church menu's Escape
    /// action. It never assumes a specific town or service return target.
    ///
    /// Go：`main.go:5501`，`func (g *Game) leaveChurch()`
    pub(crate) fn leave_church(&mut self) {
        if !self.current_node_type_is("church") {
            return;
        }
        self.camp.as_mut().unwrap().advance("");
        self.native_town_hub_return = true;
        self.enter_node();
    }

    /// leaveShop is the campaign boundary for the shop's Escape/leave action. It
    /// deliberately delegates the next node to editable campaign data instead of
    /// assuming every shop returns to the same town.
    ///
    /// Go：`main.go:5550`，`func (g *Game) leaveShop()`
    pub(crate) fn leave_shop(&mut self) {
        if !self.current_node_type_is("shop") {
            return;
        }
        let return_selection = self.native_shop_variant;
        self.native_shop_ui_job = None;
        self.native_shop_mode.clear();
        self.native_shop_variant = 0;
        self.camp.as_mut().unwrap().advance("");
        self.native_town_hub_return = true;
        self.enter_node();
        if self.current_node_type_is("town") && matches!(return_selection, 1 | 3 | 5) {
            // 0x2e341 returns to the same town selector that dispatched the
            // weapon/item/secret branch. In particular, DOSBox E2 confirms
            // secret variant 5 restores the revealed hidden selection 5.
            self.camp_sel = return_selection;
        }
    }

    /// leaveHotel returns through the authored campaign edge. It never assumes a
    /// specific town or converts an unresolved raw hotel callee into gameplay.
    ///
    /// Go：`main.go:5588`，`func (g *Game) leaveHotel()`
    pub(crate) fn leave_hotel(&mut self) {
        if !self.current_node_type_is("hotel") {
            return;
        }
        self.native_hotel_mode.clear();
        self.native_hotel_prompt_return = false;
        self.camp.as_mut().unwrap().advance("");
        self.native_town_hub_return = true;
        self.enter_node();
    }

    /// applyHotelServiceSelection is deliberately raw: official 0x2fc85 proves
    /// selector/resource/callee order, but not the high-level service names. It
    /// records the route for an eventual indexed/UI consumer and never mutates
    /// party, gold, or campaign state on its own.
    ///
    /// Go：`main.go:5575`，`func (g *Game) applyHotelServiceSelection(selector byte) bool`（`r8`，E 組隨
    /// `campInput`、`handleNativeHotelInput` 補移植）
    pub(crate) fn apply_hotel_service_selection(&mut self, selector: u8) -> bool {
        let Some(route) = fdother::resolve_native_hotel_service_route(selector) else {
            self.msg = format!("旅館 raw selector {selector} 無效");
            return false;
        };
        (self.hotel_route, self.hotel_has_route) = (route, true);
        self.msg = String::from("酒店介面素材不完整，保留目前狀態");
        true
    }

    /// ringInput 暫定四向命令選單 + 法術選單輸入。回傳 true = 已攔截。
    /// 方向配對已由 Docker Capstone 0x18d8c switch 釘死：↑0攻擊/←1法術/→2物品/↓3待機。
    ///
    /// Go：`main.go:5614`，`func (g *Game) ringInput() bool`
    pub(crate) fn ring_input(&mut self) -> bool {
        self.pointer_overlay_rejects_click(); // Go `main.go:5623`（r5i）：作廢點在不可用方向上的 Enter
        let input = self.tick_input.clone();
        let just = |k: Key| input.is_just_pressed(k);
        let enter = just(Key::Enter) || just(Key::Space);
        let esc = just(Key::Escape) || just(Key::Backspace);
        if let Some(info) = self.native_system_info_ui.as_ref() {
            if info.phase == "steady" && input.any_just_pressed() {
                self.close_native_system_info_ui();
            }
            return true;
        }
        if self.native_system_end_turn_ui.is_some() && self.native_class_ui_job.is_some() {
            return true;
        }
        if self.native_system_end_turn_delay > 0 {
            return true;
        }
        if let Some(treasure) = self
            .native_system_end_turn_ui
            .as_mut()
            .and_then(|state| state.treasure.as_mut())
            .filter(|treasure| treasure.await_ack)
        {
            if enter || esc {
                treasure.await_ack = false;
                self.native_system_end_turn_delay = 1;
            }
            return true;
        }
        if self.native_system_end_turn_confirm {
            if self.native_class_ui_job.is_some() {
                return true;
            }
            if let Some(state) = self.native_system_end_turn_ui.as_mut().filter(|state| state.treasure.is_some()) {
                if just(Key::ArrowLeft) {
                    state.choice = 0;
                } else if just(Key::ArrowRight) {
                    state.choice = 1;
                }
                if enter || esc {
                    let accepted = !esc && state.choice == 0;
                    self.finish_native_treasure_choice(accepted);
                }
                return true;
            }
            if esc {
                self.cancel_native_system_end_turn();
                return true;
            }
            if let Some(state) = self.native_system_end_turn_ui.as_mut() {
                if just(Key::ArrowLeft) {
                    state.choice = 0;
                }
                if just(Key::ArrowRight) {
                    state.choice = 1;
                }
            }
            if enter {
                if self.native_system_end_turn_ui.as_ref().is_some_and(|state| state.choice == 1) {
                    self.cancel_native_system_end_turn();
                } else {
                    self.confirm_native_system_end_turn();
                }
            }
            return true;
        }
        if self.native_system_options_open {
            if !self.ring {
                self.native_system_options_open = false;
                self.native_system_cursor_overlay = false;
                return false;
            }
            if self.action_overlay_blocks_input() {
                return true;
            }
            self.apply_native_overlay_direction_input();
            if esc {
                self.begin_action_overlay_close(Some(Box::new(|g: &mut Game| {
                    g.native_system_options_open = false;
                    g.native_system_cursor_overlay = false;
                    g.msg = String::new();
                })));
                return true;
            }
            if enter && !self.toggle_native_system_option(self.ring_sel) {
                self.msg = String::from("原版系統設定值無效，未變更");
            }
            return true;
        }
        if self.native_system_nested_open {
            if !self.ring {
                self.native_system_nested_open = false;
                self.native_system_cursor_overlay = false;
                return false;
            }
            if self.action_overlay_blocks_input() {
                return true;
            }
            self.apply_native_overlay_direction_input();
            if esc {
                self.begin_action_overlay_close(Some(Box::new(|g: &mut Game| {
                    g.native_system_nested_open = false;
                    g.native_system_cursor_overlay = false;
                    g.msg = String::new();
                })));
                return true;
            }
            if enter {
                let selector = self.ring_sel;
                match selector {
                    1 => {
                        self.activate_native_system_direction_one();
                    }
                    0 => match self.prepare_native_system_info_ui() {
                        Ok(state) => self.begin_action_overlay_close(Some(Box::new(move |g: &mut Game| {
                            g.native_system_nested_open = false;
                            g.native_system_cursor_overlay = false;
                            g.native_system_info_ui = Some(state);
                        }))),
                        Err(_) => self.msg = String::from("原版戰場資訊來源不完整，未開啟畫面"),
                    },
                    3 if !self.begin_native_nested_system_exit() => {
                        self.msg = String::from("原版離場確認資產不完整，未離開遊戲");
                    }
                    2 if !self.begin_native_nested_current_load() => {
                        self.msg = String::from("原版目前戰況讀檔來源或資產不完整，保留目前戰場");
                    }
                    _ => {}
                }
            }
            return true;
        }
        if self.native_system_cursor_overlay {
            if !self.ring {
                self.native_system_cursor_overlay = false;
                return false;
            }
            if self.action_overlay_blocks_input() {
                return true;
            }
            self.apply_native_overlay_direction_input();
            if esc {
                self.begin_action_overlay_close(Some(Box::new(|g: &mut Game| {
                    g.native_system_cursor_overlay = false;
                    g.msg = String::new();
                })));
                return true;
            }
            if enter {
                let selector = self.ring_sel;
                match selector {
                    0 => {
                        if self.native_nested_system_overlay_state().is_none() {
                            self.msg = String::from("原版巢狀系統選單圖格或狀態不完整");
                            return true;
                        }
                        self.begin_action_overlay_close(Some(Box::new(|g: &mut Game| {
                            g.native_system_nested_open = true;
                            g.begin_action_overlay_open(0);
                        })));
                    }
                    3 if !self.begin_native_system_end_turn() => {
                        self.msg = String::from("原版 END 確認資產不完整，未結束回合");
                    }
                    2 => {
                        if !self.native_system_options_ready() {
                            self.msg = String::from("原版系統設定圖格不完整，未開啟選單");
                            return true;
                        }
                        self.begin_action_overlay_close(Some(Box::new(|g: &mut Game| {
                            g.native_system_options_open = true;
                            g.begin_action_overlay_open(0);
                        })));
                    }
                    1 => {
                        self.activate_native_system_direction_one();
                    }
                    _ => {}
                }
            }
            return true;
        }
        if self.item_open {
            if self.sel.is_none() {
                self.item_open = false;
                self.clear_native_item_panel();
                return false;
            }
            return self.native_item_panel_input(enter, esc);
        }
        if self.native_command_open {
            return self.native_command_grid_input(enter, esc);
        }
        if self.spell_open {
            // 法術選單
            let Some(sel) = self.sel else {
                self.spell_open = false;
                return false;
            };
            let spells = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic").units[sel].spells.clone();
            if just(Key::ArrowUp) && self.spell_sel > 0 {
                self.spell_sel -= 1;
            }
            if just(Key::ArrowDown) && self.spell_sel < spells.len() as i64 - 1 {
                self.spell_sel += 1;
            }
            if esc {
                self.spell_open = false;
                self.begin_action_overlay_open(self.ring_sel);
            }
            if enter && self.spell_sel < spells.len() as i64 {
                let id = spells[self.spell_sel as usize];
                let mp = self.st.as_ref().unwrap().units[sel].mp;
                if let Some(i) = self.spells.iter().position(|spell| spell.id == id) {
                    if mp < self.spells[i].mp {
                        if let (message, true) = self.locale_message("battle.mp.insufficient", &[]) {
                            self.msg = message;
                        }
                        return true;
                    }
                    // Go `g.castSp = &g.spells[i]`：Rust 存那一筆的複本。
                    self.cast_sp = Some(self.spells[i].clone());
                    self.spell_open = false;
                    let (name, dist) = (self.spells[i].name.clone(), self.spells[i].dist);
                    if let (message, true) =
                        self.locale_message("battle.spell.target_prompt", &[name.into(), dist.into()])
                    {
                        self.msg = message;
                    }
                }
            }
            return true;
        }
        let Some(sel) = self.sel.filter(|_| self.ring) else {
            return false;
        };
        if self.action_overlay_blocks_input() {
            return true;
        }
        // 環導航：方向寫入 [0x53c57] 的閘門在 sub_177FC 0x17835..0x17897，
        // 該方向的 availability 不為零時原版根本不改變選擇。
        self.apply_native_overlay_direction_input();
        if esc {
            // ESC = 取消(doc13):先播放0x176b4四幀，再退回移動前位置
            self.begin_action_overlay_close(Some(Box::new(|g: &mut Game| {
                g.msg = String::new();
                // Go 在延續執行時才讀 `g.sel`（nil 時解參考 panic）。
                let sel = g.sel.expect("Go：g.sel 為 nil 時解參考 panic");
                let (sel_orig_x, sel_orig_y) = (g.sel_orig_x, g.sel_orig_y);
                let st = g.st.as_mut().unwrap();
                let unit = &mut st.units[sel];
                if unit.x == sel_orig_x && unit.y == sel_orig_y {
                    g.clear_native_player_movement();
                    (g.sel, g.reach, g.moved) = (None, BTreeMap::new(), false);
                } else {
                    let dir = unit.dir;
                    unit.set_map_placement(sel_orig_x, sel_orig_y, dir);
                    g.moved = false;
                    g.reach = st.reachable(sel);
                    g.restore_player_movement_cursor();
                    if g.st.as_ref().unwrap().has_native_map_view_state
                        && native_map_assets_available(g.native_map_assets.as_ref())
                        && let Err(err) = g.prepare_native_player_movement(sel)
                    {
                        g.load_err = err.message().to_string();
                    }
                }
            })));
            return true;
        }
        if enter {
            // 0x177fc accepts a direction only when its corresponding 0x18d8c
            // availability word is zero.  Rendering a disabled FDOTHER cell without
            // enforcing the same input gate would let the remake execute an action
            // the native chooser rejects.
            let availability = self.action_overlay_availability();
            if !native_action_selectable(availability, self.ring_sel) {
                if let (message, true) = self.locale_message("battle.command.unavailable", &[]) {
                    self.msg = message;
                }
                return true;
            }
            match self.ring_sel {
                0 => {
                    // 攻擊 → 關環,進選目標(游標移到攻擊範圍內的敵人;範圍依武器射程,doc32)
                    self.begin_action_overlay_close(Some(Box::new(|g: &mut Game| g.begin_player_attack_targeting())));
                }
                1 => {
                    // 法術(原版 0x1cff0；有法術者才可用)
                    let unit = &self.st.as_ref().unwrap().units[sel];
                    let ids = unit.native_command_ids();
                    if !ids.is_empty() && self.locale_entities.is_some() && self.native_ui_palette.len() >= 0xce {
                        // Native 0x18d8c disables its command action when raw unit+0x27
                        // is nonzero.  NativeTransient[5] preserves exactly that byte;
                        // legacy Sealed is a separate normalized compatibility status.
                        if native_command_action_blocked(Some(unit)) {
                            if let (message, true) = self.locale_message("battle.command.native_unavailable", &[]) {
                                self.msg = message;
                            }
                            return true;
                        }
                        self.begin_action_overlay_close(Some(Box::new(|g: &mut Game| {
                            (g.native_command_open, g.native_command_sel) = (true, 0);
                        })));
                        return true;
                    }
                    let (has_spells, sealed) = (!unit.spells.is_empty(), unit.sealed);
                    if has_spells {
                        if sealed {
                            if let (message, true) = self.locale_message("battle.spell.sealed", &[]) {
                                self.msg = message;
                            }
                        } else {
                            self.begin_action_overlay_close(Some(Box::new(|g: &mut Game| {
                                (g.spell_open, g.spell_sel) = (true, 0);
                            })));
                        }
                    } else if let (message, true) = self.locale_message("battle.spell.none", &[]) {
                        self.msg = message;
                    }
                }
                2 => {
                    // 物品(原版 0x1bbdc；完整 item action 仍 fail-closed)
                    self.begin_action_overlay_close(Some(Box::new(|g: &mut Game| {
                        (g.item_open, g.item_sel) = (true, 0);
                        (g.item_anim_step, g.item_closing) = (0, false);
                        if let Some(actor) = g.sel {
                            g.prepare_native_item_panel(actor);
                        }
                    })));
                }
                3 => {
                    // 休息回復／格子互動(原版 0x13fd4→0x190ac)
                    self.begin_action_overlay_close(Some(Box::new(|g: &mut Game| g.finish_selected_wait())));
                }
                _ => {}
            }
        }
        true
    }

    /// Go：`main.go:6201`，`func (g *Game) resetNativeTargetField() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn reset_native_target_field(&mut self) -> bool {
        let Some(st) = self.st.as_mut() else {
            return false;
        };
        if st.w <= 0 || st.h <= 0 || st.native_tile_blit_modes.len() as i64 != st.w * st.h {
            return false;
        }
        // 0x4dbfc runs immediately after 0x115b6 returns.
        st.native_tile_blit_modes.fill(0xff);
        true
    }

    /// finishSelectedWait 對應原版行動選單第四項「下／休息」。未移動時走
    /// 0x13fd4：current HP != max HP 且 raw +0x25/+0x26 都為零才增加
    /// floor(max HP/5)，不保證最少回復 1。其後 0x190ac 檢查當格寶物；
    /// 不是踩上格子立即開箱，也不是道具指令。
    ///
    /// Go：`main.go:6239`，`func (g *Game) finishSelectedWait()`
    pub(crate) fn finish_selected_wait(&mut self) {
        let Some(u) = self.sel else {
            return;
        };
        self.ring = false;
        let (sel_orig_x, sel_orig_y) = (self.sel_orig_x, self.sel_orig_y);
        let st = self.st.as_mut().expect("Go：g.st 為 nil 時解參考 panic");
        let unit = &mut st.units[u];
        if unit.x == sel_orig_x
            && unit.y == sel_orig_y
            && unit.hp != unit.max_hp
            && unit.native_transient[3] == 0
            && unit.native_transient[4] == 0
        {
            let heal = unit.max_hp / 5;
            unit.hp += heal;
            if unit.hp > unit.max_hp {
                unit.hp = unit.max_hp;
            }
        }
        let (ux, uy, inventory_len) = (unit.x, unit.y, unit.inventory.len());
        let (before, exists) = st.treasure_at(ux, uy);
        if exists {
            if st.has_native_map_view_state && ((before.kind == "item" && inventory_len < 8) || before.kind == "gold") {
                if let Err(err) = self.begin_native_treasure_prompt(u, before) {
                    self.load_err = err.message().to_string();
                }
                return;
            }
            let (got, ok) = st.claim_treasure(Some(u), ux, uy);
            if ok {
                if got.kind == "gold" {
                    self.gold += got.value;
                    if let (message, true) = self.locale_message("battle.treasure.gold", &[got.value.into()]) {
                        self.msg = message;
                    }
                } else if let (message, true) = self.locale_message("battle.treasure.item", &[got.value.into()]) {
                    self.msg = message;
                }
            } else if (before.kind == "item" || before.kind == "event")
                && inventory_len >= 8
                && let (message, true) = self.locale_message("battle.treasure.inventory_full", &[])
            {
                self.msg = message;
            }
        }
        self.finish_successful_unit_action(
            Some(u),
            Some(Box::new(|g: &mut Game| {
                (g.sel, g.reach, g.moved) = (None, BTreeMap::new(), false);
            })),
        );
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
        let Some(actor) = actor else {
            return;
        };
        // Go 的 `after` 同時被四個續行閉包捕捉（每個閉包都以同一個 `after` 重入本函式），沒有被存起來的閉包
        // 就丟掉；`FnOnce` 只能交出一次，所以放在共用的格子裡，由真正執行的那一條取出。
        let after: SharedAfter = Arc::new(std::sync::Mutex::new(after));
        if self.begin_native_action_death_presentation(actor, &after) {
            return;
        }
        if let Some(st) = self.st.as_mut() {
            // 0x1DB65：行動結算後把 HP 歸零的記錄標成 +5＝1，指令傷害的擊殺才會從
            // AI 目標掃描與回合回復裡消失（第四章 r9 收據第 5 回合 0x13 的目標選擇）。
            st.mark_native_dead_records();
        }
        // 0x134E4：每個行動收尾（0x13E77 的 AI 合流、0x1566A 攻擊路徑、玩家 0x18890）都把
        // **全部**記錄的 raw +3 歸零，不只行動者。掉落訊息（r6 seq 633）與升級對話（r4 seq
        // 968、r6 seq 1836）的底圖上攻擊者已是 pose 0；第十回合 event 25 的 ACTING 30 讓凱麗
        // 停在 pose 3，敵方第一個行動收尾就把她歸零（r6 seq 1962→1963）。冪等。
        self.reset_all_native_map_poses();
        // 0x1AA1D 成功臂（在 0x18890 的 handler 裡、bit7 之前：r4 seq 279 攻擊者還沒變灰）：
        // 先播「得到 X」訊息（等鍵、關框），再回來收尾。
        if self.run_pending_native_death_reward_messages(actor, Some(finish_successful_unit_action_then(actor, &after)))
        {
            return;
        }
        if self.run_pending_native_death_rewards(Some(finish_successful_unit_action_then(actor, &after))) {
            return;
        }
        // 死亡程式在攻擊演出之後、行動收尾之前執行（0x1CFF0 → 0x1B6B7／0x1AA1D）。
        if self.run_pending_death_programs(actor, Some(finish_successful_unit_action_then(actor, &after))) {
            return;
        }
        // handler 成功返回臂（0x13512）：Acted、靜止姿態、+5 bit7。這些是冪等的，對話回來
        // 再進一次也不會變。
        let (sel, ai_busy) = (self.sel, self.ai_busy);
        let st = self.st.as_mut().expect("Go：actor 指向 g.st.Units");
        let has_view = st.has_native_map_view_state;
        let unit = &mut st.units[actor];
        unit.acted = true;
        let player_actor =
            sel == Some(actor) || (!ai_busy && unit.has_native_record_byte6 && unit.native_record_byte6 == 2);
        if player_actor && has_view && unit.has_native_record_byte5 {
            unit.native_record_byte5 |= 0x80;
        }
        // 0x11951 0x11CAC 重繪之後才 0x1196D 0x1E292：經驗／升級對話看到的是攻擊者已變灰、
        // 屍體已移除、姿勢已靜止的地圖（r4 seq 968、r6 seq 1836）；章節函式表、0x13565 自動
        // 換手與 0x1198A 格子事件都在它之後。
        if self.run_pending_native_level_up_dialogues(Some(finish_successful_unit_action_then(actor, &after))) {
            return;
        }
        if self.sel == Some(actor) {
            self.clear_native_player_movement();
        }
        if let Some(after) = take_shared_after(&after) {
            after(self);
        }
        // 0x11985 的 0x13565 在 0x1198A 之前：最後一個我方單位收尾就直接進 0x1A30B 換手，
        // [0x51A8F] 在友軍 AI 迴圈第一筆（0x1D853）就被寫回 0xff，這次踏到的格子事件不會分派。
        if self.auto_end_player_phase(Some(actor)) {
            self.native_field_event_pending = None;
            return;
        }
        // 0x1198A（玩家）／0x1D855 等（AI）：分派走行時記下的 selector 0 格子事件，或收尾格
        // 以 selector 1 查到的 event 61／75（0x18B0C／0x18B66 同樣寫 [0x51A8F]）。
        if self.dispatch_native_field_event_pending(Some(actor)) {
            return;
        }
        if self.begin_native_field_event80(Some(actor)) {
            return;
        }
        if self.begin_native_field_event61(Some(actor), None) {
            return;
        }
        self.begin_native_field_event75(Some(actor), None);
    }

    /// resetAllNativeMapPoses 是 0x134E4：對 [0x53BEB] 筆記錄逐筆 `+3 = 0`，再 delay(0x14) 毫秒。
    ///
    /// Go：`main.go:6369`，`func (g *Game) resetAllNativeMapPoses()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn reset_all_native_map_poses(&mut self) {
        let Some(st) = self.st.as_mut() else {
            return;
        };
        for u in &mut st.units {
            if u.has_native_map_presentation {
                u.set_map_pose(NATIVE_MAP_REST_POSE);
            }
        }
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
        let (Some(st), Some(actor)) = (self.st.as_ref(), actor) else {
            return false;
        };
        if st.units[actor].camp != battle::OWN
            || self.ai_busy
            || !self.result.is_empty()
            || self.native_turn_staging.is_some()
            || self.battle_event.is_some()
        {
            return false;
        }
        for u in &st.units {
            if u.camp != battle::OWN || !u.on_field || !u.alive() || u.acted {
                continue;
            }
            if u.has_native_record_byte5 && u.native_record_byte5 & 0x81 != 0 {
                continue;
            }
            if let (transient, true) = u.native_transient_duration(0x26)
                && transient != 0
            {
                continue;
            }
            return false;
        }
        self.end_turn();
        true
    }

    /// awardDeathReward 在擊倒當下處理死亡效果：型態 0／1 直接給（只給原版陣營 2 的
    /// 擊殺者，見 grantNativeDeathReward），型態 2／3 登記成死亡程式，等行動收尾依記錄
    /// 順序執行。每個死亡 transition 只處理一次。
    ///
    /// Go：`main.go:6410`，`func (g *Game) awardDeathReward(dead, killer *battle.Unit)`
    ///
    /// R4 B 組（跨組介面，本體仍歸 D 組）：`killer` 改 `Option<UnitIdx>`。B 組的 `completeTurn` 以
    /// `g.awardDeathReward(u, nil)` 處理狀態異常致死，骨架的 `UnitIdx` 傳不出 nil。
    pub(crate) fn award_death_reward(&mut self, dead: Option<UnitIdx>, killer: Option<UnitIdx>) {
        let Some(dead) = dead else {
            return;
        };
        let st = self.st.as_ref().expect("Go：dead 指向 g.st.Units");
        if st.units[dead].alive() {
            return;
        }
        if self.death_rewarded.get(&dead).copied().unwrap_or(false) {
            return;
        }
        self.death_rewarded.insert(dead, true);
        if let Some(r) = st.units[dead].death_reward.as_ref()
            && (r.r#type == 0 || r.r#type == 1)
        {
            let (kind, value) = (r.r#type, r.value);
            self.grant_native_death_reward(kind, value, killer);
        }
        self.queue_native_death_program(Some(dead), killer);
    }
}

/// R4 D 組新增，沒有 Go 對應：`finishSuccessfulUnitAction` 的 `after` 被多個續行閉包共用（見該函式）。
pub(crate) type SharedAfter = Arc<std::sync::Mutex<Option<Box<dyn FnOnce(&mut Game) + Send>>>>;

/// 取出共用的 `after`（只有第一個取的拿得到）。
pub(crate) fn take_shared_after(after: &SharedAfter) -> Option<Box<dyn FnOnce(&mut Game) + Send>> {
    after.lock().expect("after 格子沒有被中斷的持有者").take()
}

/// Go `finishSuccessfulUnitAction` 的四個續行閉包：`func() { g.finishSuccessfulUnitAction(actor, after); g.checkResult() }`。
fn finish_successful_unit_action_then(actor: UnitIdx, after: &SharedAfter) -> Box<dyn FnOnce(&mut Game) + Send> {
    let after = Arc::clone(after);
    Box::new(move |g: &mut Game| {
        g.finish_successful_unit_action(Some(actor), take_shared_after(&after));
        g.check_result();
    })
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
    ///
    /// Go 的 `finish` 閉包捕捉 `w := g.walk`；呼叫當下 `g.walk == w`，Rust 在 finish 開頭把它取出
    /// （跨組介面見 `step_battle_walk_segment` 的說明）。
    pub(crate) fn step_battle_walk(&mut self) {
        let Some(w) = self.walk.as_ref() else {
            return;
        };
        if self.m.is_none() || w.native_group_march_paused {
            return;
        }
        fn finish(g: &mut Game, pose: i64) {
            let Some(mut w) = g.walk.take() else {
                return;
            };
            if w.native_group_march_event != w.native_group_march_events.len() as i64 {
                g.load_err = String::from("native system group-march ended before planned field events");
                // Go：g.walk = nil（已取出）
                g.native_system_group_march = None;
                return;
            }
            let last = *w.path.last().expect("Go：w.path 為空時索引 panic");
            if let (Some(idx), Some(st)) = (w.u, g.st.as_mut()) {
                st.units[idx].set_map_placement(last.x, last.y, pose);
            }
            match w.then.take() {
                Some(then) => then(g),
                None => {
                    g.moved = true;
                    g.begin_battle_action_overlay();
                }
            }
        }
        while self.step_battle_walk_segment(&mut finish) {}
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
    ///
    /// R4 A 組（跨組介面需求，主代理整合時與 B 組定案）：Go 的 `w` 與 `g.walk` 是同一個指標，segment 內的
    /// `nativeCh12UpWalkFrame` 讀 `g.walk`、`beginNativeSystemGroupMarchFieldEvent` 比對 `g.walk != walk`、
    /// 錯誤路徑寫 `g.walk = nil`；`finish` 也要 `&mut Game`。骨架的 `(w: &mut WalkAnim, finish: &mut dyn FnMut(i64))`
    /// 無法同時滿足，A 組先改成 walk 留在 `self.walk`、`finish` 收 `&mut Game`：segment 以
    /// `self.walk.as_mut()` 存取 Go 的 `w`，在 Go 呼叫 `finish(pose)` 的地方呼叫 `finish(self, pose)`。
    ///
    /// R4 B 組照上面的介面實作：Go 的 `w` 一律讀寫 `self.walk`，單位 `w.u` 是 `st.units` 的索引。
    /// `self.walk` 為空時（Go 不會發生：呼叫端剛取過 `g.walk`）直接回 false。
    pub(crate) fn step_battle_walk_segment(&mut self, finish: &mut dyn FnMut(&mut Game, i64)) -> bool {
        let Some(w) = self.walk.as_ref() else {
            return false;
        };
        let u = w.u;
        if w.path.len() < 2 || w.seg >= w.path.len() as i64 - 1 {
            // 沒有實際走過任何一格：保留目前姿態，不憑空補一次寫入。
            let mut pose = self.walk_unit(u).dir;
            if !(0..=3).contains(&pose) {
                pose = 0;
            }
            finish(self, pose);
            return false;
        }
        let (a, b) = (w.path[w.seg as usize], w.path[w.seg as usize + 1]);
        let pose = dir_toward(a.x, a.y, b.x, b.y);
        let (tile_w, tile_h) = {
            let m = self.m.as_ref().expect("Go：g.m 為 nil 時解參考 panic");
            (m.tile_w, m.tile_h)
        };
        let w = self.walk.as_mut().unwrap();
        w.tick += 1;
        let tick = w.tick;
        if tick <= NATIVE_MAP_GRID_MOTION_FRAMES {
            let unit = self.walk_unit_mut(u);
            (unit.x, unit.y) = (a.x, a.y);
            unit.set_map_pose(pose);
            unit.off_x = (b.x - a.x) as f64 * tile_w as f64 * tick as f64 / NATIVE_MAP_GRID_MOTION_FRAMES as f64;
            unit.off_y = (b.y - a.y) as f64 * tile_h as f64 * tick as f64 / NATIVE_MAP_GRID_MOTION_FRAMES as f64;
            unit.set_native_map_grid_motion(pose, tick);
            // 13185 redraws each motion before committing the next grid position.
            // Keep the work writer even when Update runs without a subsequent Draw.
            let (_, _, ok) = self.native_ch12_up_walk_frame();
            if ok && let Err(err) = self.compose_native_map_frame() {
                self.load_err = format!("native up-step redraw: {err}");
                self.walk = None;
            }
            return false;
        }
        {
            let unit = self.walk_unit_mut(u);
            if !unit.finish_native_map_grid_step(pose, b.x, b.y) {
                unit.set_map_placement(b.x, b.y, pose);
            }
        }
        // 走行步進家族在格邊界也搬視圖：絕對游標跟著單位走一格，可見游標或鏡頭
        // 二選一。只有玩家自己的走行才跟；AI 與劇情走位由各自的 owner 決定。
        let w = self.walk.as_ref().unwrap();
        if w.then.is_none() || w.follow_view {
            self.follow_native_map_cursor_step(a.x, a.y, b.x - a.x, b.y - a.y);
        }
        // 0x1300D 每提交一格就以新座標呼叫 0x13A44(x, y, selector0)，把格子事件 id 記進
        // [0x51A8F]，行動收尾才分派（第七章 r2：直線向上走進 (12,15) 也觸發 event 26）。
        // 全軍移動（selector 1）另有預演過的 owner，走原本的分段路徑。
        let Some(w) = self.walk.as_ref() else {
            // Go 的 `w` 仍指向原 job；`followNativeMapCursorStep` 不會清 `g.walk`，這裡只防禦。
            return false;
        };
        let selector = w.native_event_selector.unwrap_or(0);
        if selector == 0 {
            self.note_native_field_event_step(u, b.x, b.y);
        } else if b.x == a.x - 1 && b.y == a.y {
            let (event_id, bound) = battle::native_field_event_id_at(self.st.as_ref(), b.x, b.y, selector);
            let w = self.walk.as_ref().unwrap();
            let next_event = w.native_group_march_events.get(w.native_group_march_event as usize);
            let planned = w.native_group_march_event < w.native_group_march_events.len() as i64
                && next_event.is_some_and(|e| e.path_index == w.seg + 1);
            if bound != planned || (planned && next_event.is_some_and(|e| e.event_id != event_id)) {
                self.load_err = String::from("native system group-march event changed after atomic preflight");
                self.walk = None;
                self.native_system_group_march = None;
                return false;
            }
            if planned {
                let w = self.walk.as_mut().unwrap();
                w.seg += 1;
                w.tick = 0;
                w.native_group_march_paused = true;
                todo!("R8: action_overlay_runtime.go:beginNativeSystemGroupMarchFieldEvent（第一章沒有執行到）")
            }
        }
        let w = self.walk.as_mut().unwrap();
        w.seg += 1;
        w.tick = 0;
        if w.seg >= w.path.len() as i64 - 1 {
            // 原版在整段移動抵達時把 raw +3 姿態重設為 0、+4 動作重設為 0，
            // 不停在最後一格的行走方向。dosgolem 原版收據（10 萬指令粒度）：
            // 走第一格 pose=2／motion 1..6、第二格 pose=2／motion 重設後 1..6、
            // 抵達時 pose=0／motion=0 並保持到玩家下指令。
            finish(self, NATIVE_MAP_REST_POSE);
            return false;
        }
        // 還有下一段：原版跨格那一幀就是下一段的 `+4 = 1`，不會停在 0。
        true
    }

    /// R4 B 組新增：Go `w.u`（`*battle.Unit`）。走行單位一律在 `st.units` 裡；Go 在 `u` 為 nil 時解參考 panic。
    fn walk_unit(&self, u: Option<UnitIdx>) -> &battle::Unit {
        &self.st.as_ref().expect("Go：walk 單位在 g.st.Units").units[u.expect("Go：w.u 為 nil 時解參考 panic")]
    }

    /// R4 B 組新增：[`Game::walk_unit`] 的可寫版本。
    fn walk_unit_mut(&mut self, u: Option<UnitIdx>) -> &mut battle::Unit {
        &mut self.st.as_mut().expect("Go：walk 單位在 g.st.Units").units[u.expect("Go：w.u 為 nil 時解參考 panic")]
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
    ///
    /// 讀檔、JSON 與 PNG 的錯誤文字是 Rust 的（教訓 `rust-port-no-go-stdlib`）；接受與拒收與 Go 相同。
    /// Go 以 `SubImage` 切出共用像素的圖塊；Rust 複製成獨立的 `Picture`（像素相同）。
    pub(crate) fn load_map(&mut self, dir: &str) -> Result<()> {
        self.modern_map_tileset_loaded = false;
        let dir = asset_path(dir);
        let map_path = format!("{dir}/map.json");
        let raw = std::fs::read(&map_path).map_err(|e| game_err!("open {map_path}: {e}"))?;
        let m: MapData = serde_json::from_slice(&raw).map_err(|e| game_err!("{e}"))?;
        let png_path = format!("{dir}/tileset.png");
        let png_raw = std::fs::read(&png_path).map_err(|e| game_err!("open {png_path}: {e}"))?;
        let img = decode_png_picture(&png_raw).ok_or_else(|| GameError::new("image: unknown format"))?;
        let img = Arc::new(img);
        self.base_map_tileset = Some(img.clone());
        self.current_map_id = -1;
        // Go `filepath.Base(filepath.Clean(dir))`
        let base = std::path::Path::new(dir.trim_end_matches('/'))
            .file_name()
            .map(|b| b.to_string_lossy().into_owned())
            .unwrap_or_default();
        let mut tileset = img;
        if let Some(set) = self.modern_story_portraits.as_ref() {
            let mut map_id = -1;
            if base == "assets" {
                map_id = 0;
            } else if let Some(rest) = base.strip_prefix("map")
                && let Ok(parsed) = rest.parse::<i64>()
            {
                map_id = parsed;
            }
            if map_id >= 0 {
                self.current_map_id = map_id;
                if let Some(modern) = set.map_tilesets.get(&map_id) {
                    // Go map 值是 *ebiten.Image；nil 項目照樣採用（與 Go 相同）。
                    if let Some(modern) = modern.clone() {
                        tileset = modern;
                    }
                    self.modern_map_tileset_loaded = true;
                }
            }
        } else if base == "assets" {
            self.current_map_id = 0;
        } else if let Some(rest) = base.strip_prefix("map") {
            // Go `g.currentMapID, _ = strconv.Atoi(...)`：解析失敗寫 0。
            self.current_map_id = rest.parse::<i64>().unwrap_or(0);
        }
        self.tiles = Vec::new();
        let ts_w = tileset.width;
        let cols = if m.cols == 0 { ts_w / m.tile_w } else { m.cols };
        let n = (tileset.height / m.tile_h) * cols;
        for i in 0..n {
            let sx = (i % cols) * m.tile_w;
            let sy = (i / cols) * m.tile_h;
            self.tiles.push(Some(Arc::new(crop_picture(&tileset, sx, sy, m.tile_w, m.tile_h))));
        }
        self.tileset = Some(tileset);
        self.m = Some(m);
        // Keep the existing PNG map playable, but expose native resources only as
        // an all-or-nothing bundle. The indexed presentation bridge consumes this
        // field later; no partial resource is allowed to affect gameplay.
        self.native_map_assets = None;
        self.modern_hd_map_frame = None;
        self.modern_hd_map_terrain.clear();
        self.native_map_dac = Vec::new();
        self.native_palette_ramp = None;
        self.native_palette_pulse = None;
        self.native_ch28_post_present = None;
        if let Ok(Some(native)) = load_native_map_assets(&dir)
            && native_map_assets_available(Some(&native))
        {
            self.native_map_dac = native.palette_dac.clone();
            self.native_map_assets = Some(native);
        }
        Ok(())
    }
}

/// battleFPT 戰鬥演出播放速度(tick/幀):環境變數 FD2_BATTLE_FPT 可調(調慢=數字大),預設 3。
///
/// Go：`main.go:6664`，`func battleFPT() int`
pub(crate) fn battle_fpt() -> i64 {
    // Go `strconv.Atoi` 接受前置正負號與十進位數字；Rust `parse::<i64>` 同樣接受 `+`／`-`，超出 int 範圍兩者都拒收。
    if let Ok(v) = std::env::var("FD2_BATTLE_FPT").unwrap_or_default().parse::<i64>()
        && v > 0
    {
        return v;
    }
    3
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
        if !self.native_attack_presentation_available(atk_group, def_group) {
            return None;
        }
        let fpt = battle_fpt();
        let af = figani_index(atk_group) + 1;
        let frames_len = self.figani.get(&af).map_or(0, Vec::len);
        let delays = self.figani_delays.get(&af)?;
        if frames_len == 0 || delays.len() != frames_len {
            // Never invent a 15-frame animation or silently fall back to a fixed
            // delay: an unpaired PNG/export schedule is not an original attack.
            return None;
        }
        let df = figani_index(def_group);
        let def_frames_len = self.figani.get(&df).map_or(0, Vec::len);
        let def_delays = self.figani_delays.get(&df)?;
        if def_frames_len == 0 || def_delays.len() != def_frames_len {
            // The defender idle figure is a native presentation input too; do not
            // fall back to a guessed fixed-period breathing loop.
            return None;
        }
        let timeline = figani::DisplayScheduler::new(delays, fpt).ok()?;
        let body_ticks = timeline.body_ticks();
        let total = body_ticks + 4 * fpt; // 尾段停格 4 幀時間
        Some(AtkAnim {
            atk_fig: af,
            def_fig: figani_index(def_group),
            atk_name: atk_name.to_string(),
            def_name: def_name.to_string(),
            atk_hp,
            atk_max,
            atk_lv,
            atk_mp,
            atk_max_mp,
            def_lv,
            def_mp,
            def_max_mp,
            def_hp0,
            def_hp1,
            def_max,
            timer: total,
            total,
            fpt,
            terrain,
            atk_own,
            figani_timeline: Some(timeline),
            body_ticks,
            ..Default::default()
        })
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
        if !self.native_attack_presentation_available(counter_atk_group, counter_def_group) {
            return None;
        }
        let fpt = battle_fpt();
        let af = figani_index(counter_atk_group) + 1;
        let delays = self.figani_delays.get(&af)?;
        if delays.len() != self.figani.get(&af).map_or(0, Vec::len) {
            return None;
        }
        let timeline = figani::DisplayScheduler::new(delays, fpt).ok()?;
        let body_ticks = timeline.body_ticks();
        Some(AtkStage {
            atk_fig: af,
            def_fig: figani_index(counter_def_group),
            timeline: Some(timeline),
            body_ticks,
            total: body_ticks + 4 * fpt,
            def_hp0,
            def_hp1,
            def_max,
        })
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
        let (Some(a), Some(actor), Some(target), Some(counter)) = (a, actor, target, result.counter.as_ref()) else {
            return;
        };
        let st = self.st.as_ref().expect("Go：actor、target 指向 g.st.Units");
        let (actor_hp, actor_max_hp, actor_fig) =
            (st.units[actor].hp, st.units[actor].max_hp, st.units[actor].battle_fig);
        let before = actor_hp + counter.amount;
        a.atk_hp = before;
        a.counter = self.new_counter_stage(st.units[target].battle_fig, actor_fig, before, actor_hp, actor_max_hp);
    }

    /// nativeAttackPresentationAvailable 執行 newAtkAnim 所需的完整資產與排程檢查，
    /// 但不消耗戰鬥亂數，也不修改 HP；敵方 AI 以此作為失敗即關閉的交易預檢。
    ///
    /// Go：`main.go:6753`，`func (g *Game) nativeAttackPresentationAvailable(atkGroup, defGroup int) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_attack_presentation_available(&self, atk_group: i64, def_group: i64) -> bool {
        let fpt = battle_fpt();
        let af = figani_index(atk_group) + 1;
        let frames_len = self.figani.get(&af).map_or(0, Vec::len);
        let Some(delays) = self.figani_delays.get(&af) else {
            return false;
        };
        if frames_len == 0 || delays.len() != frames_len {
            return false;
        }
        let df = figani_index(def_group);
        let def_frames_len = self.figani.get(&df).map_or(0, Vec::len);
        let Some(def_delays) = self.figani_delays.get(&df) else {
            return false;
        };
        if def_frames_len == 0 || def_delays.len() != def_frames_len {
            return false;
        }
        figani::DisplayScheduler::new(delays, fpt).is_ok()
    }
}

/// attackPresentationTickMillis 是一個顯示 tick 的長度。原版 `sub_2939D` 對每一格
/// 重複 `cell+6` 次 `sub_17AA9(1)`，每次一個 BIOS tick；fpt 是重製端在一個原版延遲
/// 單位裡插幾個顯示 tick 的解析度，所以一個顯示 tick 就是 BIOS tick 除以 fpt。
///
/// Go：`main.go:6777`，`func attackPresentationTickMillis(fpt int) float64`
pub(crate) fn attack_presentation_tick_millis(fpt: i64) -> f64 {
    if fpt <= 0 {
        return indexedmap::PHASE_BANNER_STEP_MILLIS;
    }
    indexedmap::PHASE_BANNER_STEP_MILLIS / fpt as f64
}

/// attackPresentationTicks 把累積的毫秒換成這一畫格要推進幾個顯示 tick，並回傳
/// 還沒換掉的餘數。餘數要留著——丟掉的話 60 Hz 與 BIOS tick 的不整除會讓整段
/// 演出愈跑愈短。
///
/// Go：`main.go:6787`，`func attackPresentationTicks(accumMillis float64, fpt int) (ticks int, rest float64)`
pub(crate) fn attack_presentation_ticks(accum_millis: f64, fpt: i64) -> (i64, f64) {
    let step = attack_presentation_tick_millis(fpt);
    let (mut ticks, mut rest) = (0, accum_millis);
    while rest >= step {
        rest -= step;
        ticks += 1;
    }
    (ticks, rest)
}

/// errAttackPresentationYield 不是失敗：它表示這一幀要在演出收尾之後立刻交還，
/// 因為接手的事件流程（field event 61 或 battleEvent）已經接管。
///
/// Go：`main.go:6799`
/// Go `errors.New` 的哨兵錯誤；`errors.Is` 以 [`GameError::is`] 判斷。
pub(crate) static ERR_ATTACK_PRESENTATION_YIELD: GameError =
    GameError::sentinel("attack presentation yielded to an event");

/// Rust 新增（R4 A 組）：Go `ebiten.Termination`（`errors.New("regular termination")`）。`Update` 回傳它
/// 表示正常結束主迴圈（截圖完成、玩家選擇離開）；驅動端以 [`GameError::is`] 判斷。
pub(crate) static ERR_TERMINATION: GameError = GameError::sentinel("regular termination");

impl Game {
    /// stepAttackPresentationTick 推進一個**顯示 tick**（不是一個畫格）：走一步
    /// FIGANI 時間軸、判三段音效、時間到就收掉演出。
    ///
    /// Go：`main.go:6803`，`func (g *Game) stepAttackPresentationTick() error`
    ///
    /// `preludeDrawn`／`leadDrawn` 由呈現回報設定（Go 在 `Draw` 裡設，見 `PORTING.md`
    /// 「Draw 中改狀態的位置」）。
    pub(crate) fn step_attack_presentation_tick(&mut self) -> Result<()> {
        let Some(a) = self.atk.as_mut() else {
            return Ok(());
        };
        if let Some(scene) = a.native_scene.as_mut() {
            if scene.prelude_frame < scene.prelude_images.len() as i64 {
                if scene.prelude_drawn {
                    scene.prelude_drawn = false;
                    scene.prelude_frame += 1;
                }
                return Ok(());
            }
            if scene.body.is_some() {
                let millis = attack_presentation_tick_millis(a.fpt);
                // Go 把 a.nativeScene 指標交給 stepNativePhysicalBodyMillis；Rust 先取出，呼叫後若
                // 演出仍在就放回（演出被收掉時 Go 的 scene 也不再可達）。
                let mut scene = a.native_scene.take().unwrap();
                let result = self.step_native_physical_body_millis(&mut scene, millis);
                if let Some(a) = self.atk.as_mut()
                    && a.native_scene.is_none()
                {
                    a.native_scene = Some(scene);
                }
                return result;
            }
            if scene.lead_frame < scene.lead_images.len() as i64 {
                if !scene.lead_drawn {
                    return Ok(());
                }
                scene.lead_ticks += 1;
                let mut wait = 1;
                if scene.lead_frame < scene.departure.len() as i64 {
                    wait = scene.lead_delays[scene.lead_frame as usize].max(1);
                    wait *= a.fpt;
                    scene.lead_body_ticks += 1;
                }
                if scene.lead_ticks >= wait {
                    scene.lead_frame += 1;
                    (scene.lead_ticks, scene.lead_drawn) = (0, false);
                    if scene.lead_frame == scene.lead_images.len() as i64 {
                        // E1 尾段從 header2 接續；首次 departure 不再重播。
                        for _ in 0..scene.lead_body_ticks {
                            let Some(timeline) = a.figani_timeline.as_mut() else {
                                return Err(GameError::new("native physical departure timeline unavailable"));
                            };
                            timeline.step()?;
                        }
                        a.timer -= scene.lead_body_ticks;
                        a.frame_index = scene.attack_start;
                    }
                }
                return Ok(());
            }
        }
        a.timer -= 1;
        let prog = a.total - a.timer;
        if prog <= a.body_ticks
            && let Some(timeline) = a.figani_timeline.as_mut()
        {
            match timeline.step() {
                Ok((frame, presented, _)) => {
                    if presented {
                        a.frame_index = frame;
                    }
                }
                Err(err) => {
                    // The timeline was validated at load time; a runtime failure must
                    // not run the action continuation with an unpresented frame.
                    a.after = None;
                    self.atk = None;
                    let err = GameError::from(err);
                    self.load_err = format!("FIGANI attack timeline: {}", err.message());
                    return Err(err);
                }
            }
        }
        let a = self.atk.as_mut().unwrap();
        let (fpt, timer, def_hp1, atk_fig) = (a.fpt, a.timer, a.def_hp1, a.atk_fig);
        if fpt > 0 {
            // 三段音效(== 比對每 tick 遞增的 prog,各觸發一次)
            let frames = self.figani.get(&atk_fig).map_or(0, |f| f.len() as i64);
            let swing_at = (frames - 4) * fpt;
            if prog == swing_at {
                // 揮擊(蓄力揮出)
                let sample = self.sfx_swing.clone();
                self.play_raw(&sample);
            } else if prog == swing_at + 3 * fpt {
                // 命中(劈中、守方 HP 抽乾)
                let sample = self.sfx_impact.clone();
                self.play_raw(&sample);
            }
        }
        if timer == fpt && def_hp1 <= 0 {
            // 收勢那幀:守方陣亡音
            let sample = self.sfx_death.clone();
            self.play_raw(&sample);
        }
        if timer <= 0 {
            if let Some(a) = self.atk.as_mut().filter(|a| a.counter.is_some()) {
                a.begin_counter_stage();
                return Ok(());
            }
            self.finish_attack_presentation();
            if self.native_field_event61.is_some() || self.battle_event.is_some() {
                return Err(ERR_ATTACK_PRESENTATION_YIELD.clone());
            }
        }
        Ok(())
    }

    /// Go：`main.go:6918`，`func (g *Game) finishAttackPresentation()`
    ///
    /// Go 對演出圖片呼叫 `Dispose()`；Rust 的畫面由 `Arc` 管理，丟棄即釋放。
    pub(crate) fn finish_attack_presentation(&mut self) {
        let Some(a) = self.atk.as_ref() else {
            return;
        };
        if let Some(scene) = a.native_scene.as_ref() {
            let return_ramp = scene.body.as_ref().is_some_and(|b| b.return_ramp_started);
            // 正常原生演出在11CAC(1)／caller續行前明確清除VGA。
            // 契約見fd2_physical_background_selection_20261004.json的physical_map_return_spec；
            // 不推定重新malloc的地圖work初值。
            self.native_map_vga.fill(0); // 0x290AC..0x290BD: memset(0xA0000,0,0xFA00)。
            // 290C2的11CAC(1)在290D4停止音效之前返回。
            // nil bundle保留既有PNG相容路徑；已宣告bundle不得略過合成失敗。
            if self.native_map_assets.is_some()
                && let Err(err) = self.compose_native_map_frame_for_physical_return()
            {
                self.stop_native_physical_sound();
                self.atk = None;
                self.load_err = format!("native physical map return: {}", err.message());
                return;
            }
            self.stop_native_physical_sound();
            if return_ramp && self.native_map_assets.is_some() {
                // 2910A→1F525在重建map後以delta64..0漸亮；最後Draw才返回caller。
                // READY：physical_owner_return_dac_spec。孤立map helper未走body，不外推owner。
                let then: Box<dyn FnOnce(&mut Game) + Send> = Box::new(|g: &mut Game| g.complete_attack_presentation());
                if let Err(err) = self.start_native_palette_ramp(64, 0, 2, Some(then)) {
                    self.atk = None;
                    self.load_err = format!("native physical return fade-in: {}", err.message());
                }
                return;
            }
        }
        self.complete_attack_presentation();
    }

    /// Go：`main.go:6962`，`func (g *Game) completeAttackPresentation()`
    pub(crate) fn complete_attack_presentation(&mut self) {
        let Some(mut a) = self.atk.take() else {
            return;
        };
        if let Some(after) = a.after.take() {
            after(self);
        }
    }
}

impl AtkAnim {
    /// beginCounterStage 把演出換成反擊那一段：資源與 HP 變化換成第二次 `sub_2939D`
    /// 的參數，資訊條左右對調（陣營換邊），時間軸重新開始。畫面不重新滑入——原版
    /// 的 `sub_29164` 整段只跑一次。
    ///
    /// Go：`main.go:6894`，`func (a *atkAnim) beginCounterStage()`（`r8`：第一章沒有反擊；
    /// `stepAttackPresentationTick` 需要它，隨呼叫端補移植）。
    fn begin_counter_stage(&mut self) {
        let Some(next) = self.counter.take() else {
            return;
        };
        if let Some(scene) = self.native_scene.as_mut() {
            scene.counter_active = true;
        }
        (self.atk_fig, self.def_fig) = (next.atk_fig, next.def_fig);
        std::mem::swap(&mut self.atk_name, &mut self.def_name);
        std::mem::swap(&mut self.atk_lv, &mut self.def_lv);
        std::mem::swap(&mut self.atk_mp, &mut self.def_mp);
        std::mem::swap(&mut self.atk_max_mp, &mut self.def_max_mp);
        // 第二段的攻方就是上一段的守方，HP 停在上一段結算後的值。
        (self.atk_hp, self.atk_max) = (self.def_hp1, self.def_max);
        (self.def_hp0, self.def_hp1, self.def_max) = (next.def_hp0, next.def_hp1, next.def_max);
        self.atk_own = !self.atk_own;
        self.figani_timeline = next.timeline;
        (self.body_ticks, self.total, self.timer) = (next.body_ticks, next.total, next.total);
        self.frame_index = 0;
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
    fig.wrapping_mul(3)
}

/// loadSprites 載入 assets/sprites/fig_NNN_fMM.png,按 fig index 分組成幀序列。
///
/// Go：`main.go:6981`，`func loadSprites() map[int][]*ebiten.Image`
pub(crate) fn load_sprites() -> BTreeMap<i64, Vec<Option<Arc<Picture>>>> {
    load_numbered_frames("assets/sprites/fig_*_f*.png", "fig_", "_f")
}

/// Go `fmt.Sscanf(name, "<prefix>%d<mid>%d.png", &a, &b)` 的子集：`%d` 是可帶正負號的十進位整數，
/// 字面部分逐字比對（Go 不檢查格式之後多出的字元，這裡相同）。
fn scan_numbered_frame(name: &str, prefix: &str, mid: &str) -> Option<(i64, i64)> {
    fn int(s: &str) -> Option<(i64, &str)> {
        let sign = usize::from(s.starts_with(['+', '-']));
        let digits = s[sign..].bytes().take_while(u8::is_ascii_digit).count();
        if digits == 0 {
            return None;
        }
        Some((s[..sign + digits].parse().ok()?, &s[sign + digits..]))
    }
    let (a, rest) = int(name.strip_prefix(prefix)?)?;
    let (b, rest) = int(rest.strip_prefix(mid)?)?;
    rest.starts_with(".png").then_some((a, b))
}

/// Rust 新增：`loadSprites`／`loadPortraits`／`loadFIGANI` 共用的本體（Go 三個函式除了樣式與
/// Sscanf 格式之外逐行相同，`main.go:6981`、`7017`、`7053`）：讀不到或解不開的檔案略過，依
/// (編號, 幀) 排序後分組。
fn load_numbered_frames(pattern: &str, prefix: &str, mid: &str) -> BTreeMap<i64, Vec<Option<Arc<Picture>>>> {
    let mut frames: Vec<(i64, i64, Arc<Picture>)> = Vec::new();
    for fp in asset_glob(pattern) {
        let base = std::path::Path::new(&fp).file_name().map(|b| b.to_string_lossy().into_owned()).unwrap_or_default();
        let Some((idx, fno)) = scan_numbered_frame(&base, prefix, mid) else {
            continue;
        };
        let Ok(raw) = std::fs::read(&fp) else {
            continue;
        };
        let Some(im) = decode_png_picture(&raw) else {
            continue;
        };
        frames.push((idx, fno, Arc::new(im)));
    }
    // Go `sort.Slice` 不穩定；鍵重複（同編號同幀）時 Go 的先後不定，Rust 用穩定排序保留檔名順序。
    frames.sort_by_key(|f| (f.0, f.1));
    let mut out: BTreeMap<i64, Vec<Option<Arc<Picture>>>> = BTreeMap::new();
    for (idx, _, img) in frames {
        out.entry(idx).or_default().push(Some(img));
    }
    out
}

/// loadPortraits 載入 assets/portraits/DATO_NNN_mM.png,按肖像 id 分組成 4 嘴型幀。
///
/// Go：`main.go:7017`，`func loadPortraits() map[int][]*ebiten.Image`
pub(crate) fn load_portraits() -> BTreeMap<i64, Vec<Option<Arc<Picture>>>> {
    load_numbered_frames("assets/portraits/DATO_*_m*.png", "DATO_", "_m")
}

/// loadFIGANI 載入 assets/figani/FIGANI_NNN_fNN.png,按 fig id 分組成攻擊全身分鏡。
///
/// Go：`main.go:7053`，`func loadFIGANI() map[int][]*ebiten.Image`
pub(crate) fn load_figani() -> BTreeMap<i64, Vec<Option<Arc<Picture>>>> {
    load_numbered_frames("assets/figani/FIGANI_*_f*.png", "FIGANI_", "_f")
}

/// loadFIGANIDelays loads descriptor +6 values exported from the fixed
/// player-provided FIGANI.DAT. It is deliberately separate from the PNG
/// presentation assets: a missing, malformed, or mismatched schedule must not
/// be replaced with a guessed frame count or delay.
///
/// Go：`main.go:7092`，`func loadFIGANIDelays() (map[int][]int, error)`
///
/// Go 以 `range encoded`（map，順序隨機）核對；Rust 依鍵的字串排序。只有多筆同時無效時回報哪一筆
/// 會不同（KNOWN_DIFFERENCES `figani-delays-map-order`）。
pub(crate) fn load_figani_delays() -> Result<BTreeMap<i64, Vec<i64>>> {
    let path = asset_path("assets/figani/delays.json");
    let raw = std::fs::read(&path).map_err(|e| game_err!("open {path}: {e}"))?;
    let encoded: Option<BTreeMap<String, Option<Vec<i64>>>> =
        serde_json::from_slice(&raw).map_err(|e| game_err!("{e}"))?;
    let mut out = BTreeMap::new();
    for (key, delays) in encoded.unwrap_or_default() {
        // Go `strconv.Atoi`：可帶正負號的十進位。
        let id = match key.parse::<i64>() {
            Ok(id) if id >= 0 => id,
            _ => return Err(game_err!("invalid FIGANI delay resource {}", battle::go_quote(&key))),
        };
        let delays = delays.unwrap_or_default();
        if delays.is_empty() {
            return Err(game_err!("FIGANI {id} has no frame delays"));
        }
        for (frame, &delay) in delays.iter().enumerate() {
            if delay < 0 {
                return Err(game_err!("FIGANI {id} frame {frame} has invalid delay {delay}"));
            }
        }
        out.insert(id, delays);
    }
    if out.is_empty() {
        return Err(GameError::new("FIGANI delay table is empty"));
    }
    Ok(out)
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
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct LegacyScene {
        #[serde(deserialize_with = "battle::null_as_default")]
        label: String,
        #[serde(deserialize_with = "battle::null_as_default")]
        lines: Vec<campaign::Line>,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct LegacyFile {
        #[serde(deserialize_with = "battle::null_as_default")]
        scenes: Vec<LegacyScene>,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Legacy {
        speaker: Option<i64>,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Extensions {
        #[serde(deserialize_with = "battle::null_as_default")]
        legacy: Legacy,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct CanonicalLine {
        #[serde(deserialize_with = "battle::null_as_default")]
        line_id: String,
        #[serde(deserialize_with = "battle::null_as_default")]
        text: String,
        #[serde(deserialize_with = "battle::null_as_default")]
        extensions: Extensions,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct CanonicalScene {
        #[serde(deserialize_with = "battle::null_as_default")]
        lines: Vec<CanonicalLine>,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct CanonicalFile {
        #[serde(deserialize_with = "battle::null_as_default")]
        scenes: Vec<CanonicalScene>,
    }
    let legacy_path = asset_path(path);
    let raw = std::fs::read(&legacy_path).map_err(|e| game_err!("open {legacy_path}: {e}"))?;
    let mut f: LegacyFile = serde_json::from_slice(&raw).map_err(|e| game_err!("{e}"))?;
    let base = std::path::Path::new(path).file_name().map(|b| b.to_string_lossy().into_owned()).unwrap_or_default();
    let canonical_path = join_path(&["assets", "editor-canonical", "story", &base]);
    let quoted = battle::go_quote(&canonical_path);
    let canonical_raw = std::fs::read(asset_path(&canonical_path))
        .map_err(|e| game_err!("canonical story {quoted}: open {}: {e}", asset_path(&canonical_path)))?;
    let canonical: CanonicalFile =
        serde_json::from_slice(&canonical_raw).map_err(|e| game_err!("canonical story {quoted}: {e}"))?;
    let story = battle::go_quote(path);
    if f.scenes.len() != canonical.scenes.len() {
        return Err(game_err!("story {story} scene count {} != canonical {}", f.scenes.len(), canonical.scenes.len()));
    }
    for (scene_number, (legacy_scene, canonical_scene)) in f.scenes.iter_mut().zip(&canonical.scenes).enumerate() {
        let (legacy_lines, canonical_lines) = (&mut legacy_scene.lines, &canonical_scene.lines);
        if legacy_lines.len() != canonical_lines.len() {
            return Err(game_err!(
                "story {story} scene {scene_number} line count {} != canonical {}",
                legacy_lines.len(),
                canonical_lines.len()
            ));
        }
        for (line_number, (line, identity)) in legacy_lines.iter_mut().zip(canonical_lines).enumerate() {
            if identity.line_id.is_empty()
                || identity.text != line.text
                || identity.extensions.legacy.speaker != Some(line.speaker)
            {
                return Err(game_err!(
                    "story {story} scene {scene_number} line {line_number} canonical identity mismatch"
                ));
            }
            line.line_id = identity.line_id.clone();
        }
    }
    if let Some(&index) = scene_index {
        if index < 0 || index >= f.scenes.len() as i64 {
            return Err(game_err!("story {story} scene index {index} out of range"));
        }
        return Ok(std::mem::take(&mut f.scenes[index as usize].lines));
    }
    if scene.is_empty() {
        return Ok(f.scenes.into_iter().flat_map(|sc| sc.lines).collect());
    }
    match f.scenes.into_iter().find(|sc| sc.label == scene) {
        Some(sc) => Ok(sc.lines),
        None => Err(game_err!("story {story} scene {} not found", battle::go_quote(scene))),
    }
}

/// handlerStoryPath converts a StoryIndexMap script path (relative to
/// assets/story) into the normal asset lookup path.  Existing authored beats
/// may already carry assets/... or an absolute path, which stays untouched.
///
/// Go：`main.go:7240`，`func handlerStoryPath(script string) string`
///
/// R4 B 組：`filepath.Join` 會詞法整理（Clean）路徑，Rust 用 A 組的 `join_path`（只略過空段）；
/// StoryIndexMap 的 script 都是單純相對檔名（KNOWN_DIFFERENCES `handler-story-path-no-clean`）。
pub(crate) fn handler_story_path(script: &str) -> String {
    // Go 在 linux 上 `filepath.ToSlash` 不改字串、`filepath.IsAbs` 看開頭的 `/`。
    if script.starts_with('/') || script.starts_with("assets/") {
        return script.to_string();
    }
    join_path(&["assets", "story", script])
}

/// loadFigMeta 載入 FIGANI 每幀內嵌絕對螢幕座標 (dx,dy)@320(assets/figani/meta.json;doc06:
/// 幀標頭 +0/+2,動畫的走位/伸擊/突刺全靠逐幀 (dx,dy) 變化,引擎不需錨點/位移計算)。
///
/// Go：`main.go:7249`，`func loadFigMeta() map[int][][2]int`
///
/// Go 的 `[2]int` 接受長度不足（補零）或過長（截斷）的 JSON 陣列；Rust 只接受長度 2，其他整份視為
/// 解碼失敗（KNOWN_DIFFERENCES `figani-meta-pair-length`，正式匯出檔都是長度 2）。
pub(crate) fn load_fig_meta() -> BTreeMap<i64, Vec<[i64; 2]>> {
    let mut out = BTreeMap::new();
    let Ok(raw) = std::fs::read(asset_path("assets/figani/meta.json")) else {
        return out;
    };
    let Ok(m) = serde_json::from_slice::<Option<BTreeMap<String, Option<Vec<[i64; 2]>>>>>(&raw) else {
        return out;
    };
    for (k, v) in m.unwrap_or_default() {
        if let Ok(id) = k.parse::<i64>() {
            out.insert(id, v.unwrap_or_default());
        }
    }
    out
}

/// Go：`main.go:7293`，`func absInt(v int) int`
pub(crate) fn abs_int(v: i64) -> i64 {
    if v < 0 {
        return v.wrapping_neg();
    }
    v
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
        let (Some(actor_idx), Some(target_idx)) = (actor, target) else {
            return Err(game_err!("physical attack context unavailable"));
        };
        if self.st.is_none() {
            return Err(game_err!("physical attack context unavailable"));
        }
        if self.rng.is_none() {
            return Err(game_err!("physical attack RNG unavailable"));
        }
        // 傷害、反擊與原生單位的經驗／升級成長都走原版的全域 `0x627B8`
        // （g.nativeRNGState）；g.rng 只剩沒有原版 `+6` 的舊可編輯單位在用。
        let st = self.st.as_mut().unwrap();
        let initial_acted_bit = st.units[actor_idx].native_record_byte5 & 0x80;
        let death_before = capture_native_action_death_flags(st, &[actor, target]);
        let rng = self.rng.as_mut().map(|rng| rng as &mut dyn battle::GoRand);
        let result = st.attack_native_physical_with_experience(actor, target, self.native_rng_state, rng)?;
        self.queue_native_action_death_presentation(&death_before);
        // 純結算先標行動完成，正式 UI 則由 0x13512 的既有尾端 owner 發布 bit7。
        // 保留入口位元直到死亡事件／掉落對話結束，避免 delay 期間重繪提前變灰。
        let (sel, ai_busy) = (self.sel, self.ai_busy);
        let unit = &mut self.st.as_mut().unwrap().units[actor_idx];
        if sel == Some(actor_idx) && !ai_busy && unit.has_native_record_byte5 {
            unit.native_record_byte5 = unit.native_record_byte5 & !0x80 | initial_acted_bit;
        }
        self.native_rng_state = result.rng_state;
        // 0x1196D／0x1566A：兩段交鋒結束後才 0x1E292；經驗非零就有對話（行動收尾時播）。
        self.queue_native_level_up_dialogue(Some(actor_idx), result.attack.exp_gained as i64, &result.attack.level_ups);
        if result.counter_exp_gained > 0 {
            self.queue_native_level_up_dialogue(Some(target_idx), result.counter_exp_gained, &result.counter_level_ups);
        }
        Ok(result)
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
        if !self.debug {
            return;
        }
        self.msg = message.to_string();
    }
}

/// Go：`main.go:7367`，`func playerPhysicalAttackMessage(catalog *localization.Catalog, actor, target *battle.Unit, result battle.AttackResult) (string, error)`
pub(crate) fn player_physical_attack_message(
    catalog: Option<&localization::Catalog>,
    actor: Option<&battle::Unit>,
    target: Option<&battle::Unit>,
    result: &battle::AttackResult,
) -> Result<String> {
    let (Some(catalog), Some(actor), Some(target)) = (catalog, actor, target) else {
        return Err(game_err!("physical attack message context unavailable"));
    };
    let mut actor_name = actor.name.clone();
    if actor_name.is_empty() {
        actor_name = actor.cls_name.clone();
    }
    let mut target_name = target.name.clone();
    if target_name.is_empty() {
        target_name = target.cls_name.clone();
    }
    if actor_name.is_empty() || target_name.is_empty() {
        return Err(game_err!("physical attack message name unavailable"));
    }
    use localization::FormatArg;
    if result.missed {
        return Ok(catalog.format("battle.attack.miss", &[FormatArg::Str(actor_name), FormatArg::Str(target_name)])?);
    }
    let mut message = catalog.format(
        "battle.attack.hit",
        &[FormatArg::Str(actor_name), FormatArg::Str(target_name), FormatArg::Int(result.amount)],
    )?;
    if result.crit {
        message += &catalog.format("battle.attack.critical_suffix", &[])?;
    }
    if result.exp_gained > 0.0 {
        message += &catalog.format("battle.attack.exp_suffix", &[FormatArg::Float(result.exp_gained)])?;
    }
    Ok(message)
}

/// dirToward 從 (ax,ay) 朝 (tx,ty) 的方向:0下 1左 2上 3右(FDICON 方向幀)。
///
/// Go：`main.go:7407`，`func dirToward(ax, ay, tx, ty int) int`
pub(crate) fn dir_toward(ax: i64, ay: i64, tx: i64, ty: i64) -> i64 {
    let (dx, dy) = (tx.wrapping_sub(ax), ty.wrapping_sub(ay));
    if abs_int(dx) > abs_int(dy) {
        if dx > 0 {
            return 3;
        }
        return 1;
    }
    if dy > 0 {
        return 0;
    }
    2
}

impl Game {
    /// confirm 處理 Enter/Space:選取我方單位顯示移動範圍,或移動到可達格 / 原地待機。
    ///
    /// Go：`main.go:7422`，`func (g *Game) confirm()`
    pub(crate) fn confirm(&mut self) {
        if self.st.is_none() || self.walk.is_some() || self.ai_busy {
            return;
        }
        self.play_sfx(SFX_CONFIRM);
        let cur = battle::Cell { x: self.cur_x, y: self.cur_y };
        let Some(sel) = self.sel else {
            // 選我方單位
            let u = self.st.as_ref().unwrap().unit_at(self.cur_x, self.cur_y);
            self.consume_native_continue_opening_confirm();
            if let Some(u) = u
                && self.st.as_ref().unwrap().units[u].has_native_record_byte6
            {
                // 0x118EE：0x12C0D 返回單位索引即清零，早於檢視／行動 gate。
                self.st.as_mut().unwrap().native_experience_accumulator = 0;
            }
            if let Some(u) = u
                && self.st.as_ref().unwrap().has_native_map_view_state
            {
                let (handled, err) = match self.inspect_native_player_unit(u, false) {
                    Ok(handled) => (handled, false),
                    Err(err) => {
                        self.load_err = err.message().to_string();
                        // Go 失敗時回傳 (true, err)。
                        (true, true)
                    }
                };
                if handled || err {
                    return;
                }
            }
            let Some(u) = u else {
                if self.native_system_overlay_ready() {
                    // 0x117E7 在 0x12C0D 回傳 -1 時直接呼叫 0x16F55；這是共用
                    // 玩家控制器，不是 chapter0 CONTINUE 特例。ActionOverlayOrigin
                    // 以 visible map cursor 為基準，不借用選取單位座標。
                    self.native_system_cursor_overlay = true;
                    // 0x16f55 將 action selector 初始化為 0；此值只保留輸入游標
                    // 的原始順序，並不替四格指派動作語意。
                    self.begin_action_overlay_open(0);
                }
                return;
            };
            let unit = &self.st.as_ref().unwrap().units[u];
            if unit.camp == battle::OWN && unit.paralyzed {
                if let (message, true) = self.locale_message("battle.unit.paralyzed", &[]) {
                    self.msg = message;
                }
                return;
            }
            if unit.camp == battle::OWN && !unit.acted {
                let (ux, uy) = (unit.x, unit.y);
                self.sel = Some(u);
                self.native_field_event_pending = None; // 0x188BF：[0x51A8F] = 0xff
                (self.sel_orig_x, self.sel_orig_y) = (ux, uy); // 記移動前位置(ESC 取消退回,playfix #4)
                self.moved = false;
                self.reach = self.st.as_ref().unwrap().reachable(u);
                if self.st.as_ref().unwrap().has_native_map_view_state
                    && native_map_assets_available(self.native_map_assets.as_ref())
                    && let Err(err) = self.prepare_native_player_movement(u)
                {
                    (self.sel, self.reach) = (None, BTreeMap::new());
                    self.load_err = format!("native player movement: {}", err.message());
                }
            }
            return;
        };
        if self.cast_sp.is_some() {
            todo!("R8: main.go:7475 施法目標確認（CastArea、localizedSpellResult；第一章重播沒有執行到）");
        }
        if self.native_item_targeting {
            let item_id = self.native_item_target_id;
            let target = self.st.as_ref().and_then(|st| st.unit_at(self.cur_x, self.cur_y));
            let applied = match target.map(|t| self.apply_native_target_item(t)) {
                Some(Ok(applied)) => applied,
                Some(Err(err)) => {
                    self.msg = format!("物品 {item_id:02X}h：{}", err.message());
                    return;
                }
                None => false,
            };
            if !applied {
                if self.native_item_relocating
                    && let (message, true) =
                        self.locale_message("battle.item.choose_destination", &[localization::FormatArg::Int(item_id)])
                {
                    self.msg = message;
                }
                return;
            }
            let key = if self.native_ai_item_presentation.is_some() {
                "battle.item.damage_presenting"
            } else {
                "battle.item.recovery_complete"
            };
            if let (message, true) = self.locale_message(key, &[localization::FormatArg::Int(item_id)]) {
                self.msg = message;
            }
            return;
        }
        if self.native_item_relocating {
            todo!("R8: main.go:7562 物品／傳送術目的地確認（applyNativeRelocationDestination；第一章重播沒有執行到）");
        }
        if self.native_command0_targeting {
            let Some((id, target)) = self.native_command_target_confirmation() else {
                return;
            };
            // 各指令的玩家演出入口（Go `main.go:7631`..`8193`）逐一補移植；尚未接上的指令與 Go 執行失敗時相同，
            // 留在選目標並顯示原因，Esc 回指令格。
            if !self.start_native_player_command(id, target) {
                self.msg = format!("原始指令 {id}：玩家演出尚未移植");
            }
            return;
        }
        if !self.moved {
            // 移動階段
            let unit = &self.st.as_ref().unwrap().units[sel];
            if self.cur_x == unit.x && self.cur_y == unit.y {
                // 原地 → 不移動,開指令環
                self.clear_native_player_movement();
                self.moved = true;
                self.reach = BTreeMap::new();
                self.begin_battle_action_overlay();
            } else if self.reach.get(&cur).copied().unwrap_or(false)
                && self.st.as_ref().unwrap().unit_at(self.cur_x, self.cur_y).is_none()
            {
                // 移動到可達空格:沿路徑逐格走
                let mut p = self.st.as_ref().unwrap().path(sel, self.cur_x, self.cur_y);
                if let Some(plan) = self.native_move_plan.as_ref() {
                    p = match plan.path(cur) {
                        Ok(p) => p,
                        Err(err) => {
                            self.load_err = err.message().to_string();
                            return;
                        }
                    };
                    self.reset_native_target_field();
                    self.native_move_panel = None;
                }
                if p.len() >= 2 {
                    self.walk = Some(WalkAnim { u: Some(sel), path: p, ..Default::default() });
                    // 原版確認之後游標回到單位所在格：0x18960 先存下游標，
                    // 0x18A26 以那個值呼叫 0x12CEA，逐格走回去。走的是鍵盤處理器，所以
                    // 可見游標與鏡頭一起動——收據 fd2-move-confirm-cursor-20260909 的 cp0071
                    // 量到游標 14→16 的同時可見游標 1→2（第三格由接著開始的走行
                    // 步進先扣掉），正是「成對搬動」的形狀。0x18A14 先把 [0x51A83] 寫 0，
                    // 0x18A2E 才寫回 1：走回去的每一格只動可見游標時不重繪，HUD anchor
                    // 不評估（第六章 r3 seq 817：走回 (7,23) 可見 (2,6) 沒把小窗翻到右邊）。
                    if let Some(st) = self.st.as_mut() {
                        st.materialize_native_map_range_mode(0);
                    }
                    self.focus_native_map_cursor_on_unit(Some(sel));
                    if let Some(st) = self.st.as_mut() {
                        st.materialize_native_map_range_mode(1);
                    }
                } else {
                    // 理論上不會(reach 內必可達),保底瞬移
                    let unit = &mut self.st.as_mut().unwrap().units[sel];
                    let dir = unit.dir;
                    unit.set_map_placement(self.cur_x, self.cur_y, dir);
                    self.moved = true;
                    self.begin_battle_action_overlay();
                }
                self.reach = BTreeMap::new();
            }
            return;
        }
        // 原生0x18F76共用0x115B6確認閘門。自己格不能在攻擊目標階段直接待機。
        if self.st.as_ref().unwrap().has_native_map_view_state && !self.native_physical_attack_confirmation_allowed() {
            return;
        }
        // 攻擊階段：合法敵方格進攻擊；無raw來源的相容場景保留原地待機。
        let st = self.st.as_ref().unwrap();
        let target = st.unit_at(self.cur_x, self.cur_y).filter(|&tgt| {
            tgt != sel && st.units[tgt].camp != battle::OWN && st.in_attack_range(sel, self.cur_x, self.cur_y)
        });
        if let Some(tgt) = target {
            // 0x115b6 確認回來，0x18f86 的 0x4dbfc 先於攻擊演出清掉射程標記。
            self.reset_native_target_field();
            // The original transaction requires both attack and defender-idle FIGANI
            // inputs. Preflight the complete separated pair before direction, RNG, HP,
            // EXP or acted state changes; a missing presentation must not become a
            // successful invisible attack.
            let (sel_fig, tgt_fig) = {
                let units = &self.st.as_ref().unwrap().units;
                (units[sel].battle_fig, units[tgt].battle_fig)
            };
            if let Err(err) = self.ensure_native_attack_presentation(sel_fig, tgt_fig) {
                self.load_err =
                    format!("FIGANI attack presentation unavailable: {sel_fig} -> {tgt_fig}: {}", err.message());
                return;
            }
            let scene = match self.prepare_native_physical_scene(Some(sel), tgt) {
                Ok(scene) => scene,
                Err(err) => {
                    self.load_err = format!("physical attack scene unavailable: {}", err.message());
                    return;
                }
            };
            // 攻擊者面向目標(FDICON 方向幀)
            let (cur_x, cur_y) = (self.cur_x, self.cur_y);
            let units = &mut self.st.as_mut().unwrap().units;
            let pose = dir_toward(units[sel].x, units[sel].y, cur_x, cur_y);
            units[sel].set_map_pose(pose);
            let target_unit = &units[tgt];
            let mut nm = target_unit.name.clone();
            if nm.is_empty() {
                nm = target_unit.cls_name.clone();
            }
            let mut anm = units[sel].name.clone();
            if anm.is_empty() {
                anm = units[sel].cls_name.clone();
            }
            let def_hp0 = target_unit.hp;
            let full_result = match self.resolve_physical_attack_full(Some(sel), Some(tgt)) {
                Ok(result) => result,
                Err(err) => {
                    self.msg = format!("攻擊：{}", err.message());
                    return;
                }
            };
            let attack_result = full_result.attack.clone();
            self.award_death_reward(Some(tgt), Some(sel));
            let units = &self.st.as_ref().unwrap().units;
            let message = player_physical_attack_message(
                self.locale_catalog.as_ref(),
                Some(&units[sel]),
                Some(&units[tgt]),
                &attack_result,
            );
            let message = match message {
                Ok(message) => message,
                Err(err) => {
                    self.load_err = format!("physical attack locale: {}", err.message());
                    return;
                }
            };
            self.publish_physical_attack_message(&message);
            let actor = sel;
            let units = &self.st.as_ref().unwrap().units;
            let (a, t) = (&units[actor], &units[tgt]);
            let terrain = self.terrain_at(cur_x, cur_y);
            self.atk = self.new_atk_anim(
                a.battle_fig,
                t.battle_fig,
                &anm,
                &nm,
                a.hp,
                a.max_hp,
                a.lv,
                a.mp,
                a.max_mp,
                t.lv,
                t.mp,
                t.max_mp,
                def_hp0,
                t.hp,
                t.max_hp,
                terrain,
                true,
            );
            // 反擊接成同一段演出的第二半，與原版一樣不重新滑入。Go 的 `g.atk` 與這裡的 `a` 是同一個指標；
            // 兩個被呼叫端都不讀 `g.atk`，Rust 先取出、改完再放回。
            let mut atk = self.atk.take();
            self.attach_counter_presentation(atk.as_mut(), Some(actor), Some(tgt), &attack_result);
            if let Some(mut a) = atk {
                a.native_scene = scene;
                if let Err(err) = self.attach_native_physical_body(a.native_scene.as_mut(), &full_result) {
                    self.load_err = format!("physical attack body: {}", err.message());
                    self.atk = None;
                    return;
                }
                a.after = Some(Box::new(move |g: &mut Game| {
                    g.finish_successful_unit_action(Some(actor), None);
                }));
                self.atk = Some(a);
            } else {
                // The complete pair was preflighted above. Reaching this branch means
                // the cache contract changed between settlement and player creation;
                // stop rather than pretending the invisible presentation succeeded.
                let units = &self.st.as_ref().unwrap().units;
                self.load_err = format!(
                    "FIGANI attack presentation disappeared after preflight: {} -> {}",
                    units[actor].battle_fig, units[tgt].battle_fig
                );
                return;
            }
            (self.sel, self.reach, self.moved) = (None, BTreeMap::new(), false);
            self.check_result();
        } else if self.cur_x == st.units[sel].x && self.cur_y == st.units[sel].y {
            // 原地待命
            self.reset_native_target_field();
            self.finish_selected_wait();
        }
    }

    /// checkResult 檢查勝負(失敗條件:索爾死;勝利:敵全滅,doc28 第1章)。
    ///
    /// Go：`main.go:8308`，`func (g *Game) checkResult()`
    pub(crate) fn check_result(&mut self) {
        if !self.result.is_empty() || self.sc.is_none() || self.native_chapter_result.is_some() {
            return;
        }
        // 死亡程式還沒跑完（例如頭目的死亡台詞）就先不判；跑完的續行會再判一次。
        if self.native_death_programs_pending() {
            return;
        }
        if self.native_action_death_presentation_pending() || self.native_ch28_post_present.is_some() {
            return;
        }
        if !self.sc.as_ref().unwrap().native_result_rules.is_empty() {
            self.begin_native_chapter_result();
            return;
        }
        let mut protect = String::from("索爾");
        if let Some(n) = self.camp.as_ref().and_then(|camp| camp.node())
            && !n.protect.is_empty()
        {
            protect = n.protect.clone();
        }
        let alive = match self.protected_battle_unit_alive(&protect) {
            Ok(alive) => alive,
            Err(err) => {
                self.load_err = format!("battle protection: {}", err.message());
                return;
            }
        };
        if !alive {
            self.result = String::from("lose");
            self.begin_native_defeat_if_configured();
            return;
        }
        // 章節 handler（第十章 0x20707）在 0x205BE 之後覆寫結果碼 1。
        if self.sc.as_ref().unwrap().native_result_code1(self.st.as_ref()) {
            self.result = String::from("lose");
            self.begin_native_defeat_if_configured();
            return;
        }
        let r = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic").result("");
        if !r.is_empty() {
            self.result = r;
            self.begin_native_defeat_if_configured();
        }
    }

    /// Go：`main.go:8370`，`func (g *Game) Update() error`
    ///
    /// Go 的四個 `defer`（`syncBalanceState`、清除玩家面板請求、`recordInputAuditState`、`traceRunLog`）
    /// 在每條 return 路徑上依後進先出執行；Rust 把本體放在 [`Game::update_frame`]，回來後依同一順序呼叫。
    /// `ebiten.Termination` 對應 [`ERR_TERMINATION`]（驅動端以 `GameError::is` 判斷）。
    pub(crate) fn update(&mut self) -> Result<()> {
        self.sync_balance_state();
        self.player_poster_cycle_requested = false;
        let result = self.update_frame();
        self.trace_run_log();
        self.record_input_audit_state();
        (self.player_theme_requested, self.player_locale_requested, self.player_interface_requested) =
            (false, false, false);
        (self.player_speed_requested, self.player_font_requested, self.player_fullscreen_requested) =
            (false, false, false);
        self.sync_balance_state();
        result
    }

    /// Rust 新增：Go `Update` 的本體（`main.go:8380` 起，`defer` 之後的部分）。輸入一律讀 `tick_input`。
    fn update_frame(&mut self) -> Result<()> {
        let input = self.tick_input.clone();
        let just = |k: Key| input.is_just_pressed(k);
        let enter_or_space = just(Key::Enter) || just(Key::Space);
        self.frame += 1;
        if self.player_fullscreen_requested {
            // Go `ebiten.SetFullscreen(!ebiten.IsFullscreen())`：視窗是外殼的事，這裡只留請求。
            self.fullscreen_toggle_requested = true;
            return Ok(());
        }
        let physical = input.physical();
        if physical.is_just_pressed(Key::Enter)
            && (physical.is_pressed(Key::Alt)
                || physical.is_pressed(Key::AltLeft)
                || physical.is_pressed(Key::AltRight))
        {
            return Ok(());
        }
        if self.startup_blocked {
            return Ok(());
        }
        if self.native_defeat.is_some() {
            let now = self.game_now();
            if let Err(err) = self.step_native_defeat(now) {
                self.load_err = format!("原生敗北呈現：{err}");
                self.startup_blocked = true;
            }
            return Ok(());
        }
        self.step_sfx_voices();
        self.step_action_overlay_lifecycle();
        self.step_native_system_info_ui();
        self.step_native_player_status();
        self.step_native_system_end_turn();
        let now = self.game_now();
        self.step_native_class_ui_lifecycle(now);
        if self.native_ending.is_none() && !native_modifier_held(&input) && just(Key::F1) {
            self.help_visible = !self.help_visible;
        }
        if self.help_visible {
            if just(Key::Escape) {
                self.help_visible = false;
            }
            return Ok(());
        }
        if self.transient_ui {
            return Ok(());
        }
        if let Some(message) = self.native_system_end_turn_ui.as_ref().and_then(|s| s.reward_message.as_ref()) {
            // 0x1AA1D 的訊息是同步的：開框、0x16C57(0) 等任一鍵、關框期間 AI 與玩家輸入都不動。
            if message.await_ack && self.native_class_ui_job.is_none() && input.any_just_pressed() {
                self.acknowledge_native_death_reward_message();
            }
            return Ok(());
        }
        if self.native_system_exit_requested {
            return Err(ERR_TERMINATION.clone());
        }
        let now = self.game_now();
        self.step_native_church_ui_lifecycle(now);
        let now = self.game_now();
        self.step_native_shop_ui_lifecycle(now);
        let now = self.game_now();
        let tick = self.native_town_ui_clock.sample(now);
        self.step_native_town_ui_lifecycle(tick);
        let now = self.game_now();
        self.step_native_preparation_ui_lifecycle(now);
        self.poll_modern_theme_load();
        let global = self.native_ending.is_none() && !native_modifier_held(&input);
        if global && self.player_theme_requested {
            self.player_theme_requested = false;
            self.cycle_theme_art();
        }
        if global && self.player_interface_requested {
            self.player_interface_requested = false;
            self.cycle_hd_title_interface();
        }
        if global && self.player_font_requested {
            // 全域：切換字級
            self.cycle_font_scale();
        }
        if global && self.player_speed_requested {
            self.cycle_player_speed();
        }
        let physical = input.physical();
        if self.native_ending.is_none() && physical.is_pressed(Key::Control) && physical.is_just_pressed(Key::F12) {
            // 全域：Ctrl+F12 開發除錯 HUD 開關（實體鍵；Go `main.go:8472`，r5i）
            self.debug = !self.debug;
        }
        if self.banner_t > 0 {
            self.banner_t -= 1;
            if self.banner_t == 0 {
                self.banner_frame = Vec::new();
            }
        }
        if !self.title_phase.is_empty() && self.title_update() {
            if !self.shot_path.is_empty() && self.shot_taken {
                // 截圖模式在 title 也要能退出
                return Err(ERR_TERMINATION.clone());
            }
            return Ok(());
        }
        if self.native_ending.is_some() {
            let confirm = just(Key::Enter) || just(Key::Space);
            self.update_native_ending(confirm, just(Key::Escape), input.any_just_pressed())?;
            if !self.shot_path.is_empty() && self.shot_taken {
                return Err(ERR_TERMINATION.clone());
            }
            return Ok(());
        }
        if self.native_field_event61.is_some() {
            // Go：`g.stepNativeFieldEvent61Tick(g.nativeFieldEvent61.clock.Sample(g.gameNow()))`（`r8`）。
            todo!("R8: native_field_event61.go:stepNativeFieldEvent61Tick（地圖事件 61）")
        }
        if self.native_end_turn_recovery.is_some() {
            let now = self.game_now();
            self.step_native_end_turn_recovery(now);
            return Ok(());
        }
        if self.native_ai_idle_recovery.is_some() {
            self.step_native_ai_idle_recovery();
            return self.shot_termination();
        }
        if self.native_ch20_sky_key.is_some() {
            todo!("R8: native_ch20_sky_key.go:stepNativeCh20SkyKey（第二十章天空之鑰）")
        }
        if self.native_ch23_loop.is_some() {
            self.step_native_ch23_loop();
            return self.shot_termination();
        }
        if self.native2189_a.is_some() {
            self.step_native_2189a();
            return self.shot_termination();
        }
        if self.native_unit_present.is_some() {
            self.step_native_unit_present();
            return self.shot_termination();
        }
        if self.native_ch28_post_present.is_some() {
            self.step_native_ch28_post_present();
            return self.shot_termination();
        }
        // 攻擊演出推進(FIGANI 全身分鏡;演出期間鎖玩家輸入)
        //
        // 一個顯示 tick 不是一個 60 Hz 畫格。原版 `sub_2939D` 對每一格重複
        // `cell+6` 次 `sub_17AA9(1)`，也就是每格停留 `cell+6` 個 BIOS tick；
        // fpt 是重製端在一個原版延遲單位裡插幾個顯示 tick 的解析度，所以一個顯示
        // tick 的長度是「一個 BIOS tick ÷ fpt」。照 60 Hz 一格一 tick 會讓整段快
        // 約 9%（50 對 54.925 毫秒）。BIOS tick 常數與回合橫幅同源，見
        // docs/knowledge-base/105-phase-banner-timing-20260910.md。
        if let Some(a) = self.atk.as_mut() {
            let body_running = a
                .native_scene
                .as_ref()
                .is_some_and(|s| s.body.is_some() && s.prelude_frame >= s.prelude_images.len() as i64);
            if body_running {
                // Go 把 scene 指標交給 stepNativePhysicalBodyMillis；Rust 先取出，呼叫後若演出仍在就放回。
                let mut scene = a.native_scene.take().unwrap();
                let result = self.step_native_physical_body_millis(&mut scene, 1000.0 / 60.0);
                if let Some(a) = self.atk.as_mut()
                    && a.native_scene.is_none()
                {
                    a.native_scene = Some(scene);
                }
                if result.is_err() {
                    return Ok(());
                }
            } else {
                let (mut ticks, rest) = attack_presentation_ticks(a.accum_millis + 1000.0 / 60.0, a.fpt);
                a.accum_millis = rest;
                while ticks > 0 && self.atk.is_some() {
                    if self.step_attack_presentation_tick().is_err() {
                        return Ok(());
                    }
                    ticks -= 1;
                }
            }
        }
        // 行軍動畫(spawn_march):進場位移緩動歸零,到位轉正面待機
        if let Some(st) = self.st.as_mut() {
            for u in st.units.iter_mut() {
                if u.off_x != 0.0 {
                    u.off_x *= 0.85;
                    if u.off_x < 1.0 && u.off_x > -1.0 {
                        u.off_x = 0.0;
                    }
                }
                if u.off_y != 0.0 {
                    u.off_y *= 0.85;
                    if u.off_y < 1.0 && u.off_y > -1.0 {
                        u.off_y = 0.0;
                        u.set_map_pose(0); // 到位面向鏡頭待機
                    }
                }
            }
        }
        // 移動動畫:原版每格 unit+4=1..6，第7 tick提交目的格。
        self.step_battle_walk();
        self.ai_step(); // AI 回合驅動(aiBusy 時逐單位行走→攻擊演出)
        self.step_native_story_dialogue_progress();
        self.step_dialogue_mouth();
        // 截圖模式:到指定幀後自動退出(畫面已於 Draw 存檔)。逐幀攻擊序列
        // 只有 `FD2_SHOT_SERIES` 時也必須進入同一個 setup，否則攻擊演出永遠
        // 不會建立，逐幀工具只能得到空目錄而無法驗證 GUI 時序。
        if !self.shot_path.is_empty() || !self.shot_series.is_empty() {
            // Apply screenshot-only setup immediately before capture, after scenario
            // setup has had time to spawn its party. Frame 1 is too early for
            // FD2_SHOT_RING on battle-start event scenarios.
            if !self.shot_setup && self.frame >= self.shot_frame - 1 {
                self.shot_setup = true;
                // Go 這裡依 FD2_SHOT_* 環境變數建立截圖用的固定狀態（教會、城鎮、商店、指令環、攻擊演出等
                // 二十多種，`main.go:8628`..`8907`）。120 範圍表把截圖模式歸為除錯工具（`fd2-tools`）。
                todo!("fd2-tools: main.go:Update 截圖模式狀態設定（FD2_SHOT_*）")
            }
            if !self.shot_series.is_empty() {
                // 逐幀模式:演出播完才退出
                if self.frame > 2 && self.atk.is_none() {
                    return Err(ERR_TERMINATION.clone());
                }
            } else if self.shot_taken {
                return Err(ERR_TERMINATION.clone());
            }
        }
        if self.m.is_none() {
            return Ok(());
        }
        self.step_story_walks(); // 場景走位動畫(doc46 §5.3);storyWalks 為空時內部直接返回
        self.step_act_job_at_update_rate(); // beat「act」以約18.2065Hz來源拍投影到60Hz更新
        self.step_focus_unit(); // beat「focus_unit」依原版安全帶逐格移動游標／鏡頭
        self.step_dlg_anim(); // 對話框換人縮/展動畫(使用者回饋 #3)
        if self.dlg_scroll_t > 0 {
            self.dlg_scroll_t -= 1;
        }
        self.step_fade(); // 場景淡出/淡入轉場(doc46 §5.2;beat「fade」兩個方向都靠 then 接回下一拍)
        self.step_transition_reveal(); // native 0x24b4d alternating present loop
        self.step_native_indexed_transition(); // native 0x24618 indexed map/palette transition
        self.step_native_command_heal_presentation(); // native 0x21EB1 command 13..16 indexed presentation
        self.step_native_command_modifier_presentation(); // native 0x1D6C8 command 17..19 palette presentation
        self.step_native_ai_command_modifier_presentation(); // native 0x1541F enemy command17..19 handler tail
        self.step_native_command0_presentation(); // native 0x2A6BD→0x26152 command0 battle presentation
        self.step_native_command1_presentation(); // native 0x2A6BD→0x262EF command1 battle presentation
        self.step_native_command2_presentation(); // native 0x2A6BD→0x26528 command2 battle presentation
        self.step_native_command3_presentation(); // native 0x2A6BD→0x26795 command3 battle presentation
        self.step_native_command5_presentation(); // native 0x2A6BD→0x26BFD command5 battle presentation
        self.step_native_command6_presentation(); // native 0x2A6BD→0x26E39 command6 battle presentation
        self.step_native_command7_presentation(); // native 0x2A6BD→0x272B8 command7 battle presentation
        self.step_native_command8_presentation(); // native 0x2A6BD→0x274B0 command8 battle presentation
        self.step_native_command9_player_presentation(); // native 0x1CFF0→0x214AD player command9 map presentation
        self.step_native_command9_ai_presentation(); // native 0x2A6BD→0x275D6 enemy command9 presentation
        self.step_native_command1012_presentation(); // native 0x21527/0x2185F/0x21A9E player map presentation
        self.step_native_command24_presentation(); // native 0x276EC selector32 FIGANI presentation
        self.step_native_command29_presentation(); // native 0x276EC selector34 multi-target FIGANI presentation
        self.step_native_command32_presentation(); // native 0x27FC9→0x2111A player command32 presentation
        self.step_native_command33_presentation(); // native 0x27FC9→0x211A4 player command33 presentation
        self.step_native_command34_presentation(); // native 0x27FC9→0x22721/0x22866/0x22997 player command34 presentation
        self.step_native_command35_presentation(); // native 0x27FC9→0x22D1B×3 player command35 presentation
        self.step_native_ai_item_presentation(); // native 0x15055→0x211A4 AI item type5/13 tail
        self.step_native_town_departure(); // native 0x2d190 town caller
        self.step_native_palette_ramp(); // native 0x1f882/0x1f525 whole-DAC ramps
        self.step_native_palette_pulse(); // native 0x35E5A whole-DAC pulse
        self.step_native_spawn_intro(); // native 0x32999 twelve-pass indexed spawn transition
        self.step_native_turn_staging(); // typed raw-camp0 pre-AI staging helper
        if self.camp.is_some() && self.story_auto_advance > 0 {
            // 無對白節點自動轉場倒數(行軍蒙太奇)
            self.story_auto_advance -= 1;
            if self.story_auto_advance == 0
                && let Some(n) = self.camp.as_ref().unwrap().node().cloned()
            {
                self.advance_story_node(&n);
            }
        }
        if self.beat_delay > 0 {
            // beat「delay」倒數(doc50 0x375b2)
            self.beat_delay -= 1;
            if self.beat_delay == 0 {
                self.beat_advance();
            }
        }
        self.step_battle_event_delay();
        if self.cur_x != self.prev_cur_x || self.cur_y != self.prev_cur_y {
            // 游標移動音
            self.play_sfx(SFX_CURSOR);
            (self.prev_cur_x, self.prev_cur_y) = (self.cur_x, self.cur_y);
        }
        let (tile_w, tile_h, map_w, map_h) = {
            let m = self.m.as_ref().unwrap();
            (m.tile_w, m.tile_h, m.w, m.h)
        };
        // 相機跟隨游標(置中,夾在地圖內;先於各攔截,避免環/選單開啟時相機停擺)
        // storyBG 場景背景模式鏡頭固定(enterNode 已設 CamX/CamY),不跟游標走。
        // FollowWalk 節點例外:走位期間鏡頭鎖定走位者(原版長廊運鏡,2-1;視野=320×200 世界px)。
        if self.cam_pan.is_some() {
            self.step_cam_pan();
        } else if self.story_bg {
            if self.follow_walk && !self.story_walks.is_empty() {
                let u = &self.story_actors[self.story_walks[0].actor as usize];
                let (vw, vh) = (LOGICAL_W / STORY_ZOOM, LOGICAL_H / STORY_ZOOM);
                self.cam_x = (u.x * tile_w) as f64 + u.off_x + tile_w as f64 / 2.0 - vw as f64 / 2.0;
                self.cam_y = (u.y * tile_h) as f64 + u.off_y + tile_h as f64 / 2.0 - vh as f64 / 2.0;
                clamp(&mut self.cam_x, 0.0, (map_w * tile_w - vw) as f64);
                clamp(&mut self.cam_y, 0.0, (map_h * tile_h - vh) as f64);
                if self.cam_max_y > 0.0 && self.cam_y > self.cam_max_y {
                    // 場景鏡頭上限(王座廳擋草地;走位者可從畫面外走入)
                    self.cam_y = self.cam_max_y;
                }
            }
        } else if self.battle_event.is_none() && self.native_turn_staging.is_none() && !self.sync_native_map_view() {
            self.center_camera_on_cursor();
        }
        if self.native_heal_presentation.is_some()
            || self.native_modifier_presentation.is_some()
            || self.native_ai_command_modifier.is_some()
            || self.native_ai_item_presentation.is_some()
            || self.native_cmd0_presentation.is_some()
            || self.native_cmd1_presentation.is_some()
            || self.native_cmd2_presentation.is_some()
            || self.native_cmd3_presentation.is_some()
            || self.native_cmd5_presentation.is_some()
            || self.native_cmd6_presentation.is_some()
            || self.native_cmd7_presentation.is_some()
            || self.native_cmd8_presentation.is_some()
            || self.native_cmd9_player.is_some()
            || self.native_cmd9_ai_presentation.is_some()
            || self.native_cmd1012.is_some()
            || self.native_cmd24_presentation.is_some()
            || self.native_cmd29_presentation.is_some()
            || self.native_cmd32_presentation.is_some()
            || self.native_cmd33_presentation.is_some()
            || self.native_cmd34_presentation.is_some()
            || self.native_cmd35_presentation.is_some()
        {
            return Ok(());
        }
        if self.native_system_info_ui.is_some() {
            self.ring_input();
            return Ok(());
        }
        // 35D60的六次35E5A是同步演出；續行前不可開啟另一個END owner。
        // READY: fd2_ch29_flow_20261017.json event76_input_gate_contract。
        if self.native_palette_pulse.is_some() {
            return Ok(());
        }
        if !native_modifier_held(&input) && just(Key::F5) {
            // 快速存檔:節點進度;戰鬥中可操作時附戰場快照(battle_snapshot.go)
            self.save_game();
        }
        if !native_modifier_held(&input) && just(Key::F9) {
            // 快速讀檔
            self.load_game();
        }
        let dialogue_advance = native_battle_dialogue_advance_input(enter_or_space, just(Key::Escape));
        if self.native_level_up_dialogue.is_some() {
            self.step_native_level_up_dialogue(dialogue_advance);
            return Ok(()); // 0x1E292 同步：對話期間不進 AI 與其他輸入
        }
        if self.battle_event.is_some() || self.native_turn_staging.is_some() {
            self.handle_battle_event_dialogue_input(dialogue_advance);
            return Ok(()); // PAN/delay/dialogue sequence blocks battle input and repeated end-turn
        }
        if self.camp_input() {
            // campaign 節點(story/choice/ending/勝敗轉場)攔截輸入
            return Ok(());
        }
        // 0x1A30B is synchronous: map input resumes only after the AI phase.
        // Dialogue input still belongs to its existing owner below.
        if self.ai_busy && self.dialog.is_empty() {
            return Ok(());
        }
        if self.native_player_status.is_some() {
            if enter_or_space || just(Key::Escape) {
                todo!("R8: advanceNativePlayerStatus（玩家狀態面板翻頁）")
            }
            return Ok(());
        }
        if self.native_player_focus.is_some() {
            self.step_native_player_focus();
            return Ok(());
        }
        if self.ring_input() {
            // radial 指令環 / 法術選單
            return Ok(());
        }
        if !self.dialog.is_empty() {
            // 戰鬥起手對白(g.camp==nil 直接開局,或 campaign battle 節點無 story 攔截):Enter/Space/Esc 逐句清除
            if dialogue_advance {
                self.dlg_advance(); // 翻頁優先,翻完換句(長對白分頁,不截斷)
            }
            return Ok(());
        }
        let cancel = just(Key::Escape) || just(Key::Backspace);
        if (self.native_item_targeting || self.native_item_relocating) && cancel {
            // 原始物品第一階段 target selector：ESC 回物品列
            self.cancel_native_item_target_modal();
            return Ok(());
        }
        if (self.cast_sp.is_some() || self.native_command0_targeting) && just(Key::Escape) {
            // native target selection: ESC 回 command grid
            self.cancel_native_command_target_selection();
            return Ok(());
        }
        // 游標移動:方向鍵 / WASD(按住持續移動,keyRepeat)/ 觸控
        if self.native_move_panel.is_some() && input.any_just_pressed() {
            self.native_move_panel = None;
        }
        if key_repeat(&input, Key::ArrowLeft) || key_repeat(&input, Key::A) {
            self.move_map_cursor(-1, 0);
        }
        if key_repeat(&input, Key::ArrowRight) || key_repeat(&input, Key::D) {
            self.move_map_cursor(1, 0);
        }
        if key_repeat(&input, Key::ArrowUp) || key_repeat(&input, Key::W) {
            self.move_map_cursor(0, -1);
        }
        if key_repeat(&input, Key::ArrowDown) || key_repeat(&input, Key::S) {
            self.move_map_cursor(0, 1);
        }
        // 觸控:點哪格移到哪格（Go `main.go:9095`..`9125`；R5 階段 4 r5w 接上 `TickInput::just_pressed_touches`）。
        for &(tx, ty) in &input.just_pressed_touches {
            let (tx, ty, accepted) = self.player_touch_coordinates(tx, ty);
            if !accepted {
                continue;
            }
            // Go `int(g.camX) + tx`：float64 轉 int 向零截斷。
            let mut target_x = (self.cam_x as i64 + tx) / tile_w;
            let mut target_y = (self.cam_y as i64 + ty) / tile_h;
            if target_x < 0 {
                target_x = 0;
            } else if target_x >= map_w {
                target_x = map_w - 1;
            }
            if target_y < 0 {
                target_y = 0;
            } else if target_y >= map_h {
                target_y = map_h - 1;
            }
            while self.cur_x < target_x {
                self.move_map_cursor(1, 0);
            }
            while self.cur_x > target_x {
                self.move_map_cursor(-1, 0);
            }
            while self.cur_y < target_y {
                self.move_map_cursor(0, 1);
            }
            while self.cur_y > target_y {
                self.move_map_cursor(0, -1);
            }
        }
        self.follow_pointer_on_map(); // 滑鼠：移動或點下時游標跟到指向的格子（Go `main.go:9136`，r5i）
        // 邊界
        if self.cur_x < 0 {
            self.cur_x = 0;
        }
        if self.cur_y < 0 {
            self.cur_y = 0;
        }
        if self.cur_x >= map_w {
            self.cur_x = map_w - 1;
        }
        if self.cur_y >= map_h {
            self.cur_y = map_h - 1;
        }
        // 選取 / 移動 / 取消
        if enter_or_space {
            self.confirm();
        }
        let st_has_view = |g: &Game| g.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic").has_native_map_view_state;
        if cancel {
            if self.sel.is_some() && self.moved {
                // 已移動、正在選攻擊目標:退回指令環(取消一層,doc13;ring 的 ESC 才真正退回原位)
                self.return_player_attack_target_to_ring();
            } else if self.sel.is_none() && st_has_view(self) && just(Key::Escape) {
                self.cycle_native_player_unit();
            } else {
                if self.native_move_plan.is_some() {
                    self.restore_player_movement_cursor();
                    self.clear_native_player_movement();
                }
                (self.sel, self.reach) = (None, BTreeMap::new());
                self.msg.clear();
            }
        }
        if self.sel.is_none() && st_has_view(self) {
            if just(Key::Z) || just(Key::Numpad5) {
                self.cycle_native_player_unit();
            }
            if just(Key::Home) {
                let (x, y) = (self.cur_x, self.cur_y);
                if let Some(u) = self.st.as_ref().unwrap().unit_at(x, y)
                    && let Err(err) = self.inspect_native_player_unit(u, true)
                {
                    self.load_err = err.message().to_string();
                }
            }
        }
        // Tab 是重製端快速鍵；原版式正常入口是空游標四格系統面板的 Down→END。
        if just(Key::Tab) {
            self.end_turn();
        }
        // 相機跟隨游標(置中,夾在地圖內)
        if !self.sync_native_map_view() {
            self.center_camera_on_cursor();
        }
        Ok(())
    }

    /// Rust 新增：Go `Update` 兩處相同的「相機置中游標」（`main.go:8996`、`9171`）。
    fn center_camera_on_cursor(&mut self) {
        let m = self.m.as_ref().unwrap();
        let (tile_w, tile_h, map_w, map_h) = (m.tile_w, m.tile_h, m.w, m.h);
        self.cam_x = (self.cur_x * tile_w - LOGICAL_W / 2 + tile_w / 2) as f64;
        self.cam_y = (self.cur_y * tile_h - LOGICAL_H / 2 + tile_h / 2) as f64;
        clamp(&mut self.cam_x, 0.0, (map_w * tile_w - LOGICAL_W) as f64);
        clamp(&mut self.cam_y, 0.0, (map_h * tile_h - LOGICAL_H) as f64);
    }

    /// Rust 新增：Go 一再出現的 `if g.shotPath != "" && g.shotTaken { return ebiten.Termination }; return nil`。
    fn shot_termination(&self) -> Result<()> {
        if !self.shot_path.is_empty() && self.shot_taken {
            return Err(ERR_TERMINATION.clone());
        }
        Ok(())
    }

    /// Go：`main.go:9179`，`func (g *Game) syncNativeMapView() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn sync_native_map_view(&mut self) -> bool {
        let (Some(m), Some(st)) = (self.m.as_ref(), self.st.as_ref()) else {
            return false;
        };
        if !st.has_native_map_view_state {
            return false;
        }
        let view = st.native_map_view_state;
        (self.cur_x, self.cur_y) = (view.cursor_x, view.cursor_y);
        self.cam_x = (view.camera_x * m.tile_w) as f64;
        self.cam_y = (view.camera_y * m.tile_h) as f64;
        true
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
        if !self.st.as_ref().is_some_and(|st| st.native_map_cursor_step_redraws(before, after)) {
            return;
        }
        self.redraw_native_map_hud();
        // 第十二章的 mode3 會保留前一鏡頭底色；11CAC 不只更新 HUD。
        // 已證實 caller／literal writer 見 terrain review 的 battle_cursor_redraw。
        if self.camp.as_ref().is_some_and(|c| c.node_id() == "battle_ch12")
            && self.native_map_assets.as_ref().is_some_and(|a| a.map_index == 11)
            && !self.has_story_native_map_view
            && self.st.as_ref().is_some_and(|st| st.has_native_map_view_state)
            && let Err(err) = self.compose_native_map_frame()
        {
            self.load_err = format!("native cursor redraw: {err}");
        }
    }

    /// R4 B 組新增：`battle.State.FocusNativeMapCursorSteps(x, y, g.nativeCursorStepHUD)`。
    ///
    /// Go 把會改寫 `Game`（HUD anchor、第十二章重繪）的 `nativeCursorStepHUD` 交給 `st` 的方法當 callback；
    /// Rust 在借用 `st` 時不能再借用 `Game`，所以逐行照 fd2-rules `State::focus_native_map_cursor_steps`
    /// （Go `native_map_view.go:264`）在 `Game` 這層重寫迴圈，每走一格呼叫 `native_cursor_step_hud`。
    fn focus_native_map_cursor_steps_with_hud(&mut self, x: i64, y: i64) -> bool {
        let Some(st) = self.st.as_ref() else {
            return false;
        };
        if !st.has_native_map_view_state {
            return false;
        }
        // 先取出 callback 再呼叫，避免呼叫期間持有鎖。
        let hook = battle::NATIVE_FOCUS_TRACE_HOOK.read().unwrap_or_else(|e| e.into_inner()).clone();
        if let Some(hook) = hook {
            hook(x, y, st.native_map_view_state);
        }
        if !st.native_map_view_state.cursor_in_field(st.w, st.h) || x < 0 || x >= st.w || y < 0 || y >= st.h {
            return false;
        }
        let step_once = |g: &mut Game, dx: i64, dy: i64| -> bool {
            let st = g.st.as_mut().unwrap();
            let before = st.native_map_view_state;
            let (_, ok) = st.move_native_map_cursor(dx, dy);
            if !ok {
                return false;
            }
            let after = st.native_map_view_state;
            g.native_cursor_step_hud(before, after);
            true
        };
        loop {
            let cursor_x = self.st.as_ref().unwrap().native_map_view_state.cursor_x;
            if cursor_x == x {
                break;
            }
            let step = if cursor_x > x { -1 } else { 1 };
            if !step_once(self, step, 0) {
                return false;
            }
        }
        loop {
            let cursor_y = self.st.as_ref().unwrap().native_map_view_state.cursor_y;
            if cursor_y == y {
                break;
            }
            let step = if cursor_y > y { -1 } else { 1 };
            if !step_once(self, 0, step) {
                return false;
            }
        }
        true
    }

    /// redrawNativeMapHUD 是一次原版 0x11CAC 重繪裡 0x1ACF3→0x1AD2A 的 HUD anchor 評估：閘 A／閘 B
    /// 由 AdvanceNativeMapHUDAnchor 判，可見游標取當下視圖。重製端每幀 Draw 的整幀重組不是原版的
    /// 重繪（走行步進 0x12EAA 家族就不呼叫 0x11CAC），所以 anchor 只在對應原版重繪點的地方經這裡
    /// 評估；對照表見 56 §#40 重繪入口。
    ///
    /// Go：`main.go:9212`，`func (g *Game) redrawNativeMapHUD()`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn redraw_native_map_hud(&mut self) {
        if let Some(st) = self.st.as_mut() {
            redraw_native_map_hud_state(st);
        }
    }

    /// Go：`main.go:9220`，`func (g *Game) moveMapCursor(dx, dy int)`
    /// 持續名冊全員回滿 HP／MP，清行動、位移、增益與狀態異常。
    ///
    /// Go：`main.go`，`func (g *Game) resetPersistentRosterState()`
    pub(crate) fn reset_persistent_roster_state(&mut self) {
        for unit in self.party_roster.values_mut() {
            unit.acted = false;
            unit.hp = unit.max_hp;
            unit.mp = unit.max_mp;
            (unit.off_x, unit.off_y) = (0.0, 0.0);
            (unit.buff_ap_pct, unit.buff_dp_pct, unit.buff_hit, unit.buff_ev, unit.buff_turns) = (0, 0, 0, 0, 0);
            (unit.sealed, unit.seal_turns) = (false, 0);
            (unit.poisoned, unit.poison_turns) = (false, 0);
            (unit.paralyzed, unit.paralyze_turns) = (false, 0);
        }
    }

    pub(crate) fn move_map_cursor(&mut self, dx: i64, dy: i64) {
        if let Some(st) = self.st.as_mut().filter(|st| st.has_native_map_view_state) {
            let before = st.native_map_view_state;
            let (_, ok) = st.move_native_map_cursor(dx, dy);
            if ok {
                let after = st.native_map_view_state;
                self.native_cursor_step_hud(before, after);
                self.sync_native_map_view();
            }
            return;
        }
        self.cur_x += dx;
        self.cur_y += dy;
    }

    /// focusNativeMapCursorOnUnit 重現 0x12D7B→0x12CEA：讀單位記錄的 x/y，再把游標
    /// 先 X 後 Y 逐格移過去。逐格走的是四個鍵盤處理器，所以可見游標與鏡頭跟著同一套
    /// 安全帶規則，三組全域不會脫節。
    ///
    /// Go：`main.go:9236`，`func (g *Game) focusNativeMapCursorOnUnit(u *battle.Unit)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn focus_native_map_cursor_on_unit(&mut self, u: Option<UnitIdx>) {
        let Some(u) = u else {
            return;
        };
        let Some(st) = self.st.as_ref().filter(|st| st.has_native_map_view_state) else {
            return;
        };
        let (x, y) = (st.units[u].x, st.units[u].y);
        self.redraw_native_map_hud(); // 0x12D01：0x12CEA 開頭的 0x11CAC(0)
        if !self.focus_native_map_cursor_steps_with_hud(x, y) {
            return;
        }
        self.sync_native_map_view();
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
        let Some(st) = self.st.as_ref().filter(|st| st.has_native_map_view_state) else {
            return;
        };
        let hook = battle::NATIVE_FOCUS_TRACE_HOOK.read().unwrap_or_else(|e| e.into_inner()).clone();
        if let Some(hook) = hook {
            hook(x, y, st.native_map_view_state);
        }
        self.redraw_native_map_hud(); // 0x12D01：0x12CEA 開頭的 0x11CAC(0)
        if !self.focus_native_map_cursor_steps_with_hud(x, y) {
            return;
        }
        self.sync_native_map_view();
    }

    /// followNativeMapCursorStep 讓視圖跟著走行的單位走一格。判準是單位自己的
    /// 相對列／行，不是已存的可見游標值；寫入端清單與安全帶規則見
    /// docs/data/ida/fd2_visible_cursor_writers_ida.txt。
    ///
    /// Go：`main.go:9270`，`func (g *Game) followNativeMapCursorStep(fromX, fromY, dx, dy int)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn follow_native_map_cursor_step(&mut self, from_x: i64, from_y: i64, dx: i64, dy: i64) {
        let Some(st) = self.st.as_mut().filter(|st| st.has_native_map_view_state) else {
            return;
        };
        if !st.advance_native_map_walk_step_view(from_x, from_y, dx, dy) {
            return;
        }
        // 走行步進不推 HUD anchor：0x12EAA／0x1300D／0x13185／0x13315 只呼叫 0x1297D 局部重繪與
        // 0x11EB0／0x11EEE 拷貝，不經過 0x11CAC，所以不會進 0x1ACF3。第八章 c2 seq 1667 單位走過
        // 可見游標 (2,6)，原版小窗沒翻邊。
        self.sync_native_map_view();
    }

    /// positionScreenshotCursor drives the same recovered cursor/camera state
    /// machine as interactive input. Directly assigning curX/curY leaves the
    /// production indexed frame on stale NativeMapViewState and produces a false
    /// visual oracle.
    ///
    /// Go：`main.go:9287`，`func (g *Game) positionScreenshotCursor(x, y int) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn position_screenshot_cursor(&mut self, x: i64, y: i64) -> bool {
        let Some(st) = self.st.as_ref() else {
            return false;
        };
        if x < 0 || y < 0 || x >= st.w || y >= st.h {
            return false;
        }
        if !st.has_native_map_view_state {
            (self.cur_x, self.cur_y) = (x, y);
            return true;
        }
        // 與玩家確認移動同一條路（0x12CEA：先 X 後 Y，逐格走鍵盤處理器），
        // 所以截圖與正式路徑產生的視圖狀態一致。
        // 原版逐格重繪會留下 mode3 後續消費的底色。只合成終點會漏掉
        // 中間鏡頭的 literal writer；#52 主證據的 cursor_helper 記錄此契約。
        if !self.focus_native_map_cursor_steps_with_hud(x, y) {
            return false;
        }
        let st = self.st.as_mut().unwrap();
        if st.has_native_map_hud_state {
            let view = st.native_map_view_state;
            st.advance_native_map_hud_anchor(view.visible_cursor_x, view.visible_cursor_y);
        }
        self.sync_native_map_view();
        self.cur_x == x && self.cur_y == y
    }
}

/// R4 B 組新增：`redrawNativeMapHUD` 對 `battle.State` 的部分。Go `startNativeUnitDeathPresentMode`
/// 以 `Game{st: candidate}.redrawNativeMapHUD()` 對私有候選狀態做同一件事，兩處共用。
pub(crate) fn redraw_native_map_hud_state(st: &mut battle::State) {
    if !st.has_native_map_hud_state || !st.has_native_map_view_state {
        return;
    }
    let v = st.native_map_view_state;
    st.advance_native_map_hud_anchor(v.visible_cursor_x, v.visible_cursor_y);
}

/// keyRepeat 方向鍵按住持續觸發(playfix #1):首次按下立即動一格,按住 12 tick 後
/// 每 5 tick 再動一格(可玩性手感參數,原版掃描碼節奏未逐格量測,見 doc13 §游標)。
///
/// Go：`main.go:9312`，`func keyRepeat(k ebiten.Key) bool`
/// Go 讀全域的 `inpututil.KeyPressDuration`；Rust 收本幀的 `TickInput`（簽名變更見 `PORTING.md`）。
pub(crate) fn key_repeat(input: &TickInput, k: Key) -> bool {
    let d = input.key_press_duration(k);
    if d == 0 {
        return false;
    }
    if d == 1 {
        return true;
    }
    if d < 12 {
        return false;
    }
    (d - 12) % 5 == 0
}

/// Go：`main.go:9326`，`func clamp(v *float64, lo, hi float64)`
pub(crate) fn clamp(v: &mut f64, lo: f64, mut hi: f64) {
    if hi < lo {
        hi = lo;
    }
    if *v < lo {
        *v = lo;
    }
    if *v > hi {
        *v = hi;
    }
}

/// Rust 新增：`Game::battle_text_draws` 列出的一次 `g.font.Draw(screen, text, x, y, scale, clr)`。
#[derive(Debug, Clone, PartialEq)]
pub(crate) struct BattleTextDraw {
    pub(crate) text: String,
    pub(crate) x: f64,
    pub(crate) y: f64,
    pub(crate) scale: f64,
    pub(crate) clr: ColorRgba,
}

impl Game {
    /// Go：`main.go:9355`，`func (g *Game) Draw(screen *ebiten.Image)`
    ///
    /// R5 階段 1 的分派骨架（`PORTING.md`「R5 階段 1」）：照 Go 的 early-return 順序選分支。開場與第一章
    /// 執行到的分支呼叫對應函式（R5 階段 2 移植完畢，R5 階段 3 起不再有 `todo!("r5:…")`）；HD 分支 `todo!("r6:…")`；
    /// 開場與第一章沒有走到的分支 `todo!("r8:…")`。分支是否執行到依階段 0 的 `Draw` 覆蓋率
    /// （`work/r5-stage0-20261115/r1/cover-*.out`）。
    ///
    /// 開頭的重設是 R4 的 `begin_present_frame`（Go 9356..9372，含 `defer g.beginModernHDText(screen)()`；忠實模式下
    /// 收尾是空函式）；結尾的 `defer` 是 [`Game::draw_deferred`]。Draw 裡改流程狀態的地方照 `PORTING.md`
    /// 「Draw 中改狀態的位置」由各 `draw*` 函式自己做（與 Go 相同），不經呈現回報。章重播的 `ack_presents` 在 Go
    /// 呼叫 `g.Draw` 的三處（城鎮出發、有原生場景的攻擊演出、END 回收）也呼叫本函式（R5 階段 3）。
    pub(crate) fn draw(&mut self, screen: &mut Picture) {
        self.begin_present_frame();
        self.draw_frame(screen);
        self.draw_deferred(screen);
    }

    /// Go `Draw` 的第二個 `defer`（`main.go:9376`..`9393`）：每條 return 路徑都執行。
    fn draw_deferred(&mut self, screen: &mut Picture) {
        const WHITE: ColorRgba = ColorRgba { r: 0xff, g: 0xff, b: 0xff, a: 0xff };
        const BLACK: ColorRgba = ColorRgba { r: 0, g: 0, b: 0, a: 0xff };
        if self.native_full_dac_white {
            screen.fill(WHITE);
        }
        if self.native_full_dac_black {
            screen.fill(BLACK);
        }
        if self.native_defeat.is_none()
            && self.title_phase.is_empty()
            && self.native_palette_ramp.is_none()
            && native_dacis_black(&self.native_map_dac)
        {
            screen.fill(BLACK);
        }
        // 白閃的 drawn 與說明畫面的 clearModernHDText（R6）：與呈現回報共用。
        self.finish_present_frame();
        if self.help_visible {
            // R5 階段 4（r5w）：121 範圍表把說明畫面列在 R5 的外殼，接 F1 時移植。
            self.draw_player_help(screen);
        }
    }

    /// Go `g.shotPath != "" && !g.shotTaken && g.frame >= g.shotFrame` 時的 `captureShot`（截圖模式，fd2-tools）。
    pub(crate) fn draw_capture_shot_point(&mut self, screen: &mut Picture) {
        if !self.shot_path.is_empty() && !self.shot_taken && self.frame >= self.shot_frame {
            let _ = screen;
            todo!("r8:main.go:captureShot（FD2_SHOT 截圖模式；R5 的畫格擷取走 parity_replay 的擷取點）")
        }
    }

    /// Go `Draw` 的 if 鏈（`main.go:9395`..`9797`）與尾段（`main.go:9799`..`10188`）。
    fn draw_frame(&mut self, screen: &mut Picture) {
        const BLACK: ColorRgba = ColorRgba { r: 0, g: 0, b: 0, a: 0xff };
        if self.startup_blocked {
            screen.fill(ColorRgba { r: 0x12, g: 0x16, b: 0x20, a: 0xff });
            if let Some(font) = self.font.as_mut() {
                font.draw(screen, &self.load_err, 16.0, 28.0, 1.0, ColorRgba { r: 255, g: 255, b: 255, a: 255 });
            }
            return;
        }
        if let Some(job) = self.native_defeat.as_mut() {
            job.present(Some(screen));
            return;
        }
        if !self.title_phase.is_empty() {
            self.draw_title(screen);
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_ending.is_some() {
            self.draw_native_ending_preview(Some(screen));
            return;
        }
        if self.native_ch20_sky_key.is_some() {
            screen.fill(BLACK);
            todo!("r8:native_ch20_sky_key.go:drawNativeCh20SkyKey／failNativeCh20SkyKey");
        }
        if self.native_ch23_loop.is_some() {
            screen.fill(BLACK);
            if !self.draw_native_ch23_loop(Some(screen)) {
                self.fail_native_ch23_loop("presentation unavailable");
            }
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native2189_a.is_some() {
            screen.fill(BLACK);
            if !self.draw_native_2189a(Some(screen)) {
                self.fail_native_2189a("indexed frame unavailable");
            }
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_unit_present.is_some() {
            screen.fill(BLACK);
            if !self.draw_native_unit_present(Some(screen)) {
                self.fail_native_unit_present(&game_err!("indexed frame unavailable"));
            }
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_ch28_post_present.is_some() {
            // 第一章執行到（階段 0 覆蓋率 1064 次），失敗路徑沒有。
            screen.fill(BLACK);
            if !self.draw_native_ch28_post_present(screen) {
                todo!("r8:native_ch28_post_present.go:failNativeCh28PostPresent 與 DebugPrint（第一章沒有走到）");
            }
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.transition_reveal.is_some() {
            screen.fill(BLACK);
            if self.transition_reveal.as_ref().is_some_and(|j| !j.hd_frames.is_empty()) {
                todo!("R6: native_24b4d.go:146 HD 列震動幀");
            }
            self.draw_native_24b4d(Some(screen));
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.m.is_none() {
            todo!("r8:main.go:9483 缺地圖時的 ebitenutil.DebugPrint");
        }
        if self.indexed_transition.is_some() {
            screen.fill(BLACK);
            if self.modern_story_portraits.as_ref().is_some_and(|set| set.map_sprites_hd.len() == 125) {
                todo!("R6: native_indexed_transition.go:336 HD 索引轉場幀");
            }
            self.draw_native_indexed_transition(screen);
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_heal_presentation.is_some() {
            screen.fill(BLACK);
            self.present_native_command_heal(Some(screen));
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_modifier_presentation.is_some() {
            screen.fill(BLACK);
            todo!("r8:native_command_modifier_presentation.go:drawNativeCommandModifierPresentation");
        }
        if self.native_ai_command_modifier.is_some() {
            screen.fill(BLACK);
            self.present_native_ai_command_modifier(Some(screen));
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_cmd0_presentation.is_some() {
            screen.fill(BLACK);
            if !self.native_cmd0_presentation.as_mut().unwrap().present(Some(screen)) {
                self.fail_native_command0_presentation(&game_err!("指令 0 演出畫面無效"));
            }
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_cmd5_presentation.is_some() {
            screen.fill(BLACK);
            self.present_native_indexed_damage_command(Some(screen));
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_cmd2_presentation.is_some() {
            screen.fill(BLACK);
            self.present_native_command2(Some(screen));
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_cmd8_presentation.is_some() {
            screen.fill(BLACK);
            self.present_native_command8(Some(screen));
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_cmd9_ai_presentation.is_some() {
            screen.fill(BLACK);
            self.present_native_command9_ai(Some(screen));
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_cmd6_presentation.is_some() {
            screen.fill(BLACK);
            self.present_native_command6(Some(screen));
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_cmd7_presentation.is_some() {
            screen.fill(BLACK);
            self.present_native_command7(Some(screen));
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_cmd3_presentation.is_some() {
            screen.fill(BLACK);
            self.present_native_command3(Some(screen));
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_cmd6_presentation.is_some()
            || self.native_cmd1_presentation.is_some()
            || self.native_cmd2_presentation.is_some()
            || self.native_cmd3_presentation.is_some()
            || self.native_cmd7_presentation.is_some()
            || self.native_cmd8_presentation.is_some()
            || self.native_cmd9_player.is_some()
            || self.native_cmd9_ai_presentation.is_some()
            || self.native_cmd1012.is_some()
            || self.native_cmd24_presentation.is_some()
            || self.native_cmd29_presentation.is_some()
            || self.native_cmd32_presentation.is_some()
            || self.native_cmd33_presentation.is_some()
            || self.native_cmd34_presentation.is_some()
            || self.native_cmd35_presentation.is_some()
        {
            // Go main.go:9528..9701 依序 17 個指令演出分支，各自 Fill 黑、draw、失敗時 fail＋DebugPrint。
            screen.fill(BLACK);
            todo!("r8:main.go:9528..9701 指令 0～35 演出（第一章沒有）");
        }
        if self.native_ai_item_presentation.is_some() {
            screen.fill(BLACK);
            if !self.draw_native_ai_item_presentation(screen) {
                self.fail_native_ai_item_presentation(&game_err!("敵方道具演出畫面無效"));
            }
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_town_departure.is_some() {
            screen.fill(BLACK);
            self.draw_native_town_departure(screen);
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_palette_ramp.is_some() {
            screen.fill(BLACK);
            if !self.draw_native_palette_ramp(screen) {
                todo!("r8:main.go:9718 調色盤斜坡畫不出來時的 DebugPrint");
            }
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_palette_pulse.is_some() {
            screen.fill(BLACK);
            self.draw_native_palette_pulse(screen);
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.spawn_intro_transition.is_some() {
            screen.fill(BLACK);
            if !self.draw_native_spawn_intro(screen) {
                todo!("r8:main.go:9738 登場轉場畫不出來時的 DebugPrint");
            }
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_turn_staging.as_ref().is_some_and(|j| j.indexed) {
            screen.fill(BLACK);
            self.draw_native_turn_staging(Some(screen));
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_field_event61.is_some() {
            screen.fill(BLACK);
            todo!("r8:native_field_event61.go:drawNativeFieldEvent61");
        }
        if self.native_end_turn_recovery.is_some() {
            screen.fill(BLACK);
            if !self.draw_native_end_turn_recovery(screen) {
                self.cancel_native_end_turn_recovery(&game_err!("native END recovery: Draw unavailable"));
                todo!("r8:main.go:9769 ebitenutil.DebugPrint(loadErr)");
            }
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.native_ai_idle_recovery.is_some() {
            screen.fill(BLACK);
            if !self.draw_native_ai_idle_recovery(screen) {
                // Go main.go:9779..9783；與 present.rs 的呈現回報相同。
                let job = self.native_ai_idle_recovery.take().unwrap();
                let st = self.st.as_mut().unwrap();
                st.native_map_range_mode = job.before_range;
                st.has_native_map_range_mode_state = job.before_has;
                self.load_err = "native AI 0x13fd4 presentation unavailable".to_string();
                self.ai_busy = false;
                todo!("r8:main.go:9784 ebitenutil.DebugPrint");
            }
            self.draw_capture_shot_point(screen);
            return;
        }
        if self.atk.is_some() {
            self.draw_battle_scene(screen);
            self.draw_capture_shot_point(screen);
            return;
        }
        self.draw_field(screen);
    }

    /// Go `Draw` 尾段（`main.go:9799`..`10188`）：地圖、原生整幀、回合橫幅、對話、文字層、戰役介面、指令環與
    /// 系統介面、舊 fade 疊層。R5 階段 3：`present.rs` 的呈現回報在尾段也呼叫它（狀態改寫與 `Draw` 同一份程式）。
    pub(crate) fn draw_field(&mut self, screen: &mut Picture) {
        screen.fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
        let mut native_map_presented = false;
        let campaign_battle_view =
            self.camp.is_none() || self.camp.as_ref().and_then(|c| c.node()).is_some_and(|n| n.r#type == "battle");
        // Go modernHDMapFrameAdmission（HD 守門）：modernStoryPortraits 為 nil 時兩條路徑都回 false。
        let legacy_viewport = self.story_bg || self.battle_event.is_some();
        let modern_hd_requested = self.modern_hd_map_frame_admission(legacy_viewport, campaign_battle_view);
        if !modern_hd_requested {
            // main.go:9814..9950：正規化地圖層。FD2_UNIT_LABELS、F3 除錯字與 drawUnitHUD 在開場與第一章沒有走到（r8）。
            self.draw_normalized_world_layer(screen, legacy_viewport);
            self.draw_field_cursor_hud(screen, legacy_viewport, campaign_battle_view);
        }
        if modern_hd_requested {
            match self.draw_modern_hd_map_frame(screen) {
                Ok(()) => native_map_presented = true,
                Err(err) => {
                    self.load_err = err.to_string();
                    return;
                }
            }
        } else if self.native_map_frame_admission(self.story_bg, campaign_battle_view) {
            native_map_presented = self.draw_native_map_frame(screen);
        }
        self.draw_field_map_overlays(screen, native_map_presented, legacy_viewport, campaign_battle_view);
        let native_story_dialogue_drawn = self.draw_native_story_dialogue(screen);
        self.posters.native_dialogue_visible =
            native_story_dialogue_drawn && !modern_hd_requested && self.has_hd_story_map_source();
        if self.font.is_some() {
            if let Some(st) = self.st.as_ref()
                && !legacy_viewport
                && !native_map_presented
                && st.unit_at(self.cur_x, self.cur_y).is_some()
            {
                // main.go:9974..9984 游標格上方的單位中文名；開場與第一章條件成立時都沒有游標單位。
                todo!("r8:main.go:9975 選中單位中文名（開場與第一章沒有走到）");
            }
            if !native_story_dialogue_drawn
                && !self.dialog.is_empty()
                && !(self.story_bg && self.walk_first && !self.story_walks.is_empty())
                && self.dlg_shown != DLG_NONE
            {
                self.draw_legacy_story_dialogue(screen);
            }
            // main.go:10122..10147：TTF 戰場短訊與勝負字樣（R5 r5f；拆成 `battle_text_draws` 讓局部黃金呼叫同一份程式）。
            let (draws, finished) = self.battle_text_draws();
            for d in &draws {
                self.font.as_mut().unwrap().draw(screen, &d.text, d.x, d.y, d.scale, d.clr);
            }
            if !finished {
                return;
            }
        }
        self.draw_campaign_ui(screen);
        if !self.draw_field_command_ui(screen, native_map_presented) {
            return;
        }
        self.draw_story_fade_overlay(screen);
        self.draw_capture_shot_point(screen);
    }

    /// Go `Draw` 本體 `main.go:9923..9941`（`!modernHDRequested` 區塊內、正規化世界層之後）：游標白框與左下 HUD。
    ///
    /// R5 階段 2 r5h：從 [`Game::draw_field`] 拆出來，只為了讓局部黃金（`parity_replay/rust_golden_r5h_layers_test.rs`）
    /// 呼叫同一份程式。`drawUnitHUD` 後備與 F3 除錯字在開場與第一章沒有走到（`r8`）。
    pub(crate) fn draw_field_cursor_hud(
        &mut self,
        screen: &mut Picture,
        legacy_viewport: bool,
        campaign_battle_view: bool,
    ) {
        let (tw, th) = {
            let m = self.m.as_ref().expect("draw_frame 已檢查 g.m");
            (m.tile_w, m.tile_h)
        };
        // 游標(白框):指令環/法術選單開啟時不顯示(原版該狀態下選取指示只在環上的選中圖示,
        // 常駐白框會疊在中央、與環的選中框混淆,見 playfix #5)
        let cur_px = (self.cur_x * tw) as f64 - self.cam_x;
        let cur_py = (self.cur_y * th) as f64 - self.cam_y;
        if !self.ring && !self.spell_open && !legacy_viewport && campaign_battle_view {
            draw_cursor(screen, cur_px, cur_py, tw as f64, th as f64);
        }
        // HUD(對照原版 orig_04/08):游標單位資訊=左下面板(非常駐頂列);回合切換=中央大字橫幅。
        if self.st.is_some() && self.font.is_some() && !legacy_viewport {
            let cursor_unit = self.st.as_ref().unwrap().unit_at(self.cur_x, self.cur_y);
            if cursor_unit.is_some() && !self.draw_native_map_hud(screen) {
                todo!("r8:main.go:drawUnitHUD（原生左下面板畫不出來時的後備；開場與第一章沒有走到）");
            }
            if self.debug {
                todo!("r8:main.go:9939 F3 除錯字 ebitenutil.DebugPrintAt（開場與第一章沒有開）");
            }
        }
    }

    /// Go `Draw` 本體 `main.go:9964..9972`：原生整幀之後的移動面板與回合橫幅（TTF 後備）。
    ///
    /// R5 階段 2 r5h：從 [`Game::draw_field`] 拆出來，理由同 [`Game::draw_field_cursor_hud`]。
    pub(crate) fn draw_field_map_overlays(
        &mut self,
        screen: &mut Picture,
        native_map_presented: bool,
        legacy_viewport: bool,
        campaign_battle_view: bool,
    ) {
        if native_map_presented {
            self.draw_native_movement_panel(screen);
        }
        // 回合橫幅(PLAYER/ENEMY PHASE，transient)必須畫在原生整幀之後：
        // drawNativeMapFrame 是覆蓋全畫面的 320×200 blit，畫在它前面等於沒畫。
        // 這一層是重製端的近似呈現，原版自己的回合字樣仍未解出，不宣稱一致。
        if !legacy_viewport && campaign_battle_view {
            self.draw_phase_banner(screen);
        }
    }

    /// Go `Draw` 本體 `main.go:10156..10174`：設施／班級介面工作、END 對話、指令環、指令格、法術與物品選單、
    /// 系統資訊與角色狀態。回 false 表示 Go 在 `transientUI && drawNativeClassUIJob` 成立時 `return` 出 `Draw`。
    ///
    /// R5 階段 2 r5h：從 [`Game::draw_field`] 拆出來，理由同 [`Game::draw_field_cursor_hud`]。
    pub(crate) fn draw_field_command_ui(&mut self, screen: &mut Picture, native_map_presented: bool) -> bool {
        // Command UI is an indexed-sprite layer; it must not depend on an optional
        // Chinese font being available in the runtime environment.
        if self.transient_ui && self.draw_native_class_ui_job(screen) {
            return false;
        }
        if !self.draw_native_system_end_turn(screen) {
            if self.native_action_overlay_in_map_frame(native_map_presented) {
                // The admitted indexed frame already owns icons then cursor unit.
                // A second drawRing changes their placement and covers that unit.
                self.mark_action_overlay_drawn();
            } else {
                self.draw_ring(screen);
            }
            self.draw_native_command_grid(screen);
            self.draw_spell_menu(screen);
            self.draw_item_menu(screen);
        }
        // 原版戰場資訊是 320×200 indexed 全畫面，最後覆蓋戰場與選單層。
        self.draw_native_system_info_ui(screen);
        self.draw_native_player_status(screen);
        true
    }

    /// Go `Draw` 本體 `main.go:9812..9919`：正規化世界層，到游標之前為止。
    ///
    /// story 場景與原版阻塞 battle event 走 320×200 離屏（`g.storyView`）再放大 `storyZoom` 倍；一般可操作戰場在
    /// 640×400 直繪。依序畫圖塊、移動範圍格、施法／攻擊射程格、單位（`drawUnitSprite`，已行動單位 `ColorScale`
    /// 變暗）、storyActors，離屏時放大貼回 `screen`。R5 階段 2 `r5p` 逐行移植；從 [`Game::draw_field`] 拆出來，
    /// 只是讓局部黃金（`parity_replay/rust_golden_r5p_layers_test.rs`）呼叫同一份程式。`g.storyView` 暫時取出成
    /// 區域變數，讓畫到它時仍可呼叫 `&self` 的 `tile_at`／`draw_unit_sprite`。
    pub(crate) fn draw_normalized_world_layer(&mut self, screen: &mut Picture, legacy_viewport: bool) {
        let (tw, th) = {
            let m = self.m.as_ref().expect("draw_frame 已檢查 g.m");
            (m.tile_w, m.tile_h)
        };
        let (view_w, view_h) =
            if legacy_viewport { (LOGICAL_W / STORY_ZOOM, LOGICAL_H / STORY_ZOOM) } else { (LOGICAL_W, LOGICAL_H) };
        // Go `g.storyView` 只在第一次需要時建立，之後每幀 Fill 黑重用。
        let mut story_view = if legacy_viewport {
            let mut view = self
                .story_view
                .take()
                .unwrap_or_else(|| Arc::new(Picture::new_image(LOGICAL_W / STORY_ZOOM, LOGICAL_H / STORY_ZOOM)));
            Arc::make_mut(&mut view).fill(ColorRgba { r: 0, g: 0, b: 0, a: 0xff });
            Some(view)
        } else {
            None
        };
        {
            let target: &mut Picture = match story_view.as_mut() {
                Some(view) => Arc::make_mut(view),
                None => &mut *screen,
            };
            let m = self.m.as_ref().expect("draw_frame 已檢查 g.m");
            // 只畫可見範圍
            let x0 = (self.cam_x as i64) / tw;
            let y0 = (self.cam_y as i64) / th;
            let x1 = (self.cam_x as i64 + view_w) / tw + 1;
            let y1 = (self.cam_y as i64 + view_h) / th + 1;
            let mut cy = y0;
            while cy <= y1 && cy < m.h {
                let mut cx = x0;
                while cx <= x1 && cx < m.w {
                    if cy < 0 || cx < 0 {
                        cx += 1;
                        continue;
                    }
                    if let Some(t) = self.tile_at(m.tiles[(cy * m.w + cx) as usize]) {
                        let mut op = DrawImageOptions::default();
                        op.geo_m.translate((cx * tw) as f64 - self.cam_x, (cy * th) as f64 - self.cam_y);
                        target.draw_image(&t, &op);
                    }
                    cx += 1;
                }
                cy += 1;
            }
            // 移動範圍高亮(已選單位:藍色半透明格)
            if let Some(sel) = self.sel {
                let mut hl = Picture::new_image(tw, th);
                hl.fill(ColorRgba { r: 0x40, g: 0x80, b: 0xff, a: 0x66 });
                // Go 走訪 map 的順序不固定；每格位置不同、互不重疊，順序不影響像素。
                for c in self.reach.keys() {
                    let mut op = DrawImageOptions::default();
                    op.geo_m.translate((c.x * tw) as f64 - self.cam_x, (c.y * th) as f64 - self.cam_y);
                    target.draw_image(&hl, &op);
                }
                let st = self.st.as_ref().expect("g.sel 非 nil 時有 g.st");
                if let Some(sp) = self.cast_sp.as_ref() {
                    // 施法射程高亮(紫)
                    let mut ch = Picture::new_image(tw, th);
                    ch.fill(ColorRgba { r: 0xa0, g: 0x50, b: 0xe0, a: 0x5c });
                    for y in 0..m.h {
                        for x in 0..m.w {
                            if st.in_cast_range(sel, sp, x, y) {
                                let mut op = DrawImageOptions::default();
                                op.geo_m.translate((x * tw) as f64 - self.cam_x, (y * th) as f64 - self.cam_y);
                                target.draw_image(&ch, &op);
                            }
                        }
                    }
                }
                // 原生物品／command 0 目標 modal 只由完整的 0x11CAC 索引畫面
                // 呈現。缺少 raw target field、LUT 或 range sprite 時，不畫沒有
                // 原版證據的 RGBA 色塊作後備，避免把 renderer 失敗偽裝成成功。
                // 攻擊射程高亮(紅;已移動、選攻擊、尚未選中目標的階段,doc32 武器攻擊距離接線)
                if self.cast_sp.is_none() && self.moved && !self.ring && !self.spell_open {
                    let mut ah = Picture::new_image(tw, th);
                    ah.fill(ColorRgba { r: 0xe0, g: 0x30, b: 0x30, a: 0x5c });
                    for y in 0..m.h {
                        for x in 0..m.w {
                            if st.in_attack_range(sel, x, y) {
                                let mut op = DrawImageOptions::default();
                                op.geo_m.translate((x * tw) as f64 - self.cam_x, (y * th) as f64 - self.cam_y);
                                target.draw_image(&ah, &op);
                            }
                        }
                    }
                }
            }
            // 單位層(M1:FIGANI 待機動畫 sprite + 陣營腳標 + HP bar;無 sprite 退回色塊)
            if let Some(st) = self.st.as_ref() {
                for (i, u) in st.units.iter().enumerate() {
                    if !u.on_field || !u.alive() {
                        // 待命(未進場)單位不畫
                        continue;
                    }
                    let ux = (u.x * tw) as f64 - self.cam_x;
                    let uy = (u.y * th) as f64 - self.cam_y;
                    if ux < -(tw as f64) || ux > view_w as f64 || uy < -(th as f64) || uy > view_h as f64 {
                        continue;
                    }
                    self.draw_unit_sprite(target, ux, uy, tw as f64, th as f64, u, self.map_sprite_group(i));
                }
            }
            // storyBG 場景靜態角色(doc23 §4:王座廳國王/王后/主角等 cutscene 擺位,同一 sprite 繪法無戰鬥邏輯)
            for u in &self.story_actors {
                if !u.on_field || !u.alive() {
                    continue;
                }
                let ux = (u.x * tw) as f64 - self.cam_x;
                let uy = (u.y * th) as f64 - self.cam_y;
                self.draw_unit_sprite(target, ux, uy, tw as f64, th as f64, u, u.fig);
            }
        }
        if let Some(view) = story_view {
            // 離屏世界層放大貼回畫布(48px/格,原版取景)
            let mut op = DrawImageOptions::default();
            op.geo_m.scale(STORY_ZOOM as f64, STORY_ZOOM as f64);
            screen.draw_image(&view, &op);
            self.story_view = Some(view);
            if self.unit_labels {
                todo!("r8:main.go:9911 FD2_UNIT_LABELS 的 ebitenutil.DebugPrintAt（開場與第一章沒有開）");
            }
        }
    }

    /// Go `Draw` 本體 `main.go:10173..10183`：場景淡出/淡入轉場(doc46 §5.2)，全螢幕黑色疊層，alpha 隨 `fade.t`
    /// 漸變。R5 階段 2 `r5p` 逐行移植；從 [`Game::draw_field`] 拆出來的理由同 [`Game::draw_normalized_world_layer`]。
    /// `main.go:10122..10147`（Draw 尾段 `if g.font != nil` 裡）：依序列出要以 `g.font.Draw` 畫的 TTF 戰場短訊與
    /// 勝負字樣。第二個值是 false 時 Go 在這裡 `return` 出 `Draw`（語系訊息缺鍵；列出的字仍先畫）。
    ///
    /// Go 沒有這個函式邊界：Go 是邊判斷邊 `g.font.Draw`，畫字不改判斷用到的狀態，所以先列出再畫結果相同。
    /// 拆出來只為了讓局部黃金（`rust_golden_ttf_layers_test.rs`）呼叫同一份程式。
    pub(crate) fn battle_text_draws(&mut self) -> (Vec<BattleTextDraw>, bool) {
        let mut draws = Vec::new();
        if self.should_draw_transient_battle_message() {
            // 攻擊等短訊(無對話框時)
            let clr = ColorRgba { r: 0xff, g: 0xf0, b: 0xb4, a: 0xff };
            draws.push(BattleTextDraw { text: self.msg.clone(), x: 8.0, y: LOGICAL_H as f64 - 30.0, scale: 1.2, clr });
        }
        if !self.result.is_empty() {
            // 勝負(中央大字)
            let (mut t, ok) = self.locale_message("battle.result.win", &[]);
            if !ok {
                return (draws, false);
            }
            let mut c = ColorRgba { r: 0xff, g: 0xdc, b: 0x50, a: 0xff };
            if self.result == "lose" {
                let (lose, ok) = self.locale_message("battle.result.lose", &[]);
                if !ok {
                    return (draws, false);
                }
                t = lose;
                c = ColorRgba { r: 0xff, g: 0x70, b: 0x70, a: 0xff };
            }
            let (cx, cy) = (LOGICAL_W as f64 / 2.0, LOGICAL_H as f64 / 2.0);
            draws.push(BattleTextDraw { text: t, x: cx - 78.0, y: cy - 30.0, scale: 3.0, clr: c });
            if self.camp.is_some() {
                let (continue_text, ok) = self.locale_message("battle.result.continue", &[]);
                if !ok {
                    return (draws, false);
                }
                let gray = ColorRgba { r: 0xe0, g: 0xe0, b: 0xe0, a: 0xff };
                draws.push(BattleTextDraw { text: continue_text, x: cx - 70.0, y: cy + 36.0, scale: 1.0, clr: gray });
            }
        }
        (draws, true)
    }

    pub(crate) fn draw_story_fade_overlay(&self, screen: &mut Picture) {
        if let Some(fade) = self.fade.as_ref() {
            let frac = fade.t as f64 / fade.total as f64;
            let mut alpha = frac;
            if !fade.out {
                alpha = 1.0 - frac;
            }
            let mut ov = Picture::new_image(LOGICAL_W, LOGICAL_H);
            // Go `uint8(alpha * 0xff)`：amd64 先截成 int64 再取低 8 位元（Rust 的 `as u8` 會飽和，範圍外結果不同）。
            ov.fill(ColorRgba { r: 0, g: 0, b: 0, a: (alpha * 255.0) as i64 as u8 });
            screen.draw_image(&ov, &DrawImageOptions::default());
        }
    }

    /// Go：`main.go:11586`，`func (g *Game) Layout(outsideW, outsideH int) (int, int)`（`r8`：Ebiten 主迴圈的
    /// 畫布尺寸回呼，重播測試不呼叫；只回傳固定的 640×400，照移）。
    pub(crate) fn layout(&self, outside_w: i64, outside_h: i64) -> (i64, i64) {
        let _ = (outside_w, outside_h);
        (LOGICAL_W, LOGICAL_H)
    }

    /// Go：`main.go:10305`，`func (g *Game) usesModernBattleActionIcons() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn uses_modern_battle_action_icons(&mut self) -> bool {
        self.interface_asset_theme().is_some_and(|style| style.battle_action_icons.len() == 4)
    }

    /// Go：`main.go:10339`，`func (g *Game) actionOverlayAvailability() [4]int`
    pub(crate) fn action_overlay_availability(&mut self) -> [i64; 4] {
        let mut availability = [0i64; 4];
        let (Some(sel), Some(st)) = (self.sel, self.st.as_ref()) else {
            return [1, 1, 1, 1];
        };
        let unit = &st.units[sel];
        let mut attack = false;
        // 0x1b83d first requires an equipped inventory entry whose ID is < 0x80.
        // The following target geometry is still the current remake range model;
        // the original item-record +0xb/+0xc calculation is not yet an adapter.
        if has_native_equipped_weapon(Some(unit)) {
            for other in &st.units {
                if other.on_field
                    && other.hp > 0
                    && other.camp != battle::OWN
                    && st.in_attack_range(sel, other.x, other.y)
                {
                    attack = true;
                    break;
                }
            }
        }
        if !attack {
            availability[0] = 1;
        }
        if has_native_command(Some(unit)) {
            // A raw command inventory follows the original unit+0x27 gate.  The
            // normalized Spells list is retained only for old editable scenarios;
            // it has its own legacy Sealed gate and is not evidence about FD2 ABI.
            if native_command_action_blocked(Some(unit)) {
                availability[1] = 1;
            }
        } else if unit.sealed || unit.spells.is_empty() {
            availability[1] = 1;
        }
        // 0x18a5f..0x18a72：真的走過路（0x4e1a6 步數非 0／0xff，經 0x13488）之後才進
        // 0x18d8c 的那一條，會先把表的 +4 設 1，只放過 raw +7 為 0x12／0x13／0x22 的
        // 單位；原地開環走 0x18b33 沒有這一條。第四章 r9 收據 seq 459：有指令的單位
        // 走完五格，指令格是紅的。
        if self.moved
            && (unit.x != self.sel_orig_x || unit.y != self.sel_orig_y)
            && !native_command_allowed_after_move(unit.battle_fig)
        {
            availability[1] = 1;
        }
        if unit.native_inventory_flags.len() == 8 {
            match battle::native_inventory_available_count(&unit.native_inventory_flags) {
                Ok(count) if count != 0 => {}
                _ => availability[2] = 1,
            }
        } else if unit.inventory.is_empty() {
            // Legacy editable JSON has no raw eight-cell flags; this is explicitly
            // a compatibility approximation, not native item availability.
            availability[2] = 1;
        }
        availability
    }
}

/// hasNativeEquippedWeapon is the recovered 0x1b83d inventory precondition.
/// Inventory/Equipped are the remake's compact projection of the native eight
/// slots; a missing equipped entry is deliberately not treated as a weapon.
///
/// Go：`main.go:10413`，`func hasNativeEquippedWeapon(unit *battle.Unit) bool`
pub(crate) fn has_native_equipped_weapon(unit: Option<&battle::Unit>) -> bool {
    let Some(unit) = unit else {
        return false;
    };
    if unit.native_inventory_flags.len() == 8 && unit.inventory_slots.len() == 8 {
        let slot = battle::native_equipped_inventory_slot(&unit.native_inventory_flags, &unit.inventory_slots, 0);
        return slot.is_ok_and(|slot| slot >= 0);
    }
    for (i, &item_id) in unit.inventory.iter().enumerate() {
        if (0..0x80).contains(&item_id) && i < unit.equipped.len() && unit.equipped[i] {
            return true;
        }
    }
    false
}

/// nativeCommandAllowedAfterMove 是 0x18a5f..0x18a70 的三個 raw +7 常數；其餘
/// 單位移動之後指令格一律不可選。
///
/// Go：`main.go:10431`，`func nativeCommandAllowedAfterMove(battleFig int) bool`
pub(crate) fn native_command_allowed_after_move(battle_fig: i64) -> bool {
    matches!(battle_fig, 0x12 | 0x13 | 0x22)
}

/// hasNativeCommand uses only the exact 0x1c269 bit inventory.  It must not
/// infer native command availability from the normalized editable Spells list.
///
/// Go：`main.go:10442`，`func hasNativeCommand(unit *battle.Unit) bool`
pub(crate) fn has_native_command(unit: Option<&battle::Unit>) -> bool {
    unit.is_some_and(|unit| unit.native_command_mask.iter().any(|&bits| bits != 0))
}

/// nativeCommandActionBlocked is the raw action-menu gate recovered from
/// 0x18d8c: after 0x1c269 finds a command, unit+0x27 nonzero disables the
/// command direction. Command 22 is a verified writer of this duration byte,
/// but its gameplay name and exhaustive producer set remain intentionally
/// unknown; NativeTransient[5] is the fail-closed storage of that byte.
///
/// Go：`main.go:10459`，`func nativeCommandActionBlocked(unit *battle.Unit) bool`
pub(crate) fn native_command_action_blocked(unit: Option<&battle::Unit>) -> bool {
    unit.is_none_or(|unit| unit.native_transient[5] != 0)
}

/// nativeActionSelectable mirrors the final 0x177fc availability check after
/// a direction is chosen: only a zero disabled-word may dispatch an action.
///
/// Go：`main.go:10465`，`func nativeActionSelectable(availability [4]int, direction int) bool`
pub(crate) fn native_action_selectable(availability: [i64; 4], direction: i64) -> bool {
    direction >= 0 && direction < availability.len() as i64 && availability[direction as usize] == 0
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
        if self.native_system_options_open {
            return match self.current_native_system_options().action_overlay_state() {
                Ok(state) => state,
                Err(_) => fdother::ActionOverlayState { availability: [-1, -1, -1, -1], ..Default::default() },
            };
        }
        if self.native_system_nested_open {
            todo!("R8: main.go:10525 nativeNestedSystemOverlayState（第一章重播沒有執行到）");
        }
        if self.native_system_cursor_overlay {
            return fdother::native_continue_action_overlay_state();
        }
        let availability = self.action_overlay_availability();
        fdother::battle_action_overlay_state(availability)
    }

    /// drawBattleScene 全螢幕戰鬥演出(對照原版 orig_05:守方左面右/攻方右土台/斬擊弧/血條/命中閃紅抽血)。
    ///
    /// Go：`main.go:11231`，`func (g *Game) drawBattleScene(screen *ebiten.Image)`
    ///
    /// R5 階段 2 r5b：開場與第一章每次都在原生前奏／本體（`drawNativePhysicalPrelude`）返回（階段 0 覆蓋率 6488 次，
    /// 其後 0 次）。其後的近似畫面（黑底、背景、`drawNativePhysicalBase` 與失敗時的 `loadErr`、TTF 狀態欄、FIGANI 守方／
    /// 台座／攻方圖層、紅色剪影）用到的 `drawNativePhysicalBase`、`battleSceneLayerOrder`、`redSilhouette` 等都是 r8，
    /// 整段 `todo!("r8:…")`。
    pub(crate) fn draw_battle_scene(&mut self, screen: &mut Picture) {
        if self.draw_native_physical_prelude(screen) {
            return;
        }
        todo!(
            "r8:main.go:11240..11391 drawBattleScene 原生前奏與本體之外的戰鬥畫面（drawNativePhysicalBase、TTF 狀態欄、FIGANI 圖層；開場與第一章沒有走到）"
        )
    }
}

pub(crate) const NATIVE_ACTION_OVERLAY_CELL_COUNT: i64 = 78;

/// loadNativeCommandLabels reads the editable export of FDTXT_000 command
/// labels. It is optional because the export is player-provided original text;
/// absence deliberately leaves the normalized presentation names intact.
///
/// Go：`main.go:11595`，`func loadNativeCommandLabels() map[int]string`
///
/// Go 失敗時回 nil map；Rust 回空 map（呼叫端只查鍵，兩者相同）。
pub(crate) fn load_native_command_labels() -> BTreeMap<i64, String> {
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Entry {
        #[serde(deserialize_with = "battle::null_as_default")]
        command_id: i64,
        #[serde(deserialize_with = "battle::null_as_default")]
        label: String,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Document {
        #[serde(deserialize_with = "battle::null_as_default")]
        entries: Vec<Entry>,
    }
    let Ok(raw) = std::fs::read(asset_path("assets/data/command_labels.json")) else {
        return BTreeMap::new();
    };
    let Ok(Some(document)) = serde_json::from_slice::<Option<Document>>(&raw) else {
        return BTreeMap::new();
    };
    document
        .entries
        .into_iter()
        .filter(|e| (0..40).contains(&e.command_id) && !e.label.is_empty())
        .map(|e| (e.command_id, e.label))
        .collect()
}

/// Go：`main.go:11619`，`func loadGame() *Game`
///
/// Go 一律回傳非 nil 的 `*Game`（錯誤寫在 `loadErr`）；Rust 一律回傳 `Some`。
///
/// 和 Go 不同的地方：
/// - 亂數種子：Go 沒有 `FD2_SEED` 時用 `time.Now().UnixNano()`；`fd2-game` 不讀主機時間，改用
///   `VIRTUAL_CLOCK_ORIGIN` 的奈秒值（KNOWN_DIFFERENCES `load-game-seed-no-wall-clock`），要不同種子的
///   外殼設定 `FD2_SEED` 或在載入後換掉 `rng`。
/// - HD 美術讀取 `FD2_THEME` 或設定的 `modern-handpainted-a`。一般啟動將素材驗證交給背景執行緒。
///   HD 介面、結局預覽、晚期整備截圖、截圖隊伍及快轉另行移植。
pub(crate) fn load_game() -> Option<Game> {
    fn env(name: &str) -> String {
        std::env::var(name).unwrap_or_default()
    }
    fn png_picture(rel: &str) -> Option<Arc<Picture>> {
        let raw = std::fs::read(asset_path(rel)).ok()?;
        decode_png_picture(&raw).map(Arc::new)
    }
    let mut g = Game {
        shot_frame: 20,
        native_map_hud_persistent: battle::initial_native_map_hud_persistent_state(),
        ..Default::default()
    };
    let configured = load_settings();
    g.player_speed = normalize_player_speed(configured.game_speed);
    g.player_battle_speed = normalize_battle_animation_speed(configured.battle_animation_speed);
    g.player_hd_quality = parse_hd_texture_quality(&configured.hd_texture_quality);
    let quality = env("FD2_HD_TEXTURE_QUALITY");
    if !quality.is_empty() {
        if quality != "full" && quality != "balanced" && quality != "light" {
            g.load_err = format!("HD texture quality: unsupported value {quality}");
            return Some(g);
        }
        g.player_hd_quality = parse_hd_texture_quality(&quality);
    }
    g.player_clock.speed = g.player_speed;
    g.bgm_source = configured.bgm_source.clone(); // 音源設定(預設 fm=Sound Blaster)
    g.locale_id = configured.locale_id.clone();
    g.apply_font_scale(configured.font_scale);
    let locale = env("FD2_LOCALE");
    if !locale.is_empty() {
        if LOCALE_DISPLAY_NAME.get(&locale).is_none_or(|n| n.is_empty()) {
            g.load_err = format!("locale setting: unsupported locale {locale}");
            return Some(g);
        }
        g.locale_id = locale;
    }
    match load_official_locale(&g.locale_id) {
        Ok(catalog) => g.locale_catalog = catalog,
        Err(err) => {
            g.load_err = format!("locale setting: {}", err.message());
            return Some(g);
        }
    }
    match load_official_locale_content(&g.locale_id) {
        Ok(content) => g.locale_content = content,
        Err(err) => {
            g.load_err = format!("locale content: {}", err.message());
            return Some(g);
        }
    }
    match load_official_locale_entities(&g.locale_id) {
        Ok(entities) => g.locale_entities = entities,
        Err(err) => {
            g.load_err = format!("locale entities: {}", err.message());
            return Some(g);
        }
    }
    let mut theme_id = env("FD2_THEME");
    if theme_id.is_empty() {
        theme_id = configured.theme_id.clone();
    }
    let modern_art = theme_id == MODERN_HANDPAINTED_THEME_ID;
    if !modern_art && !theme_id.is_empty() && theme_id != "faithful" {
        g.load_err = format!("modern theme: unsupported theme {theme_id}");
        return Some(g);
    }
    let mode = env("FD2_INTERFACE_MODE");
    if !mode.is_empty() && mode != PLAYER_INTERFACE_ORIGINAL && mode != PLAYER_INTERFACE_HD {
        g.load_err = format!("interface: unsupported mode {mode}");
        return Some(g);
    }
    g.interface_mode = PLAYER_INTERFACE_ORIGINAL.into();
    let bgm = env("FD2_BGM_SOURCE");
    if !bgm.is_empty() && BGM_SOURCE_NAME.get(&bgm).is_some_and(|n| !n.is_empty()) {
        g.bgm_source = bgm; // 覆寫(截圖/測試用)
    }
    g.unit_labels = !env("FD2_UNIT_LABELS").is_empty();
    g.cutscene_log = !env("FD2_CUTSCENE_LOG").is_empty();
    g.approximate_mode = env("FD2_APPROXIMATE") == "1";
    g.shot_path = env("FD2_SHOT");
    g.shot_series = env("FD2_SHOT_SERIES");
    if let Ok(n) = env("FD2_SHOT_FRAME").parse::<i64>() {
        g.shot_frame = n;
    }
    let cur = env("FD2_SHOT_CUR");
    if !cur.is_empty() {
        // Go `fmt.Sscanf(v, "%d,%d", &x, &y)`：逐個掃描，第一個成功就寫入。
        let mut parts = cur.splitn(2, ',');
        if let Some(Ok(x)) = parts.next().map(|p| p.trim_start().parse::<i64>()) {
            g.shot_cur_x = x;
            if let Some(Ok(y)) = parts.next().map(|p| p.trim_start().parse::<i64>()) {
                g.shot_cur_y = y;
            }
        }
    }
    if let Ok(n) = env("FD2_SHOT_TURN").parse::<i64>() {
        g.shot_turn = n;
    }
    if !env("FD2_ENDING_PREFIX").is_empty() {
        match new_native_ending_preview() {
            Ok(preview) => g.native_ending = Some(preview),
            Err(err) => {
                g.load_err = format!("native ending: {}", err.message());
                return Some(g);
            }
        }
        // 直接預覽刻意略過地圖／戰鬥載入；抵達已還原文字分支後，
        // 0x2c39b owner 仍只使用玩家提供的 DATO、FDOTHER 與 FDTXT 資產。
        g.portraits = load_portraits();
        g.font = load_font();
        return Some(g);
    }
    if let Err(err) = g.load_map("assets") {
        g.load_err = err.message().to_string();
        return Some(g);
    }
    // 載入單位(M1)
    match battle::load(&asset_path("assets/map0_units.json")) {
        Ok(mut st) => {
            if let Err(err) = g.bind_native_future_item_rows(Some(&mut st)) {
                g.load_err = format!("native future constructor item rows: {}", err.message());
            } else if let Err(err) = g.bind_native_movement_cost_rows(Some(&mut st)) {
                g.load_err = format!("native movement rows: {}", err.message());
            }
            g.st = Some(st);
        }
        Err(err) => {
            if g.load_err.is_empty() {
                g.load_err = format!("units: {}", err.message());
            }
        }
    }
    // 載入劇本 + 套用初始(待命 group + on_battle_start 主角隊進場,doc 25/29)
    if g.st.is_some() {
        match battle::load_scenario(&asset_path("assets/scenarios/ch01.json")) {
            Ok(sc) => {
                g.sc = Some(sc);
                g.bind_native_death_programs();
                let (mut sc, mut st) = (g.sc.take().unwrap(), g.st.take().unwrap());
                let result = sc.setup_checked(Some(&mut st));
                (g.sc, g.st) = (Some(sc), Some(st));
                match result {
                    Ok(dialogue) => g.dialog.extend(dialogue),
                    Err(err) => g.load_err = format!("scenario setup: {}", err.message()),
                }
                g.focus_on_party();
            }
            Err(err) => {
                if g.load_err.is_empty() {
                    g.load_err = format!("scenario: {}", err.message());
                }
            }
        }
    }
    g.sprites = load_sprites();
    // HD 美術在 draw_unit_sprite 取用主題圖組，原版圖組仍由 sprites 保有。
    g.portraits = load_portraits();
    g.figani = load_figani();
    match load_figani_delays() {
        Ok(delays) => g.figani_delays = delays,
        Err(err) => {
            if g.load_err.is_empty() {
                g.load_err = format!("FIGANI delays: {}", err.message());
            }
        }
    }
    g.fig_meta = load_fig_meta();
    g.native_ui_palette = load_native_ui_palette();
    if let Ok((font, glyphs)) = load_native_battle_name_assets() {
        (g.native_battle_font, g.native_battle_glyphs) = (font, glyphs);
    }
    if let Ok(panel_assets) = battle::load_native_battle_panel_value_assets(&separated_asset_path("")) {
        g.native_battle_panel = Some(panel_assets);
        g.native_battle_values = BTreeMap::new();
    }
    (g.native_action_cells, g.native_action_cells_raw) = load_native_action_cells_with_raw(&g.native_ui_palette);
    if let Ok(system_info) = load_native_system_info_assets() {
        g.native_system_info_assets = system_info;
    }
    if let Ok(load_slots_ui) = load_native_load_slots_ui_assets() {
        g.native_load_slots_ui = load_slots_ui;
    }
    if let Ok(Some(class_ui)) = load_native_class_ui_assets() {
        if let Ok(preparation_ui) = load_native_preparation_ui_assets() {
            g.native_preparation_ui = preparation_ui;
        }
        if let Ok(town_ui) = load_native_town_ui_assets() {
            g.native_town_ui = town_ui;
        }
        if let Ok(shop_ui) = load_native_shop_ui_assets(Some(&class_ui)) {
            g.native_shop_ui = shop_ui;
        }
        if let Ok(hotel_ui) = load_native_hotel_ui_assets(Some(&class_ui)) {
            g.native_hotel_ui = hotel_ui;
        }
        g.native_class_ui = Some(class_ui);
    }
    g.bg = png_picture("assets/bg/bg.png"); // 戰鬥背景(BG.DAT)
    g.tai = png_picture("assets/tai/tai_004.png"); // 我方台座(TAI_004 綠草橢圓)
    g.panel = png_picture("assets/ui/panel.png"); // 狀態欄框(LMI1 #22)
    g.dlg_box = png_picture("assets/ui/dialog.png"); // 對話框框(LMI1 #21)
    // Native 0x18d8c result order is attack, spell, item, wait.  There is no
    // separately proven spell icon in the extracted set; keep the historical
    // status asset as the spell slot until the resource table is decoded.
    for (i, nm) in ["attack", "status", "item", "wait"].into_iter().enumerate() {
        if let Some(icon) = png_picture(&format!("assets/ui/ring_{nm}.png")) {
            g.ring_icons[i] = Some(icon);
        }
    }
    // Go 以 g.modernStoryPortraits.battleActionIcons 覆寫 ringIcons（HD 主題，R6）。
    if let Ok(sp) = battle::load_spells(&asset_path("assets/spells.json")) {
        // 法術表(EXE dump)
        g.native_command_labels = load_native_command_labels();
        match localized_spell_book(g.locale_entities.as_ref(), &sp) {
            Ok(sp) => {
                if let Some(st) = g.st.as_mut() {
                    st.spell_book = sp.clone();
                }
                g.spells = sp;
            }
            Err(err) => {
                g.load_err = format!("localized spell book: {}", err.message());
                return Some(g);
            }
        }
    }
    match battle::load_native_command_records(&asset_path("assets/spells.json")) {
        Ok(records) => {
            g.native_command_book = records;
            let mut st = g.st.take();
            g.bind_native_command_book(st.as_mut());
            g.st = st;
        }
        Err(err) => {
            if g.load_err.is_empty() {
                g.load_err = format!("native command records: {}", err.message());
            }
        }
    }
    match battle::load_native_command_resistances(&asset_path("assets/data/native_command_resistances.json")) {
        Ok(resistances) => {
            g.native_command_resistances = resistances;
            let mut st = g.st.take();
            g.bind_native_command_resistances(st.as_mut());
            g.st = st;
        }
        Err(err) => {
            if g.load_err.is_empty() {
                g.load_err = format!("native command resistances: {}", err.message());
            }
        }
    }
    let mut learn_path = asset_path("assets/data/command_learn.json");
    if !file_exists(&learn_path) {
        learn_path = String::from("../docs/data/exe_tables/command_learn.json");
    }
    match battle::load_command_learn(&learn_path) {
        Ok(table) => g.command_learn = table,
        Err(err) => {
            if g.load_err.is_empty() {
                g.load_err = format!("command learn: {}", err.message());
            }
        }
    }
    let mut learn_selector_path = asset_path("assets/data/class_change_growth.json");
    if !file_exists(&learn_selector_path) {
        learn_selector_path = String::from("../docs/data/exe_tables/growth.json");
    }
    match battle::load_command_learn_selectors(&learn_selector_path) {
        Ok(selectors) => g.command_learn_selectors = selectors,
        Err(err) => {
            if g.load_err.is_empty() {
                g.load_err = format!("command learn selectors: {}", err.message());
            }
        }
    }
    // 原生0x1E292以byte +7直接取槽；68列後的相鄰讀取與負跨距見第十五章#81。
    // 角色轉職／學招仍使用作者的68列；原生成長承接完整256槽的明示資料。
    let mut native_growth_path = asset_path("assets/data/native_growth_slots.json");
    if !file_exists(&native_growth_path) {
        native_growth_path = String::from("../docs/data/exe_tables/native_growth_slots.json");
    }
    match battle::load_native_growth_slots(&native_growth_path) {
        Ok(rows) => g.native_growth_rows = rows,
        Err(err) => {
            if g.load_err.is_empty() {
                g.load_err = format!("native growth rows: {}", err.message());
            }
        }
    }
    let mut st = g.st.take();
    g.bind_command_learn(st.as_mut());
    g.st = st;
    if let Ok(commands) = campaign::load_ai_command_spell_map(&asset_path("assets/data/item.json"))
        && let Some(st) = g.st.as_mut()
    {
        st.ai_command_spell = commands;
    }
    if let Ok((types, equip)) = campaign::load_shop_eligibility(
        &asset_path("assets/data/item.json"),
        &asset_path("assets/data/class_equip_types.json"),
    ) {
        (g.shop_item_types, g.shop_equip_types) = (types, equip);
    }
    if let Ok(prices) = campaign::load_item_prices(&asset_path("assets/data/item.json")) {
        g.shop_item_prices = prices;
    }
    if let Ok(stats) = campaign::load_item_stats(&asset_path("assets/data/item.json")) {
        g.shop_item_stats = stats;
    }
    if let Ok(rates) = campaign::load_revive_fee_rates(&asset_path("assets/data/revive_fee_rates.json")) {
        g.revive_fee_rates = rates;
    }
    let mut class_table_path = asset_path("assets/data/class_change_targets.json");
    if std::fs::metadata(&class_table_path).is_err() {
        class_table_path = String::from("docs/data/exe_tables/class_change_targets.json");
    }
    if let Ok(table) = campaign::load_class_change_table(&class_table_path) {
        g.class_change_table = table;
    }
    let mut growth_path = asset_path("assets/data/class_change_growth.json");
    if std::fs::metadata(&growth_path).is_err() {
        growth_path = String::from("docs/data/exe_tables/growth.json");
    }
    if let Ok(growth) = campaign::load_class_change_growth(&growth_path) {
        g.class_change_growth = growth;
    }
    if let Ok(table) =
        campaign::load_native_join_constructor_table(&asset_path("assets/data/native_join_constructor.json"))
    {
        g.native_join_constructor = table;
        g.has_native_join_constructor = true;
    }
    if let Ok(table) = campaign::load_native_join_base_table(&asset_path("assets/data/native_join_base_units.json")) {
        g.native_join_bases = table;
        g.has_native_join_bases = true;
    }
    if let Ok(rows) =
        battle::load_native_item_effect_row_prefix(&asset_path("assets/data/native_item_effect_rows.json"))
    {
        g.native_join_item_effect_rows = rows;
    }
    if let Ok(table) =
        battle::load_native_command_palette_flash_table(&asset_path("assets/data/native_command_palette_flash.json"))
    {
        g.native_command_palette_flash = Some(table);
    }
    match battle::load_native_command_scene_table(&asset_path("assets/data/native_command_scene.json")) {
        Ok(table) => g.native_command_scene = Some(table),
        Err(err) => {
            if g.load_err.is_empty() {
                g.load_err = format!("native command scene: {}", err.message());
            }
        }
    }
    let mut st = g.st.take();
    g.initialize_equipment_bases(st.as_mut());
    g.st = st;
    g.font = load_font();
    // 狀態欄名字專用整數尺寸 face(scale 1.0 繪製,避免非整數縮放模糊);orig 名墨高 13px→face 28
    g.font_nm = load_font_sized(28.0);
    let font_scale = g.font_scale;
    g.apply_font_scale(font_scale);
    for k in 0..10 {
        // 原版數字 cell(LMI1 #31-40)
        if let Some(digit) = png_picture(&format!("assets/ui/digit_{k}.png")) {
            g.digits[k] = Some(digit);
        }
    }
    g.gold = 0; // 普通 START 戰況資訊為零；92 保存原版畫面，移除早期猜測的 1000。
    let mut seed = VIRTUAL_CLOCK_ORIGIN.0;
    if let Ok(v) = env("FD2_SEED").parse::<i64>() {
        seed = v;
    }
    g.rng = Some(battle::GoMathRand::new(seed));
    match load_sfx() {
        Ok(bank) => g.sfx = bank,
        Err(err) => {
            if g.load_err.is_empty() {
                g.load_err = format!("separated UI sounds: {}", err.message());
            }
        }
    }
    match load_separated_command_sound_banks() {
        Ok(banks) => g.install_separated_command_sounds(&banks),
        Err(err) => {
            if g.load_err.is_empty() {
                g.load_err = format!("separated command sounds: {}", err.message());
            }
        }
    }
    // 戰鬥音效:揮擊/命中/陣亡三段(真素材;attack_id→池 精確對照 doc36 未 RE,故命中/陣亡池為近似選擇)
    g.sfx_swing = load_wav("assets/sfx/battle_48_00.wav"); // 揮擊(池 sub0,七池共用)
    g.sfx_impact = load_wav("assets/sfx/battle_64_00.wav"); // 命中(最短最尖池)
    // FDMUS_018 的已閉合用途是 boot／標題，不在此猜測戰場或商店曲目。
    let title = env("FD2_TITLE");
    if title == "1" || (g.shot_path.is_empty() && title != "0") {
        // 開頭動畫+主選單(headless 截圖預設跳過)
        match load_title_assets() {
            Ok(ta) => g.title_assets = ta,
            Err(err) => {
                g.load_err = err.message().to_string();
                g.startup_blocked = true;
                return Some(g);
            }
        }
        if env("FD2_NOCUT").is_empty() {
            g.title_phase = String::from("cutscene"); // 分離 ANI 已在 loadTitleAssets 一次預檢；完整原版排程仍非 E2
        } else {
            g.title_phase = String::from("scroll"); // 明確測試／除錯旗標略過 AFM 過場
            g.scroll_y = 535.0;
        }
    }
    if env("FD2_SHOT_TITLE_MENU") == "1" {
        if g.shot_path.is_empty() || g.title_assets.is_none() {
            g.load_err = String::from("FD2_SHOT_TITLE_MENU requires FD2_SHOT and loaded title assets");
            return Some(g);
        }
        // 有界畫面 oracle：只跳過開場排程，不改正式玩家預設流程。
        g.enter_title_menu();
        g.title_flash = 0;
    }
    if !env("FD2_SHOT_LOAD_STATE").is_empty() {
        todo!("R8: native_load_slots_ui.go:parseNativeLoadSlotShotState（FD2_SHOT_LOAD_STATE 截圖）")
    }
    let mut cp = env("FD2_CAMPAIGN");
    if !cp.is_empty() {
        // 劇本節點圖模式(doc 19;放最後,story 對白不被開場 Setup 蓋掉)
        if cp == "1" {
            cp = String::from("assets/scenarios/campaign.json");
        }
        match load_player_campaign(&cp) {
            Ok(Some(c)) => {
                g.camp = Some(campaign::new_runner(&c));
                if env("FD2_SHOT_LATE_PREPARATION") == "1" {
                    todo!("R8: prepareReviewedLatePreparationShot（FD2_SHOT_LATE_PREPARATION 截圖）")
                }
                if !env("FD2_CAMP_CLASS_FIXTURE").is_empty() {
                    todo!("R8: FD2_CAMP_CLASS_FIXTURE（教會轉職畫面 oracle 的固定名冊）")
                }
                if !env("FD2_CAMP_PREP_BATTLE").is_empty() {
                    todo!("R8: FD2_CAMP_PREP_BATTLE（戰後 handler 截圖 oracle 的前置戰場）")
                }
                if !env("FD2_SHOT_PARTY_BINDING").is_empty() {
                    todo!("R8: materializeShotPartyFromBinding（FD2_SHOT_PARTY_BINDING 截圖）")
                }
                let node = env("FD2_CAMP_NODE");
                if !node.is_empty() && c.nodes.contains_key(&node) {
                    // 驗證鉤子:直接跳指定節點
                    let camp = g.camp.as_mut().unwrap();
                    camp.cur = node;
                    camp.flags.insert(String::from("found_secret"), !env("FD2_CAMP_SECRET").is_empty());
                }
                g.enter_node();
                // 只在明確的截圖證據模式壓縮既有 cutscene 的等待時間。一般
                // 玩家輸入、正常戰役與存檔永遠不會走這條分支；若任何 beat
                // 或原生 renderer 尚未閉合，保留 loadErr 而停止，不產生假正式畫面。
                if !g.shot_path.is_empty()
                    && env("FD2_SHOT_FAST_FORWARD") == "1"
                    && g.camp.as_ref().is_some_and(|c| c.cur == "story_ch00_handler")
                    && g.load_err.is_empty()
                {
                    todo!("R8: main.go:fastForwardShotCampaign（FD2_SHOT_FAST_FORWARD 截圖）")
                }
            }
            Ok(None) => {}
            Err(err) => g.load_err = format!("campaign: {}", err.message()),
        }
    }
    if g.load_err.is_empty() {
        if (mode == PLAYER_INTERFACE_HD || (mode.is_empty() && configured.interface_mode == PLAYER_INTERFACE_HD))
            && let Err(err) = g.enable_hd_title_interface()
        {
            g.msg = format!("介面切換失敗：{err}");
        }
        if modern_art {
            if modern_theme_startup_deferred() {
                g.request_modern_theme_art(false);
            } else if let Err(err) = g.load_startup_modern_theme() {
                g.load_err = format!("modern theme: {err}");
            }
        }
    }
    Some(g)
}

pub(crate) const PHASE_BANNER_ENEMY_HOLD_MILLIS: i64 = 20;

pub(crate) const PHASE_BANNER_PLAYER_HOLD_MILLIS: i64 = 150;

pub(crate) const PHASE_BANNER_PLAYER_TEXT: &str = "PLAYER PHASE";

/// Go `main.go:12142` `phaseBannerEnemyText`（R5 階段 2 補：測試與 `blitPhaseBannerGlyphs` 的對照用）。
pub(crate) const PHASE_BANNER_ENEMY_TEXT: &str = "ENEMY PHASE";

// 橫幅字樣的原版來源：FDOTHER #5 的 LMI1 cell。`sub_1F1CC` 由 `[0x53A81]`
// （`0x25CE5` 以 `sub_111BA("FDOTHER.DAT", …, 5)` 載入）取三格——`0x50` PLAYER、
// `0x51` PHASE、`0x52` ENEMY——再用 `sub_15F0E` 畫成兩塊。
//
// 兩塊的座標寫在 `sub_1F42D`（0x1F43F 起）：第一塊 x ＝ `0x55 − 參數`，第二塊
// （固定 `0x51` PHASE）x ＝ `參數 + 0xA5`，兩塊 y 都是 `0x52`。**兩塊是相向
// 移動的**——參數由 100 遞減到 0，第一塊由左往右、第二塊由右往左，在中間會合。
// 那組座標是離屏緩衝區的（`sub_11EB0` 之後整塊複製到 VGA 的 `0xA0504`，也就是
// (4,4)），換算成 VGA 就是停住時的 (0x59,0x56) 與 (0xA9,0x56)。
//
// 索引 0 是透明：字身用 2..9（落在 DAC 暗化不動的 0x00..0x0F），左下的投影
// 陰影用 0xFE。所以走 BlitAtClipped 的透明語意，不是 BlitOpaqueAt。
// Go：`main.go:12206`..`12213`（R5 階段 2 補）。
pub(crate) const PHASE_BANNER_GLYPH_PLAYER: i64 = 0x50;
pub(crate) const PHASE_BANNER_GLYPH_PHASE: i64 = 0x51;
pub(crate) const PHASE_BANNER_GLYPH_ENEMY: i64 = 0x52;
pub(crate) const PHASE_BANNER_GLYPH_FIRST_X: i64 = 0x59;
pub(crate) const PHASE_BANNER_GLYPH_SECOND_X: i64 = 0xA9;
pub(crate) const PHASE_BANNER_GLYPH_Y: i64 = 0x56;

/// phaseBannerHoldMillis 回傳中段停留的毫秒數。未知字樣時採敵方回合的 20 毫秒，
/// 這是兩者中較短的一個，不會憑空拉長演出。
///
/// Go：`main.go:12132`，`func phaseBannerHoldMillis(banner string) float64`
pub(crate) fn phase_banner_hold_millis(banner: &str) -> f64 {
    if banner == PHASE_BANNER_PLAYER_TEXT {
        return PHASE_BANNER_PLAYER_HOLD_MILLIS as f64;
    }
    PHASE_BANNER_ENEMY_HOLD_MILLIS as f64
}

/// phaseBannerTotalMillis 是整段橫幅的原版長度：字樣滑入 7 步、馬賽克 33 步、
/// 滑出 5 步，各一個 BIOS tick，加上中段停留。
///
/// Go：`main.go:12141`，`func phaseBannerTotalMillis(banner string) float64`
pub(crate) fn phase_banner_total_millis(banner: &str) -> f64 {
    let steps = indexedmap::PHASE_BANNER_SLIDE_IN_STEPS
        + indexedmap::PHASE_BANNER_STEPS
        + indexedmap::PHASE_BANNER_SLIDE_OUT_STEPS;
    steps as f64 * indexedmap::PHASE_BANNER_STEP_MILLIS + phase_banner_hold_millis(banner)
}

/// phaseBannerFrames 把原版長度換成 60 Hz 的重製端幀數(無條件進位，寧可多留一
/// 幀也不提前收掉最後一步)。
///
/// Go：`main.go:12175`，`func phaseBannerFrames(banner string) int`
pub(crate) fn phase_banner_frames(banner: &str) -> i64 {
    (phase_banner_total_millis(banner) * 60.0 / 1000.0).ceil() as i64
}

impl Game {
    /// Go：`main.go:12254`，`func (g *Game) showBanner(s string)`
    pub(crate) fn show_banner(&mut self, s: &str) {
        if !self.shot_path.is_empty() {
            return;
        }
        (self.banner, self.banner_t) = (s.to_string(), phase_banner_frames(s));
        // 原版在 `sub_1F1CC` 開頭 memcpy(0xA0000) 保存整幀，之後整段不重繪；快照
        // 在橫幅第一次繪製時取，那時的 nativeMapVGA 才是玩家看到的那一張。
        self.banner_frame = Vec::new();
    }

    /// drawNativeMapHUD presents the proven indexed 0x1acf3 panel/terrain/AP/DP
    /// subpasses at the native 320x200 surface. It returns false on any missing or
    /// unverified raw input, preserving the legacy approximation without partial
    /// drawing. Raw gates/anchor and optional unit/HP must all come from the
    /// materialized battle runtime; this function never hardcodes native globals.
    ///
    /// Go：`main.go:12356`，`func (g *Game) nativeMapHUDInput() (indexedmap.NativeMapHUDInput, bool)`
    pub(crate) fn native_map_hud_input(&mut self) -> (indexedmap::NativeMapHUDInput, bool) {
        let a = self.native_map_assets.as_ref();
        let (Some(m), Some(st)) = (self.m.as_ref(), self.st.as_ref()) else {
            self.native_map_hud_input_reason =
                String::from("資產、HUD／cycle 狀態、selector cache 或游標界線其一不成立");
            return (indexedmap::NativeMapHUDInput::default(), false);
        };
        if !native_map_assets_available(a)
            || !st.has_native_map_hud_state
            || !st.has_native_map_cycle_state
            || st.native_map_selector_cache.is_none()
            || self.cur_x < 0
            || self.cur_y < 0
            || self.cur_x >= m.w
            || self.cur_y >= m.h
        {
            self.native_map_hud_input_reason =
                String::from("資產、HUD／cycle 狀態、selector cache 或游標界線其一不成立");
            return (indexedmap::NativeMapHUDInput::default(), false);
        }
        let a = a.unwrap();
        let tile = if st.has_native_map_event_grid {
            let (t, ok) = st.native_map_tile_at(self.cur_x, self.cur_y);
            if !ok {
                self.native_map_hud_input_reason = String::from("原生可變地圖緩衝損壞");
                return (indexedmap::NativeMapHUDInput::default(), false);
            }
            t
        } else {
            let index = self.cur_y * m.w + self.cur_x;
            if index >= m.tiles.len() as i64 {
                self.native_map_hud_input_reason = String::from("初始地圖圖塊資料不足");
                return (indexedmap::NativeMapHUDInput::default(), false);
            }
            m.tiles[index as usize]
        };
        let descriptors = a.controls.len() as i64 / 4;
        if tile < 0 || tile >= descriptors || tile > 0x3ff {
            self.native_map_hud_input_reason = format!("terrain descriptor {} outside 0..{}", tile, descriptors - 1);
            return (indexedmap::NativeMapHUDInput::default(), false);
        }
        let control = a.controls[(tile * 4 + 1) as usize];
        let raw_hud = st.native_map_hud_state;
        let mut r#in = indexedmap::NativeMapHUDInput {
            display_gate_a: raw_hud.display_gate_a != 0,
            display_gate_b: raw_hud.display_gate_b != 0,
            anchor_x: raw_hud.anchor_x,
            terrain_descriptor: tile,
            terrain_control: control,
            optional_unit: None,
        };
        // 與 BlitNativeMapHUD 的原版顯示閘門一致；故事隱藏 HUD 時不消費
        // 游標下角色的血量／職業資料，不能因此拒絕整張背景。
        if !r#in.display_gate_a || !r#in.display_gate_b {
            return (r#in, true);
        }
        if let Some(ui) = st.unit_at(self.cur_x, self.cur_y) {
            let u = &st.units[ui];
            if !u.has_map_selector_slot
                || !u.has_battle_fig
                || u.battle_fig < 0
                || u.battle_fig > 0xff
                || !u.has_native_record_race
                || !u.has_native_record_byte6
                || !u.has_native_record_word42
                || u.hp < 0
                || u.hp > 0xffff
            {
                self.native_map_hud_input_reason = format!(
                    "游標下單位缺少 raw 出處：slot={} fig={}({}) race={} byte6={} word42={} hp={}",
                    u.has_map_selector_slot,
                    u.has_battle_fig,
                    u.battle_fig,
                    u.has_native_record_race,
                    u.has_native_record_byte6,
                    u.has_native_record_word42,
                    u.hp
                );
                return (indexedmap::NativeMapHUDInput::default(), false);
            }
            if indexedmap::native_map_hud_optional_unit_eligible(
                u.battle_fig as u8,
                u.native_record_race,
                u.native_record_byte6,
            ) {
                r#in.optional_unit = Some(indexedmap::NativeMapHUDOptionalUnit {
                    selector_slot: u.map_selector_slot,
                    raw_state: st.native_map_cycle_state.idle,
                    current: u.hp as u16,
                    maximum: u.native_record_word42,
                });
            }
        }
        (r#in, true)
    }

    /// Go：`main.go:12527`，`func (g *Game) composeNativeMapFrame() error`
    pub(crate) fn compose_native_map_frame(&mut self) -> Result<()> {
        self.compose_native_map_frame_for_action_background(false)
    }

    /// composeNativeMapFrameForActionBackground 的 closedAction 投影只供 0x190AC
    /// 取寶提示底圖；依 17643 的背景恢復與 seq3820 的同狀態畫面，保留原始
    /// selector，省略互動 range／selection。契約見 SDD #53，不能套用其他對話。
    ///
    /// Go：`main.go:12534`，`func (g *Game) composeNativeMapFrameForActionBackground(closedAction bool) error`
    pub(crate) fn compose_native_map_frame_for_action_background(&mut self, closed_action: bool) -> Result<()> {
        self.compose_native_map_frame_for_caller(closed_action, true)
    }

    /// 正常物理返回呼叫11CAC(1)，只略過4DFCC；其他地圖層及時鐘照常。
    /// 契約見physical_return_palette_spec，不用closedAction省略range／selection。
    ///
    /// Go：`main.go:12540`，`func (g *Game) composeNativeMapFrameForPhysicalReturn() error`
    pub(crate) fn compose_native_map_frame_for_physical_return(&mut self) -> Result<()> {
        self.compose_native_map_frame_for_caller(false, false)
    }

    /// Go：`main.go:12544`，`func (g *Game) composeNativeMapFrameForCaller(closedAction, updateSteadyPalette bool) error`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn compose_native_map_frame_for_caller(
        &mut self,
        closed_action: bool,
        update_steady_palette: bool,
    ) -> Result<()> {
        let mut now = self.game_now();
        if let Some(mut frozen) = self.native_map_frozen_now.take() {
            // FnMut 觀察者：呼叫完放回（Go 每次都讀同一個 func 欄位）。
            now = frozen(self);
            if self.native_map_frozen_now.is_none() {
                self.native_map_frozen_now = Some(frozen);
            }
        }
        if (!self.shot_path.is_empty() || !self.shot_series.is_empty())
            && std::env::var("FD2_SHOT_DETERMINISTIC").is_ok_and(|v| v == "1")
        {
            // 截圖證據只取固定 60 Hz 虛擬時鐘；一般玩家仍使用實際 BIOS
            // 時鐘。這避免 Xvfb 排程差異把同一狀態存成不同動畫幀。
            // Go `time.Unix(0, frame*int64(time.Second/60))` 不是零值時間；Rust 加上虛擬時鐘起點，
            // 相鄰兩幀的差（16666666 ns）與 Go 相同。
            now = VIRTUAL_CLOCK_ORIGIN + GoDuration(self.frame.wrapping_mul(GoDuration::SECOND.0 / 60));
        }
        self.compose_native_map_frame_at_with_palette_cycle(now, closed_action, update_steady_palette)
    }

    /// Go：`main.go:12570`，`func (g *Game) composeNativeMapFrameAtWithPaletteCycle(now time.Time, closedAction, updateSteadyPalette bool) error`
    pub(crate) fn compose_native_map_frame_at_with_palette_cycle(
        &mut self,
        now: GameInstant,
        closed_action: bool,
        update_steady_palette: bool,
    ) -> Result<()> {
        let Some(st) = self.st.as_ref() else {
            return Err(game_err!("native map frame: battle state is unavailable"));
        };
        let _ = st;
        let (mut hud, ok) = self.native_map_hud_input();
        if !ok {
            let st = self.st.as_ref().unwrap();
            return Err(game_err!(
                "native map frame: HUD input unavailable（{}；assets={} hud={} cycle={} selector_cache={} cursor=({},{}) map={}）",
                self.native_map_hud_input_reason,
                native_map_assets_available(self.native_map_assets.as_ref()),
                st.has_native_map_hud_state,
                st.has_native_map_cycle_state,
                st.native_map_selector_cache.is_some(),
                self.cur_x,
                self.cur_y,
                self.m.is_some(),
            ));
        }
        let st = self.st.as_ref().unwrap();
        if !st.has_native_map_range_mode_state || st.native_map_range_mode < 0 || st.native_map_range_mode > 5 {
            return Err(game_err!(
                "native map frame: drawable selector state unavailable（has={} mode={}）",
                st.has_native_map_range_mode_state,
                st.native_map_range_mode
            ));
        }
        // Go 另建 `candidateGame := Game{st: &candidateState, nativeMapClock: candidateClock}` 推進時鐘；
        // `advanceNativeMapClock` 只讀寫 `st` 與 `nativeMapClock`，Rust 暫時把候選值換進 `self` 再換回。
        let candidate_state = st.clone();
        let original_st = self.st.replace(candidate_state);
        let original_clock = self.native_map_clock;
        let advanced = self.advance_native_map_clock(now);
        let candidate_state = std::mem::replace(&mut self.st, original_st).unwrap();
        let candidate_clock = std::mem::replace(&mut self.native_map_clock, original_clock);
        if !advanced {
            return Err(game_err!("native map frame: timing state unavailable"));
        }
        let (raw_tick, ok) = candidate_clock.current();
        if !ok {
            return Err(game_err!("native map frame: auxiliary timing unavailable"));
        }
        let mut aux_phase = raw_tick % 16;
        if aux_phase < 0 {
            aux_phase += 16;
        }
        if let Some(phase) = self.native_chapter_aux_phase_override {
            aux_phase = phase & 15;
        }
        let mut current_dac = &self.native_map_dac;
        if current_dac.is_empty() && native_map_assets_available(self.native_map_assets.as_ref()) {
            // loadMap normally owns this initialization. Direct compositor owners
            // may materialize the same complete original bundle without passing
            // through loadMap; the immutable bundled baseline is the only safe
            // equivalent initial DAC, not a synthesized palette.
            current_dac = &self.native_map_assets.as_ref().unwrap().palette_dac;
        }
        if current_dac.len() != 256 * 3 {
            return Err(game_err!("native map frame: current DAC unavailable"));
        }
        let mut candidate_dac = current_dac.clone();
        let (mut candidate_palette_phase, mut candidate_palette_tick) =
            (self.native_fdother_palette_phase, self.native_fdother_palette_tick);
        if update_steady_palette {
            (candidate_palette_phase, candidate_palette_tick, _) = fdother::advance_native_dac_palette_cycle_e0ef(
                &mut candidate_dac,
                candidate_palette_phase,
                candidate_palette_tick,
                raw_tick,
            )
            .map_err(|e| GameError::from(e).wrap("native map frame: steady DAC cycle: "))?;
        }
        // 1297D先於1ACF3；預檢的HUD圖示需讀本次candidate idle。
        // 複製optional值，保留前置快照與失敗交易的原子性。契約見physical_map_hud_cycle_spec。
        if let Some(unit) = hud.optional_unit.as_mut() {
            unit.raw_state = candidate_state.native_map_cycle_state.idle;
        }
        // Go 在 buildNativeMapFrameInput 成功之後才建立選取覆蓋層；Rust 的覆蓋層需要 `&mut self`（延遲載入
        // 單位視窗素材），而輸入借用 `self`，所以先建立、把錯誤留到同一個位置才回報。先建立只多做一次素材快取。
        let overlay = if closed_action { Ok(None) } else { self.native_map_selection_overlay(Some(&candidate_state)) };
        let modern_icons = self.ring && self.uses_modern_battle_action_icons();
        let mut work = std::mem::take(&mut self.native_map_work);
        let mut vga = std::mem::take(&mut self.native_map_vga);
        let composed = self.compose_native_map_frame_buffers(
            &candidate_state,
            hud,
            aux_phase,
            closed_action,
            overlay,
            modern_icons,
            raw_tick,
            &candidate_dac,
            &mut work,
            &mut vga,
        );
        let (new_work, new_vga, stage, modern, poster_terrain) = match composed {
            Ok(outcome) => outcome,
            Err(err) => {
                // Go 的 `work`／`vga` 與 `g.nativeMapWork`／`g.nativeMapVGA` 是同一塊記憶體（除非長度不對或要複製），
                // 合成到一半失敗時已寫入的像素留在 g 上；Rust 照樣放回。
                (self.native_map_work, self.native_map_vga) = (work, vga);
                return Err(err);
            }
        };
        if self.posters.native_scenery.as_deref() != poster_terrain.as_ref() {
            self.posters.native_scenery = poster_terrain.map(Arc::new);
        }
        if let Some(stage) = stage {
            self.native_ch23_state = Some(stage);
        }
        if let Some(frame) = modern {
            self.modern_hd_map_terrain.clone_from(&frame.terrain_seed);
            self.modern_hd_map_frame = Some(frame);
        }
        self.native_interface_hud = None;
        self.native_map_work = new_work.unwrap_or(work);
        self.native_map_vga = new_vga.unwrap_or(vga);
        // Go 在不走行、沒有回合橫幅時把這一幀的輸入 `in` 發布到 `g.nativeMapFrameSource`。它借用素材與戰場狀態，
        // Rust 不保存；讀它的都是 HD 呈現（`modern_hd_*`、`player_interface`、`player_poster`，R6），
        // 由那些呼叫端以同一組資源重組。見 PORTING.md「C 組：原生地圖整幀」。
        self.native_map_frame_source = None;
        self.st = Some(candidate_state);
        self.native_map_clock = candidate_clock;
        self.native_map_dac = candidate_dac;
        self.native_fdother_palette_phase = candidate_palette_phase;
        self.native_fdother_palette_tick = candidate_palette_tick;
        Ok(())
    }

    /// `composeNativeMapFrameAtWithPaletteCycle` 中借用素材與候選狀態的那一段（Rust 拆出）：組輸入、
    /// 掛選取覆蓋層、ch23 暫存與三種合成路徑。回傳新配置的工作／VGA 緩衝（`None` 表示就地寫進傳入的
    /// 緩衝，對應 Go 與 `g` 共用同一塊記憶體）與 ch23 暫存狀態。
    fn compose_native_map_frame_buffers(
        &self,
        candidate_state: &battle::State,
        hud: indexedmap::NativeMapHUDInput,
        aux_phase: i64,
        closed_action: bool,
        overlay: Result<Option<NativeMapSelectionOverlay>>,
        modern_icons: bool,
        raw_tick: i64,
        candidate_dac: &[u8],
        work: &mut Vec<u8>,
        vga: &mut Vec<u8>,
    ) -> Result<(
        Option<Vec<u8>>,
        Option<Vec<u8>>,
        Option<NativeCh23AdapterState>,
        Option<ModernHDMapFrame>,
        Option<Picture>,
    )> {
        let overlay_fn;
        let mut in_ = build_native_map_frame_input(
            self.native_map_assets.as_ref(),
            self.m.as_ref(),
            Some(candidate_state),
            NativeMapFrameRuntime { hud, chapter_aux_phase: aux_phase },
        )?;
        if closed_action {
            in_.frame.range_mode = 0;
        } else if let Some(overlay) = overlay? {
            overlay_fn =
                move |buf: &mut [u8], stride: i64, base: i64| overlay.render(self, candidate_state, buf, stride, base);
            in_.frame.selection_overlay = Some(&overlay_fn);
            if self.ring {
                // 指令環貼在走行結束時的畫面上，0x1741C 的迴圈不再進 0x122DC：畫面上
                // 沒有游標白框（第四章 r9 seq 208 收據），只有四張圖示。
                in_.frame.range_mode = 0;
            }
        }
        if modern_icons {
            in_.frame.selection_overlay = None;
            in_.frame.range_mode = 0;
        }
        let stage =
            self.prepare_native_ch23_steady_stage(raw_tick).map_err(|e| e.wrap("native map frame: ch23 staging: "))?;
        let work_size = indexedmap::NATIVE_UNIT_PRESENT_WORK_SIZE as usize;
        let vga_size = indexedmap::NATIVE_MAP_VGA_SIZE as usize;
        let copy = stage.is_some() || self.modern_hd_map_composing;
        let mut own_work = (work.len() != work_size || copy)
            .then(|| if work.len() != work_size { vec![0u8; work_size] } else { work.clone() });
        let mut own_vga = (vga.len() != vga_size || copy)
            .then(|| if vga.len() != vga_size { vec![0u8; vga_size] } else { vga.clone() });
        let work_buf: &mut Vec<u8> = own_work.as_mut().unwrap_or(work);
        let vga_buf: &mut Vec<u8> = own_vga.as_mut().unwrap_or(vga);
        if let Some(stage) = &stage {
            indexedmap::seed_native_ch23_staging(work_buf, &stage.staging)?;
        }
        // 走行中的重繪是另一條路徑，不是整幀排程關掉幾層：原版每一幀只進地形、
        // 單位與前景，完全不進 0x11CAC、0x122DC 與 0x1AD72，所以畫面上沒有游標
        // 白框也沒有左下 HUD 面板。收據見
        // docs/knowledge-base/99-move-confirm-cursor-20260909.md。
        // 回合橫幅期間也是同一種畫面：0x1A30B 在 0x1F1CC 之前用 0x11EB0 把離屏地圖
        // 緩衝區（地形、單位、前景）整塊搬到 VGA (4,4)，蓋掉左下 HUD；游標白框也不
        // 在裡面（第四章 r9 收據 seq 636／913：全員行動完自動換手，橫幅下沒有白框
        // 與地形面板）。橫幅本身對這張快照做馬賽克，之後不重繪。
        let motion = if let (unit_y, phase, true) = self.native_ch12_up_walk_frame() {
            ModernHDMapMotion::North { unit_y, phase }
        } else if self.walk.is_some() || self.banner_t > 0 {
            ModernHDMapMotion::Step
        } else {
            ModernHDMapMotion::Steady
        };
        let modern = if self.modern_hd_map_composing {
            let theme =
                self.modern_story_portraits.as_ref().ok_or_else(|| game_err!("modern HD map: missing theme"))?;
            Some(self.prepare_hd_map(&in_, stage.as_ref(), candidate_dac, motion, false, theme)?)
        } else {
            None
        };
        let mut poster_terrain = None;
        match motion {
            ModernHDMapMotion::North { unit_y, phase } => {
                indexedmap::compose_native_up_step_frame(work_buf, vga_buf, &in_.frame, unit_y, phase)?
            }
            ModernHDMapMotion::Step => indexedmap::compose_native_step_frame(work_buf, vga_buf, &in_.frame)?,
            ModernHDMapMotion::Steady if self.posters.active() && modern.is_none() => {
                let palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(candidate_dac)?);
                let mut observe = |stage, work: &mut [u8], _: &mut [u8]| {
                    if stage == indexedmap::FRAME_STAGE_TERRAIN {
                        poster_terrain = poster_terrain_base(work, NATIVE_MAP_STEADY_BASE as usize, &palette);
                    }
                    Ok(())
                };
                indexedmap::compose_native_frame_observed(work_buf, vga_buf, &mut in_, Some(&mut observe))?;
            }
            ModernHDMapMotion::Steady => indexedmap::compose_native_frame(work_buf, vga_buf, &mut in_)?,
        }
        if self.modern_story_portraits.is_none()
            && self.modern_hd_interface_enabled()
            && self.font.is_some()
            && self.walk.is_none()
            && self.banner_t <= 0
        {
            todo!("R6: main.go:12731 prepareModernHDMapHUDPanel（HD 介面 HUD）");
        }
        Ok((own_work, own_vga, stage, modern, poster_terrain))
    }

    /// endTurn 是 0x1A30B 的入口（系統選單 END／YES、全員行動完的 0x13565、全軍移動收尾
    /// 都走這裡）。原版順序：selector 1 回合事件（0x1A813(1)）→ selector 1 暫時狀態
    /// → 友軍 AI → selector 0 回合事件 → selector 0 暫時狀態 → 敵方 AI → 回合數加一。
    /// 可編輯的回合事件依 NativeTurnPhaseSelector 插在對應位置；沒有 selector 的手寫
    /// 事件仍留在敵方回合之後（finishTurn）。
    ///
    /// Go：`main.go:12766`，`func (g *Game) endTurn()`
    pub(crate) fn end_turn(&mut self) {
        if self.st.is_none()
            || !self.result.is_empty()
            || self.ai_busy
            || self.native_end_turn_recovery.is_some()
            || self.native_ch28_post_present.is_some()
            || self.native_turn_staging.is_some()
            || self.battle_event.is_some()
            || self.native_palette_pulse.is_some()
        {
            return;
        }
        let st = self.st.as_mut().unwrap();
        if st.has_native_runtime_unit_projection || st.has_native_map_view_state {
            let result = self.begin_native_end_turn_recovery(Some(Box::new(|g: &mut Game| {
                g.run_editable_turn_events(1, Some(Box::new(|g: &mut Game| g.end_turn_after_selector1_events())));
            })));
            if let Err(err) = result {
                self.load_err = err.message().to_string();
            }
            return;
        }
        // 兩條 END 路徑（系統選單 0x1726B、全員行動完 0x135B4）進 0x1A30B 之前都把 HUD
        // 顯示閘門 B [0x51AAC] 寫 0，返回後寫 1：整個換手、敵方回合到 PLAYER PHASE 橫幅
        // 都不畫小窗，0x1ACF3 的 anchor 分支也不跑。
        if st.has_native_map_hud_state {
            st.native_map_hud_state.display_gate_b = 0;
        }
        // 沒有正式原版 runtime/view 的舊可編輯劇本保留同步數值路徑。
        let records = st.apply_native_end_turn_recovery();
        if std::env::var("FD2_SHOT_AI").is_ok_and(|v| !v.is_empty()) {
            let st = self.st.as_ref().unwrap();
            for rec in records {
                // Go `rec.Unit` 是指標；Rust 的 `rec.unit` 是 `st.units` 的索引。
                if let Some(u) = rec.unit.map(|i| &st.units[i]) {
                    eprintln!("END recovery: {}({},{}) hp {}→{}", u.cls_name, u.x, u.y, rec.before, rec.after);
                }
            }
        }
        self.run_editable_turn_events(1, Some(Box::new(|g: &mut Game| g.end_turn_after_selector1_events())));
    }

    /// Go：`main.go:12793`，`func (g *Game) endTurnAfterSelector1Events()`
    pub(crate) fn end_turn_after_selector1_events(&mut self) {
        let started = match self.start_native_raw_camp0_turn_events() {
            Ok(started) => started,
            Err(err) => {
                self.load_err = format!("native raw camp0 phase: {}", err.message());
                return;
            }
        };
        if started {
            return;
        }
        // 0x1A30B：selector 1 回合事件 → sub_1A866(1) → 友軍 AI（0x1D80B）→ 橫幅 → 0x13536 →
        // selector 0 回合事件 → sub_1A866(0) → 敵軍兩遍。selector 0 的掃描在 aiStep 的橫幅之後
        // 才跑（aiPhaseSelector0SweepPending），不和 selector 1 併在友軍 AI 之前。
        if self.native_transient_sweep_available() {
            let result = self.begin_native_transient_phases(
                &mut [1],
                Some(Box::new(|g: &mut Game| g.begin_enemy_phase_after_turn_events())),
            );
            if let Err(err) = result {
                self.load_err = err.message().to_string();
            }
            return;
        }
        self.begin_enemy_phase_after_turn_events();
    }

    /// nativeTransientSweepAvailable：sub_1A866 的掃描只在名冊每筆都有 raw +5／+6 出處時
    /// 跑（原版名冊或 saved runtime 投影）；沒有 raw 出處的舊可編輯劇本沒有 +0x22..+0x27，
    /// 跳過。第六章 r4 收據：敵方法師的強化 +0x22 每回合 −1、歸零後才會再被 0x1598A
    /// 的指令 17 評分選中，不遞減就永遠不再施放。
    ///
    /// Go：`main.go:12818`，`func (g *Game) nativeTransientSweepAvailable() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_transient_sweep_available(&self) -> bool {
        let Some(st) = self.st.as_ref().filter(|st| !st.units.is_empty()) else {
            return false;
        };
        if st.has_native_runtime_unit_projection {
            return true;
        }
        st.units.iter().all(|unit| unit.has_native_record_byte5 && unit.has_native_record_byte6)
    }

    /// beginEnemyPhaseAfterTurnEvents 進敵方回合。0x1A30B 的順序是橫幅（0x1A3A2）→
    /// 0x13536 清已行動 → 0x1A813(0) 的 selector 0 回合事件 → 0x1D8BA 敵方 AI，所以
    /// selector 0 的事件（第五章第 3 回合的 event 14：AI 模式改寫＋對白）要等橫幅播完才跑，
    /// 由 aiStep 在清 bit7 之後接手；不跑 AI 的截圖模式直接在這裡跑完再收回合。
    ///
    /// Go：`main.go:12837`，`func (g *Game) beginEnemyPhaseAfterTurnEvents()`
    pub(crate) fn begin_enemy_phase_after_turn_events(&mut self) {
        if !self.shot_path.is_empty() && std::env::var("FD2_SHOT_AI").unwrap_or_default().is_empty() {
            self.run_editable_turn_events(0, Some(Box::new(|g: &mut Game| g.begin_enemy_phase())));
            return;
        }
        self.begin_enemy_phase();
    }

    /// runEditableTurnEvents 觸發 on_turn_end 裡 phase selector 相符的事件，跑完再 then。
    ///
    /// Go：`main.go:12846`，`func (g *Game) runEditableTurnEvents(selector int, then func())`
    /// Go 的 func 參數 `then` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。
    pub(crate) fn run_editable_turn_events(&mut self, selector: i64, then: Option<Box<dyn FnOnce(&mut Game) + Send>>) {
        if self.sc.is_none() || self.st.is_none() {
            if let Some(then) = then {
                then(self);
            }
            return;
        }
        let sc = self.sc.as_mut().unwrap();
        let st = self.st.as_ref();
        let mut accept = |e: &battle::Event| e.native_turn_phase_selector() == selector;
        let mut actions = sc.trigger_actions_where(st, "on_turn_end", "", Some(&mut accept));
        // 由 control_turn 動態啟用的控制列（第九章事件 30 排的事件 31）同樣由 0x1A813 在這個
        // selector 分派，接在靜態列降成的劇本事件之後。
        for event in sc.native_turn_action_events_at(st, selector as u8) {
            actions.extend(event.actions);
        }
        if actions.is_empty() {
            if let Some(then) = then {
                then(self);
            }
            return;
        }
        self.start_battle_event(&mut actions, then);
    }

    /// Go：`main.go:12866`，`func (g *Game) beginEnemyPhase()`
    pub(crate) fn begin_enemy_phase(&mut self) {
        if let Some(st) = self.st.as_mut() {
            st.reset_native_ai_scan();
        }
        if self.shot_path.is_empty() || !std::env::var("FD2_SHOT_AI").unwrap_or_default().is_empty() {
            // 截圖模式預設跳 AI;FD2_SHOT_AI=1 強制驗證 AI 行走
            self.ai_busy = true; // AI 階段:逐單位行走動畫(Update 內 aiStep 驅動),播完 finishTurn
            // 0x1A30B：友軍 AI（0x1D80B，raw +6==1）在橫幅之前；aiStep 跑完 pass 0 才
            // 進 ENEMY PHASE 橫幅、0x13536、selector 0 事件與敵軍兩遍。
            self.ai_ally_phase_pending = true;
            return;
        }
        self.finish_turn();
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
        if actions.is_empty() {
            if let Some(then) = then {
                then(self);
            }
            return;
        }
        self.battle_event = Some(BattleEventRun { actions: actions.to_vec(), index: -1, then });
        self.advance_battle_event();
    }

    /// R4 B 組隨呼叫端補移植（原標 `r8`）：Go `main.go:12892`，`func (g *Game) finishBattleEventWithError(message string)`。
    /// 戰鬥事件的錯誤路徑；`failNativeChapterResult` 是 `r8`，有原生章節結果時停在 `todo!`。
    pub(crate) fn finish_battle_event_with_error(&mut self, message: &str) {
        self.load_err = format!("battle event: {message}");
        if self.native_chapter_result.is_some() {
            todo!("R8: native_chapter_result.go:failNativeChapterResult（第一章沒有執行到）")
        }
        (self.battle_event, self.battle_event_delay, self.cam_pan) = (None, 0, None);
    }

    /// Go：`main.go:12900`，`func (g *Game) advanceBattleEvent()`
    ///
    /// R4 B 組：Go 持有 `run := g.battleEvent` 指標逐項推進；Rust 每一輪重新借用 `self.battle_event`，
    /// 收尾時取出再呼叫延續。動作先 `clone()` 出來再執行（執行期間會借用 `Game`）。
    pub(crate) fn advance_battle_event(&mut self) {
        if self.battle_event.is_none() {
            return;
        }
        loop {
            let Some(run) = self.battle_event.as_mut() else {
                return;
            };
            run.index += 1;
            if run.index >= run.actions.len() as i64 {
                let mut run = self.battle_event.take().unwrap();
                if let Some(then) = run.then.take() {
                    then(self);
                }
                return;
            }
            let action = run.actions[run.index as usize].clone();
            if let Some(when) = action.native_when.as_ref() {
                match when.r#match(self.st.as_ref()) {
                    Err(err) => {
                        self.finish_battle_event_with_error(&err.to_string());
                        return;
                    }
                    Ok(false) => continue,
                    Ok(true) => {}
                }
            }
            if action.r#type == "dialogue" && action.native_dialogue_ref.is_some() {
                if let Err(err) = self.start_native_event_dialogue(&action) {
                    self.finish_battle_event_with_error(err.message());
                }
                return;
            }
            match action.r#type.as_str() {
                "native_death_op" => {
                    if self.run_native_death_op(&action) {
                        return;
                    }
                    continue;
                }
                "reset_pose" => {
                    let Some(st) = self.st.as_mut() else {
                        self.finish_battle_event_with_error("重設姿態缺少戰場");
                        return;
                    };
                    for unit in &mut st.units {
                        unit.set_map_pose(0);
                    }
                    self.battle_event_delay = 1;
                    return;
                }
                "redraw" => {
                    self.native_map_vga = Vec::new();
                    if let Err(err) = self.compose_native_map_frame() {
                        self.finish_battle_event_with_error(err.message());
                        return;
                    }
                    self.battle_event_delay = 1;
                    return;
                }
                "native_staging_present" => {
                    if action.native_event_id != Some(82) || action.native_source.is_empty() {
                        self.finish_battle_event_with_error("staging 缺少已證實caller");
                        return;
                    }
                    let spec = action.native_staging_present;
                    let then: Box<dyn FnOnce(&mut Game) + Send> = Box::new(|g: &mut Game| g.advance_battle_event());
                    if let Err(err) = self.start_native_battle_staging_present(spec.as_ref(), Some(then)) {
                        self.finish_battle_event_with_error(&err.to_string());
                    }
                    return;
                }
                "native_range_zero" | "native_range_one" => {
                    if self.st.is_none() || action.native_source.is_empty() {
                        self.finish_battle_event_with_error("range 寫入缺少來源");
                        return;
                    }
                    let value = if action.r#type == "native_range_one" { 1 } else { 0 };
                    let st = self.st.as_mut().unwrap();
                    (st.native_map_range_mode, st.has_native_map_range_mode_state) = (value, true);
                }
                "pan" => {
                    let (Some(grid), Some(m)) = (action.grid, self.m.as_ref().filter(|m| m.tile_w > 0 && m.tile_h > 0))
                    else {
                        self.finish_battle_event_with_error("pan 缺少有效 grid/map");
                        return;
                    };
                    self.cam_pan = Some(CamPanJob {
                        from_x: self.cam_x,
                        from_y: self.cam_y,
                        to_x: (grid[0] * m.tile_w) as f64,
                        to_y: (grid[1] * m.tile_h) as f64,
                        tile_step: true,
                        then: Some(Box::new(|g: &mut Game| g.advance_battle_event())),
                        ..Default::default()
                    });
                    return;
                }
                "delay" => {
                    let mut frames = action.ms * 60 / 1000;
                    if frames <= 0 {
                        frames = 1;
                    }
                    self.battle_event_delay = frames;
                    return;
                }
                "native_acting" => {
                    let Some(unit_count) = self.st.as_ref().map(|st| st.units.len() as i64) else {
                        self.finish_battle_event_with_error("native acting 缺少 runtime battle state");
                        return;
                    };
                    match self.load_native_battle_following_acting(action.native_acting.as_ref(), unit_count) {
                        Ok(mut frames) => self.start_native_battle_following_acting(
                            &mut frames,
                            Some(Box::new(|g: &mut Game| g.advance_battle_event())),
                        ),
                        Err(err) => self.finish_battle_event_with_error(err.message()),
                    }
                    return;
                }
                _ => {
                    let (call, is_native_intro) = match native_battle_intro_call(&action) {
                        Ok(v) => v,
                        Err(err) => {
                            self.finish_battle_event_with_error(err.message());
                            return;
                        }
                    };
                    if is_native_intro {
                        if let Err(err) = self.start_native_battle_spawn_intro(
                            &action,
                            &call,
                            Some(Box::new(|g: &mut Game| g.advance_battle_event())),
                        ) {
                            self.finish_battle_event_with_error(err.message());
                        }
                        return;
                    }
                    let sc = self.sc.as_mut().expect("Go：g.sc 為 nil 時解參考 panic");
                    let (dialogue, is_dialogue) = match sc.execute_action_checked(self.st.as_mut(), &action) {
                        Ok(v) => v,
                        Err(err) => {
                            self.finish_battle_event_with_error(&err.to_string());
                            return;
                        }
                    };
                    self.apply_scenario_party_joins();
                    if !self.load_err.is_empty() {
                        let message = self.load_err.clone();
                        self.finish_battle_event_with_error(&message);
                        return;
                    }
                    if is_dialogue {
                        self.dialog = vec![dialogue];
                        self.dlg_page = 0;
                        return;
                    }
                }
            }
        }
    }

    /// Go：`main.go:13040`，`func (g *Game) stepBattleEventDelay()`
    pub(crate) fn step_battle_event_delay(&mut self) {
        if self.battle_event_delay <= 0 {
            return;
        }
        self.battle_event_delay -= 1;
        if self.battle_event_delay == 0 {
            self.advance_battle_event();
        }
    }

    /// finishTurn starts the ordered on_turn_end event. Turn/status bookkeeping is
    /// deferred until the complete visual sequence finishes.
    ///
    /// Go：`main.go:13052`，`func (g *Game) finishTurn()`
    pub(crate) fn finish_turn(&mut self) {
        if self.battle_event.is_some() {
            return;
        }
        if let Some(sc) = self.sc.as_mut() {
            // 帶 phase selector 的事件已在 endTurn／敵方 AI 前／下一回合玩家輸入前
            // 各自觸發；這裡只剩沒有 selector 的手寫事件。
            let mut accept = |e: &battle::Event| e.native_turn_phase_selector() == -1;
            let mut actions = sc.trigger_actions_where(self.st.as_ref(), "on_turn_end", "", Some(&mut accept));
            if !actions.is_empty() {
                self.start_battle_event(&mut actions, Some(Box::new(|g: &mut Game| g.complete_turn())));
                return;
            }
        }
        self.complete_turn();
    }

    /// Go：`main.go:13070`，`func (g *Game) completeTurn()`
    pub(crate) fn complete_turn(&mut self) {
        let st = self.st.as_mut().expect("Go：g.st 為 nil 時解參考 panic");
        st.turn += 1;
        // Keep the executable [0x53bef] counter only for states that entered with
        // explicit native provenance; hand-authored tests/legacy states remain
        // unproven (zero) and therefore fail closed for native round predicates.
        if st.native_round_counter > 0 {
            st.native_round_counter += 1;
        }
        let count = st.units.len();
        for i in 0..count {
            let st = self.st.as_mut().unwrap();
            let has_view = st.has_native_map_view_state;
            let u = &mut st.units[i];
            u.acted = false;
            if has_view && u.has_native_record_byte5 {
                u.native_record_byte5 &= 0x7f;
            }
            u.tick_status(); // buff/封咒/中毒/麻痺回合遞減+中毒扣血(doc02 §6.4)
            self.award_death_reward(Some(i), None); // poison/status death shares the same once-only reward path
        }
        match self.start_native_raw_camp2_turn_events(Some(Box::new(|g: &mut Game| g.complete_turn_player_phase()))) {
            Err(err) => {
                self.load_err = format!("native raw camp2 phase: {err}");
            }
            Ok(true) => {}
            Ok(false) => self.complete_turn_player_phase(),
        }
    }

    /// Go：`main.go:13097`，`func (g *Game) completeTurnPlayerPhase()`
    pub(crate) fn complete_turn_player_phase(&mut self) {
        // 0x1A78B→0x1A797 passes selector 2 after the phase presentation and
        // immediately before range mode 1/player input. Keep the raw selector;
        // do not substitute normalized Camp.
        if self.native_transient_sweep_available() {
            if let Err(err) = self.begin_native_transient_phases(
                &mut [2],
                Some(Box::new(|g: &mut Game| g.finish_native_transient_player_phase())),
            ) {
                self.load_err = err.message().to_string();
            }
            return;
        }
        self.finish_native_transient_player_phase();
    }

    /// Go：`main.go:13110`，`func (g *Game) finishNativeTransientPlayerPhase()`
    pub(crate) fn finish_native_transient_player_phase(&mut self) {
        if self.result.is_empty() {
            self.show_banner("PLAYER PHASE");
        }
        // 0x1A78B→0x1A797：phase 演出之後、玩家輸入之前的 selector 2 回合事件，
        // 此時回合數已加一，所以 when.turn 對的是新回合。
        self.run_editable_turn_events(2, Some(Box::new(|g: &mut Game| g.finish_native_transient_player_phase_input())));
    }

    /// Go：`main.go:13119`，`func (g *Game) finishNativeTransientPlayerPhaseInput()`
    pub(crate) fn finish_native_transient_player_phase_input(&mut self) {
        (self.sel, self.reach, self.moved) = (None, BTreeMap::new(), false);
        // 0x1A79F [0x51A83]=1 → 0x1A7AB 0x12D7B(0) 聚焦記錄 0 都還在 0x1A30B 裡，閘 B 仍是 0，
        // 逐格重繪不評估 HUD anchor；0x1A30B 返回後 0x17277／0x135D4 才寫回 1（第九章 r1 seq 814：
        // 聚焦途中可見游標 (2,6)，原版小窗沒翻到右側）。
        if let Some(st) = self.st.as_mut().filter(|st| st.has_native_map_view_state && !st.units.is_empty()) {
            // 0x1A79F：聚焦前 [0x51A83]=1，逐格都重繪（閘 B 仍是 0，anchor 不動）。
            st.materialize_native_map_range_mode(1);
            self.native_next_player_index = 0;
            if self.begin_native_player_focus(Some(0)) {
                self.native_gate_b_after_focus = true;
                self.check_result();
                return;
            }
        }
        self.restore_native_display_gate_b();
        self.check_result();
    }

    /// restoreNativeDisplayGateB 是 0x1A30B 返回後的 0x17277／0x135D4 閘 B 寫回；輸入迴圈接著的
    /// 0x11CAC 重繪經 0x1ACF3 依當下可見游標評估 anchor。
    ///
    /// Go：`main.go:13140`，`func (g *Game) restoreNativeDisplayGateB()`
    pub(crate) fn restore_native_display_gate_b(&mut self) {
        self.native_gate_b_after_focus = false;
        let Some(st) = self.st.as_mut().filter(|st| st.has_native_map_hud_state) else {
            return;
        };
        st.native_map_hud_state.display_gate_b = 1;
        self.redraw_native_map_hud();
    }

    /// aiStep AI 回合驅動:一次取一個單位的行動計畫,播行走動畫→到位攻擊(全螢幕演出)。
    /// 全單位動完 → finishTurn。
    ///
    /// Go：`main.go:13151`，`func (g *Game) aiStep()`
    pub(crate) fn ai_step(&mut self) {
        // 回合橫幅（0x1F1CC）是同步播完才回到 0x1A30B 往下走；橫幅期間 AI 不動。
        if self.banner_t > 0 {
            return;
        }
        if self.ai_phase_clear_bit7_pending
            && self.ai_busy
            && let Some(st) = self.st.as_mut()
        {
            // 0x13536：橫幅之後、敵方 AI 之前把全部記錄的 +5 bit7 清掉（含 END 回復
            // 剛設的與玩家已行動位）；戰鬥在敵方回合中結束時，持續紀錄裡就不會留著它。
            for u in &mut st.units {
                if u.has_native_record_byte5 {
                    u.native_record_byte5 &= 0x7f;
                }
                u.acted = false;
            }
            self.ai_phase_clear_bit7_pending = false;
        }
        // 回合事件（對白、鏡頭、演出）進行中 AI 不動；0x1A813(0) 是同步呼叫，回來才進 AI。
        if self.battle_event.is_some()
            || self.cam_pan.is_some()
            || self.act_job.is_some()
            || self.native_turn_staging.is_some()
            || self.native_level_up_dialogue.is_some()
            || self.native_system_end_turn_ui.is_some()
        {
            return;
        }
        if self.ai_phase_selector0_pending && self.ai_busy && self.st.is_some() {
            self.ai_phase_selector0_pending = false;
            // 79的兩個bit7必須在13536之後發布，否則會被同一階段清掉。
            if let Err(err) = self.apply_native_raw_camp0_pair_event() {
                self.load_err = format!("native event79 phase: {}", err.message());
                self.ai_busy = false;
                return;
            }
            self.run_editable_turn_events(0, Some(Box::new(|_: &mut Game| {})));
            if self.battle_event.is_some() {
                return;
            }
        }
        if self.ai_phase_selector0_sweep_pend && self.ai_busy && self.st.is_some() {
            // sub_1A866(0)：selector 0 回合事件之後、敵方 AI 之前掃 camp 0 的暫時狀態。
            self.ai_phase_selector0_sweep_pend = false;
            if self.native_transient_sweep_available() {
                if let Err(err) = self.begin_native_transient_phases(&mut [0], None) {
                    self.load_err = err.message().to_string();
                    self.ai_busy = false;
                    return;
                }
                if self.native_transient_presentation_active() {
                    return;
                }
            }
        }
        if !self.ai_busy
            || self.walk.is_some()
            || self.atk.is_some()
            || self.native_ai_idle_recovery.is_some()
            || self.native_end_turn_recovery.is_some()
            || self.native_heal_presentation.is_some()
            || self.native_modifier_presentation.is_some()
            || self.native_ai_command_modifier.is_some()
            || self.native_ai_item_presentation.is_some()
            || self.native_cmd0_presentation.is_some()
            || self.native_cmd1_presentation.is_some()
            || self.native_cmd2_presentation.is_some()
            || self.native_cmd3_presentation.is_some()
            || self.native_cmd5_presentation.is_some()
            || self.native_cmd6_presentation.is_some()
            || self.native_cmd7_presentation.is_some()
            || self.native_cmd8_presentation.is_some()
            || self.native_cmd9_player.is_some()
            || self.native_cmd9_ai_presentation.is_some()
            || self.native_cmd1012.is_some()
            || self.native_cmd24_presentation.is_some()
            || self.native_cmd29_presentation.is_some()
            || self.native_cmd32_presentation.is_some()
            || self.native_cmd33_presentation.is_some()
            || self.native_cmd34_presentation.is_some()
            || self.native_cmd35_presentation.is_some()
            || !self.result.is_empty()
        {
            if !self.result.is_empty() {
                self.ai_busy = false;
            }
            return;
        }
        let plan = if self.ai_ally_phase_pending {
            let (ally_plan, handled) = self.st.as_mut().expect("Go：g.st 為 nil 時解參考 panic").next_ally_ai_plan();
            match ally_plan.filter(|_| handled) {
                Some(plan) => Some(plan),
                None => {
                    // 友軍那一遍跑完（或名冊沒有 raw +6，沒有獨立的友軍遍）：進橫幅。
                    self.ai_ally_phase_pending = false;
                    self.show_banner("ENEMY PHASE");
                    self.ai_phase_clear_bit7_pending = true;
                    self.ai_phase_selector0_pending = true;
                    self.ai_phase_selector0_sweep_pend = true;
                    return;
                }
            }
        } else {
            self.st.as_mut().expect("Go：g.st 為 nil 時解參考 panic").next_ai_plan()
        };
        let Some(plan) = plan else {
            self.ai_busy = false;
            self.finish_turn();
            return;
        };
        if plan.native_error.is_none()
            && let Some(mut observer) = self.ai_plan_observer.take()
        {
            // FnMut 觀察者：呼叫完放回（Go 每次都讀同一個 func 欄位）。
            observer(self, &plan);
            if self.ai_plan_observer.is_none() {
                self.ai_plan_observer = Some(observer);
            }
        }
        self.native_field_event_pending = None; // 0x1D855 等：每個 AI 單位行動前 [0x51A8F] = 0xff
        if plan.native_mode_fallback_active && plan.native_mode_fallback == 8 && plan.native_error.is_none() {
            // 0x13D9A→0x1317D直接返回，跳過0x13512／selector1／姿態重設。
            // 不設Acted，第二遍仍能依raw入場；原生掃描游標已前進，不會原地重試。
            return;
        }
        if let Some(err) = plan.native_error.as_ref() {
            // Native mode 2 有明確的原始來源閘門；閘門失敗時不可消耗行動，也不可
            // 靜默替換成正規化 AI。
            self.load_err = format!("native AI: {}", err.message());
            self.ai_busy = false;
            return;
        }
        if !plan.native_mode11_stages.is_empty() {
            self.start_native_ai_mode11(plan);
            return;
        }
        if let Some(decision) = plan.native_idle_recovery {
            let u = plan.u;
            let after: Box<dyn FnOnce(&mut Game) + Send> =
                Box::new(move |g: &mut Game| g.finish_successful_unit_action(u, None));
            if let Err(err) = self.begin_native_ai_idle_recovery(plan.u, decision, Some(after)) {
                self.load_err = format!("native AI 0x13fd4: {}", err.message());
                self.ai_busy = false;
            }
            return;
        }
        let u = plan.u.expect("Go：plan.U 為 nil 時解參考 panic");
        if plan.path.len() >= 2 {
            if std::env::var("FD2_SHOT_AI").is_ok_and(|v| !v.is_empty()) {
                let unit = &self.st.as_ref().unwrap().units[u];
                let (first, last) = (plan.path[0], plan.path[plan.path.len() - 1]);
                eprintln!(
                    "AI walk: {}({},{})→({},{}) 段數{} 目標={} mode={} intended=({},{}) blocked=({},{},{}) opposite={:?} candidates={:?}",
                    unit.cls_name,
                    first.x,
                    first.y,
                    last.x,
                    last.y,
                    plan.path.len() - 1,
                    plan.target.is_some(),
                    plan.native_mode_fallback,
                    plan.native_mode_intended.x,
                    plan.native_mode_intended.y,
                    plan.native_mode_blocked_cell.x,
                    plan.native_mode_blocked_cell.y,
                    plan.native_mode_blocked_found,
                    plan.native_mode_opposite,
                    plan.native_mode_candidates
                );
            }
            // 0x13BA5／0x14203：走之前先把游標走到自己身上。
            let (ux, uy) = {
                let unit = &self.st.as_ref().unwrap().units[u];
                (unit.x, unit.y)
            };
            self.ai_focus_cursor(ux, uy);
            let path = plan.path.clone();
            let plan = Arc::new(plan);
            self.walk = Some(WalkAnim {
                u: Some(u),
                path,
                then: Some(Box::new(move |g: &mut Game| g.ai_step_act(&plan))),
                follow_view: true,
                ..Default::default()
            });
        } else {
            if plan.native_mode_focus_actor && plan.target.is_none() {
                // 已在目的地：分派器照樣在 0x14B78 之前 0x12D7B 聚焦自己。mode 4／7／10 在聚焦
                // 前寫 [0x51A83]=0（0x13BE1／0x13D2E／0x13DF8）；mode 0／3／9 沒有這個前置寫入。
                if plan.native_mode_write_range_zero && plan.native_mode_fallback != 3 {
                    let st = self.st.as_mut().unwrap();
                    st.native_map_range_mode = 0;
                    st.has_native_map_range_mode_state = true;
                }
                let (ux, uy) = {
                    let unit = &self.st.as_ref().unwrap().units[u];
                    (unit.x, unit.y)
                };
                self.ai_focus_cursor(ux, uy);
            }
            self.ai_step_act(&Arc::new(plan));
        }
    }

    /// Go `aiStep` 的閉包 `act`（`main.go:13265`..`13419`）：走完（或不用走）之後執行計畫。`plan` 是共用的計畫值
    /// （Go 的 `*battle.AIPlan` 指標）。
    fn ai_step_act(&mut self, plan: &Arc<battle::AIPlan>) {
        let u = plan.u.expect("Go：plan.U 為 nil 時解參考 panic");
        if plan.native_mode_write_range_zero {
            // The dispatcher writes the raw map-range global [0x51a83] on
            // these branches. Keep the state field explicitly raw; it is not
            // a normalized command-selection value.
            let st = self.st.as_mut().expect("Go：g.st 為 nil 時解參考 panic");
            st.native_map_range_mode = 0;
            st.has_native_map_range_mode_state = true;
        }
        if plan.spell_id >= 0 {
            todo!("R8: ai_spell_execute.go:executeAISpell（第一章重播沒有敵方法術）");
        }
        if plan.native_action_kind == battle::NATIVE_AI_ACTION_ITEM
            || plan.native_action_kind == battle::NATIVE_AI_ACTION_COMMAND
        {
            if let Err(err) = self.execute_native_ai_action(plan) {
                let kind = if plan.native_action_kind == battle::NATIVE_AI_ACTION_ITEM {
                    "敵方道具"
                } else {
                    "敵方指令"
                };
                self.load_err = format!("{kind}：{}", err.message());
                self.ai_busy = false;
            }
            return;
        }
        let target = plan.target.filter(|&t| self.st.as_ref().unwrap().units[t].alive());
        let Some(tgt) = target else {
            self.ai_step_finish(plan);
            return;
        };
        let st = self.st.as_ref().unwrap();
        let (unit, target_unit) = (&st.units[u], &st.units[tgt]);
        let nm = if target_unit.name.is_empty() { target_unit.cls_name.clone() } else { target_unit.name.clone() };
        let anm = if unit.name.is_empty() { unit.cls_name.clone() } else { unit.name.clone() };
        let (u_fig, t_fig) = (unit.battle_fig, target_unit.battle_fig);
        if let Err(err) = self.ensure_native_attack_presentation(u_fig, t_fig) {
            self.load_err =
                format!("AI FIGANI attack presentation unavailable: {} -> {}: {}", u_fig, t_fig, err.message());
            self.ai_busy = false;
            return;
        }
        let scene = match self.prepare_native_physical_scene(Some(u), tgt) {
            Ok(scene) => scene,
            Err(err) => {
                self.load_err = format!("AI physical scene unavailable: {}", err.message());
                self.ai_busy = false;
                return;
            }
        };
        // 0x1548E：聚焦自己（0x154AD）→ 0x14B78 移動 → 聚焦目標（0x154DE）再進 0x1F04A
        // 演出。有走路時自己的聚焦已在走之前做過（原位），走完不再聚焦自己（第八章 c3
        // eip-trace seq 1692：(20,18) 0x154B2 → (16,22) 0x154E3，中間沒有 (18,20)）。
        let (ux, uy, tx, ty) = {
            let units = &self.st.as_ref().unwrap().units;
            (units[u].x, units[u].y, units[tgt].x, units[tgt].y)
        };
        if plan.path.len() < 2 {
            self.ai_focus_cursor(ux, uy);
        }
        self.ai_focus_cursor(tx, ty);
        let st = self.st.as_mut().unwrap();
        let (ux, uy) = (st.units[u].x, st.units[u].y);
        st.units[u].set_map_pose(dir_toward(ux, uy, tx, ty));
        let hp0 = st.units[tgt].hp;
        let rng_before = self.native_rng_state;
        let full_result = match self.resolve_physical_attack_full(Some(u), Some(tgt)) {
            Ok(result) => result,
            Err(err) => {
                // The normalized planner must not consume an action when the
                // production RNG boundary is unavailable.
                self.load_err = format!("AI physical attack: {}", err.message());
                self.ai_busy = false;
                return;
            }
        };
        let attack_result = full_result.attack.clone();
        if std::env::var("FD2_SHOT_AI").is_ok_and(|v| !v.is_empty()) {
            let units = &self.st.as_ref().unwrap().units;
            let counter = match attack_result.counter.as_ref() {
                Some(c) => format!(
                    "dmg={} miss={} exp={} lvups={:?}",
                    c.amount, c.missed, full_result.counter_exp_gained, full_result.counter_level_ups
                ),
                None => String::from("none"),
            };
            eprintln!(
                "AI attack: {}({},{}) hp={} → ({},{}) id={} hp {}→{} dmg={} miss={} counter={} rng {}→{}",
                units[u].cls_name,
                units[u].x,
                units[u].y,
                units[u].hp,
                units[tgt].x,
                units[tgt].y,
                units[tgt].native_identity,
                hp0,
                units[tgt].hp,
                attack_result.amount,
                attack_result.missed,
                counter,
                rng_before,
                self.native_rng_state
            );
        }
        self.award_death_reward(Some(tgt), Some(u));
        // 反擊把攻方打倒：0x1562F 0x1B6B7 → 0x1AA1D([0x53C4B]) 把掉落給被打的那個
        // 單位（第四章 r9 酒店存檔收據 rec2 多出 0xCE／0xCA 兩件）。
        self.award_death_reward(Some(u), Some(tgt));
        let units = &self.st.as_ref().unwrap().units;
        let message = match player_physical_attack_message(
            self.locale_catalog.as_ref(),
            Some(&units[u]),
            Some(&units[tgt]),
            &attack_result,
        ) {
            Ok(message) => message,
            Err(err) => {
                self.load_err = format!("AI physical attack locale: {}", err.message());
                self.ai_busy = false;
                return;
            }
        };
        self.publish_physical_attack_message(&message);
        let units = &self.st.as_ref().unwrap().units;
        let (a, t) = (&units[u], &units[tgt]);
        let terrain = self.terrain_at(t.x, t.y);
        let mut atk = self.new_atk_anim(
            a.battle_fig,
            t.battle_fig,
            &anm,
            &nm,
            a.hp,
            a.max_hp,
            a.lv,
            a.mp,
            a.max_mp,
            t.lv,
            t.mp,
            t.max_mp,
            hp0,
            t.hp,
            t.max_hp,
            terrain,
            a.camp == battle::OWN,
        );
        // Go 的 `g.atk` 就是這個演出；兩個被呼叫端都不讀 `g.atk`，Rust 改完再放進去。
        self.attach_counter_presentation(atk.as_mut(), Some(u), Some(tgt), &attack_result);
        if let Some(mut a) = atk {
            a.native_scene = scene;
            if let Err(err) = self.attach_native_physical_body(a.native_scene.as_mut(), &full_result) {
                self.load_err = format!("AI physical attack body: {}", err.message());
                self.atk = None;
                self.ai_busy = false;
                return;
            }
            let plan = Arc::clone(plan);
            a.after = Some(Box::new(move |g: &mut Game| g.ai_step_finish(&plan)));
            self.atk = Some(a);
        } else {
            self.atk = None;
            self.load_err = format!("AI FIGANI attack presentation unavailable: {} -> {}", u_fig, t_fig);
            self.ai_step_finish(plan);
        }
        self.check_result();
    }

    /// 敵方指令／道具的已閉合 owner 分派（Go `executeNativeAIActionWithContinuation` 的 switch）。
    pub(crate) fn execute_native_ai_action(&mut self, plan: &battle::AIPlan) -> Result<()> {
        if plan.native_action_kind == battle::NATIVE_AI_ACTION_ITEM {
            return self.execute_native_ai_item_restore(plan);
        }
        match plan.native_command_id {
            0 => self.execute_native_ai_command0(plan),
            1 | 4..=5 => self.execute_native_ai_indexed_damage_command(plan),
            2 => self.execute_native_ai_command2(plan),
            8 => self.execute_native_ai_command8(plan),
            9 => self.execute_native_ai_command9(plan),
            6 => self.execute_native_ai_command6(plan),
            7 => self.execute_native_ai_command7(plan),
            3 => self.execute_native_ai_command3(plan),
            13..=16 => self.execute_native_ai_command_heal(plan),
            17..=19 => self.execute_native_ai_command_modifier(plan),
            20..=22 | 26 | 27 => self.execute_native_ai_command2022(plan),
            id => Err(game_err!("指令 {id} 尚無執行入口")),
        }
    }

    /// Go `aiStep` 的閉包 `finish`（`main.go:13291`..`13340`）。
    fn ai_step_finish(&mut self, plan: &Arc<battle::AIPlan>) {
        let u = plan.u.expect("Go：plan.U 為 nil 時解參考 panic");
        if plan.native_mode_event_active {
            let audio_cue = battle::native_ai_mode5_audio_cue_for_raw_tail();
            if !os_mute_or_shot(Some(self)) && self.sfx.get(&audio_cue.index).is_none_or(|s| s.is_empty()) {
                self.load_err = format!(
                    "native AI mode 5 raw sample unavailable: resource={} index={}",
                    audio_cue.resource_id, audio_cue.index
                );
                self.ai_busy = false;
                return;
            }
            if std::env::var("FD2_SHOT_AI").is_ok_and(|v| !v.is_empty()) {
                let unit = &self.st.as_ref().unwrap().units[u];
                eprintln!(
                    "0x13a9f mode5 event: unit id={} at ({},{}) event={} dest=({},{}) before={:?}",
                    unit.native_identity,
                    unit.x,
                    unit.y,
                    plan.native_mode_event_id,
                    plan.native_mode_event_destination.x,
                    plan.native_mode_event_destination.y,
                    unit.native_record_death_effect
                );
            }
            // Go 在 ApplyNativeAIMode5EventWithAudioCue 的回呼裡直接 playSFX；Rust 借用 `st` 時不能再借用 `Game`，
            // 先收集通過檢查的 cue，套用完再依序播放（`mode5-audio-after-apply`）。
            let mut cues = Vec::new();
            let mut emit = |cue: battle::NativeAIMode5AudioCue| {
                // Native 0x13D0D calls 0x25B45([0x53EE8],12,1).
                // The extracted FDOTHER #31 sample index is the only
                // accepted mapping; do not substitute a normalized SFX.
                if cue.handle_linear_address == battle::NATIVE_AI_MODE5_AUDIO_HANDLE
                    && cue.resource_id == battle::NATIVE_AI_MODE5_AUDIO_RESOURCE
                    && cue.index == battle::NATIVE_AI_MODE5_AUDIO_INDEX
                    && cue.loop_count == battle::NATIVE_AI_MODE5_AUDIO_LOOP_COUNT
                {
                    cues.push(cue.index);
                }
            };
            let applied = self.st.as_mut().unwrap().apply_native_ai_mode5_event_with_audio_cue(
                Some(u),
                plan.native_mode_event_id,
                plan.native_mode_event_destination,
                Some(&mut emit),
            );
            for index in cues {
                self.play_sfx(index);
            }
            if let Err(err) = applied {
                self.load_err = format!("native AI mode event: {}", err.message());
                self.ai_busy = false;
                return;
            }
        }
        let write_byte5 = plan.native_mode_write_byte5;
        if let Some(decision) = plan.native_fallback_idle_recovery {
            let after: Box<dyn FnOnce(&mut Game) + Send> = Box::new(move |g: &mut Game| {
                if !g.ai_step_write_byte5(u, write_byte5) {
                    return;
                }
                g.finish_successful_unit_action(Some(u), None);
                g.ai_step_departed(write_byte5);
            });
            if let Err(err) = self.begin_native_ai_idle_recovery(Some(u), decision, Some(after)) {
                self.load_err = format!("native AI mode fallback 0x13fd4: {}", err.message());
                self.ai_busy = false;
            }
            return;
        }
        if !self.ai_step_write_byte5(u, write_byte5) {
            return;
        }
        self.finish_successful_unit_action(Some(u), None);
        self.ai_step_departed(write_byte5);
    }

    /// Go `aiStep` 的閉包 `writeByte5`。
    fn ai_step_write_byte5(&mut self, u: UnitIdx, write: bool) -> bool {
        if write {
            // 0x32975 writes the complete runtime +0x05 byte only after
            // mode 7's raw destination comparison succeeds (0x13D60, after
            // the optional 0x13FD4).  The record leaves the field: on_field
            // follows +5 bit0 so the result rule no longer counts it.
            // Acted remains the engine projection updated by the common
            // completion owner below.
            let st = self.st.as_mut().expect("Go：u 指向 g.st.Units");
            if let Err(err) = st.mark_native_unit_inactive(u) {
                self.load_err = format!("native AI mode 7 departure: {}", err.message());
                self.ai_busy = false;
                return false;
            }
        }
        true
    }

    /// Go `aiStep` 的閉包 `departed`：0x1D80B／0x1D8BA／0x1D988 每個單位走完 0x13A9F 都呼叫
    /// 章節處理器 [0x51B19]（0x1D8A0／0x1D96C／0x1D9FC），[0x53ECC] 非零就停下敵方迴圈；撤退的
    /// 最後一名敵軍抵達離場的當下就判勝負。收尾若開了格子事件（0x51B91 在 0x51B19 之前），
    /// 等它播完；玩家回合開始的 check_result 仍會補判。
    fn ai_step_departed(&mut self, write: bool) {
        if write && self.battle_event.is_none() {
            self.check_result();
        }
    }
}

// R4 A 組：A 組在 main.go 名下函式的測試（檔名帶組別，避免與別組的 main 測試模組衝突）。
#[cfg(test)]
#[path = "main/main_a_test.rs"]
mod main_a_test;

#[cfg(test)]
#[path = "main/main_b_test.rs"]
mod main_b_test;

#[cfg(test)]
#[path = "main/main_d_test.rs"]
mod main_d_test;

// R5 階段 2 r5m：地圖與單位組的測試。
#[cfg(test)]
#[path = "main/main_r5m_test.rs"]
mod main_r5m_test;

// R5 階段 2 r5p：Draw 本體的正規化地圖層與舊 fade 疊層的測試。
#[cfg(test)]
#[path = "main/main_r5p_test.rs"]
mod main_r5p_test;

// R5 階段 2 r5c：城鎮與設施組的測試。
#[cfg(test)]
#[path = "main/main_r5c_test.rs"]
mod main_r5c_test;

// R5 階段 2 r5b：戰鬥演出組的測試。
#[cfg(test)]
#[path = "main/main_r5b_test.rs"]
mod main_r5b_test;
// R5 階段 2 r5h：HUD 與選單組的測試。
#[cfg(test)]
#[path = "main/main_r5h_test.rs"]
mod main_r5h_test;

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`main.go:8363`，`func (g *Game) tileAt(idx int) *ebiten.Image`
    /// Go 的 `g.tiles[idx]` 可能是 nil（切圖失敗的格），Rust 是 `None`。
    pub(crate) fn tile_at(&self, idx: i64) -> Option<Arc<Picture>> {
        if idx < 0 || idx >= self.tiles.len() as i64 {
            return None;
        }
        self.tiles[idx as usize].clone()
    }

    /// Go：`main.go:9338`，`func (g *Game) nativeMapFrameAdmission(legacyViewport, campaignBattleView bool) bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn native_map_frame_admission(&self, legacy_viewport: bool, campaign_battle_view: bool) -> bool {
        if legacy_viewport || !campaign_battle_view {
            return false;
        }
        // 現代主題由正規化 renderer 疊加真彩圖集與人物；原版完整索引幀會把它們
        // 全部蓋掉，因此兩種呈現管線必須互斥。
        if self.modern_story_portraits.is_some() {
            return false;
        }
        if self.spell_open || self.item_open || self.cast_sp.is_some() {
            return false;
        }
        self.sel.is_none()
            || self.native_command0_targeting
            || self.native_item_targeting
            || self.native_item_relocating
            || self.ring
            || self.native_command_open
            || self.native_move_plan.is_some()
    }

    /// shouldDrawTransientBattleMessage 不讓診斷／結果文字覆蓋已還原的原生目標選擇器。
    /// 未修改原版的 command 0 玩家軌跡在 LUT 相位循環中只呈現索引地圖、游標與 HUD；
    /// modal 關閉後仍保留 g.msg，供拒絕或結果 owner 使用。其餘既有戰鬥短訊在各自
    /// caller-specific UI 閉合前維持原本呈現。
    ///
    /// Go：`main.go:10194`，`func (g *Game) shouldDrawTransientBattleMessage() bool`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn should_draw_transient_battle_message(&self) -> bool {
        !self.msg.is_empty() && self.dialog.is_empty() && !self.native_command0_targeting
    }

    /// drawRing shows the native FDOTHER #2 action overlay in the faithful theme.
    /// A fully admitted modern theme deliberately bypasses that original overlay so
    /// its catalog-verified action icons remain visible in local complete packages.
    ///
    /// Go：`main.go:10221`，`func (g *Game) drawRing(screen *ebiten.Image)`
    ///
    /// R5 階段 2 r5h。開場與第一章只走到開頭的守門（階段 0 覆蓋率 8408／6355 次全部在守門返回；第一章指令環
    /// 開著的幀都由 `nativeActionOverlayInMapFrame` 承認，不呼叫本函式）。守門之後：HD 指令環兩條是
    /// `todo!("r6:…")`；原版 `drawNativeActionOverlay` 與 `actionOverlayAnchor`（`r8`）之後的重製端後備環
    /// 標 `todo!("r8:…")`，不輸出近似畫面。
    pub(crate) fn draw_ring(&mut self, screen: &mut Picture) {
        if !self.ring || self.m.is_none() || (self.sel.is_none() && !self.native_system_cursor_overlay) {
            return;
        }
        if self.modern_hd_world_visible && self.uses_modern_battle_action_icons() {
            todo!("r6:modern_hd_map_gpu.go:drawModernHDMapRing");
        }
        if self.native_map_presented && self.uses_modern_battle_action_icons() {
            todo!("r6:modern_hd_action_icon.go:drawModernNativeMapRing");
        }
        let _ = screen;
        todo!(
            "r8:main.go:10237..10303 actionOverlayAnchor、drawNativeActionOverlay 與重製端後備環（開場與第一章沒有走到）"
        )
    }

    /// drawSpellMenu 法術選單(名稱 + MP;↑↓選、Enter 施放、ESC 回環)。
    ///
    /// Go：`main.go:10537`，`func (g *Game) drawSpellMenu(screen *ebiten.Image)`
    ///
    /// R5 階段 2 r5h 逐行移植。開場與第一章只走到守門（`spellOpen` 都是 false），本體只由單元測試驗。
    pub(crate) fn draw_spell_menu(&mut self, screen: &mut Picture) {
        let Some(sel) = self.sel else {
            return;
        };
        if !self.spell_open || self.font.is_none() {
            return;
        }
        let (sel_mp, sel_spells) = {
            let u = &self.st.as_ref().expect("g.sel 指向 g.st.Units").units[sel];
            (u.mp, u.spells.clone())
        };
        let h = 44.0 + sel_spells.len() as f64 * 30.0;
        let mut boxed = Picture::new_image(230, h as i64);
        boxed.fill(ColorRgba { r: 0x10, g: 0x1c, b: 0x40, a: 0xee });
        let mut op = DrawImageOptions::default();
        op.geo_m.translate(20.0, 60.0);
        screen.draw_image(&boxed, &op);
        let (title, ok) = self.locale_message("battle.spell.menu_title", &[sel_mp.into()]);
        if ok {
            self.font.as_mut().unwrap().draw(
                screen,
                &title,
                34.0,
                68.0,
                1.0,
                ColorRgba { r: 0xff, g: 0xe0, b: 0x90, a: 0xff },
            );
        }
        for (i, id) in sel_spells.iter().enumerate() {
            let Some(sp) = self.spells.iter().find(|s| s.id == *id) else {
                continue;
            };
            let mut c = ColorRgba { r: 0xd0, g: 0xd8, b: 0xe8, a: 0xff };
            let mut pre = "　";
            if i as i64 == self.spell_sel {
                c = ColorRgba { r: 0xff, g: 0xff, b: 0xff, a: 0xff };
                pre = "▶";
            }
            if sel_mp < sp.mp {
                c = ColorRgba { r: 0x80, g: 0x80, b: 0x90, a: 0xff }; // MP 不足變暗
            }
            let line = format!("{}{}  MP{}", pre, sp.name, sp.mp);
            self.font.as_mut().unwrap().draw(screen, &line, 32.0, 96.0 + i as f64 * 30.0, 1.0, c);
        }
    }

    /// drawItemMenu prefers the proven indexed native panel. The text shell below
    /// is only a legacy/missing-original-assets fallback; using an item remains
    /// blocked when its 0x20c6f effect/target transaction is unavailable.
    ///
    /// Go：`main.go:10577`，`func (g *Game) drawItemMenu(screen *ebiten.Image)`
    ///
    /// R5 階段 2 r5h。開場與第一章只走到守門（`itemOpen` 都是 false）；守門之後先呼叫的
    /// `drawNativeItemPanel`（`native_item_panel_ui.go`）與後備用的 `nativeItemMenuSlot` 都是 `r8`，整段標
    /// `todo!("r8:…")`。
    pub(crate) fn draw_item_menu(&mut self, screen: &mut Picture) {
        if !self.item_open || self.sel.is_none() {
            return;
        }
        if self.draw_native_item_panel(screen) {
            return;
        }
        todo!("r8:main.go:10594..10609 TTF 後備物品清單（缺原版素材時）")
    }

    /// drawNativeCommandGrid renders the recovered 0x1ceed four-row layout only
    /// when its player-provided labels and VGA palette are both available.
    ///
    /// Go：`main.go:10617`，`func (g *Game) drawNativeCommandGrid(screen *ebiten.Image)`
    ///
    /// R5 階段 2 r5h。開場與第一章只走到守門（`nativeCommandOpen` 都是 false）；格子文字要
    /// `localizedNativeCommandLabel`（`native_command_locale.go`，`r8`），守門之後標 `todo!("r8:…")`。
    pub(crate) fn draw_native_command_grid(&mut self, screen: &mut Picture) {
        if !self.native_command_open || self.sel.is_none() || self.font.is_none() || self.native_ui_palette.len() < 0xce
        {
            return;
        }
        let _ = screen;
        todo!("r8:main.go:10621..10634 localizedNativeCommandLabel 與格子文字（開場與第一章沒有走到）")
    }

    /// drawCampaignUI campaign 節點 UI:choice 選單 / ending 結語 / game over。
    ///
    /// Go：`main.go:10638`，`func (g *Game) drawCampaignUI(screen *ebiten.Image)`
    ///
    /// R5 階段 2 r5c。第一章走到的分支（階段 0 覆蓋率）只有各節點的原生呈現：城鎮 `drawNativeTown`、商店
    /// `drawNativeShop`、酒店 `drawNativeHotel`、整備 `drawNativePreparation`、教會 `drawNativeChurchUIJob`，各自成立就
    /// return；其他節點型別（戰場、故事等）沒有分支。原生呈現不成立時的 TTF 後備版面（`fillBox`＋`Font.Draw`）、流程
    /// 結束的 GAME OVER、`choice` 選單、教會的其餘子畫面與 `ending` 結語都沒有走到，`todo!("r8:…")`；分支表見
    /// `PORTING.md`「R5 階段 2：r5c」。Go 的 `fillBox` 閉包只在後備分支使用，這裡不建立。
    pub(crate) fn draw_campaign_ui(&mut self, screen: &mut Picture) {
        if self.camp.is_none() || self.font.is_none() {
            return;
        }
        let Some(node_type) = self.camp.as_ref().unwrap().node().map(|n| n.r#type.clone()) else {
            todo!("r8:main.go:10652 流程結束的 GAME OVER（fillBox＋TTF；開場與第一章沒有走到）");
        };
        match node_type.as_str() {
            "choice" | "town" => {
                if node_type == "town" && self.draw_native_town(screen) {
                    return;
                }
                todo!("r8:main.go:10660..10679 choice／town 的 TTF 後備選單（開場與第一章沒有走到）");
            }
            "shop" => {
                if self.draw_native_shop(screen) {
                    return;
                }
                todo!("r8:main.go:10685..10764 商店的 TTF 後備版面（開場與第一章沒有走到）");
            }
            "hotel" => {
                if self.draw_native_hotel(screen) {
                    return;
                }
                todo!("r8:main.go:10769..10794 酒店的 TTF 後備版面（開場與第一章沒有走到）");
            }
            "preparation" => {
                if self.draw_native_preparation(screen) {
                    return;
                }
                todo!("r8:main.go:10799..10889 整備的 TTF 後備版面（開場與第一章沒有走到）");
            }
            "church" => {
                if self.draw_native_church_ui_job(screen) {
                    return;
                }
                if self.draw_native_class_ui_job(screen) {
                    return;
                }
                self.draw_native_church_service(screen);
            }
            "ending" => todo!("r8:main.go:11099..11122 ending 結語（開場與第一章沒有走到）"),
            _ => {}
        }
    }

    /// mapSpriteGroup chooses the exact native raw FDICON key only for a battle
    /// State whose whole construction sequence materialized successfully. Story
    /// actors remain on their editable legacy Fig path by construction.
    ///
    /// Go：`main.go:11514`，`func (g *Game) mapSpriteGroup(u *battle.Unit) int`
    /// 所有權：`u` 是 `self.st.units` 的索引（Go 傳 `*battle.Unit`，唯一的呼叫端只傳 `g.st.Units` 的單位）；
    /// Go 在 `g.st == nil` 時直接讀 `u.Fig`，Rust 沒有 `st` 就沒有 `u`。
    pub(crate) fn map_sprite_group(&self, u: UnitIdx) -> i64 {
        let st = self.st.as_ref().expect("Go：u 指向 g.st.Units");
        let (key, ok) = st.native_map_sprite_key(Some(u));
        if ok {
            return key;
        }
        st.units[u].fig
    }

    /// drawUnitSprite 以 FDICON Q 版 sprite 呈現單位；正式 sprite 路徑不另加重製端
    /// HP bar／腳標，fallback 色塊則仍有自己的除錯資訊。完整戰場畫面另依 E2 驗收。
    /// 用方向走動分鏡(FDICON 12幀=4方向×3:站/抬左手/抬右手);行軍時套用 OffX/OffY 位移。
    /// spriteGroup is either a proven native raw key or an explicit legacy Fig;
    /// this helper never infers one from the other.
    ///
    /// Go：`main.go:11528`，`func (g *Game) drawUnitSprite(screen *ebiten.Image, x, y, w, h float64, u *battle.Unit, spriteGroup int)`
    ///
    /// 簽名（R5 階段 2）：`u` 改成 `&battle::Unit`、接收者改成 `&self`。Go 的呼叫端傳 `g.st.Units` 的單位、
    /// `&g.storyActors[i]`（`main.go:9894`、`9905`）與 `g.sel`（`main.go:10257`），storyActors 不是 `st.units`
    /// 的索引；函式本身不改 `Game`。
    #[allow(clippy::too_many_arguments)]
    pub(crate) fn draw_unit_sprite(
        &self,
        screen: &mut Picture,
        x: f64,
        y: f64,
        w: f64,
        h: f64,
        u: &battle::Unit,
        sprite_group: i64,
    ) {
        let x = x + u.off_x; // 行軍/移動位移
        let y = y + u.off_y;
        let frames = self
            .modern_story_portraits
            .as_ref()
            .and_then(|theme| theme.map_sprites.get(&sprite_group))
            .or_else(|| self.sprites.get(&sprite_group))
            .map(Vec::as_slice)
            .unwrap_or_default();
        if frames.is_empty() {
            // Go `drawUnit(screen, x, y, w, h, campColor(u.Camp), u)`：開場與第一章沒有走到（階段 0 覆蓋率 0）。
            let _ = (w, h);
            todo!("r8:main.go:drawUnit（fallback 色塊）");
        }
        // 方向走動幀:dir(0下1左2上3右)×3 + 走動相位;不足 12 幀(只導下方向)則退回
        let f = (self.frame / 8) % 3;
        let mut idx = u.dir * 3 + f;
        if idx >= frames.len() as i64 {
            idx = f % frames.len() as i64;
        }
        let frame = frames[usize::try_from(idx).expect("Go：frames[idx] 負索引會 panic")]
            .as_ref()
            .expect("Go：DrawImage(nil) 會 panic");
        let mut op = DrawImageOptions::default();
        op.geo_m.translate(x, y - 4.0); // 略上移讓單位「站」在格上
        if u.acted {
            op.color_scale.scale(0.55, 0.55, 0.6, 1.0); // 已行動變暗(對映原版灰階)
        }
        screen.draw_image(frame, &op);
    }

    /// phaseBannerElapsedMillis 是這一段橫幅已經走了多少毫秒。
    ///
    /// Go：`main.go:12148`，`func (g *Game) phaseBannerElapsedMillis() float64`
    pub(crate) fn phase_banner_elapsed_millis(&self) -> f64 {
        (phase_banner_frames(&self.banner) - self.banner_t) as f64 * 1000.0 / 60.0
    }

    /// phaseBannerSlideOffset 回傳字樣相對停住位置的 x 偏移（原版像素），以及字樣
    /// 這一刻要不要畫。滑入在馬賽克之前、滑出在馬賽克之後，都是每步一個 BIOS tick。
    ///
    /// Go：`main.go:12154`，`func (g *Game) phaseBannerSlideOffset() (int, bool)`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn phase_banner_slide_offset(&self) -> (i64, bool) {
        if self.banner_t <= 0 || self.banner.is_empty() {
            return (0, false);
        }
        let elapsed = self.phase_banner_elapsed_millis();
        let slide_in = indexedmap::PHASE_BANNER_SLIDE_IN_STEPS as f64 * indexedmap::PHASE_BANNER_STEP_MILLIS;
        if elapsed < slide_in {
            return (
                indexedmap::phase_banner_slide_in_offset((elapsed / indexedmap::PHASE_BANNER_STEP_MILLIS) as i64),
                true,
            );
        }
        let mosaic = indexedmap::PHASE_BANNER_STEPS as f64 * indexedmap::PHASE_BANNER_STEP_MILLIS
            + phase_banner_hold_millis(&self.banner);
        if elapsed < slide_in + mosaic {
            return (0, true);
        }
        (
            indexedmap::phase_banner_slide_out_offset(
                ((elapsed - slide_in - mosaic) / indexedmap::PHASE_BANNER_STEP_MILLIS) as i64,
            ),
            true,
        )
    }

    /// blitPhaseBannerGlyphs 把兩塊原版字樣畫進索引畫面，回傳是否畫成。缺分離素材時
    /// 回 false，由 drawPhaseBanner 用重製端字型頂著——那是可玩性的後備，不是原版
    /// 版面。
    ///
    /// Go：`main.go:12203`，`func (g *Game) blitPhaseBannerGlyphs(dst []byte, offset int) bool`
    pub(crate) fn blit_phase_banner_glyphs(&self, dst: &mut [u8], offset: i64) -> bool {
        let Some(assets) = self.native_map_assets.as_ref() else {
            return false;
        };
        if assets.command_heal_digits.len() as i64 <= PHASE_BANNER_GLYPH_ENEMY {
            return false;
        }
        let mut first = PHASE_BANNER_GLYPH_ENEMY;
        if self.banner == PHASE_BANNER_PLAYER_TEXT {
            first = PHASE_BANNER_GLYPH_PLAYER;
        }
        // offset 是第一塊的偏移（負值＝還在左邊畫面外），第二塊取相反數往右退。
        blit_phase_banner_glyph(dst, &assets.command_heal_digits[first as usize], PHASE_BANNER_GLYPH_FIRST_X + offset)
            && blit_phase_banner_glyph(
                dst,
                &assets.command_heal_digits[PHASE_BANNER_GLYPH_PHASE as usize],
                PHASE_BANNER_GLYPH_SECOND_X - offset,
            )
    }

    /// phaseBannerGlyphsAvailable 回報這一輪橫幅會不會用原版字樣。
    ///
    /// Go：`main.go:12249`，`func (g *Game) phaseBannerGlyphsAvailable() bool`
    pub(crate) fn phase_banner_glyphs_available(&self) -> bool {
        self.native_map_assets.as_ref().is_some_and(|a| a.command_heal_digits.len() as i64 > PHASE_BANNER_GLYPH_ENEMY)
    }

    /// phaseBannerStep 回傳目前的馬賽克步驟；沒有橫幅時回 -1。
    ///
    /// 三段對應原版的三段：進場 16 步、中段 `delay()`(畫面停在進場最後一步)、
    /// 退場 17 步。
    ///
    /// Go：`main.go:12268`，`func (g *Game) phaseBannerStep() int`
    /// Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn phase_banner_step(&self) -> i64 {
        if self.banner_t <= 0 || self.banner.is_empty() {
            return -1;
        }
        // 馬賽克夾在字樣滑入與滑出之間，滑動那兩段畫的是正常地圖。
        let slide_in = indexedmap::PHASE_BANNER_SLIDE_IN_STEPS as f64 * indexedmap::PHASE_BANNER_STEP_MILLIS;
        let elapsed = self.phase_banner_elapsed_millis() - slide_in;
        if elapsed < 0.0 {
            return -1;
        }
        // Go 的 `PhaseBannerEnterSteps * PhaseBannerStepMillis` 是無型別常數運算；16 是 2 的冪，與 f64 乘法結果相同。
        let enter_millis = indexedmap::PHASE_BANNER_ENTER_STEPS as f64 * indexedmap::PHASE_BANNER_STEP_MILLIS;
        let hold = phase_banner_hold_millis(&self.banner);
        let mosaic = indexedmap::PHASE_BANNER_STEPS as f64 * indexedmap::PHASE_BANNER_STEP_MILLIS + hold;
        if elapsed >= mosaic {
            return -1;
        }
        let mut step = if elapsed < enter_millis {
            (elapsed / indexedmap::PHASE_BANNER_STEP_MILLIS) as i64
        } else if elapsed < enter_millis + hold {
            indexedmap::PHASE_BANNER_ENTER_STEPS - 1
        } else {
            indexedmap::PHASE_BANNER_ENTER_STEPS
                + ((elapsed - enter_millis - hold) / indexedmap::PHASE_BANNER_STEP_MILLIS) as i64
        };
        if step < 0 {
            step = 0;
        }
        if step >= indexedmap::PHASE_BANNER_STEPS {
            step = indexedmap::PHASE_BANNER_STEPS - 1;
        }
        step
    }

    /// drawPhaseBanner 是回合字樣的**後備路徑**：拿得到 FDOTHER #5 的三格字模時，
    /// 字樣已經由 blitPhaseBannerGlyphs 畫進索引畫面，這裡直接讓開；只有分離素材
    /// 缺件時才用重製端字型頂著，那時的字寬與落點都不是原版版面。
    ///
    /// 原版整段期間不重繪地圖，畫面變化來自對進場時那張快照重算的馬賽克，以及每步
    /// 套在 DAC 上的減量(`sub_11D40`，只動索引 0x10..0xFF，所以字樣不變暗)。馬賽克
    /// 前後另有字樣滑入 7 步與滑出 5 步(`sub_1F42D`)，每步同樣等一個 BIOS tick，
    /// 偏移就是那七／五個參數(見 indexedmap 的 PhaseBannerSlideInOffset／
    /// PhaseBannerSlideOutOffset；兩塊是相向移動的)。
    /// 見 docs/knowledge-base/105-phase-banner-timing-20260910.md。
    ///
    /// 落點與相向滑動來自 `sub_1F42D` 的 `0x55 − 參數` 與 `參數 + 0xA5`，並由原版
    /// 逐幀畫面逐像素比對確認(滑入 7 步、停留 33 步、滑出 5 步全中，收據
    /// docs/data/ui-traces/fd2-phase-banner-glyph-parity-20260910.json)。
    ///
    /// Go：`main.go:12317`，`func (g *Game) drawPhaseBanner(screen *ebiten.Image)`
    ///
    /// R5 階段 2 r5h 逐行移植。第一章每次都在 `phaseBannerGlyphsAvailable` 讓開（階段 0 覆蓋率 1999 次），TTF 後備
    /// 只由單元測試驗。
    pub(crate) fn draw_phase_banner(&mut self, screen: &mut Picture) {
        if self.banner_t <= 0 || self.banner.is_empty() || self.font.is_none() {
            return;
        }
        if self.phase_banner_glyphs_available() {
            // 原版字樣已經畫進索引畫面了，不要再疊一層自訂字型。
            return;
        }
        let (offset, visible) = self.phase_banner_slide_offset();
        if !visible {
            return;
        }
        let banner = self.banner.clone();
        let font = self.font.as_mut().unwrap();
        let w = font.width(&banner, 2.2);
        let c = ColorRgba { r: 0xff, g: 0xc8, b: 0x50, a: 0xff };
        // 原版座標是 320 寬，重製端的邏輯畫布是它的兩倍。
        let x = (LOGICAL_W as f64 - w) / 2.0 + (offset * LOGICAL_W / 320) as f64;
        font.draw(screen, &banner, x, LOGICAL_H as f64 / 2.0 - 24.0, 2.2, c);
    }

    /// Go：`main.go:12421`，`func (g *Game) drawNativeMapHUD(screen *ebiten.Image) bool`
    ///
    /// R5 階段 2 r5h 逐行移植：`BlitNativeMapHUD` 畫進 `fdicon.NativeMapStride`×200 的工作緩衝，取每列前 320 個
    /// 位元組成 320×200 索引畫格；調色盤索引 0 換成同色 alpha 0（`color.NRGBA`，`NewImageFromImage` 後是全透明），
    /// 再 `GeoM.Scale(2, 2)` 疊上去。
    pub(crate) fn draw_native_map_hud(&mut self, screen: &mut Picture) -> bool {
        let (r#in, ok) = self.native_map_hud_input();
        if !ok {
            return false;
        }
        let a = self.native_map_assets.as_ref().expect("nativeMapHUDInput 成功時有 nativeMapAssets");
        let st = self.st.as_ref().expect("nativeMapHUDInput 成功時有 g.st");
        let stride = fdicon::NATIVE_MAP_STRIDE as usize;
        let mut frame = vec![0u8; stride * 200];
        if indexedmap::blit_native_map_hud(
            &a.frames,
            a.terrain.as_deref(),
            a.units.as_ref(),
            st.native_map_selector_cache.as_ref(),
            &mut frame,
            &r#in,
        )
        .is_err()
        {
            return false;
        }
        let mut overlay_palette = a.palette.clone();
        // Go `color.NRGBA{R, G, B, A: 0}`：`color.RGBAModel` 轉換後四個通道都是 0。
        overlay_palette[0] = ColorRgba::default();
        let mut pix = vec![0u8; 320 * 200];
        for y in 0..200 {
            pix[y * 320..y * 320 + 320].copy_from_slice(&frame[y * stride..y * stride + 320]);
        }
        let img = IndexedFrame { width: 320, height: 200, pix, palette: overlay_palette };
        let mut op = DrawImageOptions::default();
        op.geo_m.scale(2.0, 2.0);
        screen.draw_image(&Picture::from_indexed(&img), &op);
        true
    }

    /// drawNativeMapFrame is the production 0x11cac bridge for the currently
    /// materialized neutral tactical state. It composes terrain, range, units,
    /// foreground and HUD atomically, then presents the corrected 312x192 viewport
    /// at VGA (4,4) on a 320x200 paletted surface.
    ///
    /// Go：`main.go:12450`，`func (g *Game) drawNativeMapFrame(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_map_frame(&mut self, screen: &mut Picture) -> bool {
        if self.compose_native_map_frame().is_err() {
            return false;
        }
        let (img, ok) = self.native_map_frame_image();
        if !ok {
            return false;
        }
        let img = img.expect("nativeMapFrameImage 成功時有影像");
        let mut op = DrawImageOptions::default();
        op.geo_m.scale(2.0, 2.0);
        screen.draw_image(&Picture::from_indexed(&img), &op);
        self.native_interface_hud_visible = self.native_interface_hud.is_some();
        self.native_map_presented = true;
        true
    }

    /// nativeMapFrameImage 把已組好的 nativeMapVGA 配上當前 DAC，再依橫幅狀態
    /// （bannerT）疊上馬賽克與字樣，回傳玩家看到的 320×200 索引畫面。重播測試
    /// 也走這裡取橫幅期間的畫面。
    ///
    /// Go：`main.go:12469`，`func (g *Game) nativeMapFrameImage() (*image.Paletted, bool)`
    pub(crate) fn native_map_frame_image(&mut self) -> (Option<Arc<IndexedFrame>>, bool) {
        let size = indexedmap::NATIVE_MAP_VGA_SIZE as usize;
        let Some(a) = self.native_map_assets.as_ref() else {
            return (None, false);
        };
        if self.native_map_vga.len() < size {
            return (None, false);
        }
        let mut palette = a.palette.clone();
        if self.native_map_dac.len() == 256 * 3
            && let Ok(current) = fdother::vga_palette_from_dac(&self.native_map_dac)
        {
            palette = color_palette_from_nrgba(&current);
        }
        // 回合橫幅期間，原版在 `sub_1F1CC` 開頭 memcpy 整幀螢幕，之後整段不重繪，
        // 每一步都對那張快照重算馬賽克再整幀寫回。這裡照同一個模型：第一次繪製
        // 時凍結一份，之後只從它取樣，不再跟著當前畫面走。
        let step = self.phase_banner_step();
        // 原版在 `sub_1F1CC` 一進來就 memcpy 螢幕，那是**字樣滑入之前**；滑入期間
        // 地圖本身不重繪，所以在橫幅一開始就凍結與原版等價。
        // Go `g.bannerFrame == nil` 在 Rust 是空 Vec（showBanner 設回空；這裡的複本至少一幀，不會是空）。
        if self.banner_t > 0 && !self.banner.is_empty() && self.banner_frame.is_empty() {
            self.banner_frame = self.native_map_vga.clone();
        }
        // 每一步另外把 DAC 索引 0x10..0xFF 減去「方塊邊長 − 1」(`sub_11D40`)，
        // 索引 0x00..0x0F 不動，所以字樣不隨地圖變暗。
        if step >= 0 {
            let level = indexedmap::phase_banner_dim(step);
            let a = self.native_map_assets.as_ref().expect("上面已確認 nativeMapAssets");
            if level > 0 && self.native_map_dac.len() == 256 * 3 && a.palette_dac.len() == 256 * 3 {
                let mut dimmed = self.native_map_dac.clone();
                if fdother::apply_vga_palette_subtraction(
                    &mut dimmed,
                    &a.palette_dac,
                    indexedmap::PHASE_BANNER_DIM_FIRST_INDEX,
                    indexedmap::PHASE_BANNER_DIM_LAST_INDEX,
                    level,
                )
                .is_ok()
                    && let Ok(current) = fdother::vga_palette_from_dac(&dimmed)
                {
                    palette = color_palette_from_nrgba(&current);
                }
            }
        }
        let mut img = IndexedFrame { width: 320, height: 200, pix: vec![0; size], palette };
        img.pix.copy_from_slice(&self.native_map_vga[..size]);
        if step >= 0 {
            let src = if self.banner_frame.len() < size { &self.native_map_vga } else { &self.banner_frame };
            if indexedmap::mosaic_native_map_viewport(&mut img.pix, src, indexedmap::phase_banner_block(step)).is_err()
            {
                return (None, false);
            }
        }
        // 字樣疊在馬賽克之後，與原版同序（`sub_4E809` 之後才 `sub_15F0E`）。
        let (offset, visible) = self.phase_banner_slide_offset();
        if visible {
            self.blit_phase_banner_glyphs(&mut img.pix, offset);
        }
        (Some(Arc::new(img)), true)
    }
}

/// Go：`main.go:11571`，`func drawCursor(dst *ebiten.Image, x, y, w, h float64)`
///
/// R5 階段 2 r5h 逐行移植：四條 2 像素寬的白條（`image.White`，`Fill` 為 `0xff` 四通道），各自 `NewImage`
/// 後以 `GeoM.Translate` 疊上去；`int(b.w)` 是 Go 的向零截斷。
pub(crate) fn draw_cursor(dst: &mut Picture, x: f64, y: f64, w: f64, h: f64) {
    let col = ColorRgba { r: 0xff, g: 0xff, b: 0xff, a: 0xff };
    let t = 2.0;
    let bars = [(x, y, w, t), (x, y + h - t, w, t), (x, y, t, h), (x + w - t, y, t, h)];
    for (bx, by, bw, bh) in bars {
        let mut sub = Picture::new_image(bw as i64, bh as i64);
        sub.fill(col);
        let mut op = DrawImageOptions::default();
        op.geo_m.translate(bx, by);
        dst.draw_image(&sub, &op);
    }
}

/// blitPhaseBannerGlyph 把一格字模畫在索引畫面的 y ＝ phaseBannerGlyphY 上，
/// **裁在 0x11EB0 的搬運窗格內**。
///
/// 原版把字樣畫進離屏緩衝區，再由 `sub_11EB0` 把 312×192 搬到 VGA 的 (4,4)；
/// 超出窗格的部分不會出現在畫面上。滑入第一步的 `ENEMY` 落在 x ＝ −11，若照著
/// 整幀去畫就會多畫到左邊那四欄黑邊——原版那四欄留著先前的內容。
///
/// Go：`main.go:12225`，`func blitPhaseBannerGlyph(dst []byte, cell fdother.LMI1Entry, x int) bool`
pub(crate) fn blit_phase_banner_glyph(dst: &mut [u8], cell: &fdother::LMI1Entry, x: i64) -> bool {
    if cell.width <= 0 || cell.height <= 0 || cell.pixels.len() as i64 != cell.width * cell.height {
        return false;
    }
    for row in 0..cell.height {
        let y = PHASE_BANNER_GLYPH_Y + row;
        if !(indexedmap::NATIVE_MAP_VIEWPORT_Y0..=indexedmap::NATIVE_MAP_VIEWPORT_Y1).contains(&y) {
            continue;
        }
        for col in 0..cell.width {
            let dx = x + col;
            if !(indexedmap::NATIVE_MAP_VIEWPORT_X0..=indexedmap::NATIVE_MAP_VIEWPORT_X1).contains(&dx) {
                continue;
            }
            // 索引 0 是透明；字身 2..9 落在 DAC 暗化不動的 0x00..0x0F。
            let v = cell.pixels[(row * cell.width + col) as usize];
            if v != 0 {
                dst[(y * 320 + dx) as usize] = v;
            }
        }
    }
    true
}

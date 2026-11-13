//! FD2.SAV 的原版儲存 ABI。
//!
//! 移植自 Go `remake/internal/fdsave/save.go`（凍結於 v.1.1.11）。尚未閉合語意的名冊
//! 位元組維持原樣，不賦予遊戲意義；位址與證據註解沿用 Go 版。錯誤訊息與 Go 版逐字相同，
//! 以便黃金差分直接比對。

use sha2::{Digest, Sha256};
use std::fmt;

pub const FILE_SIZE: usize = 0x59cb;
pub const CHECKSUM_OFF: usize = FILE_SIZE - 4;
pub const SLOT_OFFSET: usize = 0x312b;
pub const SLOT_SIZE: usize = 0xa28;
pub const SLOT_COUNT: usize = 4;
pub const ROSTER_SIZE: usize = 0xa00;
pub const UNIT_SIZE: usize = 0x50;
pub const ROSTER_UNITS: usize = ROSTER_SIZE / UNIT_SIZE;
pub const METADATA_SIZE: usize = SLOT_SIZE - ROSTER_SIZE;

pub const CURRENT_PERSISTENT_ROSTER_OFFSET: usize = 0x08a3;
pub const CURRENT_RUNTIME_ROSTER_OFFSET: usize = 0x12a3;
pub const CURRENT_FIELD_CONTROL_OFFSET: usize = 0x0000;
pub const CURRENT_FIELD_CONTROL_SIZE: usize = 0x08a3;
pub const CURRENT_FIELD_CONTROL_UNIT_OFFSET: usize = 0x0083;
pub const CURRENT_FIELD_CONTROL_UNIT_SIZE: usize = 0x001a;
pub const CURRENT_FIELD_CONTROL_UNIT_CAP: usize =
    (CURRENT_FIELD_CONTROL_SIZE - CURRENT_FIELD_CONTROL_UNIT_OFFSET) / CURRENT_FIELD_CONTROL_UNIT_SIZE;
pub const CURRENT_NATIVE_EVENT_STATE_OFFSET: usize = 0x30a3;
pub const CURRENT_NATIVE_EVENT_STATE_SIZE: usize = 0x20;
pub const CURRENT_RUNTIME_HEADER_OFFSET: usize = 0x30c3;
pub const CURRENT_RUNTIME_HEADER_SIZE: usize = 18;

/// 錯誤種類；`Display` 與 Go 版的錯誤字串逐字相同。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SaveError {
    Size {
        want: usize,
        got: usize,
    },
    ChecksumMismatch {
        expected: u32,
        got: u32,
    },
    SlotRange(usize),
    InvalidPlainSize,
    RosterSize(usize),
    MetadataSize(usize),
    EmptyChapterSlot,
    RosterCountExceeds(u8),
    RuntimeCountExceeds(u8),
    PersistentCountExceeds(u8),
    ContinueRuntimeKey {
        index: usize,
        key: u8,
    },
    ContinueActivePresentation(usize),
    /// 其餘固定訊息的拒收（CONTINUE 前置條件與目前快照寫入）。
    Rejected(&'static str),
}

impl fmt::Display for SaveError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            SaveError::Size { want, got } => write!(f, "fdsave: want {want:#x} bytes, got {got:#x}"),
            SaveError::ChecksumMismatch { expected, got } => {
                write!(f, "fdsave: checksum mismatch: expected {expected:#010x}, got {got:#010x}")
            }
            SaveError::SlotRange(slot) => write!(f, "fdsave: slot {slot} outside 0..{}", SLOT_COUNT - 1),
            SaveError::InvalidPlainSize => f.write_str("fdsave: invalid plaintext size"),
            SaveError::RosterSize(got) => write!(f, "fdsave: roster size={got:#x}, want {ROSTER_SIZE:#x}"),
            SaveError::MetadataSize(got) => write!(f, "fdsave: metadata size={got:#x}, want {METADATA_SIZE:#x}"),
            SaveError::EmptyChapterSlot => f.write_str("fdsave: native chapter slot is empty"),
            SaveError::RosterCountExceeds(count) => {
                write!(f, "fdsave: roster count {count} exceeds native capacity {ROSTER_UNITS}")
            }
            SaveError::RuntimeCountExceeds(count) => {
                write!(f, "fdsave: runtime count {count} exceeds native capacity {}", ROSTER_UNITS * 3)
            }
            SaveError::PersistentCountExceeds(count) => {
                write!(f, "fdsave: persistent count {count} exceeds native capacity {ROSTER_UNITS}")
            }
            SaveError::ContinueRuntimeKey { index, key } => {
                write!(f, "fdsave: CONTINUE runtime record {index} has invalid FDICON key {key}")
            }
            SaveError::ContinueActivePresentation(index) => {
                write!(f, "fdsave: CONTINUE active runtime record {index} has invalid map presentation")
            }
            SaveError::Rejected(message) => f.write_str(message),
        }
    }
}

impl std::error::Error for SaveError {}

pub type Result<T> = std::result::Result<T, SaveError>;

fn rol16(v: u16, n: u32) -> u16 {
    v.rotate_left(n)
}

fn le_u16(b: &[u8]) -> u16 {
    u16::from_le_bytes([b[0], b[1]])
}

fn le_u32(b: &[u8]) -> u32 {
    u32::from_le_bytes([b[0], b[1], b[2], b[3]])
}

fn sha256_hex(data: &[u8]) -> String {
    Sha256::digest(data).iter().map(|b| format!("{b:02x}")).collect()
}

/// 原版 `0x4dbd8` 的 XOR 外層；自身即反函數。
pub fn xor_envelope(data: &[u8]) -> Vec<u8> {
    let mut state: u16 = 0x00a5;
    data.iter()
        .map(|&b| {
            state = rol16(state.wrapping_add(0x9014), 3);
            b ^ state as u8
        })
        .collect()
}

pub fn checksum(plain: &[u8]) -> Result<u32> {
    if plain.len() != FILE_SIZE {
        return Err(SaveError::Size { want: FILE_SIZE, got: plain.len() });
    }
    Ok(plain[..CHECKSUM_OFF].iter().fold(0u32, |sum, &b| sum.wrapping_add(b as u32)))
}

/// 驗大小、解外層並核對尾端 little-endian 校驗和；不改動呼叫端緩衝區。
pub fn decode(stored: &[u8]) -> Result<Vec<u8>> {
    if stored.len() != FILE_SIZE {
        return Err(SaveError::Size { want: FILE_SIZE, got: stored.len() });
    }
    let plain = xor_envelope(stored);
    let want = le_u32(&plain[CHECKSUM_OFF..]);
    let got = checksum(&plain)?;
    if got != want {
        return Err(SaveError::ChecksumMismatch { expected: want, got });
    }
    Ok(plain)
}

/// 寫入原版校驗和後套上可逆外層。
pub fn encode(plain: &[u8]) -> Result<Vec<u8>> {
    if plain.len() != FILE_SIZE {
        return Err(SaveError::Size { want: FILE_SIZE, got: plain.len() });
    }
    let mut out = plain.to_vec();
    let sum = checksum(&out)?;
    out[CHECKSUM_OFF..].copy_from_slice(&sum.to_le_bytes());
    Ok(xor_envelope(&out))
}

pub fn slot_bounds(slot: usize) -> Result<(usize, usize)> {
    if slot >= SLOT_COUNT {
        return Err(SaveError::SlotRange(slot));
    }
    let start = SLOT_OFFSET + slot * SLOT_SIZE;
    Ok((start, start + SLOT_SIZE))
}

/// 一筆原版章節槽紀錄。名冊與後設資料維持原始位元組，只有部分欄位語意已證實。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Slot {
    pub roster: Vec<u8>,
    pub metadata: Vec<u8>,
}

/// 回傳名冊與後設資料區的複本。
pub fn read_slot(plain: &[u8], slot: usize) -> Result<Slot> {
    if plain.len() != FILE_SIZE {
        return Err(SaveError::InvalidPlainSize);
    }
    let (start, end) = slot_bounds(slot)?;
    Ok(Slot { roster: plain[start..start + ROSTER_SIZE].to_vec(), metadata: plain[start + ROSTER_SIZE..end].to_vec() })
}

/// 取代明文存檔映像中的一筆章節槽。先驗證再複製，格式錯誤不會只改一半；
/// 之後必須呼叫 [`encode`] 重建校驗和與外層。
pub fn write_slot(plain: &[u8], slot: usize, replacement: &Slot) -> Result<Vec<u8>> {
    if plain.len() != FILE_SIZE {
        return Err(SaveError::InvalidPlainSize);
    }
    if replacement.roster.len() != ROSTER_SIZE {
        return Err(SaveError::RosterSize(replacement.roster.len()));
    }
    if replacement.metadata.len() != METADATA_SIZE {
        return Err(SaveError::MetadataSize(replacement.metadata.len()));
    }
    let (start, end) = slot_bounds(slot)?;
    let mut out = plain.to_vec();
    out[start..start + ROSTER_SIZE].copy_from_slice(&replacement.roster);
    out[start + ROSTER_SIZE..end].copy_from_slice(&replacement.metadata);
    Ok(out)
}

/// 只公開寫入端與讀取端都已閉合的欄位。原版 `0x30012` 寫入、`0x2602c` 還原 +0..+9；
/// 最後三個位元組沿用位址名稱，+10..+39 維持不透明。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct VerifiedMetadata {
    pub chapter: u8,
    pub roster_count: u8,
    pub currency: u32,
    pub hud_gate_a: u8,
    pub raw_53af9: u8,
    pub raw_51e61: u8,
    pub raw_51e62: u8,
}

pub fn read_verified_metadata(plain: &[u8], slot: usize) -> Result<VerifiedMetadata> {
    let s = read_slot(plain, slot)?;
    let m = &s.metadata;
    Ok(VerifiedMetadata {
        chapter: m[0],
        roster_count: m[1],
        currency: le_u32(&m[2..6]),
        hud_gate_a: m[6],
        raw_53af9: m[7],
        raw_51e61: m[8],
        raw_51e62: m[9],
    })
}

/// 一筆完整的原版 0x50 位元組名冊紀錄。個別偏移須在寫入端與消費端都證實後才解碼；
/// 保留整筆是因為原版 `0x2604a` 會先複製全部 32 筆，再載入選定槽的後設資料。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PersistentRecord {
    pub raw: [u8; UNIT_SIZE],
}

impl Default for PersistentRecord {
    fn default() -> Self {
        PersistentRecord { raw: [0; UNIT_SIZE] }
    }
}

/// 一格原版道具欄。旗標維持原始值，只有個別消費端（例如裝備的 0x40 位元）已閉合。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct PersistentInventoryCell {
    pub flags: u8,
    pub item_id: u8,
}

/// 只公開有直接建構與介面消費端的偏移。`raw_presentation_key` 刻意不稱為頭像、
/// 地圖槽或角色身分；`raw_identity` 另有 +8 名冊查詢契約。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct PersistentRecordView {
    pub raw_byte5: u8,
    pub raw_camp: u8,
    pub raw_presentation_key: u8,
    pub raw_identity: u8,
    pub inventory: [PersistentInventoryCell; 8],
    pub command_mask: [u8; 5],
    pub race: u8,
    pub class: u8,
    pub level: u8,
    pub transient: [u8; 6],
    pub raw_byte34: u8,
    pub raw_byte35: u8,
    pub raw_byte36: u8,
    pub base_ap: i16,
    pub base_dp: i16,
    pub movement: u8,
    pub experience: u8,
    pub dx: i16,
    pub hp: i16,
    pub max_hp: i16,
    pub mp: i16,
    pub max_mp: i16,
    pub ap: i16,
    pub dp: i16,
    pub hit: i16,
    pub ev: i16,
}

impl PersistentRecord {
    /// 已證實欄位的唯讀投影；原始位元組陣列維持權威，不會被修改。
    pub fn view(&self) -> PersistentRecordView {
        let r = &self.raw;
        let i16_at = |offset: usize| le_u16(&r[offset..]) as i16;
        let mut view = PersistentRecordView {
            raw_byte5: r[5],
            raw_camp: r[6],
            raw_presentation_key: r[7],
            raw_identity: r[8],
            race: r[0x1f],
            class: r[0x20],
            level: r[0x21],
            raw_byte34: r[0x34],
            raw_byte35: r[0x35],
            raw_byte36: r[0x36],
            base_ap: i16_at(0x37),
            base_dp: i16_at(0x39),
            movement: r[0x3b],
            experience: r[0x3c],
            dx: i16_at(0x3e),
            hp: i16_at(0x40),
            max_hp: i16_at(0x42),
            mp: i16_at(0x44),
            max_mp: i16_at(0x46),
            ap: i16_at(0x48),
            dp: i16_at(0x4a),
            hit: i16_at(0x4c),
            ev: i16_at(0x4e),
            ..Default::default()
        };
        for (slot, cell) in view.inventory.iter_mut().enumerate() {
            let offset = 0x0a + slot * 2;
            *cell = PersistentInventoryCell { flags: r[offset], item_id: r[offset + 1] };
        }
        view.command_mask.copy_from_slice(&r[0x1a..0x1f]);
        view.transient.copy_from_slice(&r[0x22..0x28]);
        view
    }
}

/// 一個原版章節槽的安全匯入邊界，對應 `0x2602c..0x26098` 複製的選定紀錄。
/// `roster_count` 的容量檢查是重製端的安全不變式，原版載入路徑沒有這個檢查。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ChapterSlotSnapshot {
    pub slot: usize,
    pub verified: VerifiedMetadata,
    pub metadata: [u8; METADATA_SIZE],
    pub records: [PersistentRecord; ROSTER_UNITS],
    /// 完整來源出處；不寫入原版 SAV。
    pub source_save_sha256: String,
}

/// 驗證並複製一個非空的原版章節槽。不把紀錄轉成戰鬥單位，因此不能單獨用來宣稱
/// 原版戰役還原成功。
pub fn inspect_chapter_slot(plain: &[u8], slot: usize) -> Result<ChapterSlotSnapshot> {
    let raw = read_slot(plain, slot)?;
    let verified = read_verified_metadata(plain, slot)?;
    if verified.chapter == 0xff {
        return Err(SaveError::EmptyChapterSlot);
    }
    if verified.roster_count as usize > ROSTER_UNITS {
        return Err(SaveError::RosterCountExceeds(verified.roster_count));
    }
    let mut metadata = [0u8; METADATA_SIZE];
    metadata.copy_from_slice(&raw.metadata);
    let mut records = [PersistentRecord::default(); ROSTER_UNITS];
    for (index, record) in records.iter_mut().enumerate() {
        let start = index * UNIT_SIZE;
        record.raw.copy_from_slice(&raw.roster[start..start + UNIT_SIZE]);
    }
    Ok(ChapterSlotSnapshot { slot, verified, metadata, records, source_save_sha256: sha256_hex(&xor_envelope(plain)) })
}

impl ChapterSlotSnapshot {
    /// 原版後設資料選定的前段紀錄複本；其餘固定容量紀錄仍保存在 `records`。
    pub fn active_records(&self) -> Vec<PersistentRecord> {
        self.records.get(..self.verified.roster_count as usize).map(<[_]>::to_vec).unwrap_or_default()
    }
}

/// 原版 `0x10010` 還原的 18 位元組標頭。回合數與兩個數量欄由 `[0x53bef]`、`[0x53beb]`、
/// `[0x53bfb]` 的直接寫入證實；尾端四個選項位元組除 `hud_gate_a` 外維持原始值。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct CurrentRuntimeHeader {
    pub turn_counter: u8,
    pub runtime_count: u8,
    pub chapter: u8,
    pub camera_x: u8,
    pub camera_y: u8,
    pub cursor_x: u8,
    pub cursor_y: u8,
    pub visible_cursor_x: u8,
    pub visible_cursor_y: u8,
    pub persistent_count: u8,
    pub currency: u32,
    pub raw_53af9: u8,
    pub hud_gate_a: u8,
    pub raw_51e61: u8,
    pub raw_51e62: u8,
}

/// `0x10010` 消費的每個 FD2.SAV 直接複製區。`runtime_records` 依數量截斷；
/// `persistent_records` 保留原版完整容量，因載入器固定複製 0xa00 位元組。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CurrentSnapshot {
    pub native_field_control: [u8; CURRENT_FIELD_CONTROL_SIZE],
    pub header: CurrentRuntimeHeader,
    pub persistent_records: [PersistentRecord; ROSTER_UNITS],
    pub runtime_records: Vec<PersistentRecord>,
    pub native_event_state: [u8; CURRENT_NATIVE_EVENT_STATE_SIZE],
}

impl Default for CurrentSnapshot {
    fn default() -> Self {
        CurrentSnapshot {
            native_field_control: [0; CURRENT_FIELD_CONTROL_SIZE],
            header: CurrentRuntimeHeader::default(),
            persistent_records: [PersistentRecord::default(); ROSTER_UNITS],
            runtime_records: Vec::new(),
            native_event_state: [0; CURRENT_NATIVE_EVENT_STATE_SIZE],
        }
    }
}

impl CurrentSnapshot {
    /// 標頭數量界定的目前持久名冊複本。
    pub fn active_persistent_records(&self) -> Vec<PersistentRecord> {
        self.persistent_records.get(..self.header.persistent_count as usize).map(<[_]>::to_vec).unwrap_or_default()
    }
}

/// 驗證戰鬥中快照所需的外部資源身分。`selector_group_count` 是 FDICON.B24 圖格數除以 12。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ContinueRuntimeContext {
    pub chapter: i64,
    pub field_width: i64,
    pub field_height: i64,
    pub selector_group_count: i64,
    /// 原版 main 在 `0x25d83..0x25d8b`、標題選單之前取樣的有號 BIOS 計時器低位 word。
    /// 不存在 FD2.SAV 內，標題 CONTINUE 的呼叫端必須明確提供。
    pub title_timer_tick: i64,
    pub has_title_timer_tick: bool,
}

/// 一筆執行期紀錄，連同 `0x1036a..0x10395` 由紀錄 +7 重建的快取槽。存檔的 +2 位元組
/// 刻意不採信，原版 CONTINUE 在複製後會覆寫它。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ContinueRuntimeRecord {
    pub raw: PersistentRecord,
    pub selector_key: u8,
    pub selector_slot: u8,
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ContinueTurnEventControl {
    pub turn: u8,
    pub event_id: u8,
    pub raw_camp: u8,
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ContinueFieldEventControl {
    pub event_id: u8,
    pub selector: u8,
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ContinueChestControl {
    pub raw_type: u8,
    pub value: u16,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ContinueUnitControl {
    pub raw: [u8; CURRENT_FIELD_CONTROL_UNIT_SIZE],
}

/// 原版消費的 FDFIELD 控制資源固定版面。`raw_unit_count` 是排他上限：`0x10bcc` 在讀列前
/// 比較 index >= `[0x53be3]`。超過數量的資源填充仍留在 `native_field_control`，不公開為存活單位。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ContinueFieldControlView {
    pub raw_map_selector: u8,
    pub raw_own_deploy_count: u8,
    pub raw_unit_count: u8,
    pub turn_events: [ContinueTurnEventControl; 16],
    pub field_events: [ContinueFieldEventControl; 16],
    pub chests: [ContinueChestControl; 16],
    pub units: Vec<ContinueUnitControl>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ContinueRuntimeOwner {
    /// `0x10C50→0x1B750` 建構交易與精確靜態排程已閉合；CONTINUE 仍缺處理器改寫的回合槽
    /// 與群組公式的擁有者。
    PendingGroupBinding,
    /// 原版 main 在 `sub_10010` 返回後進入 `0x117E7`；剩下的擁有者是重製端的原子發布。
    BattleControllerHandoff,
}

impl ContinueRuntimeOwner {
    pub fn as_str(self) -> &'static str {
        match self {
            ContinueRuntimeOwner::PendingGroupBinding => "pending_group_binding",
            ContinueRuntimeOwner::BattleControllerHandoff => "battle_controller_handoff",
        }
    }
}

/// 標題選單 CONTINUE 呼叫端 `0x26130` 與 `sub_10010` 在最後 `0x4e031` 鍵盤 word 複製前
/// 建立的地圖呈現狀態。不描述戰鬥中另一個呼叫端 `0x1a251`。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ContinueMapPresentation {
    pub opening_range_mode: i64,
    pub interactive_range_mode: i64,
    pub hud_gate_b: u8,
    pub hud_anchor_x: i64,
}

/// `sub_10010` 第一次 `0x11cac` 重繪前的行程全域計時狀態。除 `sprite_last_timer_tick`
/// 取自 `title_timer_tick` 外都來自歸零的執行檔資料映像。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ContinueMapTimingSeed {
    pub sprite_idle_cycle: i64,
    pub sprite_moving_cycle: i64,
    pub sprite_last_timer_tick: i64,
    pub terrain_phase: i64,
    pub terrain_phase_last_timer_tick: i64,
    pub terrain_phase_override: i64,
    pub terrain_flip: i64,
    pub terrain_flip_last_timer_tick: i64,
    pub unit_pixel_shift: i64,
    pub unit_pixel_shift_last_timer_tick: i64,
}

/// 深複製的唯讀前置檢查結果。只證明存檔與資源邊界及選擇器重建；`unresolved_owners`
/// 讓正式 CONTINUE 閘門保持關閉，直到每個原版擁有者都有獨立證據與嚴格轉接。
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ContinueRuntimeInput {
    pub context: ContinueRuntimeContext,
    pub header: CurrentRuntimeHeader,
    pub native_field_control: [u8; CURRENT_FIELD_CONTROL_SIZE],
    pub field_control: ContinueFieldControlView,
    pub persistent_records: [PersistentRecord; ROSTER_UNITS],
    pub runtime_records: Vec<ContinueRuntimeRecord>,
    pub native_event_state: [u8; CURRENT_NATIVE_EVENT_STATE_SIZE],
    pub map_presentation: ContinueMapPresentation,
    pub map_timing_seed: ContinueMapTimingSeed,
    pub unresolved_owners: Vec<ContinueRuntimeOwner>,
    validated: bool,
}

impl ContinueRuntimeInput {
    pub fn ready_for_continue(&self) -> bool {
        self.unresolved_owners.is_empty()
    }

    /// 拒絕手動組裝的值，也拒絕 [`build_continue_runtime_input`] 回傳後被改過的公開欄位：
    /// 由來源快照重跑完整前置檢查，要求型別化結果完全相同。
    pub fn validated_for_runtime_bridge(&self) -> bool {
        if !self.validated {
            return false;
        }
        let snapshot = CurrentSnapshot {
            native_field_control: self.native_field_control,
            header: self.header,
            persistent_records: self.persistent_records,
            runtime_records: self.runtime_records.iter().map(|r| r.raw).collect(),
            native_event_state: self.native_event_state,
        };
        matches!(build_continue_runtime_input(&snapshot, self.context), Ok(rebuilt) if rebuilt == *self)
    }
}

/// 驗證資源側契約已閉合的每個目前快照欄位。不改動戰鬥狀態，也不虛構仍未解的擁有者。
pub fn build_continue_runtime_input(
    snapshot: &CurrentSnapshot,
    context: ContinueRuntimeContext,
) -> Result<ContinueRuntimeInput> {
    let header = snapshot.header;
    if context.chapter < 0 || context.chapter >= 30 || header.chapter as i64 != context.chapter {
        return Err(SaveError::Rejected("fdsave: CONTINUE chapter/resource mismatch"));
    }
    if context.field_width < 13
        || context.field_height < 8
        || context.field_width > 0x100
        || context.field_height > 0x100
    {
        return Err(SaveError::Rejected("fdsave: CONTINUE field dimensions are invalid"));
    }
    if context.selector_group_count <= 0 || context.selector_group_count > 0x100 {
        return Err(SaveError::Rejected("fdsave: CONTINUE FDICON group count is invalid"));
    }
    if !context.has_title_timer_tick || context.title_timer_tick < -0x8000 || context.title_timer_tick > 0x7fff {
        return Err(SaveError::Rejected("fdsave: CONTINUE title timer seed is invalid"));
    }
    let runtime_count = header.runtime_count as usize;
    if runtime_count > ROSTER_UNITS * 3 || snapshot.runtime_records.len() != runtime_count {
        return Err(SaveError::Rejected("fdsave: CONTINUE runtime record count mismatch"));
    }
    if header.persistent_count as usize > ROSTER_UNITS {
        return Err(SaveError::Rejected("fdsave: CONTINUE persistent record count exceeds capacity"));
    }
    if snapshot.native_field_control[2] as usize > CURRENT_FIELD_CONTROL_UNIT_CAP {
        return Err(SaveError::Rejected("fdsave: CONTINUE FDFIELD unit count exceeds control capacity"));
    }
    let (w, h) = (context.field_width, context.field_height);
    if header.camera_x as i64 > w - 13
        || header.camera_y as i64 > h - 8
        || header.cursor_x as i64 >= w
        || header.cursor_y as i64 >= h
        || header.visible_cursor_x as i64 != header.cursor_x as i64 - header.camera_x as i64
        || header.visible_cursor_y as i64 != header.cursor_y as i64 - header.camera_y as i64
    {
        return Err(SaveError::Rejected("fdsave: CONTINUE map view violates field bounds or cursor identity"));
    }

    let mut runtime = Vec::with_capacity(runtime_count);
    let mut slots: Vec<(u8, u8)> = Vec::new();
    for (index, record) in snapshot.runtime_records.iter().enumerate() {
        let key = record.raw[7];
        if key as i64 >= context.selector_group_count {
            return Err(SaveError::ContinueRuntimeKey { index, key });
        }
        let slot = match slots.iter().find(|(k, _)| *k == key) {
            Some(&(_, slot)) => slot,
            None => {
                let slot = slots.len() as u8;
                slots.push((key, slot));
                slot
            }
        };
        if record.raw[5] & 1 == 0
            && (record.raw[0] as i64 >= w || record.raw[1] as i64 >= h || record.raw[3] > 3 || record.raw[4] > 6)
        {
            return Err(SaveError::ContinueActivePresentation(index));
        }
        runtime.push(ContinueRuntimeRecord { raw: *record, selector_key: key, selector_slot: slot });
    }

    Ok(ContinueRuntimeInput {
        context,
        header,
        native_field_control: snapshot.native_field_control,
        field_control: decode_continue_field_control(&snapshot.native_field_control),
        persistent_records: snapshot.persistent_records,
        runtime_records: runtime,
        native_event_state: snapshot.native_event_state,
        map_presentation: ContinueMapPresentation {
            opening_range_mode: 0,
            interactive_range_mode: 1,
            hud_gate_b: 1,
            hud_anchor_x: continue_title_hud_anchor(header.visible_cursor_x as i64, header.visible_cursor_y as i64),
        },
        map_timing_seed: ContinueMapTimingSeed {
            sprite_last_timer_tick: context.title_timer_tick,
            terrain_phase_override: -1,
            ..Default::default()
        },
        unresolved_owners: vec![
            ContinueRuntimeOwner::PendingGroupBinding,
            ContinueRuntimeOwner::BattleControllerHandoff,
        ],
        validated: true,
    })
}

pub fn decode_continue_field_control(raw: &[u8; CURRENT_FIELD_CONTROL_SIZE]) -> ContinueFieldControlView {
    const TURN_OFFSET: usize = 3;
    const FIELD_OFFSET: usize = TURN_OFFSET + 16 * 3;
    const CHEST_OFFSET: usize = FIELD_OFFSET + 16 * 2;
    let mut view = ContinueFieldControlView {
        raw_map_selector: raw[0],
        raw_own_deploy_count: raw[1],
        raw_unit_count: raw[2],
        turn_events: [ContinueTurnEventControl::default(); 16],
        field_events: [ContinueFieldEventControl::default(); 16],
        chests: [ContinueChestControl::default(); 16],
        units: Vec::with_capacity(raw[2] as usize),
    };
    for (index, event) in view.turn_events.iter_mut().enumerate() {
        let o = TURN_OFFSET + index * 3;
        *event = ContinueTurnEventControl { turn: raw[o], event_id: raw[o + 1], raw_camp: raw[o + 2] };
    }
    for (index, event) in view.field_events.iter_mut().enumerate() {
        let o = FIELD_OFFSET + index * 2;
        *event = ContinueFieldEventControl { event_id: raw[o], selector: raw[o + 1] };
    }
    for (index, chest) in view.chests.iter_mut().enumerate() {
        let o = CHEST_OFFSET + index * 3;
        *chest = ContinueChestControl { raw_type: raw[o], value: le_u16(&raw[o + 1..o + 3]) };
    }
    for index in 0..raw[2] as usize {
        let o = CURRENT_FIELD_CONTROL_UNIT_OFFSET + index * CURRENT_FIELD_CONTROL_UNIT_SIZE;
        let mut unit = ContinueUnitControl { raw: [0; CURRENT_FIELD_CONTROL_UNIT_SIZE] };
        unit.raw.copy_from_slice(&raw[o..o + CURRENT_FIELD_CONTROL_UNIT_SIZE]);
        view.units.push(unit);
    }
    view
}

/// 由執行檔資料映像值 1 出發，套用 `0x1acf3` 僅有的兩個寫入。參數順序依存檔標頭 (X, Y)，
/// 原版比較順序是 `[0x53abd]` Y 再 `[0x53ab9]` X。
fn continue_title_hud_anchor(visible_x: i64, visible_y: i64) -> i64 {
    if visible_y > 5 {
        if visible_x < 3 {
            return 0xf2;
        }
        if visible_x > 9 {
            return 1;
        }
    }
    1
}

/// 只解碼標題 CONTINUE 分支使用、已證實的明文版面；與四個章節槽分開。
pub fn inspect_current_snapshot(plain: &[u8]) -> Result<CurrentSnapshot> {
    if plain.len() != FILE_SIZE {
        return Err(SaveError::InvalidPlainSize);
    }
    let h = &plain[CURRENT_RUNTIME_HEADER_OFFSET..CURRENT_RUNTIME_HEADER_OFFSET + CURRENT_RUNTIME_HEADER_SIZE];
    if h[1] as usize > ROSTER_UNITS * 3 {
        return Err(SaveError::RuntimeCountExceeds(h[1]));
    }
    if h[9] as usize > ROSTER_UNITS {
        return Err(SaveError::PersistentCountExceeds(h[9]));
    }
    let mut snapshot = CurrentSnapshot {
        header: CurrentRuntimeHeader {
            turn_counter: h[0],
            runtime_count: h[1],
            chapter: h[2],
            camera_x: h[3],
            camera_y: h[4],
            cursor_x: h[5],
            cursor_y: h[6],
            visible_cursor_x: h[7],
            visible_cursor_y: h[8],
            persistent_count: h[9],
            currency: le_u32(&h[10..14]),
            raw_53af9: h[14],
            hud_gate_a: h[15],
            raw_51e61: h[16],
            raw_51e62: h[17],
        },
        ..Default::default()
    };
    snapshot.native_field_control.copy_from_slice(
        &plain[CURRENT_FIELD_CONTROL_OFFSET..CURRENT_FIELD_CONTROL_OFFSET + CURRENT_FIELD_CONTROL_SIZE],
    );
    for (index, record) in snapshot.persistent_records.iter_mut().enumerate() {
        let start = CURRENT_PERSISTENT_ROSTER_OFFSET + index * UNIT_SIZE;
        record.raw.copy_from_slice(&plain[start..start + UNIT_SIZE]);
    }
    for index in 0..snapshot.header.runtime_count as usize {
        let start = CURRENT_RUNTIME_ROSTER_OFFSET + index * UNIT_SIZE;
        let mut record = PersistentRecord::default();
        record.raw.copy_from_slice(&plain[start..start + UNIT_SIZE]);
        snapshot.runtime_records.push(record);
    }
    snapshot.native_event_state.copy_from_slice(
        &plain[CURRENT_NATIVE_EVENT_STATE_OFFSET..CURRENT_NATIVE_EVENT_STATE_OFFSET + CURRENT_NATIVE_EVENT_STATE_SIZE],
    );
    Ok(snapshot)
}

/// 只取代原版 `sub_19DF7` selector 1 寫入的目前執行期區域；章節槽與其他不透明位元組
/// 沿用呼叫端映像。之後必須呼叫 [`encode`]。只接受原始紀錄，不把正規化的戰役或戰鬥值
/// 序列化進原版 ABI。
pub fn write_current_snapshot(plain: &[u8], snapshot: &CurrentSnapshot) -> Result<Vec<u8>> {
    if plain.len() != FILE_SIZE {
        return Err(SaveError::InvalidPlainSize);
    }
    let runtime_count = snapshot.header.runtime_count as usize;
    if runtime_count > ROSTER_UNITS * 3 || snapshot.runtime_records.len() != runtime_count {
        return Err(SaveError::Rejected("fdsave: current snapshot runtime record count mismatch"));
    }
    if snapshot.header.persistent_count as usize > ROSTER_UNITS {
        return Err(SaveError::Rejected("fdsave: current snapshot persistent record count exceeds capacity"));
    }
    if snapshot.native_field_control[2] as usize > CURRENT_FIELD_CONTROL_UNIT_CAP {
        return Err(SaveError::Rejected("fdsave: current snapshot field unit count exceeds capacity"));
    }
    let mut out = plain.to_vec();
    out[CURRENT_FIELD_CONTROL_OFFSET..CURRENT_FIELD_CONTROL_OFFSET + CURRENT_FIELD_CONTROL_SIZE]
        .copy_from_slice(&snapshot.native_field_control);
    for (index, record) in snapshot.persistent_records.iter().enumerate() {
        let start = CURRENT_PERSISTENT_ROSTER_OFFSET + index * UNIT_SIZE;
        out[start..start + UNIT_SIZE].copy_from_slice(&record.raw);
    }
    for (index, record) in snapshot.runtime_records.iter().enumerate() {
        let start = CURRENT_RUNTIME_ROSTER_OFFSET + index * UNIT_SIZE;
        out[start..start + UNIT_SIZE].copy_from_slice(&record.raw);
    }
    out[CURRENT_NATIVE_EVENT_STATE_OFFSET..CURRENT_NATIVE_EVENT_STATE_OFFSET + CURRENT_NATIVE_EVENT_STATE_SIZE]
        .copy_from_slice(&snapshot.native_event_state);
    let s = &snapshot.header;
    let h = &mut out[CURRENT_RUNTIME_HEADER_OFFSET..CURRENT_RUNTIME_HEADER_OFFSET + CURRENT_RUNTIME_HEADER_SIZE];
    h[..10].copy_from_slice(&[
        s.turn_counter,
        s.runtime_count,
        s.chapter,
        s.camera_x,
        s.camera_y,
        s.cursor_x,
        s.cursor_y,
        s.visible_cursor_x,
        s.visible_cursor_y,
        s.persistent_count,
    ]);
    h[10..14].copy_from_slice(&s.currency.to_le_bytes());
    h[14..18].copy_from_slice(&[s.raw_53af9, s.hud_gate_a, s.raw_51e61, s.raw_51e62]);
    Ok(out)
}

#[cfg(test)]
mod tests;

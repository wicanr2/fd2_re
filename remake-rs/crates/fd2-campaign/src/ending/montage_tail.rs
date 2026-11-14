//! Go 來源：`remake/internal/ending/montage_tail.go`（651 行）。

use super::*;

/// MontageTail is the raw, editable schedule after the party montage.  It
/// deliberately exposes byte offsets and native call shapes instead of
/// assigning names such as unit state, animation, or ending outcome.
///
/// Go：`montage_tail.go:19`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageTail {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"native_handler"`
    pub native_handler: String,
    /// Go 標籤：`json:"status"`
    pub status: String,
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"resources"`
    pub resources: Vec<MontageTailAsset>,
    /// Go 標籤：`json:"raw_tables"`
    pub raw_tables: MontageTailRawTable,
    /// Go 標籤：`json:"loop"`
    pub r#loop: MontageTailLoop,
    /// Go 標籤：`json:"gate"`
    pub gate: MontageTailGate,
}

/// Go：`montage_tail.go:30`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageTailAsset {
    /// Go 標籤：`json:"archive"`
    pub archive: String,
    /// Go 標籤：`json:"index"`
    pub index: i64,
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"role"`
    pub role: String,
}

/// Go：`montage_tail.go:37`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageTailRawTable {
    /// Record0Byte7, Record1Byte7 and Global540FF retain the three raw
    /// 20-byte tables at 0x525dc..0x52617. The byte-6 writes are not a fourth
    /// table: native code derives each from its paired byte-7 value.
    /// Go 標籤：`json:"record0_byte_7"`
    pub record0_byte7: Vec<i64>,
    /// Go 標籤：`json:"record1_byte_7"`
    pub record1_byte7: Vec<i64>,
    /// Go 標籤：`json:"global_540ff"`
    pub global540_ff: Vec<i64>,
}

/// Go：`montage_tail.go:46`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageTailLoop {
    /// Go 標籤：`json:"count"`
    pub count: i64,
    /// Go 標籤：`json:"runtime_records_global"`
    pub runtime_records_global: String,
    /// Go 標籤：`json:"runtime_record_stride"`
    pub runtime_record_stride: i64,
    /// Go 標籤：`json:"record0_byte_6_formula"`
    pub record0_byte6_formula: String,
    /// Go 標籤：`json:"record0_byte_7_source"`
    pub record0_byte7_source: String,
    /// Go 標籤：`json:"record1_byte_6_formula"`
    pub record1_byte6_formula: String,
    /// Go 標籤：`json:"record1_byte_7_source"`
    pub record1_byte7_source: String,
    /// Go 標籤：`json:"global_540ff_source"`
    pub global540_ff_source: String,
    /// Go 標籤：`json:"renderer_call"`
    pub renderer_call: String,
    /// Go 標籤：`json:"palette_call"`
    pub palette_call: String,
    /// Go 標籤：`json:"frame_call"`
    pub frame_call: String,
    /// Go 標籤：`json:"wait_before_frame_ticks"`
    pub wait_before_frame_ticks: i64,
    /// Go 標籤：`json:"wait_after_frame_ticks"`
    pub wait_after_frame_ticks: i64,
    /// Go 標籤：`json:"wait_helper"`
    pub wait_helper: String,
    /// Go 標籤：`json:"present_helper"`
    pub present_helper: String,
    /// Go 標籤：`json:"restore_helper"`
    pub restore_helper: String,
}

/// Go：`montage_tail.go:65`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageTailGate {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"reason"`
    pub reason: String,
}

/// MontageTailEntry keeps the three raw table bytes together for one native
/// loop index. It is a plan only: no renderer, runtime mutation, or campaign
/// transition is performed by this package.
///
/// Go：`montage_tail.go:73`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct MontageTailEntry {
    pub index: i64,
    pub global540_ff: u8,
    pub record0_byte6: u8,
    pub record0_byte7: u8,
    pub record1_byte6: u8,
    pub record1_byte7: u8,
}

/// MontageTailVisualResources records only the directly proven archive
/// selectors consumed by the nonzero 0x28a6c branch. It is not a decoded
/// animation, does not assign attacker/defender roles, and performs no runtime
/// mutation or drawing.
///
/// Go：`montage_tail.go:86`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct MontageTailVisualResources {
    pub index: i64,
    pub tai: i64,
    pub bg: i64,
    pub record1_figani_base: i64,
    pub record1_figani_aux: i64,
    pub record0_figani_base: i64,
    pub record0_figani_aux: i64,
}

/// 0x2c435 passes this raw selector to 0x1088d before the party montage.
/// These constants deliberately retain archive arithmetic instead of naming
/// selector 0x1e as a chapter or a character list.
pub(crate) const NATIVE_MONTAGE_TAIL_LOAD_SELECTOR: i64 = 0x1e;

/// Go 原式：`nativeMontageTailLoadSelector * 3`
pub(crate) const NATIVE_MONTAGE_TAIL_FIELD_MAP_RESOURCE: i64 = 90;

/// Go 原式：`nativeMontageTailFieldMapResource + 1`
pub(crate) const NATIVE_MONTAGE_TAIL_CONTROL_RESOURCE: i64 = 91;

/// Go 原式：`nativeMontageTailFieldMapResource + 2`
pub(crate) const NATIVE_MONTAGE_TAIL_POSITIONS_RESOURCE: i64 = 92;

pub(crate) const NATIVE_MONTAGE_TAIL_DEPLOY_RECORD_COUNT: i64 = 0x1f;

pub(crate) const NATIVE_MONTAGE_TAIL_CONTROL_GROUP_OFFSET: i64 = 21;

/// MontageTailLoaderPaths identifies the separated FDFIELD selector document
/// and FDICON bank consumed by the record-construction portion of 0x1088d(0x1e).
///
/// Go：`montage_tail.go:110`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageTailLoaderPaths {
    pub fdfield_root: String,
    pub fdicon_root: String,
}

/// MontageTailVisualPaths identifies the separated visual roots needed by the
/// source-bound bridge. It validates every selector and never falls back to
/// player archives.
///
/// Go：`montage_tail.go:118`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageTailVisualPaths {
    pub surface_root: String,
    pub animation_root: String,
}

/// MontageTailAssetPaths identifies the separated FDOTHER resources consumed
/// by 0x2c194's typed tail preflight. No archive fallback is permitted.
///
/// Go：`montage_tail.go:125`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageTailAssetPaths {
    pub surface_root: String,
    pub palette_root: String,
    pub animation_root: String,
}

/// MontageTailLoaderBaseline is the exact raw-runtime shape constructed by the
/// 0x1088d(0x1e) party-slot loop: 31 deployment records, with the active
/// persistent prefix copied at stride 0x50 and the remainder marked inactive.
///
/// It is deliberately *not* an admitted 0x28a6c call-time input.  The later
/// 0x2c548 montage can observe and potentially mutate the same runtime image;
/// no original trace yet proves its final full-record state at 0x2c2a6.  A
/// future tail renderer must require a separately proven call-time snapshot,
/// rather than treating this loader baseline as sufficient evidence.
///
/// Go：`montage_tail.go:140`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageTailLoaderBaseline {
    pub(crate) runtime_records: Vec<[u8; 80]>,
}

impl MontageTailLoaderBaseline {
    /// RuntimeCount reports the native deployment-slot count, including inactive
    /// slots. It does not expose a mutable backing slice.
    ///
    /// Go：`montage_tail.go:146`，`func (b MontageTailLoaderBaseline) RuntimeCount() int`
    pub fn runtime_count(&self) -> i64 {
        self.runtime_records.len() as i64
    }

    /// RuntimeRecord returns a value copy of one post-0x1088d deployment record.
    ///
    /// Go：`montage_tail.go:151`，`func (b MontageTailLoaderBaseline) RuntimeRecord(index int) ([fdsave.UnitSize]byte, error)`
    pub fn runtime_record(&self, index: i64) -> Result<[u8; 80]> {
        todo!("ending/montage_tail.go:MontageTailLoaderBaseline.RuntimeRecord")
    }

    /// FirstPair returns the first two post-loader records as value copies.  This
    /// convenience function does not upgrade the pair to a renderer admission;
    /// see MontageTailLoaderBaseline for the unresolved 0x2c548 boundary.
    ///
    /// Go：`montage_tail.go:161`，`func (b MontageTailLoaderBaseline) FirstPair() ([2][fdsave.UnitSize]byte, error)`
    pub fn first_pair(&self) -> Result<[[u8; 80]; 2]> {
        todo!("ending/montage_tail.go:MontageTailLoaderBaseline.FirstPair")
    }
}

/// BuildMontageTailLoaderBaseline reproduces only the proven raw record writes
/// in 0x1088d's selector-0x1e own-deployment loop.  It validates the complete
/// FDFIELD resource triplet needed by that fixed selector, validates FDICON
/// keys against the player archive, copies the complete persistent records,
/// performs the documented overwrites, and reuses the independently verified
/// 0x1b750 equipment tail.  It deliberately stops before 0x2c548 and does not
/// create a tail renderer or campaign transition.
///
/// Go：`montage_tail.go:175`，`func BuildMontageTailLoaderBaseline( tail MontageTail, persistent []fdsave.PersistentRecord, paths MontageTailLoaderPaths, itemTable []byte, ) (MontageTailLoaderBaseline, error)`
pub fn build_montage_tail_loader_baseline(
    tail: &MontageTail,
    persistent: &[save::PersistentRecord],
    paths: &MontageTailLoaderPaths,
    item_table: &[u8],
) -> Result<MontageTailLoaderBaseline> {
    todo!("ending/montage_tail.go:BuildMontageTailLoaderBaseline")
}

/// Go：`montage_tail.go:253`，`func validateMontageTailLoaderField(fieldMap, control, positions []byte) (int, error)`
pub(crate) fn validate_montage_tail_loader_field(field_map: &[u8], control: &[u8], positions: &[u8]) -> Result<i64> {
    todo!("ending/montage_tail.go:validateMontageTailLoaderField")
}

/// MontageTailAssets preserves the directly loaded FDOTHER inputs of
/// 0x2c194. The original nonzero 0x28a6c branch is proven to load and compose
/// visual resources, but the remake lacks its admitted full call-time runtime
/// records and exact renderer adapter. Merely loading these assets therefore
/// does not claim to reproduce that loop. The final #59 frame is kept separately
/// because the native code decodes it only after the loop has completed.
///
/// Go：`montage_tail.go:300`
#[derive(Debug, Clone, Default)]
pub struct MontageTailAssets {
    pub loop_palette: Vec<u8>,
    pub intro: fdother::Frame,
    pub loop_frames: Vec<fdother::Frame>,
    pub r#final: fdother::Frame,
}

/// MontageTailVisualSet is one preflighted 0x28a6c resource transaction. The
/// field names retain raw record ownership rather than assigning character or
/// attacker/defender identities to finale records 0 and 1.
///
/// Go：`montage_tail.go:310`
#[derive(Debug, Clone, Default)]
pub struct MontageTailVisualSet {
    pub plan: MontageTailVisualResources,
    pub tai: fdother::Frame,
    pub bg: fdother::Frame,
    pub record1_figani_base: Option<figani::Animation>,
    pub record1_figani_aux: Option<figani::Animation>,
    pub record0_figani_base: Option<figani::Animation>,
    pub record0_figani_aux: Option<figani::Animation>,
}

/// LoadMontageTailVisualSets performs an all-or-nothing preflight of the 20
/// original TAI/BG/FIGANI selector transactions. The player additionally
/// validates the actual header branch before admitting 0x2939d composition;
/// archive availability alone still does not prove DOS timing or sound parity.
///
/// Go：`montage_tail.go:324`，`func LoadMontageTailVisualSets(tail MontageTail, paths MontageTailVisualPaths) ([]MontageTailVisualSet, error)`
pub fn load_montage_tail_visual_sets(
    tail: &MontageTail,
    paths: &MontageTailVisualPaths,
) -> Result<Vec<MontageTailVisualSet>> {
    todo!("ending/montage_tail.go:LoadMontageTailVisualSets")
}

/// Go：`montage_tail.go:388`，`func validateMontageTailVisualFrame(frame fdother.Frame) error`
pub(crate) fn validate_montage_tail_visual_frame(frame: &fdother::Frame) -> Result<()> {
    todo!("ending/montage_tail.go:validateMontageTailVisualFrame")
}

/// LoadMontageTailAssetsArchive is retained for source-oracle tools and exact
/// archive-equivalence tests. Production callers use LoadMontageTailAssets.
///
/// Go：`montage_tail.go:401`，`func LoadMontageTailAssetsArchive(tail MontageTail, fdotherPath string) (MontageTailAssets, error)`
pub fn load_montage_tail_assets_archive(tail: &MontageTail, fdother_path: &str) -> Result<MontageTailAssets> {
    todo!("ending/montage_tail.go:LoadMontageTailAssetsArchive")
}

/// LoadMontageTailAssets verifies the separated native resource shapes before
/// the presentation adapter can use them. It intentionally does not invent a
/// meaning for the three per-entry raw bytes or the 0x28a6c rendering owner.
///
/// Go：`montage_tail.go:462`，`func LoadMontageTailAssets(tail MontageTail, paths MontageTailAssetPaths) (MontageTailAssets, error)`
pub fn load_montage_tail_assets(tail: &MontageTail, paths: &MontageTailAssetPaths) -> Result<MontageTailAssets> {
    todo!("ending/montage_tail.go:LoadMontageTailAssets")
}

impl MontageTailAssets {
    /// PresentFinal renders the final native frame and restores the immutable
    /// palette baseline that was captured by the preceding indexed ending.  The
    /// original reaches this image after a separate fade helper; this small
    /// adapter deliberately exposes only the stable terminal frame, not a claim
    /// of timing parity for that helper.
    ///
    /// Go：`montage_tail.go:524`，`func (assets MontageTailAssets) PresentFinal(compositor *IndexedCompositor) error`
    pub fn present_final(&self, compositor: Option<&mut IndexedCompositor>) -> Result<()> {
        todo!("ending/montage_tail.go:MontageTailAssets.PresentFinal")
    }
}

/// Go：`montage_tail.go:539`，`func LoadMontageTail(path string) (*MontageTail, error)`
pub fn load_montage_tail(path: &str) -> Result<MontageTail> {
    todo!("ending/montage_tail.go:LoadMontageTail")
}

impl MontageTail {
    /// Go：`montage_tail.go:554`，`func (tail MontageTail) validateNativeContract() error`
    pub(crate) fn validate_native_contract(&self) -> Result<()> {
        todo!("ending/montage_tail.go:MontageTail.validateNativeContract")
    }

    /// Plan returns the raw 20-entry schedule and the exact derived byte-6 values.
    /// It does not write a runtime record or present a frame.
    ///
    /// Go：`montage_tail.go:596`，`func (t MontageTail) Plan() ([]MontageTailEntry, error)`
    pub fn plan(&self) -> Result<Vec<MontageTailEntry>> {
        todo!("ending/montage_tail.go:MontageTail.Plan")
    }

    /// PlanVisualResources preserves the resource-index arithmetic directly used
    /// by the original nonzero 0x28a6c branch. The TAI#3 substitutions are raw
    /// selector comparisons, not named character exceptions. The source-bound E1
    /// player consumes this plan without mutating runtime records; an original E2
    /// claim still needs a separately admitted call-time record snapshot.
    ///
    /// Go：`montage_tail.go:622`，`func (t MontageTail) PlanVisualResources() ([]MontageTailVisualResources, error)`
    pub fn plan_visual_resources(&self) -> Result<Vec<MontageTailVisualResources>> {
        todo!("ending/montage_tail.go:MontageTail.PlanVisualResources")
    }
}

/// Go：`montage_tail.go:646`，`func tailRecordByte6(recordByte7 byte) byte`
pub(crate) fn tail_record_byte6(record_byte7: u8) -> u8 {
    todo!("ending/montage_tail.go:tailRecordByte6")
}

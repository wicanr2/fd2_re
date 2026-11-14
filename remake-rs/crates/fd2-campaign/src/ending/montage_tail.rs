//! Go 來源：`remake/internal/ending/montage_tail.go`（651 行）。

use super::*;

/// MontageTail is the raw, editable schedule after the party montage.  It
/// deliberately exposes byte offsets and native call shapes instead of
/// assigning names such as unit state, animation, or ending outcome.
///
/// Go：`montage_tail.go:19`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
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
    #[serde(deserialize_with = "battle::null_as_default")]
    pub resources: Vec<MontageTailAsset>,
    /// Go 標籤：`json:"raw_tables"`
    pub raw_tables: MontageTailRawTable,
    /// Go 標籤：`json:"loop"`
    pub r#loop: MontageTailLoop,
    /// Go 標籤：`json:"gate"`
    pub gate: MontageTailGate,
}

/// Go：`montage_tail.go:30`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct MontageTailRawTable {
    /// Record0Byte7, Record1Byte7 and Global540FF retain the three raw
    /// 20-byte tables at 0x525dc..0x52617. The byte-6 writes are not a fourth
    /// table: native code derives each from its paired byte-7 value.
    /// Go 標籤：`json:"record0_byte_7"`
    #[serde(rename = "record0_byte_7", deserialize_with = "battle::null_as_default")]
    pub record0_byte7: Vec<i64>,
    /// Go 標籤：`json:"record1_byte_7"`
    #[serde(rename = "record1_byte_7", deserialize_with = "battle::null_as_default")]
    pub record1_byte7: Vec<i64>,
    /// Go 標籤：`json:"global_540ff"`
    #[serde(rename = "global_540ff", deserialize_with = "battle::null_as_default")]
    pub global540_ff: Vec<i64>,
}

/// Go：`montage_tail.go:46`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct MontageTailLoop {
    /// Go 標籤：`json:"count"`
    pub count: i64,
    /// Go 標籤：`json:"runtime_records_global"`
    pub runtime_records_global: String,
    /// Go 標籤：`json:"runtime_record_stride"`
    pub runtime_record_stride: i64,
    /// Go 標籤：`json:"record0_byte_6_formula"`
    #[serde(rename = "record0_byte_6_formula")]
    pub record0_byte6_formula: String,
    /// Go 標籤：`json:"record0_byte_7_source"`
    #[serde(rename = "record0_byte_7_source")]
    pub record0_byte7_source: String,
    /// Go 標籤：`json:"record1_byte_6_formula"`
    #[serde(rename = "record1_byte_6_formula")]
    pub record1_byte6_formula: String,
    /// Go 標籤：`json:"record1_byte_7_source"`
    #[serde(rename = "record1_byte_7_source")]
    pub record1_byte7_source: String,
    /// Go 標籤：`json:"global_540ff_source"`
    #[serde(rename = "global_540ff_source")]
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
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
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
///
/// Go 版也沒有使用者（Go 允許未使用的常數）；照移植保留對照。
#[allow(dead_code)]
pub(crate) const NATIVE_MONTAGE_TAIL_CONTROL_RESOURCE: i64 = 91;

/// Go 原式：`nativeMontageTailFieldMapResource + 2`
///
/// Go 版也沒有使用者；照移植保留對照。
#[allow(dead_code)]
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
        if index < 0 || index >= self.runtime_records.len() as i64 {
            return Err(campaign_err!("ending: montage tail loader record {index} is unavailable"));
        }
        Ok(self.runtime_records[index as usize])
    }

    /// FirstPair returns the first two post-loader records as value copies.  This
    /// convenience function does not upgrade the pair to a renderer admission;
    /// see MontageTailLoaderBaseline for the unresolved 0x2c548 boundary.
    ///
    /// Go：`montage_tail.go:161`，`func (b MontageTailLoaderBaseline) FirstPair() ([2][fdsave.UnitSize]byte, error)`
    pub fn first_pair(&self) -> Result<[[u8; 80]; 2]> {
        if self.runtime_records.len() < 2 {
            return Err(campaign_err!("ending: montage tail loader lacks the first two records"));
        }
        Ok([self.runtime_records[0], self.runtime_records[1]])
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
    tail.validate_native_contract()?;
    // The fixed FDFIELD selector supplies 31 deployment slots.  0x2c405 later
    // indexes using the persistent count, so a 32nd persistent record lacks a
    // proven corresponding deployment record and must remain fail-closed.
    if persistent.len() < 2 || persistent.len() as i64 > NATIVE_MONTAGE_TAIL_DEPLOY_RECORD_COUNT {
        return Err(campaign_err!(
            "ending: montage tail loader persistent count {} is outside proven 2..{}",
            persistent.len(),
            NATIVE_MONTAGE_TAIL_DEPLOY_RECORD_COUNT
        ));
    }
    if paths.fdfield_root.is_empty() || paths.fdicon_root.is_empty() {
        return Err(campaign_err!("ending: montage tail loader source input is unavailable"));
    }

    let (field_map, control, positions) = load_separated_fdfield_selector30(&paths.fdfield_root)
        .map_err(|err| err.wrap("ending: montage tail separated FDFIELD selector 30: "))?;
    let unit_rows = validate_montage_tail_loader_field(&field_map, &control, &positions)?;

    let bank = fdicon::load_separated_bank(&paths.fdicon_root)
        .map_err(|err| CampaignError::from(err).wrap("ending: montage tail FDICON: "))?;
    if bank.sprites.is_empty() || bank.sprites.len() % 12 != 0 {
        return Err(campaign_err!("ending: montage tail FDICON has invalid selector groups"));
    }
    let selector_groups = (bank.sprites.len() / 12) as i64;

    let mut runtime = vec![[0u8; save::UNIT_SIZE]; NATIVE_MONTAGE_TAIL_DEPLOY_RECORD_COUNT as usize];
    let mut cache = fdicon::NativeSelectorCache::default();
    for (index, slot_record) in runtime.iter_mut().enumerate() {
        if index >= persistent.len() {
            slot_record[5] = 1;
            continue;
        }
        let mut record = persistent[index].raw;
        if i64::from(record[7]) >= selector_groups {
            return Err(campaign_err!(
                "ending: montage tail persistent record {index} has FDICON key {} outside {selector_groups} groups",
                record[7]
            ));
        }
        let slot = cache
            .slot_for(i64::from(record[7]))
            .map_err(|err| CampaignError::from(err).wrap("ending: montage tail selector cache: "))?;
        if slot > 0xff {
            return Err(campaign_err!("ending: montage tail selector slot {slot} is not a byte"));
        }
        let position_offset = (2 + (unit_rows + index as i64) * 6) as usize;
        record[0] = positions[position_offset];
        record[1] = positions[position_offset + 2];
        record[2] = slot as u8;
        record[3] = 0;
        record[4] = 0;
        record[6] = 2;
        record[0x31] = 0xff;
        record[0x22..0x28].fill(0);
        battle::apply_native_runtime_equipment_recalc(&mut record, item_table)
            .map_err(|err| CampaignError::from(err).wrap(format!("ending: montage tail runtime record {index}: ")))?;
        *slot_record = record;
    }
    Ok(MontageTailLoaderBaseline { runtime_records: runtime })
}

/// Go：`montage_tail.go:253`，`func validateMontageTailLoaderField(fieldMap, control, positions []byte) (int, error)`
pub(crate) fn validate_montage_tail_loader_field(field_map: &[u8], control: &[u8], positions: &[u8]) -> Result<i64> {
    if field_map.len() < 4 {
        return Err(campaign_err!("ending: montage tail FDFIELD map is too short"));
    }
    let width = i64::from(u16::from_le_bytes([field_map[0], field_map[1]]));
    let height = i64::from(u16::from_le_bytes([field_map[2], field_map[3]]));
    if width != 35 || height != 45 || field_map.len() as i64 != 4 + width * height * 4 {
        return Err(campaign_err!(
            "ending: montage tail FDFIELD#{NATIVE_MONTAGE_TAIL_FIELD_MAP_RESOURCE} geometry is not the proven 35x45 selector image"
        ));
    }
    if control.len() < 3
        || i64::from(control[0]) != NATIVE_MONTAGE_TAIL_LOAD_SELECTOR
        || i64::from(control[1]) != NATIVE_MONTAGE_TAIL_DEPLOY_RECORD_COUNT
        || control[2] != 1
    {
        return Err(campaign_err!(
            "ending: montage tail FDFIELD control header does not match selector {}",
            battle::go_sharp_x(NATIVE_MONTAGE_TAIL_LOAD_SELECTOR)
        ));
    }
    let unit_rows = i64::from(control[2]);
    let want_control_length =
        save::CURRENT_FIELD_CONTROL_UNIT_OFFSET as i64 + unit_rows * save::CURRENT_FIELD_CONTROL_UNIT_SIZE as i64;
    if control.len() as i64 != want_control_length {
        return Err(campaign_err!(
            "ending: montage tail FDFIELD control length {}, want {want_control_length}",
            control.len()
        ));
    }
    let unit_offset = save::CURRENT_FIELD_CONTROL_UNIT_OFFSET;
    if control[unit_offset + NATIVE_MONTAGE_TAIL_CONTROL_GROUP_OFFSET as usize] != 0xff {
        return Err(campaign_err!("ending: montage tail FDFIELD control row may append an unproven group-0 record"));
    }
    if positions.len() < 2 {
        return Err(campaign_err!("ending: montage tail FDFIELD positions are too short"));
    }
    let position_count = i64::from(u16::from_le_bytes([positions[0], positions[1]]));
    if position_count != unit_rows + NATIVE_MONTAGE_TAIL_DEPLOY_RECORD_COUNT
        || positions.len() as i64 != 2 + position_count * 6
    {
        return Err(campaign_err!("ending: montage tail FDFIELD positions do not match the proven 32-row layout"));
    }
    if i64::from(u16::from_le_bytes([positions[6], positions[7]])) != NATIVE_MONTAGE_TAIL_LOAD_SELECTOR {
        return Err(campaign_err!(
            "ending: montage tail FDFIELD first position key is not selector {}",
            battle::go_sharp_x(NATIVE_MONTAGE_TAIL_LOAD_SELECTOR)
        ));
    }
    for index in 0..NATIVE_MONTAGE_TAIL_DEPLOY_RECORD_COUNT {
        let offset = (2 + (unit_rows + index) * 6) as usize;
        if u16::from_le_bytes([positions[offset + 4], positions[offset + 5]]) != 0 {
            return Err(campaign_err!(
                "ending: montage tail FDFIELD deployment position {index} has unexpected raw key"
            ));
        }
    }
    Ok(unit_rows)
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
    if paths.surface_root.is_empty() || paths.animation_root.is_empty() {
        return Err(campaign_err!("ending: montage tail separated visual root is unavailable"));
    }
    let plans = tail.plan_visual_resources()?;
    let mut sets = Vec::with_capacity(plans.len());
    for (index, plan) in plans.into_iter().enumerate() {
        let tai = fdother::load_separated_single_frame(&paths.surface_root, "TAI.DAT", plan.tai).map_err(|err| {
            CampaignError::from(err).wrap(format!("ending: montage tail entry {index} TAI#{}: ", plan.tai))
        })?;
        let mut tai_at = tai.clone();
        (tai_at.x, tai_at.y) = (164, 157);
        validate_montage_tail_visual_frame(&tai_at)
            .map_err(|err| err.wrap(format!("ending: montage tail entry {index} TAI#{}: ", plan.tai)))?;
        let bg = fdother::load_separated_single_frame(&paths.surface_root, "BG.DAT", plan.bg).map_err(|err| {
            CampaignError::from(err).wrap(format!("ending: montage tail entry {index} BG#{}: ", plan.bg))
        })?;
        let mut bg_at = bg.clone();
        bg_at.y = 50;
        validate_montage_tail_visual_frame(&bg_at)
            .map_err(|err| err.wrap(format!("ending: montage tail entry {index} BG#{}: ", plan.bg)))?;
        let load_animation = |resource: i64| -> Result<figani::Animation> {
            let animation = figani::load_separated_resource(&paths.animation_root, resource)?;
            for (frame_index, frame) in animation.frames.iter().enumerate() {
                if frame.x < 0 || frame.y < 0 || frame.x + frame.width > WIDTH || frame.y + frame.height > HEIGHT {
                    return Err(campaign_err!("frame {frame_index} is outside the proven indexed viewport"));
                }
            }
            Ok(animation)
        };
        let load = |resource: i64| {
            load_animation(resource)
                .map_err(|err| err.wrap(format!("ending: montage tail entry {index} FIGANI#{resource}: ")))
        };
        let record1_base = load(plan.record1_figani_base)?;
        let record1_aux = load(plan.record1_figani_aux)?;
        let record0_base = load(plan.record0_figani_base)?;
        let record0_aux = load(plan.record0_figani_aux)?;
        sets.push(MontageTailVisualSet {
            plan,
            tai,
            bg,
            record1_figani_base: Some(record1_base),
            record1_figani_aux: Some(record1_aux),
            record0_figani_base: Some(record0_base),
            record0_figani_aux: Some(record0_aux),
        });
    }
    Ok(sets)
}

/// Go：`montage_tail.go:388`，`func validateMontageTailVisualFrame(frame fdother.Frame) error`
pub(crate) fn validate_montage_tail_visual_frame(frame: &fdother::Frame) -> Result<()> {
    if frame.width <= 0 || frame.height <= 0 || frame.width > WIDTH || frame.height > HEIGHT {
        return Err(campaign_err!("geometry {}x{} exceeds the indexed viewport", frame.width, frame.height));
    }
    let mut dst = vec![0u8; BYTES as usize];
    frame.blit(&mut dst, WIDTH, -1)?;
    Ok(())
}

/// LoadMontageTailAssetsArchive is retained for source-oracle tools and exact
/// archive-equivalence tests. Production callers use LoadMontageTailAssets.
///
/// Go：`montage_tail.go:401`，`func LoadMontageTailAssetsArchive(tail MontageTail, fdotherPath string) (MontageTailAssets, error)`
pub fn load_montage_tail_assets_archive(tail: &MontageTail, fdother_path: &str) -> Result<MontageTailAssets> {
    if fdother_path.is_empty() {
        return Err(campaign_err!("ending: montage tail FDOTHER path is unavailable"));
    }
    tail.plan()?;
    let palette = fdother::read_resource(fdother_path, 57)
        .map_err(|err| CampaignError::from(err).wrap("ending: montage tail FDOTHER#57: "))?;
    if palette.len() != 256 * 3 {
        return Err(campaign_err!("ending: montage tail FDOTHER#57 palette length = {}", palette.len()));
    }
    fdother::parse_vga_palette(&palette)
        .map_err(|err| CampaignError::from(err).wrap("ending: montage tail FDOTHER#57 palette: "))?;
    let loop_frames = fdother::decode_resource(fdother_path, 58)
        .map_err(|err| CampaignError::from(err).wrap("ending: montage tail FDOTHER#58: "))?;
    if loop_frames.len() as i64 != tail.r#loop.count {
        return Err(campaign_err!(
            "ending: montage tail FDOTHER#58 frames = {}, want {}",
            loop_frames.len(),
            tail.r#loop.count
        ));
    }
    for (i, frame) in loop_frames.iter().enumerate() {
        frame
            .blit(&mut vec![0u8; BYTES as usize], WIDTH, -1)
            .map_err(|err| CampaignError::from(err).wrap(format!("ending: montage tail FDOTHER#58 frame {i}: ")))?;
    }
    let load_single = |index: i64| -> Result<fdother::Frame> {
        let frame = fdother::decode_archive_single_frame(fdother_path, index)?;
        check_tail_single_frame(&frame)?;
        Ok(frame)
    };
    let intro = load_single(60).map_err(|err| err.wrap("ending: montage tail FDOTHER#60: "))?;
    let r#final = load_single(59).map_err(|err| err.wrap("ending: montage tail FDOTHER#59: "))?;
    Ok(MontageTailAssets { loop_palette: palette, intro, loop_frames, r#final })
}

/// LoadMontageTailAssets verifies the separated native resource shapes before
/// the presentation adapter can use them. It intentionally does not invent a
/// meaning for the three per-entry raw bytes or the 0x28a6c rendering owner.
///
/// Go：`montage_tail.go:462`，`func LoadMontageTailAssets(tail MontageTail, paths MontageTailAssetPaths) (MontageTailAssets, error)`
pub fn load_montage_tail_assets(tail: &MontageTail, paths: &MontageTailAssetPaths) -> Result<MontageTailAssets> {
    if paths.surface_root.is_empty() || paths.palette_root.is_empty() || paths.animation_root.is_empty() {
        return Err(campaign_err!("ending: montage tail separated asset roots are unavailable"));
    }
    tail.plan()?;
    let (palette, _) = fdother::load_separated_fdother_palette(&paths.palette_root, 57)
        .map_err(|err| CampaignError::from(err).wrap("ending: montage tail separated FDOTHER#57: "))?;
    let animation = figani::load_separated_archive_resource(&paths.animation_root, "FDOTHER.DAT", 58)
        .map_err(|err| CampaignError::from(err).wrap("ending: montage tail separated FDOTHER#58: "))?;
    if animation.frames.len() as i64 != tail.r#loop.count {
        return Err(campaign_err!(
            "ending: montage tail separated FDOTHER#58 frames = {}, want {}",
            animation.frames.len(),
            tail.r#loop.count
        ));
    }
    let mut loop_frames = Vec::with_capacity(animation.frames.len());
    for (index, source) in animation.frames.iter().enumerate() {
        let mask: Vec<u8> = source.mask.iter().map(|&value| if value != 0 { 0xff } else { 0 }).collect();
        // `fdother::Frame` 有 fd2-formats 私有欄位，crate 外不能用結構字面值；從零值逐欄設定。
        let mut frame = fdother::Frame::default();
        (frame.x, frame.y, frame.width, frame.height) = (source.x, source.y, source.width, source.height);
        frame.indexed = source.pixels.clone();
        frame.mask = mask;
        frame.blit(&mut vec![0u8; BYTES as usize], WIDTH, -1).map_err(|err| {
            CampaignError::from(err).wrap(format!("ending: montage tail separated FDOTHER#58 frame {index}: "))
        })?;
        loop_frames.push(frame);
    }
    let load_single = |index: i64| -> Result<fdother::Frame> {
        let frame = fdother::load_separated_single_frame(&paths.surface_root, "FDOTHER.DAT", index)?;
        check_tail_single_frame(&frame)?;
        Ok(frame)
    };
    let intro = load_single(60).map_err(|err| err.wrap("ending: montage tail separated FDOTHER#60: "))?;
    let r#final = load_single(59).map_err(|err| err.wrap("ending: montage tail separated FDOTHER#59: "))?;
    Ok(MontageTailAssets { loop_palette: palette, intro, loop_frames, r#final })
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
        let Some(compositor) = compositor.filter(|c| c.baseline_known()) else {
            return Err(campaign_err!("ending: montage tail final frame lacks palette baseline"));
        };
        if self.r#final.width != WIDTH || self.r#final.height != HEIGHT {
            return Err(campaign_err!(
                "ending: montage tail final frame geometry = {}x{}",
                self.r#final.width,
                self.r#final.height
            ));
        }
        compositor.vga.fill(0);
        compositor.palette = compositor.baseline;
        self.r#final
            .blit(&mut compositor.vga, WIDTH, -1)
            .map_err(|err| CampaignError::from(err).wrap("ending: montage tail final frame: "))
    }
}

/// Go：`montage_tail.go:539`，`func LoadMontageTail(path string) (*MontageTail, error)`
pub fn load_montage_tail(path: &str) -> Result<MontageTail> {
    let raw = battle::read_file(path)?;
    let tail: MontageTail = battle::decode_json(&raw)?;
    tail.validate_native_contract()
        .map_err(|err| err.wrap(format!("ending montage tail {}: ", battle::go_quote(path))))?;
    Ok(tail)
}

impl MontageTail {
    /// Go：`montage_tail.go:554`，`func (tail MontageTail) validateNativeContract() error`
    pub(crate) fn validate_native_contract(&self) -> Result<()> {
        let l = &self.r#loop;
        if self.schema_version != 2
            || self.native_handler != "0x2c194"
            || self.status != "mapped_post_montage_tail_fail_closed"
            || self.source != "0x2c194..0x2c39a"
            || self.resources.len() != 4
            || l.count != 20
            || l.runtime_records_global != "0x53a45"
            || l.runtime_record_stride != 0x50
            || l.record0_byte6_formula != "record0_byte_7 < 0x4c ? 2 : 0"
            || l.record0_byte7_source != "record0_byte_7"
            || l.record1_byte6_formula != "record1_byte_7 < 0x4c ? 2 : 0"
            || l.record1_byte7_source != "record1_byte_7"
            || l.global540_ff_source != "global_540ff"
            || l.renderer_call != "0x28a6c(0,1)"
            || l.palette_call != "0x11d40(0,255,0)"
            || l.frame_call != "0x2935b(resource_58,loop_index,staging,0x140,-1)"
            || l.wait_before_frame_ticks != 20
            || l.wait_after_frame_ticks != 78
            || l.wait_helper != "0x17aa9"
            || l.present_helper != "0x1f882"
            || l.restore_helper != "0x375c0"
            || self.gate.source.is_empty()
            || self.gate.reason.is_empty()
        {
            return Err(campaign_err!("has incomplete native contract"));
        }
        for (i, resource) in self.resources.iter().enumerate() {
            let want_index = [60, 58, 57, 59][i];
            if resource.archive != "FDOTHER.DAT"
                || resource.index != want_index
                || resource.source.is_empty()
                || resource.role.is_empty()
            {
                return Err(campaign_err!("resource {i} is incomplete"));
            }
        }
        // Go 以 map 迭代三張表；只影響同時有多張表不合法時先報哪一張（PORTING.md）。
        for (name, table) in [
            ("global_540ff", &self.raw_tables.global540_ff),
            ("record0_byte_7", &self.raw_tables.record0_byte7),
            ("record1_byte_7", &self.raw_tables.record1_byte7),
        ] {
            if table.len() as i64 != l.count {
                return Err(campaign_err!("table {name} has {} entries", table.len()));
            }
            for (i, value) in table.iter().enumerate() {
                if !(0..=0xff).contains(value) {
                    return Err(campaign_err!("table {name} entry {i} is not a byte"));
                }
            }
        }
        Ok(())
    }

    /// Plan returns the raw 20-entry schedule and the exact derived byte-6 values.
    /// It does not write a runtime record or present a frame.
    ///
    /// Go：`montage_tail.go:596`，`func (t MontageTail) Plan() ([]MontageTailEntry, error)`
    pub fn plan(&self) -> Result<Vec<MontageTailEntry>> {
        let count = self.r#loop.count;
        let t = &self.raw_tables;
        if count <= 0
            || t.record0_byte7.len() as i64 != count
            || t.record1_byte7.len() as i64 != count
            || t.global540_ff.len() as i64 != count
        {
            return Err(campaign_err!("ending montage tail has incomplete raw tables"));
        }
        let mut entries = Vec::with_capacity(count as usize);
        for i in 0..count as usize {
            let byte = |v: i64| (0..=0xff).contains(&v);
            if !byte(t.record0_byte7[i]) || !byte(t.record1_byte7[i]) || !byte(t.global540_ff[i]) {
                return Err(campaign_err!("ending montage tail raw table entry {i} is not a byte"));
            }
            entries.push(MontageTailEntry {
                index: i as i64,
                global540_ff: t.global540_ff[i] as u8,
                record0_byte6: tail_record_byte6(t.record0_byte7[i] as u8),
                record0_byte7: t.record0_byte7[i] as u8,
                record1_byte6: tail_record_byte6(t.record1_byte7[i] as u8),
                record1_byte7: t.record1_byte7[i] as u8,
            });
        }
        Ok(entries)
    }

    /// PlanVisualResources preserves the resource-index arithmetic directly used
    /// by the original nonzero 0x28a6c branch. The TAI#3 substitutions are raw
    /// selector comparisons, not named character exceptions. The source-bound E1
    /// player consumes this plan without mutating runtime records; an original E2
    /// claim still needs a separately admitted call-time record snapshot.
    ///
    /// Go：`montage_tail.go:622`，`func (t MontageTail) PlanVisualResources() ([]MontageTailVisualResources, error)`
    pub fn plan_visual_resources(&self) -> Result<Vec<MontageTailVisualResources>> {
        let entries = self.plan()?;
        Ok(entries
            .iter()
            .enumerate()
            .map(|(index, entry)| {
                let mut tai = i64::from(entry.global540_ff);
                if entry.record0_byte7 == 0x1a || entry.record0_byte7 == 0x36 || entry.record1_byte7 == 0x37 {
                    tai = 3;
                }
                MontageTailVisualResources {
                    index: index as i64,
                    tai,
                    bg: i64::from(entry.global540_ff),
                    record1_figani_base: i64::from(entry.record1_byte7) * 3,
                    record1_figani_aux: i64::from(entry.record1_byte7) * 3 + 1,
                    record0_figani_base: i64::from(entry.record0_byte7) * 3,
                    record0_figani_aux: i64::from(entry.record0_byte7) * 3 + 1,
                }
            })
            .collect())
    }
}

/// Go：`montage_tail.go:646`，`func tailRecordByte6(recordByte7 byte) byte`
pub(crate) fn tail_record_byte6(record_byte7: u8) -> u8 {
    if record_byte7 < 0x4c { 2 } else { 0 }
}

/// Go `LoadMontageTailAssets*` 內 `loadSingle` 的共同檢查：320×200 且能完整貼進 VGA。
fn check_tail_single_frame(frame: &fdother::Frame) -> Result<()> {
    if frame.width != WIDTH || frame.height != HEIGHT {
        return Err(campaign_err!("geometry {}x{}, want {WIDTH}x{HEIGHT}", frame.width, frame.height));
    }
    frame.blit(&mut vec![0u8; BYTES as usize], WIDTH, -1)?;
    Ok(())
}

#[cfg(test)]
pub(crate) mod montage_tail_test;

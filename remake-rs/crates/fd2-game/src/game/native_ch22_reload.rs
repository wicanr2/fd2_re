//! Go 來源：`remake/cmd/fd2/native_ch22_reload.go`（165 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_ch22_reload.go:47`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeCh22ReloadState {
    pub(crate) phase: i64,
    pub(crate) field: Option<MapData>,
    pub(crate) assets: Option<NativeMapAssets>,
    pub(crate) state: Option<battle::State>,
    pub(crate) aux: Vec<u8>,
}

/// 讀分離的 FDFIELD 地圖 JSON，並重組原版的 4+4×格數原始格（tile word、事件 byte、blit mode）。
///
/// Go：`native_ch22_reload.go:16`，`func loadSeparatedFDFIELDComposition(dir string) (*MapData, []byte, error)`
pub(crate) fn load_separated_fdfield_composition(dir: &str) -> Result<(MapData, Vec<u8>)> {
    let raw_json =
        std::fs::read(format!("{}/map.json", asset_path(dir))).map_err(|err| GameError::new(err.to_string()))?;
    let field: MapData = serde_json::from_slice(&raw_json).map_err(|err| GameError::new(err.to_string()))?;
    if field.w <= 0 || field.h <= 0 || field.w > 0xffff || field.h > 0xffff {
        return Err(game_err!("separated FDFIELD shape={}x{}", field.w, field.h));
    }
    let cells = (field.w * field.h) as usize;
    if field.tiles.len() != cells
        || field.native_composition_event_bytes.len() != cells
        || field.native_tile_blit_modes.len() != cells
    {
        return Err(game_err!(
            "separated FDFIELD shape={}x{} tiles={} events={} modes={}",
            field.w,
            field.h,
            field.tiles.len(),
            field.native_composition_event_bytes.len(),
            field.native_tile_blit_modes.len()
        ));
    }
    let mut raw = vec![0u8; 4 + 4 * cells];
    raw[0..2].copy_from_slice(&(field.w as u16).to_le_bytes());
    raw[2..4].copy_from_slice(&(field.h as u16).to_le_bytes());
    for (index, &tile) in field.tiles.iter().enumerate() {
        if !(0..=0xffff).contains(&tile) {
            return Err(game_err!("separated FDFIELD cell {index} tile={tile}"));
        }
        let offset = 4 + 4 * index;
        raw[offset..offset + 2].copy_from_slice(&(tile as u16).to_le_bytes());
        raw[offset + 2] = field.native_composition_event_bytes[index];
        raw[offset + 3] = field.native_tile_blit_modes[index];
    }
    Ok((field, raw))
}

/// 第 22 章戰後重載 map23：解出 FDFIELD #69 與 FDSHAP #46／#47，建立候選地圖、素材與戰場狀態，尚不發布。
///
/// Go：`native_ch22_reload.go:55`，`func decodeNativeCh22Reload(g *Game) (*nativeCh22ReloadState, error)`
pub(crate) fn decode_native_ch22_reload(g: &Game) -> Result<NativeCh22ReloadState> {
    let (Some(st), Some(m), Some(current)) = (g.st.as_ref(), g.m.as_ref(), g.native_map_assets.as_ref()) else {
        return Err(GameError::new("ch22 reload requires current indexed state and separated map23 assets"));
    };
    let (field_source, raw) = load_separated_fdfield_composition("assets/maps/map23")
        .map_err(|err| game_err!("separated FDFIELD #69: {}", err.message()))?;
    let (w, h) = (field_source.w, field_source.h);
    let (bank, controls) = fdicon::load_separated_fdshap_bank(separated_asset_path("tilesets/fdshap"), 23)
        .map_err(|err| game_err!("FDSHAP #46/#47: {err}"))?;
    let cells = (w * h) as usize;
    let (mut tiles, mut modes) = (vec![0i64; cells], vec![0u8; cells]);
    for index in 0..cells {
        let offset = 4 + 4 * index;
        let tile = i64::from(u16::from_le_bytes([raw[offset], raw[offset + 1]]) & 0x03ff);
        if tile >= controls.len() as i64 / 4 {
            return Err(game_err!(
                "FDFIELD #69 cell {index} tile={tile} exceeds FDSHAP #47 controls={}",
                controls.len() / 4
            ));
        }
        (tiles[index], modes[index]) = (tile, raw[offset + 3]);
    }
    let mut field = field_source;
    (field.tile_w, field.tile_h, field.cols) = (m.tile_w, m.tile_h, m.cols);
    field.tiles = tiles;
    field.native_tile_blit_modes = modes.clone();
    field.native_terrain_control = controls.clone();
    let mut assets = current.clone();
    assets.map_index = 23;
    assets.terrain = Some(Arc::new(bank));
    assets.controls = controls;
    let mut state = st.clone();
    (state.w, state.h) = (w, h);
    state.native_tile_blit_modes = modes;
    state.native_map_event_grid = raw;
    state.has_native_map_event_grid = true;
    Ok(NativeCh22ReloadState { field: Some(field), assets: Some(assets), state: Some(state), ..Default::default() })
}

impl Game {
    /// 0x24A4B／0x24A65／0x24A7F 三個 load_res 依序載入 FDFIELD #69、FDSHAP #46、#47；第一個呼叫先私下解碼候選。
    ///
    /// Go：`native_ch22_reload.go:96`，`func (g *Game) stageNativeCh22Resource(b campaign.Beat) error`
    pub(crate) fn stage_native_ch22_resource(&mut self, b: &campaign::Beat) -> Result<()> {
        let Some(resource) = b.resource_id else {
            return Err(GameError::new("missing typed resource index"));
        };
        let wants = [
            ("0x24a4b", "FDFIELD.DAT", "0x53a51", 69),
            ("0x24a65", "FDSHAP.DAT", "0x53a5d", 46),
            ("0x24a7f", "FDSHAP.DAT", "0x53a69", 47),
        ];
        let phase = self.native_ch22_reload.as_ref().map_or(0, |r| r.phase);
        let Some(&(source, archive, owner, want)) = usize::try_from(phase).ok().and_then(|p| wants.get(p)) else {
            return Err(GameError::new("ch22 reload resource order is invalid"));
        };
        if b.source != source || b.resource_archive != archive || b.resource_owner != owner || resource != want {
            return Err(game_err!(
                "ch22 reload phase{phase} got {}/{}/{}/{resource}",
                b.source,
                b.resource_archive,
                b.resource_owner
            ));
        }
        if phase == 0 {
            self.native_ch22_reload = Some(decode_native_ch22_reload(self)?);
        }
        self.native_ch22_reload.as_mut().unwrap().phase += 1;
        self.handler_resource = resource;
        Ok(())
    }

    /// Go：`native_ch22_reload.go:131`，`func (g *Game) resetNativeCh22ReloadGrid() error`
    pub(crate) fn reset_native_ch22_reload_grid(&mut self) -> Result<()> {
        let ok = self
            .native_ch22_reload
            .as_mut()
            .is_some_and(|r| r.phase == 3 && r.state.as_mut().is_some_and(|state| state.reset_native_map_event_grid()));
        if !ok {
            return Err(GameError::new("ch22 reload grid is incomplete"));
        }
        self.native_ch22_reload.as_mut().unwrap().phase = 4;
        Ok(())
    }

    /// 第 23 章輔助圖 FDOTHER #42 進 0x53A35 staging，之後一次發布候選地圖、素材與戰場格。
    ///
    /// Go：`native_ch22_reload.go:140`，`func (g *Game) prepareNativeCh22Aux() error`
    pub(crate) fn prepare_native_ch22_aux(&mut self) -> Result<()> {
        if self.native_ch22_reload.as_ref().is_none_or(|r| r.phase != 4) || self.handler_chapter != 23 {
            return Err(GameError::new("ch22 chapter23 auxiliary reload is out of order"));
        }
        let frame = fdother::load_separated_native_ch23_stage(separated_asset_path("surfaces"))
            .map_err(|err| game_err!("FDOTHER #42: {err}"))?;
        let mut staging = vec![0u8; (fdother::NATIVE_CH23_STAGE_STRIDE * fdother::NATIVE_CH23_STAGE_HEIGHT) as usize];
        fdother::blit_native_ch23_stage(&frame, &mut staging).map_err(|err| game_err!("FDOTHER #42 staging: {err}"))?;
        let mut reload = self.native_ch22_reload.take().unwrap();
        reload.aux = staging.clone();
        let state = reload.state.take().unwrap();
        self.m = reload.field.take();
        self.native_map_assets = reload.assets.take();
        let st =
            self.st.as_mut().ok_or_else(|| GameError::new("ch22 chapter23 auxiliary reload lacks battle state"))?;
        (st.w, st.h) = (state.w, state.h);
        st.native_tile_blit_modes = state.native_tile_blit_modes;
        st.native_map_event_grid = state.native_map_event_grid;
        st.has_native_map_event_grid = true;
        self.native_ch23_state = Some(NativeCh23AdapterState { staging, initial_complete: true, ..Default::default() });
        Ok(())
    }
}

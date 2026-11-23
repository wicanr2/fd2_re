//! Go 來源：`remake/cmd/fd2/native_town_departure.go`（111 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_town_departure.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeTownDepartureFrame {
    pub(crate) indexed: Vec<u8>,
    pub(crate) dac: Vec<u8>,
    pub(crate) palette: ColorPalette,
}

/// Go：`native_town_departure.go:20`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeTownDepartureJob {
    pub(crate) frames: Vec<NativeTownDepartureFrame>,
    pub(crate) step: i64,
    pub(crate) drawn: bool,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) source: Option<NativeTownFrameSource>,
}

impl Game {
    pub(crate) fn finish_native_preparation_departure(&mut self, then: Box<dyn FnOnce(&mut Game) + Send>) {
        let camp = self.camp.as_ref();
        let town_backed = camp.and_then(|c| c.node()).is_some_and(|node| !node.cancel.is_empty());
        if !town_backed || self.native_town_ui.is_none() {
            then(self);
            return;
        }
        match self.build_native_town_departure(then) {
            Ok(job) => self.native_town_departure = Some(job),
            Err(err) => self.load_err = format!("城鎮出發轉場：{}", err.message()),
        }
    }

    fn build_native_town_departure(&self, then: Box<dyn FnOnce(&mut Game) + Send>) -> Result<NativeTownDepartureJob> {
        let map = self
            .native_map_assets
            .as_ref()
            .filter(|assets| native_map_assets_available(Some(assets)))
            .ok_or_else(|| game_err!("缺少出發調色盤"))?;
        if self.native_town_departure.is_some() || self.native_palette_ramp.is_some() {
            return Err(game_err!("出發轉場仍在進行"));
        }
        let camp = self.camp.as_ref().ok_or_else(|| game_err!("缺少戰役"))?;
        let caller =
            camp.node().filter(|node| node.r#type == "preparation").ok_or_else(|| game_err!("出發轉場缺少整備來源"))?;
        let variant = camp
            .c
            .as_ref()
            .and_then(|campaign| campaign.nodes.get(&caller.cancel))
            .filter(|town| town.r#type == "town")
            .and_then(|town| town.native_town_variant)
            .ok_or_else(|| game_err!("出發轉場缺少城鎮來源"))?;
        let source = self
            .prep_town_source
            .as_ref()
            .filter(|source| {
                source.variant == variant && source.selection == 2 && source.indexed == self.prep_prompt_source
            })
            .cloned();
        let mut frames = Vec::with_capacity(11);
        for k in 1..=11 {
            let step = k.min(10);
            let delta = if k == 11 { 64 } else { k * 4 };
            let indexed = campaign::compose_native_town_departure_frame(&self.prep_prompt_source, variant, 2, step)?;
            let mut dac = vec![0; 768];
            fdother::apply_vga_palette_subtraction(&mut dac, &map.palette_dac, 0, 255, delta)?;
            let palette = color_palette_from_nrgba(&fdother::vga_palette_from_dac(&dac)?);
            frames.push(NativeTownDepartureFrame { indexed, dac, palette });
        }
        Ok(NativeTownDepartureJob { frames, source, then: Some(then), ..Default::default() })
    }

    pub(crate) fn draw_native_town_departure(&mut self, screen: &mut Picture) {
        let job = self.native_town_departure.as_ref().expect("出發轉場存在");
        let frame = job.frames[job.step as usize].clone();
        self.present_native_class_frame_with_palette(screen, &frame.indexed, &frame.palette);
        self.native_town_departure.as_mut().unwrap().drawn = true;
    }

    /// Go：`native_town_departure.go:68`，`func (g *Game) stepNativeTownDeparture()`
    pub(crate) fn step_native_town_departure(&mut self) {
        let Some(j) = self.native_town_departure.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.step += 1;
        j.drawn = false;
        if j.step < j.frames.len() as i64 {
            return;
        }
        let mut j = self.native_town_departure.take().unwrap();
        self.native_map_dac.clear();
        self.native_map_dac.extend_from_slice(&j.frames[j.frames.len() - 1].dac);
        self.native_map_vga.fill(0);
        self.native_town_load_pending = true;
        if let Some(then) = j.then.take() {
            then(self);
        }
    }
}

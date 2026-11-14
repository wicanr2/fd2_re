//! Go 來源：`remake/cmd/fd2/native_physical_body.go`（526 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 6 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativePhysicalBodyResources 在方向、HP與RNG修改前建立。
/// 正式入口與證據：docs/data/ida/fd2_physical_background_selection_20261004.json。
///
/// Go：`native_physical_body.go:20`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativePhysicalBodyResources {
    pub(crate) attack: Option<figani::Animation>,
    pub(crate) counter: Option<figani::Animation>,
    pub(crate) actor_idle: Option<figani::Animation>,
    pub(crate) target_idle: Option<figani::Animation>,
    /// 原版 54103／54107，反擊不交換。
    pub(crate) actor_bg: fdother::Frame,
    /// Go 與 `actorBG` 同列宣告，註解見該欄位。
    pub(crate) target_bg: fdother::Frame,
    pub(crate) layers: [fdother::Frame; 3],
    pub(crate) platform: fdother::Frame,
    /// 29164 返回後的共享 base。
    pub(crate) base: Vec<u8>,
    pub(crate) actor_side: u8,
    pub(crate) target_side: u8,
    pub(crate) main_sounds: BTreeMap<i64, Vec<u8>>,
    pub(crate) counter_sounds: BTreeMap<i64, Vec<u8>>,
}

/// 一個工作項目可只有cue；只有present項目才等待正式Draw。
/// body項目共用不可變的panel base，不為每個idle tick建立GPU貼圖。
///
/// Go：`native_physical_body.go:32`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativePhysicalBodyJob {
    pub(crate) hd_sources: Option<ModernHDBattleSources>,
    pub(crate) hd_phase: String,
    pub(crate) pixels: Vec<u8>,
    pub(crate) base: Vec<u8>,
    pub(crate) attack: figani::Frame,
    pub(crate) idle: figani::Frame,
    pub(crate) present: Option<battlepresent::NativePhysicalBodyPresent>,
    pub(crate) raw_side: u8,
    pub(crate) palette0: Option<ColorNrgba>,
    pub(crate) wait_millis: f64,
    pub(crate) cue: i64,
    pub(crate) sounds: BTreeMap<i64, Vec<u8>>,
    pub(crate) sound_bank: i64,
    pub(crate) hd_impact: bool,
}

/// Go：`native_physical_body.go:48`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativePhysicalBodyPlayback {
    pub(crate) jobs: Vec<NativePhysicalBodyJob>,
    pub(crate) index: i64,
    pub(crate) drawn: bool,
    pub(crate) cued: bool,
    pub(crate) elapsed_millis: f64,
    pub(crate) hd_extra_millis: f64,
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) last_present: i64,
    /// 正常2909D／2909F的17AA9(6)，與descriptor等待分開。
    pub(crate) return_wait_millis: f64,
    pub(crate) return_ramp_started: bool,
}

impl Game {
    /// Go：`native_physical_body.go:60`，`func (g *Game) prepareNativePhysicalBodyResources(scene *nativePhysicalScene, actor, target *battle.Unit, attack *figani.Animation) error`
    pub(crate) fn prepare_native_physical_body_resources(
        &mut self,
        scene: &mut NativePhysicalScene,
        actor: UnitIdx,
        target: UnitIdx,
        attack: &mut figani::Animation,
    ) -> Result<()> {
        todo!("native_physical_body.go:prepareNativePhysicalBodyResources")
    }

    /// 2BC9A索引525D6的bytes30..35；header4零表示無bank。
    /// 既有證據：docs/data/ida/fd2_command24_presentation_ida.txt。
    ///
    /// Go：`native_physical_body.go:157`，`func (g *Game) nativePhysicalSoundBank(animation *figani.Animation) (map[int][]byte, error)`
    pub(crate) fn native_physical_sound_bank(
        &mut self,
        animation: &figani::Animation,
    ) -> Result<BTreeMap<i64, Vec<u8>>> {
        todo!("native_physical_body.go:nativePhysicalSoundBank")
    }

    /// Go：`native_physical_body.go:206`，`func (g *Game) attachNativePhysicalBody(scene *nativePhysicalScene, result battle.NativePhysicalAttackResult) error`
    pub(crate) fn attach_native_physical_body(
        &mut self,
        scene: Option<&mut NativePhysicalScene>,
        result: &battle::NativePhysicalAttackResult,
    ) -> Result<()> {
        todo!("native_physical_body.go:attachNativePhysicalBody")
    }
}

impl NativePhysicalBodyJob {
    /// Go：`native_physical_body.go:407`，`func (job *nativePhysicalBodyJob) indexed() ([]byte, error)`
    pub(crate) fn indexed(&self) -> Result<Vec<u8>> {
        todo!("native_physical_body.go:indexed")
    }
}

impl Game {
    /// Go：`native_physical_body.go:414`，`func (g *Game) drawNativePhysicalBody(screen *ebiten.Image, scene *nativePhysicalScene) error`
    pub(crate) fn draw_native_physical_body(
        &mut self,
        screen: &mut Picture,
        scene: &mut NativePhysicalScene,
    ) -> Result<()> {
        todo!("native_physical_body.go:drawNativePhysicalBody")
    }

    /// stepNativePhysicalBodyMillis 以60Hz來源播放20/40ms與BIOS等待。
    /// 每個實際present必須由Draw確認；不累積缺畫面的時間、不跨格追趕。
    ///
    /// Go：`native_physical_body.go:458`，`func (g *Game) stepNativePhysicalBodyMillis(scene *nativePhysicalScene, elapsed float64) error`
    pub(crate) fn step_native_physical_body_millis(
        &mut self,
        scene: &mut NativePhysicalScene,
        elapsed: f64,
    ) -> Result<()> {
        todo!("native_physical_body.go:stepNativePhysicalBodyMillis")
    }
}

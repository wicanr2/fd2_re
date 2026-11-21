//! 外殼的 GPU 加速模式（R5 階段 5，`r5g`）。
//!
//! 開關：命令列 `--renderer=gpu`（或 `--renderer gpu`）、環境變數 `FD2_RENDERER=gpu`；預設 `cpu`。命令列優先。其他值是錯誤。
//!
//! 主世界每個顯示幀以 `fd2_game::game::display_list::record` 把 `Game::draw` 錄成顯示清單，放進共用的 [`GpuBridge`]；
//! 渲染世界在 `ExtractSchedule` 取走（與主世界同步，下一幀的清單不會蓋掉這一幀的），在 `Render` 的
//! `RenderSystems::PrepareResources` 以 [`GpuCompositor`] 合成，再把結果直接複製到外殼顯示用的貼圖（`GpuImage`），
//! 不經 CPU 上傳。腳本模式另把預乘畫格讀回，主世界照順序寫 `frames.jsonl`（雜湊與 CPU 模式比對）。
//!
//! GPU 不可用（沒有渲染世界、裝置不支援 compute、管線或合成出錯）時退回 CPU 合成，並在 stderr 與腳本模式的
//! `summary.json` 記錄原因。合成器還沒準備好的幾幀照 CPU 合成，畫面不受影響。

use std::collections::VecDeque;
use std::sync::{Arc, Mutex};

use bevy::prelude::*;
use bevy::render::render_asset::RenderAssets;
use bevy::render::renderer::{RenderAdapter, RenderAdapterInfo, RenderDevice, RenderQueue};
use bevy::render::texture::GpuImage;
use bevy::render::{ExtractSchedule, Render, RenderApp, RenderSystems};
use fd2_game::game::display_list::DisplayList;

use crate::gpu::{GpuCompositor, adapter_json};
use crate::video::{LOGICAL_H, LOGICAL_W};

/// 畫格合成方式。
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Renderer {
    Cpu,
    Gpu,
}

/// 讀命令列（`args` 不含程式名）與 `FD2_RENDERER`。
pub fn requested_renderer(args: &[String], env: Option<&str>) -> Result<Renderer, String> {
    let parse = |v: &str| match v {
        "cpu" => Ok(Renderer::Cpu),
        "gpu" => Ok(Renderer::Gpu),
        other => Err(format!("不認得的合成方式 {other:?}（只有 cpu、gpu）")),
    };
    let mut from_args = None;
    let mut it = args.iter();
    while let Some(a) = it.next() {
        if let Some(v) = a.strip_prefix("--renderer=") {
            from_args = Some(parse(v)?);
        } else if a == "--renderer" {
            from_args = Some(parse(it.next().ok_or("--renderer 後面要接 cpu 或 gpu")?)?);
        }
    }
    match (from_args, env.filter(|v| !v.is_empty())) {
        (Some(r), _) => Ok(r),
        (None, Some(v)) => parse(v),
        (None, None) => Ok(Renderer::Cpu),
    }
}

/// GPU 合成器的狀態（渲染世界寫、主世界讀）。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub enum GpuStatus {
    /// 渲染世界還沒建立合成器。
    #[default]
    Pending,
    /// 可以用；內容是轉接器說明（JSON）。
    Ready(String),
    /// 不能用；內容是原因。
    Unavailable(String),
}

/// 一幀要合成的清單。
pub struct GpuFrame {
    pub id: u64,
    pub list: DisplayList,
    /// 腳本模式：讀回預乘畫格。
    pub readback: bool,
}

/// 一幀的合成結果（只有要求讀回或出錯時才回）。
pub struct GpuResult {
    pub id: u64,
    pub pix: Result<Vec<u8>, String>,
    /// 渲染世界從上傳到讀回完成的時間（毫秒）。
    pub gpu_ms: f64,
}

#[derive(Default)]
pub struct BridgeState {
    pub status: GpuStatus,
    pub target: Option<AssetId<Image>>,
    pub pending: Option<GpuFrame>,
    pub done: VecDeque<GpuResult>,
}

/// 主世界與渲染世界共用的狀態。
#[derive(Resource, Clone, Default)]
pub struct GpuBridge(pub Arc<Mutex<BridgeState>>);

impl GpuBridge {
    pub fn lock(&self) -> std::sync::MutexGuard<'_, BridgeState> {
        self.0.lock().unwrap()
    }
}

/// 渲染世界這一幀取到的清單。
#[derive(Resource, Default)]
struct ExtractedGpuFrame(Option<GpuFrame>);

/// 渲染世界的合成器（建立失敗後不再重試）。
#[derive(Default)]
enum CompositorSlot {
    #[default]
    Untried,
    Ready(Box<GpuCompositor>),
    Failed,
}

/// 把 GPU 模式接到渲染世界。沒有渲染世界（沒有可用的渲染後端）時回原因，呼叫端退回 CPU。
pub fn install(app: &mut App, bridge: &GpuBridge) -> Result<(), String> {
    let Some(render) = app.get_sub_app_mut(RenderApp) else {
        return Err("沒有渲染世界（找不到可用的渲染後端）".into());
    };
    render
        .insert_resource(bridge.clone())
        .init_resource::<ExtractedGpuFrame>()
        .add_systems(ExtractSchedule, extract_gpu_frame)
        .add_systems(Render, composite_gpu_frame.in_set(RenderSystems::PrepareResources));
    Ok(())
}

fn extract_gpu_frame(bridge: Res<GpuBridge>, mut extracted: ResMut<ExtractedGpuFrame>) {
    extracted.0 = bridge.lock().pending.take();
}

// Bevy 系統的參數即依賴注入。
#[allow(clippy::too_many_arguments)]
fn composite_gpu_frame(
    bridge: Res<GpuBridge>,
    mut extracted: ResMut<ExtractedGpuFrame>,
    device: Res<RenderDevice>,
    queue: Res<RenderQueue>,
    adapter: Res<RenderAdapter>,
    info: Res<RenderAdapterInfo>,
    images: Res<RenderAssets<GpuImage>>,
    mut slot: Local<CompositorSlot>,
) {
    if matches!(*slot, CompositorSlot::Untried) {
        let made = GpuCompositor::new(
            device.wgpu_device().clone(),
            (**queue).clone(),
            &adapter.get_downlevel_capabilities(),
            adapter_json(&info),
            LOGICAL_W,
            LOGICAL_H,
        );
        let mut state = bridge.lock();
        match made {
            Ok(c) => {
                state.status = GpuStatus::Ready(c.adapter.clone());
                *slot = CompositorSlot::Ready(Box::new(c));
            }
            Err(e) => {
                state.status = GpuStatus::Unavailable(e);
                *slot = CompositorSlot::Failed;
            }
        }
    }
    let Some(frame) = extracted.0.take() else {
        return;
    };
    let CompositorSlot::Ready(comp) = &*slot else {
        bridge
            .lock()
            .done
            .push_back(GpuResult { id: frame.id, pix: Err("GPU 合成器不可用".into()), gpu_ms: 0.0 });
        return;
    };
    let t0 = std::time::Instant::now();
    let target = bridge.lock().target;
    let texture = target.and_then(|id| images.get(id)).map(|img| &img.texture);
    let wgpu_device = comp.device();
    let scope = wgpu_device.push_error_scope(wgpu::ErrorFilter::Validation);
    let mut encoder =
        wgpu_device.create_command_encoder(&wgpu::CommandEncoderDescriptor { label: Some("fd2 r5g shell") });
    let result = comp.encode(&frame.list, &mut encoder).map(|()| {
        if let Some(texture) = texture {
            comp.copy_display_to_texture(&mut encoder, texture);
        }
        if frame.readback {
            comp.encode_readback(&mut encoder);
        }
        comp.submit(encoder);
    });
    let result = result.and_then(|()| match bevy::tasks::block_on(scope.pop()) {
        Some(err) => Err(format!("GPU 合成失敗：{err}")),
        None => Ok(()),
    });
    let pix = match result {
        Err(e) => Err(e),
        Ok(()) if frame.readback => comp.read_back(),
        Ok(()) => return,
    };
    let gpu_ms = t0.elapsed().as_secs_f64() * 1e3;
    let mut state = bridge.lock();
    if let Err(e) = &pix {
        state.status = GpuStatus::Unavailable(e.clone());
    }
    state.done.push_back(GpuResult { id: frame.id, pix, gpu_ms });
}

/// 外殼主世界這一側的 GPU 模式狀態（只在要求 GPU 時建立）。
pub struct ShellGpu {
    bridge: GpuBridge,
    active: bool,
    /// 退回 CPU 的原因（記錄一次）。
    pub fallback: Option<String>,
    /// 可用時的轉接器說明。
    pub adapter: Option<String>,
    next_id: u64,
}

impl ShellGpu {
    /// 接到渲染世界；失敗時直接標成退回 CPU。
    pub fn install(app: &mut App) -> ShellGpu {
        let bridge = GpuBridge::default();
        let fallback = install(app, &bridge).err();
        if let Some(reason) = &fallback {
            eprintln!("GPU 模式不可用，改用 CPU 合成：{reason}");
            bridge.lock().status = GpuStatus::Unavailable(reason.clone());
        }
        ShellGpu { bridge, active: false, fallback, adapter: None, next_id: 0 }
    }

    /// 外殼顯示用貼圖（`setup` 建立後呼叫）。
    pub fn set_target(&self, id: AssetId<Image>) {
        self.bridge.lock().target = Some(id);
    }

    /// 每幀開頭更新狀態，回傳這一幀是否用 GPU 合成。
    pub fn poll_status(&mut self) -> bool {
        let state = self.bridge.lock();
        match &state.status {
            GpuStatus::Pending => {}
            GpuStatus::Ready(adapter) => {
                if !self.active && self.fallback.is_none() {
                    eprintln!("GPU 模式：{adapter}");
                    self.adapter = Some(adapter.clone());
                    self.active = true;
                }
            }
            GpuStatus::Unavailable(reason) => {
                if self.fallback.is_none() {
                    eprintln!("GPU 模式不可用，改用 CPU 合成：{reason}");
                    self.fallback = Some(reason.clone());
                }
                self.active = false;
            }
        }
        self.active
    }

    /// 合成器的狀態已經確定（可用或退回 CPU）。
    pub fn settled(&self) -> bool {
        self.fallback.is_some() || self.active
    }

    /// 交出一幀的清單，回傳序號。
    pub fn submit(&mut self, list: DisplayList, readback: bool) -> u64 {
        let id = self.next_id;
        self.next_id += 1;
        self.bridge.lock().pending = Some(GpuFrame { id, list, readback });
        id
    }

    /// 取走已完成的結果。
    pub fn take_results(&self) -> Vec<GpuResult> {
        self.bridge.lock().done.drain(..).collect()
    }

    /// 腳本模式 `summary.json` 的 `renderer` 欄。
    pub fn summary(&self) -> serde_json::Value {
        serde_json::json!({
            "requested": "gpu",
            "used": if self.active { "gpu" } else { "cpu" },
            "adapter": self.adapter.as_deref().and_then(|a| serde_json::from_str::<serde_json::Value>(a).ok()),
            "fallback": self.fallback,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn renderer_switch() {
        let a = |v: &[&str]| v.iter().map(|s| s.to_string()).collect::<Vec<_>>();
        assert_eq!(requested_renderer(&a(&[]), None), Ok(Renderer::Cpu));
        assert_eq!(requested_renderer(&a(&[]), Some("")), Ok(Renderer::Cpu));
        assert_eq!(requested_renderer(&a(&[]), Some("gpu")), Ok(Renderer::Gpu));
        assert_eq!(requested_renderer(&a(&["--renderer=gpu"]), None), Ok(Renderer::Gpu));
        assert_eq!(requested_renderer(&a(&["--renderer", "cpu"]), Some("gpu")), Ok(Renderer::Cpu), "命令列優先");
        assert!(requested_renderer(&a(&["--renderer=vulkan"]), None).is_err());
        assert!(requested_renderer(&a(&["--renderer"]), None).is_err());
        assert!(requested_renderer(&a(&[]), Some("GPU")).is_err());
    }
}

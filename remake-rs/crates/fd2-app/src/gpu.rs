//! GPU 合成器（R5 階段 5，`r5g`）：在 GPU 上執行 `fd2-game` 錄下的顯示清單（[`DisplayList`]）。
//!
//! 每個畫面像素一個 compute 呼叫，依序走過和該 32×32 區塊相交的操作：在矩形內就查 `xs`／`ys` 表取來源像素、查 `ColorScale` 表、做整數
//! source-over 或覆蓋寫入。幾何與浮點顏色換算都已在錄製時由 CPU 算好（見 `fd2-game` 的 `display_list` 模組說明），
//! 著色器只有整數運算，結果與 CPU 合成逐位元組相同。同一像素的操作在同一個呼叫裡依序執行，不需要操作之間的屏障。
//!
//! 輸出緩衝區前半是預乘 RGBA8（`frame`，與 CPU 畫格相同，供雜湊與讀回），後半是 alpha 改 255 的同一張（`display`，同
//! [`crate::video::opaque_rgba`]），直接複製到外殼顯示用的貼圖，不經 CPU。索引表與查表共用一個緩衝區、兩份輸出共用一個
//! 緩衝區，儲存緩衝區只用 4 個（WebGPU 與 downlevel 裝置的每階段下限）。
//!
//! 裝置可以是外殼 Bevy 的 `RenderDevice`（[`GpuCompositor::new`]），也可以是無視窗自建的（[`GpuCompositor::headless`]，
//! 驗證工具 `r5g_gpu_exec` 與 `fd2_frames` 用）。裝置不支援 compute 或儲存緩衝區時回錯誤，呼叫端退回 CPU。

use fd2_game::game::display_list::{DisplayList, OP_WORDS, Op, u32s_to_bytes};
use wgpu::util::DeviceExt as _;

/// WGSL 著色器（整數運算）。`Op` 的欄位順序同 `fd2-game` 的 `display_list::Op`。
pub const SHADER: &str = r#"
struct Op {
    kind: u32, x0: u32, y0: u32, x1: u32, y1: u32, color: u32,
    src_off: u32, src_w: u32, xs_off: u32, ys_off: u32, lut_off: u32, reserved: u32,
}
// aux 依序存取樣索引、顏色查表、區塊偏移及繪圖指令索引。
struct Params { width: u32, height: u32, lut_base: u32, tile_base: u32, tile_cols: u32, tile_size: u32 }

@group(0) @binding(0) var<uniform> params: Params;
@group(0) @binding(1) var<storage, read> ops: array<Op>;
@group(0) @binding(2) var<storage, read> aux: array<u32>;
@group(0) @binding(3) var<storage, read> src: array<u32>;
// 前 width*height 個是 frame（預乘），後 width*height 個是 display（alpha 255）。
@group(0) @binding(4) var<storage, read_write> outbuf: array<u32>;

const NONE: u32 = 0xffffffffu;

// llvmpipe 8 位元混色的乘法：x/255 ≈ (x + (x>>8) + 0x80) >> 8（fd2-game canvas.rs 的 mul255）。
fn mul255(x: u32) -> u32 {
    return (x + (x >> 8u) + 0x80u) >> 8u;
}

// source-over，四個通道（含 alpha）同式：min(255, s + mul255(d*(255-sa)))。
fn over(s: u32, d: u32) -> u32 {
    let inv = 255u - (s >> 24u);
    var res = 0u;
    for (var ch = 0u; ch < 4u; ch = ch + 1u) {
        let sh = ch * 8u;
        let sv = (s >> sh) & 0xffu;
        let dv = (d >> sh) & 0xffu;
        res = res | (min(255u, sv + mul255(dv * inv)) << sh);
    }
    return res;
}

fn color_scale(s: u32, lut_off: u32) -> u32 {
    if (lut_off == NONE) {
        return s;
    }
    var res = 0u;
    for (var ch = 0u; ch < 4u; ch = ch + 1u) {
        let sh = ch * 8u;
        let v = (s >> sh) & 0xffu;
        res = res | (((aux[params.lut_base + lut_off + v] >> sh) & 0xffu) << sh);
    }
    return res;
}

@compute @workgroup_size(8, 8, 1)
fn main(@builtin(global_invocation_id) id: vec3<u32>) {
    let x = id.x;
    let y = id.y;
    if (x >= params.width || y >= params.height) {
        return;
    }
    var p = 0u;
    let tile = params.tile_base + (y / params.tile_size) * params.tile_cols + x / params.tile_size;
    let end = aux[tile + 1u];
    for (var k = aux[tile]; k < end; k = k + 1u) {
        let op = ops[aux[k]];
        if (x < op.x0 || x >= op.x1 || y < op.y0 || y >= op.y1) {
            continue;
        }
        if (op.kind == 0u) {
            p = op.color;
        } else if (op.kind == 1u) {
            p = src[op.src_off + (y - op.y0) * op.src_w + (x - op.x0)];
        } else {
            let sy = aux[op.ys_off + (y - op.y0)];
            let sx = aux[op.xs_off + (x - op.x0)];
            if (sx == NONE || sy == NONE) {
                continue;
            }
            p = over(color_scale(src[op.src_off + sy * op.src_w + sx], op.lut_off), p);
        }
    }
    let i = y * params.width + x;
    outbuf[i] = p;
    outbuf[params.width * params.height + i] = p | 0xff000000u;
}
"#;

const WORKGROUP: u32 = 8;
const TILE_SIZE: u32 = 32;

fn covered_tiles(op: &Op, cols: u32) -> impl Iterator<Item = usize> + use<> {
    let x_end = op.x1.div_ceil(TILE_SIZE);
    let y_end = op.y1.div_ceil(TILE_SIZE);
    let xs = op.x0 / TILE_SIZE..x_end;
    let ys = if op.x0 < op.x1 && op.y0 < op.y1 { op.y0 / TILE_SIZE..y_end } else { 0..0 };
    ys.flat_map(move |y| xs.clone().map(move |x| (y * cols + x) as usize))
}

/// 附上每個區塊的指令範圍和索引。同一區塊內沿用原始指令順序。
fn append_tile_ops(aux: &mut Vec<u32>, ops: &[Op], cols: u32, rows: u32) {
    let base = aux.len();
    let tile_count = (cols * rows) as usize;
    let mut cursors = vec![0; tile_count];
    for op in ops {
        for tile in covered_tiles(op, cols) {
            cursors[tile] += 1;
        }
    }
    let mut end = (base + tile_count + 1) as u32;
    for cursor in &mut cursors {
        aux.push(end);
        let count = std::mem::replace(cursor, end);
        end += count;
    }
    aux.push(end);
    aux.resize(end as usize, 0);
    for (index, op) in ops.iter().enumerate() {
        for tile in covered_tiles(op, cols) {
            aux[cursors[tile] as usize] = index as u32;
            cursors[tile] += 1;
        }
    }
}

/// 一個畫面大小的 GPU 合成器。
pub struct GpuCompositor {
    device: wgpu::Device,
    queue: wgpu::Queue,
    pipeline: wgpu::ComputePipeline,
    layout: wgpu::BindGroupLayout,
    width: u32,
    height: u32,
    /// 前半 `frame`、後半 `display`。
    out: wgpu::Buffer,
    staging: wgpu::Buffer,
    /// 轉接器說明（JSON）。
    pub adapter: String,
}

fn storage_entry(binding: u32, read_only: bool) -> wgpu::BindGroupLayoutEntry {
    wgpu::BindGroupLayoutEntry {
        binding,
        visibility: wgpu::ShaderStages::COMPUTE,
        ty: wgpu::BindingType::Buffer {
            ty: wgpu::BufferBindingType::Storage { read_only },
            has_dynamic_offset: false,
            min_binding_size: None,
        },
        count: None,
    }
}

/// 輸出（`frame`＋`display`）與讀回兩個緩衝區。
fn targets(device: &wgpu::Device, width: u32, height: u32) -> [wgpu::Buffer; 2] {
    let size = (width * height * 4) as u64;
    let buffer = |label: &str, size, usage| {
        device.create_buffer(&wgpu::BufferDescriptor { label: Some(label), size, usage, mapped_at_creation: false })
    };
    [
        buffer("fd2 r5g out", 2 * size, wgpu::BufferUsages::STORAGE | wgpu::BufferUsages::COPY_SRC),
        buffer("fd2 r5g staging", size, wgpu::BufferUsages::MAP_READ | wgpu::BufferUsages::COPY_DST),
    ]
}

/// 轉接器說明（JSON 字串）。
pub fn adapter_json(info: &wgpu::AdapterInfo) -> String {
    serde_json::json!({
        "name": info.name, "backend": format!("{:?}", info.backend), "device_type": format!("{:?}", info.device_type),
        "driver": info.driver, "driver_info": info.driver_info,
    })
    .to_string()
}

impl GpuCompositor {
    /// 以現有裝置建立。裝置不支援 compute 著色器、儲存緩衝區不足或管線建立失敗時回錯誤。
    pub fn new(
        device: wgpu::Device,
        queue: wgpu::Queue,
        downlevel: &wgpu::DownlevelCapabilities,
        adapter: String,
        width: u32,
        height: u32,
    ) -> Result<GpuCompositor, String> {
        if !downlevel.flags.contains(wgpu::DownlevelFlags::COMPUTE_SHADERS) {
            return Err(format!("裝置不支援 compute 著色器（{adapter}）"));
        }
        let limits = device.limits();
        if limits.max_storage_buffers_per_shader_stage < 4 {
            return Err(format!(
                "每階段儲存緩衝區上限 {} 小於 4（{adapter}）",
                limits.max_storage_buffers_per_shader_stage
            ));
        }
        let scope = device.push_error_scope(wgpu::ErrorFilter::Validation);
        let module = device.create_shader_module(wgpu::ShaderModuleDescriptor {
            label: Some("fd2 r5g display list"),
            source: wgpu::ShaderSource::Wgsl(SHADER.into()),
        });
        let mut entries = vec![wgpu::BindGroupLayoutEntry {
            binding: 0,
            visibility: wgpu::ShaderStages::COMPUTE,
            ty: wgpu::BindingType::Buffer {
                ty: wgpu::BufferBindingType::Uniform,
                has_dynamic_offset: false,
                min_binding_size: None,
            },
            count: None,
        }];
        for b in 1..=3 {
            entries.push(storage_entry(b, true));
        }
        entries.push(storage_entry(4, false));
        let layout = device.create_bind_group_layout(&wgpu::BindGroupLayoutDescriptor {
            label: Some("fd2 r5g layout"),
            entries: &entries,
        });
        let pipeline_layout = device.create_pipeline_layout(&wgpu::PipelineLayoutDescriptor {
            label: Some("fd2 r5g pipeline layout"),
            bind_group_layouts: &[Some(&layout)],
            immediate_size: 0,
        });
        let pipeline = device.create_compute_pipeline(&wgpu::ComputePipelineDescriptor {
            label: Some("fd2 r5g pipeline"),
            layout: Some(&pipeline_layout),
            module: &module,
            entry_point: Some("main"),
            compilation_options: Default::default(),
            cache: None,
        });
        if let Some(err) = bevy::tasks::block_on(scope.pop()) {
            return Err(format!("GPU 合成管線建立失敗：{err}（{adapter}）"));
        }
        let [out, staging] = targets(&device, width, height);
        Ok(GpuCompositor { device, queue, pipeline, layout, width, height, out, staging, adapter })
    }

    /// 改變畫面大小（重建輸出緩衝區；驗證工具的清單大小不一）。
    pub fn resize(&mut self, width: u32, height: u32) {
        if (width, height) != (self.width, self.height) {
            [self.out, self.staging] = targets(&self.device, width, height);
            (self.width, self.height) = (width, height);
        }
    }

    /// 無視窗自建裝置（`WGPU_BACKEND` 等環境變數照 wgpu 的慣例）。
    pub fn headless(width: u32, height: u32) -> Result<GpuCompositor, String> {
        let instance = wgpu::Instance::new(wgpu::InstanceDescriptor::new_without_display_handle_from_env());
        let adapter = bevy::tasks::block_on(instance.request_adapter(&wgpu::RequestAdapterOptions::default()))
            .map_err(|e| format!("找不到 GPU 轉接器：{e}"))?;
        let info = adapter_json(&adapter.get_info());
        let downlevel = adapter.get_downlevel_capabilities();
        let (device, queue) = bevy::tasks::block_on(adapter.request_device(&wgpu::DeviceDescriptor {
            label: Some("fd2 r5g headless"),
            required_limits: adapter.limits(),
            ..Default::default()
        }))
        .map_err(|e| format!("建立 GPU 裝置失敗：{e}（{info}）"))?;
        GpuCompositor::new(device, queue, &downlevel, info, width, height)
    }

    pub fn width(&self) -> u32 {
        self.width
    }

    pub fn height(&self) -> u32 {
        self.height
    }

    /// 上傳清單並記錄合成指令到 `encoder`（不提交）。清單大小要與合成器相同。
    pub fn encode(&self, dl: &DisplayList, encoder: &mut wgpu::CommandEncoder) -> Result<(), String> {
        if (dl.width, dl.height) != (self.width, self.height) {
            return Err(format!("顯示清單 {}×{} 與合成器 {}×{} 不同", dl.width, dl.height, self.width, self.height));
        }
        let mut ops = Vec::with_capacity(dl.ops.len().max(1) * OP_WORDS);
        for op in &dl.ops {
            ops.extend_from_slice(&[
                op.kind,
                op.x0,
                op.y0,
                op.x1,
                op.y1,
                op.color,
                op.src_off,
                op.src_w,
                op.xs_off,
                op.ys_off,
                op.lut_off,
                op.reserved,
            ]);
        }
        // 空的儲存緩衝區要有一個元素，區塊索引不會指到這個元素。
        if ops.is_empty() {
            ops.resize(OP_WORDS, 0);
        }
        let src = if dl.src.is_empty() { &[0; 4] } else { dl.src.as_slice() };
        let init = |label: &str, contents: &[u8], usage| {
            self.device.create_buffer_init(&wgpu::util::BufferInitDescriptor { label: Some(label), contents, usage })
        };
        let mut aux = dl.tables.clone();
        aux.extend_from_slice(&dl.luts);
        let tile_base = aux.len() as u32;
        let tile_cols = self.width.div_ceil(TILE_SIZE);
        append_tile_ops(&mut aux, &dl.ops, tile_cols, self.height.div_ceil(TILE_SIZE));
        let params = u32s_to_bytes(&[self.width, self.height, dl.tables.len() as u32, tile_base, tile_cols, TILE_SIZE]);
        let storage = wgpu::BufferUsages::STORAGE;
        let bufs = [
            init("fd2 r5g params", &params, wgpu::BufferUsages::UNIFORM),
            init("fd2 r5g ops", &u32s_to_bytes(&ops), storage),
            init("fd2 r5g aux", &u32s_to_bytes(&aux), storage),
            init("fd2 r5g src", src, storage),
        ];
        let mut entries: Vec<wgpu::BindGroupEntry> = bufs
            .iter()
            .enumerate()
            .map(|(i, b)| wgpu::BindGroupEntry { binding: i as u32, resource: b.as_entire_binding() })
            .collect();
        entries.push(wgpu::BindGroupEntry { binding: 4, resource: self.out.as_entire_binding() });
        let bind_group = self.device.create_bind_group(&wgpu::BindGroupDescriptor {
            label: Some("fd2 r5g bind group"),
            layout: &self.layout,
            entries: &entries,
        });
        let mut pass =
            encoder.begin_compute_pass(&wgpu::ComputePassDescriptor { label: Some("fd2 r5g"), timestamp_writes: None });
        pass.set_pipeline(&self.pipeline);
        pass.set_bind_group(0, &bind_group, &[]);
        pass.dispatch_workgroups(self.width.div_ceil(WORKGROUP), self.height.div_ceil(WORKGROUP), 1);
        Ok(())
    }

    /// 把 `display`（alpha 255）複製到貼圖（RGBA8 系列、與合成器同大小）。
    pub fn copy_display_to_texture(&self, encoder: &mut wgpu::CommandEncoder, texture: &wgpu::Texture) {
        encoder.copy_buffer_to_texture(
            wgpu::TexelCopyBufferInfo {
                buffer: &self.out,
                layout: wgpu::TexelCopyBufferLayout {
                    offset: (self.width * self.height * 4) as u64,
                    bytes_per_row: Some(self.width * 4),
                    rows_per_image: Some(self.height),
                },
            },
            wgpu::TexelCopyTextureInfo {
                texture,
                mip_level: 0,
                origin: wgpu::Origin3d::ZERO,
                aspect: wgpu::TextureAspect::All,
            },
            wgpu::Extent3d { width: self.width, height: self.height, depth_or_array_layers: 1 },
        );
    }

    /// 記錄把 `frame` 複製到讀回緩衝區的指令。
    pub fn encode_readback(&self, encoder: &mut wgpu::CommandEncoder) {
        encoder.copy_buffer_to_buffer(&self.out, 0, &self.staging, 0, (self.width * self.height * 4) as u64);
    }

    /// 等 GPU 做完並讀回 `frame`（要先提交含 [`Self::encode_readback`] 的指令）。
    pub fn read_back(&self) -> Result<Vec<u8>, String> {
        let (tx, rx) = std::sync::mpsc::channel();
        self.staging.map_async(wgpu::MapMode::Read, .., move |r| {
            let _ = tx.send(r);
        });
        self.device
            .poll(wgpu::PollType::Wait { submission_index: None, timeout: None })
            .map_err(|e| format!("等待 GPU：{e}"))?;
        rx.recv().map_err(|e| format!("讀回：{e}"))?.map_err(|e| format!("讀回對映：{e}"))?;
        let out = self.staging.get_mapped_range(..).map_err(|e| format!("讀回範圍：{e}"))?.to_vec();
        self.staging.unmap();
        Ok(out)
    }

    /// 合成一份清單並讀回預乘 RGBA8（驗證工具用；大小不同時先 [`Self::resize`]）。
    pub fn render(&mut self, dl: &DisplayList) -> Result<Vec<u8>, String> {
        self.resize(dl.width, dl.height);
        let scope = self.device.push_error_scope(wgpu::ErrorFilter::Validation);
        let mut encoder =
            self.device.create_command_encoder(&wgpu::CommandEncoderDescriptor { label: Some("fd2 r5g render") });
        self.encode(dl, &mut encoder)?;
        self.encode_readback(&mut encoder);
        self.queue.submit([encoder.finish()]);
        if let Some(err) = bevy::tasks::block_on(scope.pop()) {
            return Err(format!("GPU 合成失敗：{err}"));
        }
        self.read_back()
    }

    /// 提交指令（外殼用）。
    pub fn submit(&self, encoder: wgpu::CommandEncoder) {
        self.queue.submit([encoder.finish()]);
    }

    pub fn device(&self) -> &wgpu::Device {
        &self.device
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tile_lists_keep_draw_order_across_boundaries() {
        let op = |x0, y0, x1, y1| Op { x0, y0, x1, y1, ..Default::default() };
        let ops = [op(0, 0, 64, 64), op(31, 31, 33, 32), op(32, 32, 33, 33), op(32, 0, 32, 64), op(0, 0, 32, 32)];
        let mut aux = vec![42];
        append_tile_ops(&mut aux, &ops, 2, 2);
        assert_eq!(aux[0], 42);
        let operations = |tile: usize| &aux[aux[tile + 1] as usize..aux[tile + 2] as usize];
        assert_eq!(operations(0), [0, 1, 4]);
        assert_eq!(operations(1), [0, 1]);
        assert_eq!(operations(2), [0]);
        assert_eq!(operations(3), [0, 2]);
    }
}

#!/usr/bin/env python3
"""R5 階段 1（docs/goal/121）：從 Ebiten 實測輸出抽出 fd2-game CPU 畫布單元測試用的測試向量。

用法：rust_r5_canvas_vectors.py <階段 0 ebiten-rounding.json> <階段 1 ebiten-sampling.json> <輸出 JSON>

抽樣規則（固定，不依執行而變）：
- 每個實驗保留重建輸入所需的參數與讀回整張 RGBA 的 SHA-256。Rust 測試照 Go 工具重建同一份輸入、
  以 CPU 畫布畫一次，再比整張讀回的雜湊；雜湊相同即每個通道值都相同。
- 另外保留「常見公式算錯的值」：階段 0 每個實驗裡，每個不是逐值相同的候選公式列出的前幾筆不符
  （inputs 與 Go 實際讀回），去重後依 (實驗, inputs) 排序。例如 alpha_sweep_black 的 d=191、a=2 → 189，
  color_scale_00 的 30×0.55 → 16。Rust 測試逐筆讀出對應像素比對，失敗時能直接看到是哪個值。
- 取樣實驗保留每軸的覆蓋範圍，失敗時列出第一個不同的目標座標。

輸出記錄兩個輸入檔的 SHA-256；輸入不變時輸出逐位元組相同。
"""
import hashlib
import json
import sys


def sha256_file(path):
    with open(path, 'rb') as f:
        return hashlib.sha256(f.read()).hexdigest()


def main():
    if len(sys.argv) != 4:
        print(__doc__, file=sys.stderr)
        return 2
    rounding_path, sampling_path, out_path = sys.argv[1:]
    rounding = json.load(open(rounding_path, encoding='utf-8'))
    sampling = json.load(open(sampling_path, encoding='utf-8'))
    if rounding.get('schema') != 'fd2_rust_r5_ebiten_rounding/1':
        raise SystemExit('階段 0 輸入的 schema 不對')
    if sampling.get('schema') != 'fd2_rust_r5_ebiten_sampling/1':
        raise SystemExit('階段 1 輸入的 schema 不對')

    blend = []
    spots = set()
    for e in rounding['experiments']:
        if not e['exact']:
            raise SystemExit(f"{e['name']} 沒有逐值相同的公式，不能當測試向量")
        params = {k: v for k, v in e['params'].items() if k != 'source'}
        if e['kind'] == 'alpha_sweep':
            params = {'rgb': e['name'].removeprefix('alpha_sweep_')}
        blend.append({'name': e['name'], 'kind': e['kind'], 'params': params, 'rgba_sha256': e['actual_rgba_sha256']})
        for c in e['candidates']:
            if c['mismatches'] == 0:
                continue
            for m in c.get('first_mismatches', []):
                spots.add((e['name'], tuple(m['inputs']), m['actual']))
    spot_list = [{'experiment': n, 'inputs': list(i), 'actual': a} for n, i, a in sorted(spots)]

    samp = []
    for s in sampling['sampling']:
        if not s['separable'] or not s['exact']:
            raise SystemExit(f"{s['name']} 沒有逐值相同的取樣規則，不能當測試向量")
        samp.append({'name': s['name'], 'src': [s['src_w'], s['src_h']], 'ops': s['geom_ops'],
                     'x_covered': s['x']['covered'], 'y_covered': s['y']['covered'], 'rgba_sha256': s['actual_rgba_sha256']})
    sem = []
    for s in sampling['semantics']:
        if s['mismatches'] != 0:
            raise SystemExit(f"{s['name']} 的讀回與預期不同，不能當測試向量")
        sem.append({'name': s['name'], 'size': [s['width'], s['height']], 'rgba_sha256': s['actual_rgba_sha256']})

    out = {
        'schema': 'fd2_game_canvas_vectors/1',
        'producer': 'tools/rust_r5_canvas_vectors.py',
        'environment': {'go': rounding['go_version'], 'ebiten': rounding['ebiten_version'],
                        'gpu': 'Xvfb 下的 Mesa llvmpipe（fd2-go-test-local）'},
        'inputs': {
            'rounding': {'file': 'work/r5-stage0-20261115/rounding/ebiten-rounding.json', 'sha256': sha256_file(rounding_path)},
            'sampling': {'file': 'work/r5-stage1-20261115/sampling/r1/ebiten-sampling.json', 'sha256': sha256_file(sampling_path)},
        },
        'sampling_target': sampling['target'],
        'blend': blend,
        'blend_spots': spot_list,
        'sampling': samp,
        'semantics': sem,
    }
    with open(out_path, 'w', encoding='utf-8') as f:
        json.dump(out, f, ensure_ascii=False, indent=1)
        f.write('\n')
    print(f'blend {len(blend)}、spots {len(spot_list)}、sampling {len(samp)}、semantics {len(sem)} → {out_path}')
    return 0


if __name__ == '__main__':
    sys.exit(main())

#!/usr/bin/env python3
"""核對同條件地圖量測，分開Go CPU樣本、Draw呼叫頻率與Windows實機驗收。"""
import argparse
import hashlib
import json
from pathlib import Path
import statistics

from verify_fd2_battle_performance import validate_rows


def sha(p):
    return hashlib.sha256(Path(p).read_bytes()).hexdigest()


def cpu_functions(path):
    result={}
    for line in path.read_text().splitlines():
        parts=line.split()
        if len(parts)>=6 and parts[1].endswith('%') and parts[4].endswith('%'):
            value=parts[3]
            assert value=='0' or value.endswith('ms')
            result[' '.join(parts[5:])]=float(value.removesuffix('ms'))
    assert 'main.(*Game).Draw' in result
    return {name:result.get(name,0) for name in ['main.(*Game).Draw',
        'main.hdTextureQuality.textureSource','main.prepareModernHUDArt']}


def load(folder,binary,tool):
    receipt=json.loads((folder/'receipt.json').read_text())
    assert receipt['status']=='PASS-OBSERVATION' and receipt['source_save_unchanged']
    assert receipt['binary_sha256']==sha(binary) and receipt['gui_tool_sha256']==sha(tool)
    rows=[json.loads(l) for l in (folder/'performance.jsonl').read_text().splitlines()]
    validate_rows(rows)
    # 操作器在完整觀測與截圖後以SIGTERM收尾；不要求Go defer的末段列。
    # CPU profile由30秒timer自行收妥，頻率比較只採完整的一秒窗。
    assert any(r.get('cpu_profile_finished') is True for r in rows)
    results=[]
    for window in receipt['windows']:
        first,last=window['first'],window['last']
        hd=window['label']=='hd-balanced'
        assert first['hd_theme'] is hd and last['hd_theme'] is hd
        assert first['node']==last['node']=='battle_ch01'
        assert not first['walking'] and not last['walking']
        selected=[r for r in rows if not r['final'] and
            window['start_elapsed_seconds']+5<=r['elapsed_seconds']<=window['end_elapsed_seconds']]
        assert len(selected)>=15
        assert all(r['graphics_library']=='OpenGL' and r['hd_texture_quality']=='balanced' for r in selected)
        duration=sum(r['interval_seconds'] for r in selected)
        result=dict(label=window['label'],windows=len(selected),seconds=duration,
            actual_fps_median=statistics.median(r['ebiten_actual_fps'] for r in selected),
            actual_fps_min=min(r['ebiten_actual_fps'] for r in selected),
            actual_fps_max=max(r['ebiten_actual_fps'] for r in selected),
            draw_hz_from_calls=sum(r['draws'] for r in selected)/duration,
            actual_tps_median=statistics.median(r['ebiten_actual_tps'] for r in selected))
        if 'map_terrain_texture_uploads_total' in selected[0]:
            result['texture_uploads']=selected[-1]['map_terrain_texture_uploads_total']-selected[0]['map_terrain_texture_uploads_total']
            result['texture_retentions']=selected[-1]['map_terrain_texture_retained_total']-selected[0]['map_terrain_texture_retained_total']
        results.append(result)
    return dict(binary_sha256=sha(binary),receipt_sha256=sha(folder/'receipt.json'),
        observations_sha256=sha(folder/'performance.jsonl'),profile_sha256=sha(folder/'world.pprof'),
        source_save_sha256=receipt['source_save_sha256'],gomaxprocs=receipt['gomaxprocs'],
        render_threads=receipt['render_threads'],window=receipt['window'],renderer=receipt['renderer'],results=results)


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ['baseline','candidate','baseline-binary','candidate-binary','tool','baseline-cpu','candidate-cpu','out']:
        p.add_argument('--'+n,type=Path,required=True)
    a=p.parse_args()
    assert not a.out.exists()
    before=load(a.baseline,a.baseline_binary,a.tool)
    after=load(a.candidate,a.candidate_binary,a.tool)
    for k in ['source_save_sha256','gomaxprocs','render_threads','window','renderer']:
        assert before[k]==after[k]
    old_cpu,new_cpu=cpu_functions(a.baseline_cpu),cpu_functions(a.candidate_cpu)
    report=dict(status='PASS-OBSERVATION-NOT-WINDOWS-ACCEPTANCE',baseline=before,candidate=after,
        cpu_cumulative_ms_30_second_profile=dict(baseline=old_cpu,candidate=new_cpu),
        texture_source_cpu_reduction_fraction=1-new_cpu['main.hdTextureQuality.textureSource']/old_cpu['main.hdTextureQuality.textureSource'],
        draw_cpu_reduction_fraction=1-new_cpu['main.(*Game).Draw']/old_cpu['main.(*Game).Draw'],
        windows_60fps_confirmed=False,verifier_sha256=sha(__file__),
        limits=['容器配額4CPU／6GiB，Xvfb／llvmpipe兩執行緒；不代表Windows硬體GPU。',
                '各觀測時間窗前五秒排除，仍有幀率波動；只報實測，不以平均值宣稱穩定60fps。',
                'CPU樣本是各30秒的累計Go CPU時間，沒有精確逐Draw延遲或GPU完成時間。'])
    a.out.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(dict(status=report['status'],texture_source_cpu_reduction=report['texture_source_cpu_reduction_fraction'],
        draw_cpu_reduction=report['draw_cpu_reduction_fraction'])))


if __name__=='__main__':main()

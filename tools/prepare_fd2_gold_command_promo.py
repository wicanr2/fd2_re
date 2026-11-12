#!/usr/bin/env python3
"""從正常指令環操作收據準備等長推廣片修訂及README實機圖。"""
import argparse
import copy
from datetime import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

def sha(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024*1024), b''):
            h.update(block)
    return h.hexdigest()


def read(path):
    return json.loads(Path(path).read_text())


def dump(path, value):
    Path(path).write_text(json.dumps(value, ensure_ascii=False, indent=2)+'\n')


def prepare(a):
    assert Path('/.dockerenv').exists() and os.getuid() != 0
    assert re.fullmatch(r'v\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}', a.version)
    datetime.strptime(a.version.rsplit('-', 1)[1], '%Y%m%d')
    assert a.out.parent.stat().st_uid == os.getuid()
    if a.phase == 'video':
        assert not a.out.exists()
    else:
        assert a.out.is_dir() and not (a.out/'SOURCE-PROOF.json').exists()
    gui = read(a.run/'receipt.json')
    recording = read(a.recording/'recording.json')
    source = read(a.source_promo/'EDIT-MANIFEST.json')
    validation = read(a.source_promo/'VALIDATION.json')
    assert gui['status'] == recording['status'] == 'PASS'
    assert gui['state_injection'] is False and gui['new_process_title_load'] is True
    assert gui['theme_and_interface_save_bytes_equal'] is True
    assert gui['binary_sha256'] == sha(a.binary)
    assert gui['catalog_sha256'] == sha(a.repo/'remake/assets/themes/modern/catalog.json')
    assert validation['status'] == 'PASS-TECHNICAL'
    assert source['movie']['sha256'] == validation['movie']['sha256'] == sha(a.source_promo/source['movie']['file'])
    assert source['duration_seconds'] == 293.9 and source['scenes'][6]['id'] == 'map-switch'
    assert source['version'] != a.version
    assert source['scenes'][6]['duration'] == 4
    assert [r['label'] for r in gui['ring_styles']] == [
        'hd-hd', 'hd-original', 'original-original', 'original-hd', 'hd-hd-restored']
    protected = lambda s: [(u['native_identity'], u['x'], u['y'], u['hp'], u['mp'], u['acted']) for u in s['units']]
    pictures = {}
    for row in gui['ring_styles']:
        name = 'ares-ring-'+row['label']+'.png'
        shot = gui['screenshots'][name]
        assert sha(a.run/name) == shot['sha256'] and shot['state']['ring']
        assert protected(row['state']) == protected(gui['arrival'])
        pictures[row['label']] = a.run/name
    for row in recording['files']:
        assert sha(a.recording/row['file']) == row['sha256']
    timing = []
    for line in (a.recording/'frame-times.jsonl').read_text().splitlines():
        row = json.loads(line)
        if timing and row['state']['frame'] < timing[-1]['state']['frame']:
            break
        timing.append(row)
    cuts = []
    for label in ['original-hd', 'hd-hd-restored']:
        frame = next(r['state']['frame'] for r in gui['ring_styles'] if r['label'] == label)
        observed = min(timing, key=lambda row: abs(row['state']['frame']-frame))
        end = observed['seconds']-.2
        assert end > 2 and observed['state']['hd_interface'] is True
        assert observed['state']['hd_theme'] is (label == 'hd-hd-restored')
        cuts.append(dict(label=label, start=end-2, end=end, frame=frame))
    video = a.out/'command-demo.mp4'
    if a.phase == 'video':
        a.out.mkdir()
        filters = ';'.join(f'[0:v]trim=start={r["start"]:.9f}:end={r["end"]:.9f},setpts=PTS-STARTPTS,fps=30[v{i}]' for i,r in enumerate(cuts))
        filters += ';[v0][v1]concat=n=2:v=1:a=0,format=yuv420p[v]'
        subprocess.run(['ffmpeg', '-nostdin', '-v', 'warning', '-threads', '2', '-filter_complex_threads', '1',
                        '-i', str(a.recording/'video.mkv'), '-filter_complex', filters,
                        '-map', '[v]', '-an', '-frames:v', '120', '-c:v', 'libx264',
                        '-preset', 'veryfast', '-crf', '16', str(video)], check=True, timeout=120)
        probe = json.loads(subprocess.check_output(['ffprobe', '-v', 'error', '-show_streams', '-of', 'json', str(video)], text=True))['streams']
        assert len(probe) == 1 and int(probe[0]['nb_frames']) == 120
        assert (probe[0]['width'], probe[0]['height']) == (1920, 1200)
        dump(a.out/'VIDEO.json', dict(status='PASS-120-VIDEO-FRAMES',cuts=cuts,
             file=video.name,sha256=sha(video),producer_sha256=sha(__file__)))
        print('PASS-120-VIDEO-FRAMES')
        return
    media = read(a.out/'VIDEO.json')
    assert media['status'] == 'PASS-120-VIDEO-FRAMES' and media['sha256'] == sha(video)
    assert media['cuts'] == cuts and media['producer_sha256'] == sha(__file__)
    from PIL import Image, ImageDraw, ImageFont

    def normalized(path):
        with Image.open(path) as image:
            assert image.size in [(1280, 800), (1920, 1200)]
            return image.convert('RGB').resize((1920, 1200), Image.Resampling.NEAREST)

    font = lambda n: ImageFont.truetype(str(a.font), n, index=3)
    old = Image.open(a.previous_terrain).convert('RGB')
    assert old.size == (1544, 1018)
    terrain = Image.new('RGB', (1544, 1017), '#101e30')
    draw = ImageDraw.Draw(terrain)
    for x, label, name in [(0, '原版美術／HD金字', 'original-hd'), (776, 'HD1像素手繪／HD金字', 'hd-hd')]:
        draw.text((x+16, 9), label, font=font(27), fill='#f1d78c')
        # Remove the transient bottom theme toast; keep all four actual buttons.
        picture = normalized(pictures[name]).crop((0, 0, 1920, 1120))
        terrain.paste(picture.resize((768, 448), Image.Resampling.LANCZOS), (x, 56))
    terrain.paste(old.crop((0, 505, 1544, 1018)), (0, 504))
    terrain.save(a.out/'hd1-pixel-terrain-normal-comparison.png', optimize=True)
    overview = Image.new('RGB', (1680, 600), '#101e30')
    draw = ImageDraw.Draw(overview)
    draw.text((24, 16), '原版金字浮雕・HD操作按鈕', font=font(38), fill='#f1d78c')
    crops = []
    for i, (label, name) in enumerate([
        ('原版美術／原版介面', 'original-original'), ('原版美術／HD金字', 'original-hd'),
        ('HD美術／原版介面', 'hd-original'), ('HD美術／HD金字', 'hd-hd')]):
        x = 16+416*i
        draw.text((x, 83), label, font=font(25), fill='#eee4ca')
        box = (1008, 600, 1536, 1080)
        picture = normalized(pictures[name]).crop(box).resize((400, 364), Image.Resampling.LANCZOS)
        overview.paste(picture, (x, 132))
        crops.append(dict(source=str(pictures[name]), source_sha256=sha(pictures[name]), normalize=[1920,1200], crop=list(box)))
    draw.text((24, 532), '同份首章進度正常LOAD、走行與切換；圖片僅裁切拼排。', font=font(27), fill='#adbed0')
    overview.save(a.out/'modern-battle-ui-overview.png', optimize=True)
    proof = dict(status='READY-CURRENT-GOLD-COMMAND-EDIT', binary_sha256=sha(a.binary),
        catalog_sha256=gui['catalog_sha256'], source_save_sha256=gui['source_sha256'],
        gui_receipt_sha256=sha(a.run/'receipt.json'), recording_receipt_sha256=sha(a.recording/'recording.json'),
        recording_video_sha256=sha(a.recording/'video.mkv'), cuts=cuts, command_demo_sha256=sha(video),
        previous_terrain=dict(file=str(a.previous_terrain),sha256=sha(a.previous_terrain)),
        font=dict(file=str(a.font),sha256=sha(a.font)),
        previous_lower_panel_retained=True, overview_crops=crops, producer_sha256=sha(__file__),
        figures=[dict(file=name,sha256=sha(a.out/name)) for name in ['hd1-pixel-terrain-normal-comparison.png','modern-battle-ui-overview.png']],
        runtime_state_injection=False, cold_load_verified=True,
        scope='同次正常錄影的兩段對照剪輯，各2秒；其他42幕、字幕及全片時長保持。README使用實際四組操作圖，不繪製假按鈕。')
    dump(a.out/'SOURCE-PROOF.json', proof)
    plan = copy.deepcopy(source)
    for key in ['movie', 'base_movie', 'segments', 'narration', 'audio_repair']:
        plan.pop(key, None)
    plan.update(version=a.version, status='CURRENT-GOLD-COMMAND-READY',
                scope=proof['scope'], gold_command_refresh=dict(file=str(a.out/'SOURCE-PROOF.json'),sha256=sha(a.out/'SOURCE-PROOF.json')))
    scene = plan['scenes'][6]
    scene.update(source=str(video), start=0, source_duration=4, minimum_playback_rate=1,
                 source_sha256=sha(video), capture_receipt=str(a.run/'receipt.json'),
                 capture_scope='原版美術與HD美術均採目前HD金字介面；同次正常LOAD／走行／切換的兩段實錄。',
                 recording_receipt=str(a.recording/'recording.json'),refresh='current-gold-command')
    inputs = {row['path']: row for row in plan['inputs']}
    for path in [video, a.out/'SOURCE-PROOF.json', a.run/'receipt.json', a.recording/'recording.json',
                 a.recording/'frame-times.jsonl', a.binary, Path(__file__)]:
        name = str(path)
        entry = dict(path=name, sha256=sha(path))
        assert name not in inputs or inputs[name] == entry
        inputs[name] = entry
    plan['inputs'] = list(inputs.values())
    dump(a.out/'PLAN.json', plan)
    print(json.dumps(dict(status=proof['status'], cuts=cuts, figures=proof['figures']),ensure_ascii=False))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument('--version', required=True)
    parser.add_argument('--phase', choices=['video', 'figures'], required=True,
                        help='video用既有FFmpeg映像，figures用既有Pillow映像，兩階段共用明示out目錄')
    for name in ['run','recording','binary','source-promo','previous-terrain','font','out']:
        parser.add_argument('--'+name, type=Path, required=True)
    prepare(parser.parse_args())

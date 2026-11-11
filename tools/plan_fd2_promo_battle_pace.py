#!/usr/bin/env python3
"""依錄影觀測的遊戲拍數校正戰鬥剪輯，音樂與旁白保持原速。"""
import argparse
import bisect
import copy
import json
import math
import os
from pathlib import Path
import re
import subprocess

from build_fd2_hd1_promo import digest


def load(path):
    return json.loads(Path(path).read_text())


def write(path, value):
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n')


def interpolate(rows, frame):
    frames = [r['state']['frame'] for r in rows]
    at = bisect.bisect_left(frames, frame)
    assert 0 < at < len(rows), ('拍點超出實錄', frame)
    a, b = rows[at - 1], rows[at]
    ratio = (frame - a['state']['frame']) / (b['state']['frame'] - a['state']['frame'])
    return a['seconds'] + ratio * (b['seconds'] - a['seconds'])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['previous', 'voice-plan', 'audio-plan', 'out']:
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--version', required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid() != 0
    assert not args.out.exists() and args.out.parent.stat().st_uid == os.getuid()
    assert re.fullmatch(r'v\.\d+\.\d+\.\d+-\d{8}', args.version)
    plan = load(args.previous)
    previous = copy.deepcopy(plan)
    assert plan['duration_seconds'] == 313 and len(plan['scenes']) == 43
    voice = load(args.voice_plan)
    audio = load(args.audio_plan)
    durations = {'gaia-hd': 2.5, 'ares': 4.5, 'yuni': 4.4, 'middy': 2.5, 'tino': 3.5, 'rodman': 2.5}
    identities = {'gaia-hd': (30, None), 'ares': (4, None), 'yuni': (9, 0),
                  'middy': (18, 30), 'tino': (2, 29), 'rodman': (19, 28)}
    old_offsets = {}; new_offsets = {}; clocks = {}; cursor = 0
    for scene in plan['scenes']:
        old_offsets[scene['id']] = cursor
        cursor += scene['duration']
    inputs = {r['path']: r['sha256'] for r in plan['inputs']}
    for scene in plan['scenes']:
        name = scene['id']
        if name not in durations:
            continue
        rows = [json.loads(line) for line in Path(scene['timing']).read_text().splitlines()]
        assert all(a['seconds'] < b['seconds'] and a['state']['frame'] < b['state']['frame']
                   for a, b in zip(rows, rows[1:]))
        identity, command = identities[name]
        def active(row):
            s = row['state']
            return (s.get('battle_scene') and s.get('battle_actor_fig') == identity) if command is None else (
                s.get('command_scene') and s.get('command_id') == command)
        action = [r for r in rows if active(r)]
        assert len(action) >= 4
        first_frame = action[0]['state']['frame'] - 18
        last_frame = first_frame + round(durations[name] * 60)
        audit = Path(scene['capture_receipt']).parent / 'input-audit.jsonl'
        raw_all = [json.loads(line) for line in audit.read_text().splitlines()]
        source_probe = json.loads(subprocess.check_output(['ffprobe', '-v', 'error', '-show_format',
            '-of', 'json', scene['source']], text=True, timeout=30))
        # X11編碼佇列可能未保存最後幾秒的等待；不可把操作紀錄當成影片長度。
        available = [r for r in rows if r['seconds'] <= float(source_probe['format']['duration'])-.05]
        if available[-1]['state']['frame'] < last_frame:
            last_frame = available[-1]['state']['frame']
            first_frame = last_frame - round(durations[name]*60)
        preceding_hd = []
        for r in reversed([r for r in raw_all if r['frame'] <= action[0]['state']['frame']]):
            if not r.get('hd_theme'):
                break
            preceding_hd.append(r['frame'])
        assert preceding_hd
        first_frame = max(first_frame, math.ceil(min(preceding_hd)/2)*2)
        hold = round(durations[name]-(last_frame-first_frame)/60, 12)
        assert 0 <= hold <= .1, ('來源不足以保留旁白及完整動作', name, hold)
        assert last_frame >= action[-1]['state']['frame'], '不得切掉攻擊收尾'
        start = interpolate(rows, first_frame)
        end = interpolate(rows, last_frame)
        observed = [r for r in rows if first_frame < r['state']['frame'] < last_frame]
        assert all(r['state'].get('hd_theme') for r in observed)
        raw = [r for r in raw_all if first_frame <= r['frame'] <= last_frame]
        assert raw and all(r.get('hd_theme') and r['game_speed'] == 1 for r in raw)
        points = [dict(source_seconds=0., movie_seconds=0., game_frame=first_frame)]
        points += [dict(source_seconds=r['seconds'] - start,
                        movie_seconds=(r['state']['frame'] - first_frame) / 60,
                        game_frame=r['state']['frame']) for r in observed]
        points += [dict(source_seconds=end - start, movie_seconds=(last_frame-first_frame)/60, game_frame=last_frame)]
        assert all(math.isfinite(p['source_seconds']) for p in points)
        assert digest(scene['source']) == scene['source_sha256']
        clocks[name] = dict(first_frame=first_frame, last_frame=last_frame,
            action_frames=action[-1]['state']['frame'] - action[0]['state']['frame'],
            previous_seconds=scene['duration'], seconds=durations[name], source_seconds=end-start,
            knots=len(points), normal_update_hz=60, tail_hold_seconds=hold)
        scene.update(start=start, source_duration=end-start, duration=durations[name], logical_clock=points,
                     clock_scope='依已擷取畫面與60拍觀測重定時；無補畫、無即時效能聲明。')
        if hold:
            scene['logical_hold_seconds'] = hold
    cursor = 0
    for scene in plan['scenes']:
        new_offsets[scene['id']] = cursor
        cursor += scene['duration']
    assert abs(cursor-293.9) < 1e-8
    old_scenes = {s['id']: s for s in previous['scenes']}
    new_scenes = {s['id']: s for s in plan['scenes']}
    for row in voice['segments']:
        name = row['id']
        assert name in old_scenes
        relative = row['start_seconds'] - old_offsets[name]
        assert relative >= 0
        if name == 'yuni':
            # 4.4秒畫面容納原3.996063秒語音；只將開口提早0.05秒。
            relative = .2
        row['start_seconds'] = new_offsets[name] + relative
        if name in durations:
            row['window_seconds'] = new_scenes[name]['duration'] - relative - .2
    voice['duration_seconds'] = cursor
    voice['timing_reference'] = dict(file=str(args.voice_plan), sha256=digest(args.voice_plan),
        policy='只改開始位置與時窗；36段文字、來源聲音與生成語速不變。')
    audio.update(version=args.version, duration_seconds=cursor, source_movie_sha256='SET-AFTER-VIDEO-ENCODE')
    for cue in audio['cues']:
        name = cue['scene']
        if name in clocks:
            local = (cue['observed_game_frame'] - clocks[name]['first_frame']) / 60
            assert 0 <= local < new_scenes[name]['duration']
            cue['at_seconds'] = new_offsets[name] + local
            cue['synchronization'] = '依觀測遊戲拍除以60定位；保留擷取與繪製觀測誤差，不宣稱硬體時序。'
        else:
            cue['at_seconds'] += new_offsets[name] - old_offsets[name]
        assert cue['playback_rate'] == 1
    for path in [args.previous, args.voice_plan, args.audio_plan, Path(__file__)]:
        inputs[str(path)] = digest(path)
    for key in ['movie', 'segments', 'narration', 'base_movie', 'previous_audio_movie', 'audio_repair']:
        plan.pop(key, None)
    plan.update(version=args.version, duration_seconds=cursor,
        inputs=[dict(path=p, sha256=h) for p, h in sorted(inputs.items())],
        pace_correction=dict(status='NORMAL-60-UPDATE-CLOCK', previous_manifest_sha256=digest(args.previous),
            previous_duration_seconds=313, changed_scenes=clocks,
            reference_voice_plan=str(args.voice_plan), reference_voice_plan_sha256=digest(args.voice_plan),
            limits=['保留目前觀測到的畫面，不生成遺失動作圖格。', '不代表Windows實機60fps。']))
    args.out.mkdir()
    write(args.out/'PLAN.json', plan)
    write(args.out/'AUDIO-PLAN.json', audio)
    write(args.out/'VOICE-PLAN.json', voice)
    write(args.out/'PACE-REPORT.json', clocks)
    print(json.dumps(dict(duration_seconds=cursor, changed_scenes=clocks), ensure_ascii=False))


if __name__ == '__main__':
    main()

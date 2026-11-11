#!/usr/bin/env python3
"""依正常操作錄影替換蓋亞兩幕，逐格核對其他畫面保持。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import struct
import subprocess


def sha(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def run(command, timeout=240):
    return subprocess.run(command, check=True, capture_output=True, text=True, timeout=timeout)


def load(path):
    return json.loads(Path(path).read_text())


def check_capture(clip, expected_save):
    receipt = load(clip['recording_receipt'])
    action = load(clip['action_receipt'])
    assert receipt['status'] == 'PASS' and receipt['verifier_exit_code'] == 0
    assert action['status'] == 'PASS-NORMAL-KEYBOARD-PROMO-ACTION'
    assert action['actor_identity'] == 30 and action['command_id'] is None
    assert action['runtime_state_injection'] is False and action['cold_load_verified'] is False
    assert action['source_save_sha256'] == expected_save == sha(clip['save'])
    assert action['binary_sha256'] == clip['binary_sha256'] == sha(clip['binary'])
    hd = clip['id'] == 'gaia-hd'
    assert action['action_art_theme'] == ('hd' if hd else 'faithful')
    scenes = action['battle_hd_scenes' if hd else 'battle_original_scenes']
    assert len(scenes) == 1
    presented = scenes[0]['presented']
    assert presented['battle_actor_fig'] == 30 and presented['battle_target_fig'] == 96
    assert presented['hd_theme'] is hd and presented['hd_interface'] is hd
    assert clip['duration'] == (7 if hd else 4)
    assert clip['start'] >= 0 and clip['source_duration'] > 0
    window = Path(clip['window_capture'])
    assert sha(window) == clip['window_capture_sha256']
    width, height = struct.unpack('>II', window.read_bytes()[16:24])
    assert clip['window_rect'] == [0, 0, width, height]
    assert width * 5 == height * 8 and width <= 1920 and height <= 1200
    assert action['screenshots'][window.name]['sha256'] == sha(window)
    source = Path(clip['source'])
    files = {row['file']: row for row in receipt['files']}
    assert files[source.name]['sha256'] == clip['source_sha256'] == sha(source)
    rows = [json.loads(line) for line in Path(clip['timing']).read_text().splitlines()]
    visible = [row['state'] for row in rows
               if clip['start'] <= row['seconds'] <= clip['start'] + clip['source_duration']
               and row['state'].get('battle_actor_fig') is not None]
    assert visible and all(row['battle_actor_fig'] == 30 and row['battle_target_fig'] == 96
                           and row['hd_theme'] is hd and row['hd_interface'] is hd for row in visible)
    assert len(visible) >= 3, '蓋亞演出的可見角色觀測不足'
    return dict(id=clip['id'], actor_identity=30, actor_fig=30, target_fig=96,
                hd_art=hd, hd_interface=hd, visible_observations=len(visible),
                window_rect=clip['window_rect'], window_capture_sha256=sha(window),
                source_sha256=sha(source), recording_receipt_sha256=sha(clip['recording_receipt']),
                action_receipt_sha256=sha(clip['action_receipt']),
                timing_sha256=sha(clip['timing']), normal_return=True)


def check_plan(plan):
    assert plan['schema'] == 'fd2-promo-gaia-repair-v1'
    assert re.fullmatch(r'v\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}', plan['version'])
    assert [row['id'] for row in plan['clips']] == ['gaia-original', 'gaia-hd']
    assert (plan['replace_start'], plan['replace_end'], plan['duration_seconds']) == (82, 93, 313)
    assert sha(plan['previous_movie']) == plan['previous_movie_sha256']
    previous = load(plan['previous_manifest'])
    assert previous['movie']['sha256'] == plan['previous_movie_sha256']
    assert len(previous['scenes']) == 43 and previous['duration_seconds'] == 313
    cursor = 0
    for scene in previous['scenes']:
        if scene['id'] == 'gaia-original':
            assert cursor == 82 and scene['duration'] == 4
        if scene['id'] == 'gaia-hd':
            assert cursor == 86 and scene['duration'] == 7
        cursor += scene['duration']
    return [check_capture(clip, plan['source_save_sha256']) for clip in plan['clips']]


def frame_hashes(movie, output):
    run(['ffmpeg', '-nostdin', '-v', 'error', '-threads', '2', '-i', str(movie),
         '-map', '0:v:0', '-an', '-f', 'framemd5', '-y', str(output)])
    return read_frame_hashes(output)


def read_frame_hashes(path):
    return [line.rsplit(',', 1)[1].strip() for line in path.read_text().splitlines()
            if line and not line.startswith('#')]


def prepare(plan, work, font):
    from build_fd2_hd1_promo import prepare as overlays
    assert font.is_file() and sha(font) == plan['font_sha256']
    overlays({'scenes': plan['clips']}, work, font)


def build(plan, work, out, evidence):
    assert work.is_dir() and not out.exists() and out.parent.stat().st_uid == os.getuid()
    previous = Path(plan['previous_movie'])
    probe = json.loads(run(['ffprobe', '-v', 'error', '-select_streams', 'v:0',
                            '-show_entries', 'stream=start_time', '-of', 'json', str(previous)]).stdout)
    video_offset = float(probe['streams'][0]['start_time'])
    common = ['ffmpeg', '-nostdin', '-v', 'warning', '-threads', '2',
              '-filter_threads', '1', '-filter_complex_threads', '1']
    prefix, suffix = work/'prefix.mp4', work/'suffix.mp4'
    for path in [prefix, suffix, work/'stitched.mp4']:
        assert not path.exists()
    run(common + ['-i', str(previous), '-map', '0:v:0', '-an', '-frames:v', '2460',
                  '-c:v', 'copy', '-output_ts_offset', str(-video_offset), str(prefix)])
    run(common + ['-ss', f'{93+video_offset:.6f}', '-i', str(previous), '-map', '0:v:0', '-an',
                  '-frames:v', '6600', '-c:v', 'copy', str(suffix)])
    segments = []
    for index, clip in enumerate(plan['clips']):
        segment = work/f'gaia-{index}.mp4'
        assert not segment.exists()
        rate = clip['source_duration']/clip['duration']
        assert .25 <= rate <= 32
        x, y, width, height = clip['window_rect']
        filters = (f'[0:v]trim=duration={clip["source_duration"]},'
                   f'setpts=(PTS-STARTPTS)/{rate:.12g},fps=30,'
                   f'crop={width}:{height}:{x}:{y},'
                   'scale=1536:960:flags=lanczos,pad=1920:1080:192:20:color=0x08121e,'
                   f'setsar=1[game];[game][1:v]overlay=shortest=1,format=yuv420p[v]')
        run(common + ['-ss', str(clip['start']), '-i', clip['source'],
                      '-loop', '1', '-framerate', '30', '-i', str(work/f'overlay-{index:02}.png'),
                      '-filter_complex', filters, '-map', '[v]', '-an',
                      '-frames:v', str(clip['duration']*30), '-c:v', 'libx264',
                      '-preset', 'veryfast', '-crf', '18', '-threads', '2', str(segment)])
        segments.append(segment)
    listing = work/'concat.txt'
    listing.write_text(''.join("file '" + str(path.resolve()) + "'\n"
                               for path in [prefix, *segments, suffix]))
    stitched = work/'stitched.mp4'
    run(common + ['-f', 'concat', '-safe', '0', '-i', str(listing), '-an',
                  '-c:v', 'copy', str(stitched)])
    baseline = plan.get('decoded_baseline')
    if baseline:
        assert baseline['movie_sha256'] == sha(previous)
        assert baseline['sha256'] == sha(baseline['file'])
        old = read_frame_hashes(Path(baseline['file']))
    else:
        old = frame_hashes(previous, work/'previous.framemd5')
    new = frame_hashes(stitched, work/'stitched.framemd5')
    expected = [value for i, segment in enumerate(segments)
                for value in frame_hashes(segment, work/f'gaia-{i}.framemd5')]
    assert len(old) == len(new) == 9390 and len(expected) == 330
    assert old[:2460] == new[:2460] and old[2790:] == new[2790:], '替換範圍外畫面改變'
    assert new[2460:2790] == expected, '蓋亞兩幕未完整保留'
    assert new[2460:2790] != old[2460:2790], '未真正替換錯誤鏡頭'
    out.mkdir()
    movie = out/f'FD2-{plan["version"]}-hd1-promo.mp4'
    # 暫用既有音軌；正式底音與旁白由既有分軌工具重新混入。
    run(common + ['-i', str(stitched), '-i', str(previous), '-map', '0:v:0', '-map', '1:a:0',
                  '-c', 'copy', '-t', '313', '-movflags', '+faststart', str(movie)])
    manifest = load(plan['previous_manifest'])
    manifest['version'] = plan['version']
    manifest['previous_movie'] = dict(manifest['movie'], path=str(previous))
    manifest['movie'] = dict(file=movie.name, bytes=movie.stat().st_size, sha256=sha(movie))
    manifest['status'] = 'GAIA-VIDEO-REPAIRED-AWAITING-AUDIO'
    manifest.pop('narration', None)
    manifest['scenes'] = [next((clip for clip in plan['clips'] if clip['id'] == scene['id']), scene)
                          for scene in manifest['scenes']]
    manifest['gaia_repair'] = dict(plan_sha256=plan['_plan_sha256'], evidence=evidence,
                                  unchanged_frames=9060, replaced_frames=330,
                                  outside_region_pixels_equal=True, narration_reused=True)
    (out/'EDIT-MANIFEST.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2)+'\n')
    proof = dict(status='PASS-GAIA-VIDEO-REPLACEMENT', captures=evidence,
                 frames=9390, replaced_frames=330, unchanged_frames=9060,
                 outside_region_pixels_equal=True, replacement_pixels_equal=True,
                 previous_video_start_seconds=video_offset,
                 binary_sha256=plan['clips'][0]['binary_sha256'],
                 source_save_sha256=plan['source_save_sha256'], producer_sha256=sha(__file__))
    (out/'GAIA-VIDEO-PROOF.json').write_text(json.dumps(proof, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(proof, ensure_ascii=False))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['check', 'prepare', 'build'])
    parser.add_argument('--plan', type=Path, required=True)
    parser.add_argument('--work', type=Path)
    parser.add_argument('--out', type=Path)
    parser.add_argument('--font', type=Path)
    args = parser.parse_args()
    assert Path('/.dockerenv').exists() and os.getuid() != 0
    plan = load(args.plan)
    plan = plan.get('promo_gaia_repair_20261111', {}).get('plan', plan)
    evidence = check_plan(plan)
    plan['_plan_sha256'] = hashlib.sha256(json.dumps(
        plan, ensure_ascii=False, sort_keys=True, separators=(',', ':')).encode()).hexdigest()
    if args.action == 'prepare':
        assert args.work and args.font
        prepare(plan, args.work, args.font)
    elif args.action == 'build':
        assert args.work and args.out
        build(plan, args.work, args.out, evidence)
    else:
        print(json.dumps(evidence, ensure_ascii=False))


if __name__ == '__main__':
    main()

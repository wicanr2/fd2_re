#!/usr/bin/env python3
"""在Docker內驗證推廣片規格、音量、黑幀、凍結及字幕安全區。"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('promo',type=Path)
    parser.add_argument('--caption-bounds',type=Path,required=True)
    args=parser.parse_args()
    assert Path('/.dockerenv').exists()
    root=args.promo
    manifest=json.loads((root/'EDIT-MANIFEST.json').read_text())
    movie=root/manifest['movie']['file']
    assert hashlib.sha256(movie.read_bytes()).hexdigest()==manifest['movie']['sha256']
    probe=json.loads(subprocess.check_output(['ffprobe','-v','error','-show_format','-show_streams','-of','json',str(movie)]))
    video=next(s for s in probe['streams'] if s['codec_type']=='video')
    audio=next(s for s in probe['streams'] if s['codec_type']=='audio')
    duration=float(probe['format']['duration'])
    assert abs(duration-manifest['duration_seconds'])<.1
    assert video['codec_name']=='h264' and (video['width'],video['height'])==(1920,1080) and video['avg_frame_rate']=='30/1'
    assert int(video['nb_frames'])==manifest['duration_seconds']*30
    assert audio['codec_name']=='aac' and audio['channels']==2 and audio['sample_rate']=='48000'
    run=subprocess.run(['ffmpeg','-nostdin','-hide_banner','-threads','2','-i',str(movie),
                        '-vf','blackdetect=d=0.2:pix_th=0.05:pic_th=0.98,freezedetect=n=-50dB:d=3',
                        '-af','volumedetect,silencedetect=n=-50dB:d=1','-f','null','-'],capture_output=True,text=True,timeout=240)
    assert run.returncode==0
    (root/'quality.log').write_text(run.stderr)
    mean=float(re.search(r'mean_volume: ([\d.-]+) dB',run.stderr)[1])
    peak=float(re.search(r'max_volume: ([\d.-]+) dB',run.stderr)[1])
    assert -40<mean and peak<-.5
    assert 'black_start:' not in run.stderr and 'silence_start:' not in run.stderr
    allowed=[];cursor=0
    for scene in manifest['scenes']:
        if scene['kind'] in ('card','still') or scene.get('intentional_static') or scene.get('group') in ['balance_editor','skill_hit']:
            allowed.append((cursor,cursor+scene['duration'],scene.get('group',scene['kind'])))
        cursor+=scene['duration']
    freezes=[];start=None
    for match in re.finditer(r'freeze_(start|end): ([\d.]+)',run.stderr):
        if match[1]=='start':start=float(match[2])
        elif start is not None:freezes.append((start,float(match[2])));start=None
    if start is not None:freezes.append((start,duration))
    classified=[]
    for start,end in freezes:
        owner=next((kind for a,b,kind in allowed if a-.1<=start and end<=b+.1),None)
        assert owner is not None,('非預期凍結',start,end)
        classified.append({'start':start,'end':end,'intentional_static':owner})
    bounds=json.loads(args.caption_bounds.read_text())
    for label in bounds['labels']:
        x,y,xx,yy=label['box'];assert 72<=x<xx<=1848 and 32<=y<yy<=1048
    (root/'CAPTION-BOUNDS.json').write_text(json.dumps(bounds,ensure_ascii=False,indent=2)+'\n')
    (root/'ffprobe.json').write_text(json.dumps(probe,ensure_ascii=False,indent=2)+'\n')
    report={'schema_version':1,'status':'PASS-TECHNICAL','version':manifest['version'],'movie':manifest['movie'],
            'duration_seconds':duration,'frames':int(video['nb_frames']),'canvas':[1920,1080],'fps':30,
            'audio':{'codec':'AAC','sample_rate':48000,'channels':2,'mean_db':mean,'peak_db':peak,'silent_intervals':[]},
            'black_intervals':[],'freeze_intervals':classified,'caption_safe_bounds':'PASS',
            'validator_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
            'limits':['音訊為原遊戲來源；既有FM渲染出處不完整，不宣稱硬體波形一致。','技術音量檢查不等於人耳試聽；視覺接觸表另行檢視。']}
    (root/'VALIDATION.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(report,ensure_ascii=False))


if __name__=='__main__':
    main()

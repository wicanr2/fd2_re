#!/usr/bin/env python3
"""觀測實際計時對話GPU，核對開收框之外仍為同份HD世界。"""
import hashlib
import json
from pathlib import Path
import subprocess
import threading
from verify_fd2_hd_gui import latest


class TimedDialogueProbe:
    def __init__(self,out,audit,capture,env):
        self.out,self.audit,self.capture,self.env=out,audit,capture,env
        self.stop=threading.Event();self.samples=[];self.rejected=[];self.errors=[]
        self.thread=threading.Thread(target=self.run,daemon=True)
        self.source_sha=hashlib.sha256(Path(__file__).read_bytes()).hexdigest()

    def run(self):
        seen=set()
        first_draw={}
        try:
            while not self.stop.is_set():
                before=latest(self.audit)
                owner=before.get('timed_dialogue_owner');phase=before.get('hd_timed_phase')
                token=(owner,phase,before.get('hd_timed_index'))
                if not owner or not before.get('hd_timed_dialogue') or not before.get('hd_theme') or token in seen:
                    self.stop.wait(.005);continue
                boundary=(owner,phase)
                draw=before.get('hd_world_frame',-1)
                first_draw.setdefault(boundary,draw)
                # The preceding FIGANI restore can still be the displayed
                # surface at the first audit. Do not compare that boundary.
                if (phase==0 and draw<first_draw[boundary]+2) or (phase==1 and before.get('hd_text_runs',0)==0):
                    self.stop.wait(.005);continue
                seen.add(token)
                # Uncompressed capture keeps PNG encoding out of the brief
                # native closing interval. Encode losslessly after sampling.
                path=self.out/('timed-gpu-'+str(len(seen))+'.ppm')
                self.capture(path);after=latest(self.audit)
                sample={'file':path.name,'sha256':hashlib.sha256(path.read_bytes()).hexdigest(),'before':before,'after':after}
                if (after.get('timed_dialogue_owner')==owner and after.get('hd_timed_dialogue') and after.get('hd_theme')
                    and after.get('hd_timed_phase')==phase and before.get('hd_world_units')==after.get('hd_world_units')):
                    self.samples.append(sample)
                else:self.rejected.append(sample)
        except Exception as e:self.errors.append(str(e))

    def finish(self,validate=False):
        self.stop.set()
        if self.thread.ident is not None:self.thread.join(timeout=15)
        assert not self.thread.is_alive()
        for sample in self.samples+self.rejected:
            raw=self.out/sample['file']
            if raw.suffix!='.ppm':continue
            target=raw.with_suffix('.png')
            subprocess.run(['convert',str(raw),str(target)],env=self.env,check=True,capture_output=True,timeout=15)
            sample['capture_sha256']=sample['sha256']
            sample['file']=target.name
            sample['sha256']=hashlib.sha256(target.read_bytes()).hexdigest()
            sample['capture_format']='PPM, then lossless PNG after state observation'
            raw.unlink()
        result={'samples':self.samples,'rejected':self.rejected,'errors':self.errors,'source_sha256':self.source_sha,
                'recording_created':False,'opening_warmup_frame_distance':2,
                'scope':'actual remake GPU; no original oracle; indices may advance within one phase'}
        if validate:
            assert not self.errors,self.errors
            comparisons=[];phases=set()
            for sample in self.samples:
                owner=sample['before']['timed_dialogue_owner'];phase=sample['before']['hd_timed_phase']
                if phase==1:continue
                refs=[s for s in self.samples if s['before']['timed_dialogue_owner']==owner and s['before']['hd_timed_phase']==1]
                if not refs:continue
                ref=refs[0]
                for s in (sample,ref):
                    shape=subprocess.check_output(['identify','-format','%wx%h',str(self.out/s['file'])],env=self.env,text=True)
                    assert shape=='1920x1200',shape
                roi='1872x642+24+24'
                r=subprocess.run(['compare','-metric','AE',str(self.out/ref['file'])+'['+roi+']',
                                  str(self.out/sample['file'])+'['+roi+']','null:'],env=self.env,capture_output=True,text=True)
                assert r.returncode in (0,1),r.stderr
                difference=float(r.stderr.strip());assert difference==0,(sample['file'],difference)
                comparisons.append({'reference':ref['file'],'target':sample['file'],'phase':phase,'crop':roi,'different_pixels':difference})
                phases.add(phase)
            assert {0,2}<=phases,'沒有同一處理器的有效開框／收框GPU'
            assert any(s['before'].get('hd_text_runs',0)>0 for s in self.samples if s['before']['hd_timed_phase']==1)
            result.update(status='PASS',outside_dialogue_gpu=comparisons)
        (self.out/'timed-dialogue-observation.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
        return result

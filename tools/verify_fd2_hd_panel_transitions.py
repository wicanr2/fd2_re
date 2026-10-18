"""正常GUI的唯讀區域擷取；CPU定位候選，實際像素與移動後HD穩定參照裁切比較。"""
import hashlib
import json
from pathlib import Path
import subprocess
import threading


class PanelTransitionProbe:
    def __init__(self, root, state, command, window, env):
        self.root, self.state, self.command, self.window, self.env = root, state, command, window, env
        self.stop = threading.Event()
        self.thread = None
        self.samples, self.errors = {}, []

    @staticmethod
    def eligible(state):
        if not state.get('hd_theme') or state.get('help_visible'):
            return None
        p = state.get('panel_transition') or {}
        if not p:
            return None
        index, kind = p['index'], p['kind']
        if kind == 2:
            step = index if index < 7 else 13-index
            return 'bottom' if 1 <= step <= 5 else None
        frame = index
        if p['domain'] != 'item' and kind == 0:
            frame = 11-index
        return 'upper' if 4 <= frame <= 8 else None

    def start(self):
        self.thread = threading.Thread(target=self.capture, daemon=True)
        self.thread.start()

    def capture(self):
        seen = set()
        try:
            while not self.stop.is_set() and len(seen) < 40:
                before = self.state()
                region = self.eligible(before)
                p = before.get('panel_transition') or {}
                token = (p.get('job'), p.get('kind'), p.get('index'), p.get('closing'))
                if not region or token in seen:
                    self.stop.wait(.005)
                    continue
                seen.add(token)
                name = 'panel-transition-'+str(len(seen))+'.png'
                roi = '108x558+942+0' if region == 'upper' else '600x636+210+564'
                self.command('import', '-window', self.window(), '-crop', roi, '+repage', str(self.root/name))
                self.samples[name] = {'before': before, 'after': self.state(), 'region': region, 'window_crop': roi,
                                      'sha256': hashlib.sha256((self.root/name).read_bytes()).hexdigest(),
                                      'kind': 'actual_gpu_window_region', 'exact_frame_claimed': False}
        except Exception as error:
            self.errors.append(str(error))

    def finish(self):
        self.stop.set()
        if self.thread:
            self.thread.join(timeout=20)
            if self.thread.is_alive():
                self.errors.append('面板擷取未結束')

    def rgb(self, path, box):
        x, y, width, height = box
        roi = f'{width}x{height}+{x}+{y}'
        return subprocess.check_output(['convert', str(path), '-crop', roi, '+repage', '-depth', '8', 'RGB:-'], env=self.env)

    def validate(self, items=False):
        self.finish()
        assert not self.errors, self.errors
        accepted, rejected = {}, []
        for name, meta in self.samples.items():
            p = meta['before']['panel_transition']
            if items:
                anchors = ['item-short-sword.png']
            elif p.get('identity') == 9:
                anchors = ['church-status-yuni.png', 'church-commands-yuni.png']
            else:
                anchors = ['church-status-sol.png']
            found = None
            for anchor in anchors:
                if not (self.root/anchor).exists():
                    continue
                for phase in range(4, 9) if meta['region'] == 'upper' else range(1, 6):
                    if meta['region'] == 'upper':
                        # NativeItemPanelFrameFor: sourceY=16*frame-48,
                        # destY=0 and height=141-16*frame for moving upper passes.
                        source_box = (942, (16*phase-48)*6, 108, (141-16*phase)*6)
                        dest_box = (0, 0, 108, source_box[3])
                        steady_box = (942, 0, 108, source_box[3])
                    else:
                        # nativeChurchBottomPanelRegions: sourceY=94,
                        # destY=94+16*step, clipped at native row200.
                        height = (106-16*phase)*6
                        source_box = (210, 564, 600, height)
                        dest_box = (0, 16*phase*6, 600, height)
                        steady_box = (210, 564+16*phase*6, 600, height)
                    expected = self.rgb(self.root/anchor, source_box)
                    actual = self.rgb(self.root/name, dest_box)
                    steady = self.rgb(self.root/anchor, steady_box)
                    colors = len(set(zip(expected[::3], expected[1::3], expected[2::3])))
                    if actual == expected and actual != steady and colors > 16:
                        found = {'anchor': anchor, 'matched_native_phase': phase,
                                 'source_rectangle': source_box, 'sample_rectangle': dest_box,
                                 'rgb_equal': True, 'differs_from_stable_position': True, 'reference_rgb_colors': colors}
                        break
                if found:
                    break
            if found:
                accepted[name] = {**meta, 'gpu_match': found}
            else:
                rejected.append({'file': name, **meta})
        self.save(accepted, rejected)
        assert any(x['region'] == 'upper' for x in accepted.values()), '未取得移動中上方面板的HD像素'
        if not items:
            assert any(x['region'] == 'bottom' for x in accepted.values()), '未取得狀態到命令的下框HD像素'
        return {'accepted': accepted, 'rejected': rejected, 'gpu_validation_performed': True,
                'scope': '穩定HD實際視窗依既有來源到目的座標裁切比對；CPU前後狀態照實保留，不宣稱精確同幀。'}

    def save(self, accepted=None, rejected=None):
        (self.root/'panel-transition-observation.json').write_text(json.dumps(
            {'samples': self.samples, 'accepted': accepted or {}, 'rejected': rejected or [], 'errors': self.errors},
            ensure_ascii=False, indent=2)+'\n')

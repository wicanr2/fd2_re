#!/usr/bin/env python3
"""以完整原始函式核對AI暫存值存放及16-bit清零，不以區域相同收件。"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tempfile


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('objects','linked','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    report=json.loads((a.linked/'report.json').read_text());image=json.loads(a.evidence.read_text());raw=a.original.read_bytes()
    assert hashlib.sha256(a.evidence.read_bytes()).hexdigest()==report['evidence_sha256']
    assert hashlib.sha256(raw).hexdigest()==report['input']['sha256']==image['input']['sha256']
    assert report['matched_addresses']==['0x1598a','0x35bba'] and len(report['trials'])==72
    for t in (t for t in report['trials'] if t['exact_interval_bytes']):
        f=next(f for f in image['functions'] if f['inventory']['start']==t['address'])
        original=b''.join(bytes.fromhex(i['loaded_bytes']) for c in f['chunks'] for i in c['instructions'])
        for c in f['chunks']:
            for i in c['instructions']:
                offset=int(i['file_offset'],16);data=bytes.fromhex(i['file_bytes']);assert raw[offset:offset+len(data)]==data
        assert (a.linked/t['stem']/'candidate.bin').read_bytes()==original and len(original)==f['inventory']['size']
    def code(macro):
        t=next(t for t in report['trials'] if t['flags']==['-mf','-3s','-d'+macro])
        return t,(a.linked/t['stem']/'candidate.bin').read_bytes()
    ai,ai_code=code('AI_POINT_3');reset,reset_code=code('RESET_WORD_2')
    assert ai['exact_interval_bytes'] and len(ai_code)==493 and ai_code[14:17]==bytes.fromhex('83ec58')
    assert ai_code[293:300]==bytes.fromhex('8944246883c418')
    assert ai_code[415:438]==bytes.fromhex('8d442d00034424440fb610895424540fb640018944244c')
    ai_before,ai_before_code=code('AI_POINT_0')
    assert not ai_before['exact_interval_bytes'] and len(ai_before_code)==493
    assert ai_before_code[293:300]==bytes.fromhex('83c41889442450')
    assert reset['exact_interval_bytes'] and len(reset_code)==56
    assert reset_code[33:40]==bytes.fromhex('66c74403400000') and reset_code[47:49]==bytes.fromhex('7ce0')
    assert 0x35bba+49+int.from_bytes(reset_code[48:49],'little',signed=True)==0x35bcb
    reset_before,reset_before_code=code('RESET_WORD_0');assert not reset_before['exact_interval_bytes'] and len(reset_before_code)==56
    rejected=[]
    with tempfile.TemporaryDirectory(prefix='fd2-ai-point-test-') as temp:
        root=Path(temp)
        for case in ('source_missing','source_bytes','source_hash'):
            objects=root/case;shutil.copytree(a.objects,objects)
            if case=='source_missing':(objects/'GAME.C').unlink()
            elif case=='source_bytes':(objects/'GAME.C').write_bytes((objects/'GAME.C').read_bytes()+b'\n/* changed */\n')
            else:
                path=objects/'compile-report.json';d=json.loads(path.read_text());d['source_sha256']='0'*64;path.write_text(json.dumps(d))
            output=root/(case+'-output')
            result=subprocess.run(['python3',str(Path(__file__).with_name('fd2_matching_game_restore.py')),'link',
                '--objects',str(objects),'--original',str(a.original),'--evidence',str(a.evidence),'--output',str(output)],
                capture_output=True,text=True,timeout=30)
            assert result.returncode and not output.exists(),(case,result.stdout,result.stderr)
            rejected.append({'case':case,'output_created':False})
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'complete_function_bytes':{'0x1598a':493,'0x35bba':56},'complete_total_bytes':549,
        'original_stack_frame_bytes':88,'result_store_before_argument_cleanup':True,'coordinate_load_bytes_exact':True,
        'clear_store_width_bits':16,'clear_loop_target':'0x35bcb','old_controls_remain_unmatched':True,
        'source_report_sha256':hashlib.sha256((a.linked/'report.json').read_bytes()).hexdigest(),'rejected_before_output':rejected}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()

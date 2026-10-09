#!/usr/bin/env python3
"""診斷完整函式的暫存器文字對應；不取代逐byte匹配或增加C覆蓋。"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import tempfile

from fd2_matching_game_restore import BINDINGS, verify_original_bindings

REGISTERS={}
for family,aliases in {'eax':('eax','ax','al','ah'),'ebx':('ebx','bx','bl','bh'),
        'ecx':('ecx','cx','cl','ch'),'edx':('edx','dx','dl','dh'),
        'esi':('esi','si'),'edi':('edi','di'),'ebp':('ebp','bp'),
        'esp':('esp','sp'),'eip':('eip','ip')}.items():
    for width,name in enumerate(aliases):REGISTERS[name]=(family,width)
REGISTER_RE=re.compile(r'\b('+'|'.join(REGISTERS)+r')\b')
DISASSEMBLY_RE=re.compile(r'^\s*([0-9a-f]+):\s+((?:[0-9a-f]{2}\s+)+)\s*(\S.*)$')


def digest(data):return hashlib.sha256(data).hexdigest()


def decode(data,address):
    with tempfile.TemporaryDirectory(prefix='fd2-register-diagnostic-') as temp:
        path=Path(temp)/'code.bin';path.write_bytes(data)
        r=subprocess.run(['objdump','-D','-b','binary','-m','i386','-M','intel','--insn-width=16',
            '--adjust-vma='+hex(address),str(path)],capture_output=True,text=True,check=True,timeout=10)
    result=[];cursor=address
    for line in r.stdout.splitlines():
        m=DISASSEMBLY_RE.match(line)
        if not m:continue
        at=int(m[1],16);raw=bytes.fromhex(m[2]);instruction=' '.join(m[3].split())
        if at!=cursor or data[at-address:at-address+len(raw)]!=raw or '(bad)' in instruction:
            raise ValueError('診斷反組譯的指令邊界不完整')
        result.append((at,len(raw),instruction));cursor+=len(raw)
    if cursor!=address+len(data) or not result:raise ValueError('診斷反組譯沒有涵蓋完整區間')
    return result


def diagnose(original,candidate,address):
    a,b=decode(original,address),decode(candidate,address)
    mapping={};reverse={};equivalent=len(a)==len(b) and all(x[:2]==y[:2] for x,y in zip(a,b))
    # 共用序言與返回保存固定ABI暫存器，不混入中間值的文字對應。
    # 區間由實際首末不同指令決定，兩側仍完整核對，沒有遮蔽任何差異。
    different=[n for n,(x,y) in enumerate(zip(a,b))
        if original[x[0]-address:x[0]-address+x[1]]!=candidate[y[0]-address:y[0]-address+y[1]]]
    first,last=(different[0],different[-1]+1) if different and equivalent else (0,len(a))
    start=a[first][0];end=a[last-1][0]+a[last-1][1]
    for left,right in zip(a[first:last],b[first:last]):
        if left[:2]!=right[:2]:equivalent=False;break
        if REGISTER_RE.sub('@',left[2])!=REGISTER_RE.sub('@',right[2]):equivalent=False;break
        for original_reg,candidate_reg in zip(REGISTER_RE.findall(left[2]),REGISTER_RE.findall(right[2])):
            dest,dwidth=REGISTERS[original_reg];source,swidth=REGISTERS[candidate_reg]
            if dwidth!=swidth or ((source in ('esp','eip') or dest in ('esp','eip')) and source!=dest):
                equivalent=False;break
            if mapping.get(source,dest)!=dest or reverse.get(dest,source)!=source:
                equivalent=False;break
            mapping[source]=dest;reverse[dest]=source
        if not equivalent:break
    return {'exact_interval_bytes':original==candidate,'original_size':len(original),'candidate_size':len(candidate),
        'original_code_sha256':digest(original),'candidate_code_sha256':digest(candidate),
        'differing_byte_offsets':[n for n,(x,y) in enumerate(zip(original,candidate)) if x!=y],
        'instruction_text_equivalent_under_register_map':equivalent,
        'candidate_to_original_registers':dict(sorted(mapping.items())) if equivalent else None,
        'register_map_interval':{'start':hex(start),'end':hex(end),'address_space':'IDA LE loader linear address'},
        'unchanged_prefix_bytes':start-address if equivalent else None,
        'unchanged_suffix_bytes':address+len(original)-end if equivalent else None,
        'instruction_count':len(a),'diagnostic_only':True,'increases_c_coverage':False,
        'scope':'只描述反組譯文字及邊界的對應，不證明ABI、行為等價、原作者宣告或逐byte匹配。'}


def prepare(a):
    report_path=a.linked/'report.json';report=json.loads(report_path.read_text());raw=a.original.read_bytes()
    image=json.loads(a.evidence.read_text());trial=next(t for t in report['trials'] if t['stem']==a.stem)
    identity={'file':'FD2.EXE','size':len(raw),'md5':hashlib.md5(raw).hexdigest(),'sha256':digest(raw)}
    if identity!=report['input'] or image['input']!=identity or digest(a.evidence.read_bytes())!=report['evidence_sha256']:
        raise ValueError('診斷輸入、IDA或連結收據未綁定')
    addresses=trial.get('addresses',[trial['address']])
    if addresses!=[trial['address']]:raise ValueError('診斷目前只支援單一完整函式')
    function=next(f for f in image['functions'] if f['inventory']['start']==trial['address'])
    verify_original_bindings([function],raw,BINDINGS)
    cursor=int(trial['address'],16);data=bytearray()
    for chunk in function['chunks']:
        for i in chunk['instructions']:
            at=int(i['ida_linear_address'],16);offset=int(i['file_offset'],16);file_code=bytes.fromhex(i['file_bytes'])
            loaded=bytes.fromhex(i['loaded_bytes'])
            if at!=cursor or raw[offset:offset+len(file_code)]!=file_code or len(loaded)!=len(file_code):
                raise ValueError('IDA完整區間或原始檔指令不符')
            data.extend(loaded);cursor+=len(loaded)
    if cursor!=int(function['inventory']['end'],16) or len(data)!=function['inventory']['size'] or len(data)!=trial['original_size']:
        raise ValueError('IDA函式邊界不完整')
    candidate=(a.linked/a.stem/'candidate.bin').read_bytes()
    if digest(candidate)!=trial['code_sha256'] or len(candidate)!=trial['candidate_size']:
        raise ValueError('候選產碼與收據不符')
    result=diagnose(bytes(data),candidate,int(trial['address'],16))
    first=next((n for n,(x,y) in enumerate(zip(data,candidate)) if x!=y),min(len(data),len(candidate)) if len(data)!=len(candidate) else None)
    if trial['exact_interval_bytes']!=result['exact_interval_bytes'] or trial['first_difference_offset']!=first:
        raise ValueError('診斷不得覆寫連結收據的匹配結果')
    result.update(schema_version=1,input=identity,tool=image['tool'],ida_linear_address=trial['address'],stem=a.stem,
        evidence_sha256=report['evidence_sha256'],link_report_sha256=digest(report_path.read_bytes()),
        objdump_version=subprocess.check_output(['objdump','--version'],text=True).splitlines()[0])
    return result


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('original','evidence','linked','output'):p.add_argument('--'+name,type=Path,required=True)
    p.add_argument('--stem',required=True);a=p.parse_args()
    if not Path('/.dockerenv').exists():raise ValueError('本工具只在Docker執行')
    result=prepare(a);a.output.mkdir(parents=True,exist_ok=False)
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()

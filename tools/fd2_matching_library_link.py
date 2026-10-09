#!/usr/bin/env python3
"""以原生Watcom linker重建固定SDK完整CODE；外部符號逐筆核對原始指令。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
from fd2_matching_library_probe import INPUT_SHA, LIBRARIES, omf_sections, sha
from le_xref import parse_le, parse_fixups, file_to_linear

WLINK_SHA = '97ff1fd068c48de2745d25b0f85057e485b3eee4df0fcf6570f1839e5d315fb5'
WDIS_SHA = 'c6d20aa8aeb550e780c4a72a367f4446225ad4ab4b7d4e8f18203c4271805015'
MEMBERS = ('memcpy', 'fprintf', 'setbuf', 'memset', 'int386', 'int386x', 'chktty', 'qread', 'fputs')


def record(kind, data):
    raw = bytes([kind]) + (len(data)+1).to_bytes(2, 'little') + data
    return raw + bytes([-sum(raw) & 255])


def omf_name(value):
    raw = value.encode('ascii')
    if not 0 < len(raw) <= 255:
        raise ValueError('OMF符號名稱長度不符')
    return bytes([len(raw)]) + raw


def zero_bindings(bindings):
    """只宣告零長度區段及offset 0符號，不含LEDATA、指令或padding。"""
    raw = record(0x80, omf_name('FD2_BIND'))
    if bindings:
        raw += record(0x96, b''.join(omf_name(n) for i in range(len(bindings)) for n in ('_FD2B'+str(i), 'FD2_BIND_'+str(i))))
    for i, (name, address) in enumerate(sorted(bindings.items())):
        if not 0 <= address <= 0xffffffff or i >= 60:
            raise ValueError('外部符號位址或數量不支援')
        raw += record(0x99, b'\x21'+bytes(4)+bytes([i*2+1, i*2+2, 0]))
        raw += record(0x91, bytes([0, i+1])+omf_name(name)+bytes(4)+b'\x00')
    return raw + record(0x8b, b'\x00')


def omf_details(raw):
    # checksum、完整CODE、THEADR／MODEND與多LNAMES由既有解析器先驗證。
    sections = omf_sections(raw)
    names = ['']; imports = []; declared = []; at = 0; easy = False
    def index(data, pos):
        if pos >= len(data): raise ValueError('OMF index截斷')
        n = data[pos]
        if n & 128:
            if pos+1 >= len(data): raise ValueError('OMF長index截斷')
            return ((n & 127) << 8) | data[pos+1], pos+2
        return n, pos+1
    def name(data, pos):
        if pos >= len(data) or pos+1+data[pos] > len(data): raise ValueError('OMF名稱截斷')
        size = data[pos]
        return data[pos+1:pos+1+size].decode('ascii'), pos+1+size
    while at < len(raw):
        kind = raw[at]; end = at+3+int.from_bytes(raw[at+1:at+3], 'little'); data = raw[at+3:end-1]
        base = kind & 0xfe; pos = 0
        if base == 0x88 and data == b'\x80\xaa80386': easy = True
        if base in (0x96, 0xca):
            while pos < len(data):
                n, pos = name(data, pos); names.append(n)
        elif base == 0x98:
            pos = 1 + (3 if data[0] >> 5 == 0 else 0)
            width = 4 if kind & 1 or easy else 2
            if pos+width > len(data): raise ValueError('OMF SEGDEF截斷')
            size = int.from_bytes(data[pos:pos+width], 'little'); pos += width
            if data[0] & 2 and width == 2 and size == 0: size = 65536
            segment, pos = index(data, pos); cls, pos = index(data, pos)
            if max(segment, cls) >= len(names): raise ValueError('OMF SEGDEF名稱超界')
            declared.append({'name':names[segment], 'class':names[cls], 'size':size})
        elif kind == 0x8c:
            while pos < len(data):
                n, pos = name(data, pos); _, pos = index(data, pos); imports.append(n)
        elif base in (0xb0, 0xb4, 0xbc):
            raise ValueError('尚未支援common／local／compact external，不猜綁定')
        at = end
    if any(s['size'] and s['class'].upper() != 'CODE' for s in declared):
        raise ValueError('SDK含非空資料段，尚未支援完整資料布局')
    if len(sections) != 1 or not sections[0]['publics'] or {p['offset'] for p in sections[0]['publics']} != {0}:
        raise ValueError('SDK不是唯一完整CODE及入口0')
    return sections[0], imports, declared


def validate_function(function, original):
    inv = function['inventory']; cursor = int(inv['start'],16); code = bytearray()
    meta = parse_le(original); fixups = parse_fixups(original,meta)
    for chunk in function['chunks']:
        for insn in chunk['instructions']:
            loaded = bytes.fromhex(insn['loaded_bytes']); file_bytes = bytes.fromhex(insn['file_bytes']); offset = int(insn['file_offset'],16)
            if not loaded or len(loaded) != len(file_bytes) or int(insn['ida_linear_address'],16) != cursor or original[offset:offset+len(file_bytes)] != file_bytes:
                raise ValueError('IDA邊界或原始指令不符')
            if file_to_linear(meta,offset) != cursor or file_to_linear(meta,offset+len(loaded)-1) != cursor+len(loaded)-1:
                raise ValueError('IDA線性位址與檔案映射不符')
            relocated = bytearray(file_bytes)
            for position,target in fixups.items():
                if offset <= position < offset+len(loaded):
                    at = position-offset
                    if at+4 > len(loaded): raise ValueError('LE重定位跨指令邊界')
                    relocated[at:at+4] = target.to_bytes(4,'little')
            if loaded != bytes(relocated): raise ValueError('IDA載入位元組與原始LE重定位不符')
            code.extend(loaded); cursor += len(loaded)
    if cursor != int(inv['end'],16) or len(code) != inv['size']:
        raise ValueError('IDA不是完整連續函式')
    return bytes(code)


def verify_imports(function, original, bindings):
    """SDK採原始精確名稱；不把多個前置底線視為同一個名稱。"""
    fixups = parse_fixups(original, parse_le(original)); refs = []
    for chunk in function['chunks']:
        for insn in chunk['instructions']:
            tokens = set(re.findall(r'\b[_A-Za-z][_A-Za-z0-9]*\b',insn['instruction'].split(';',1)[0]))
            code = bytes.fromhex(insn['loaded_bytes']); offset = int(insn['file_offset'],16)
            file_bytes = bytes.fromhex(insn['file_bytes'])
            if original[offset:offset+len(file_bytes)] != file_bytes: raise ValueError('SDK綁定的原始指令不符')
            value = None; kind = None
            if len(code) == 5 and code[0] in (0xe8,0xe9):
                value = int(insn['ida_linear_address'],16)+5+int.from_bytes(code[1:],'little',signed=True); kind = 'direct_rel32'
            else:
                positions = [(p,v) for p,v in fixups.items() if offset <= p < offset+len(code)]
                if len(positions) == 1:
                    p,value = positions[0]; relative = p-offset
                    if relative+4 > len(code) or int.from_bytes(code[relative:relative+4],'little') != value:
                        raise ValueError('原始LE重定位欄位不符')
                    kind = 'original_le_fixup'
            for symbol, target in bindings.items():
                aliases = {symbol} | ({'sub_36CD7'} if symbol == '__CHK' else set())
                if aliases & tokens:
                    if value != target: raise ValueError('SDK外部符號與原始參照不符：'+symbol)
                    refs.append({'symbol':symbol,'ida_linear_address':insn['ida_linear_address'],'value':hex(target),'kind':kind,
                                 'original_operand':insn['instruction']})
    if set(bindings) != {r['symbol'] for r in refs}:
        raise ValueError('SDK外部符號缺少原始caller／fixup證據')
    return refs


def plan(args):
    if sha(args.original) != INPUT_SHA or sha(args.library) != LIBRARIES['9.5'][0]: raise ValueError('原版或SDK版本不符')
    if sha(args.wlink) != WLINK_SHA or sha(args.wdis) != WDIS_SHA: raise ValueError('原生工具與固定版本不符')
    image = json.loads(args.evidence.read_text(encoding='utf-8'))
    if image['input']['sha256'] != INPUT_SHA or image['tool']['version'] != '9.4': raise ValueError('IDA版本或原版不符')
    original = args.original.read_bytes(); library = args.library.read_bytes(); by_name = {}
    for f in image['functions']: by_name.setdefault(f['inventory']['ida_analysis_name'],[]).append(f)
    plans = []
    for member in args.members:
        obj = args.objects/(member+'.o'); raw = obj.read_bytes(); position = library.find(raw)
        if position < 0 or position != library.rfind(raw): raise ValueError('SDK完整物件不是LIB中的唯一相同序列')
        section, imports, declared = omf_details(raw)
        public = [p['name'] for p in section['publics']]
        targets = {f['inventory']['start']:f for n in public for f in by_name.get(n,[])}
        if len(targets) != 1: raise ValueError('原廠公開符號沒有唯一原始IDA函式')
        function = next(iter(targets.values())); inv = function['inventory']
        if inv['classification']['value'] != 'runtime' or section['size'] != inv['size'] or inv['size'] < 32:
            raise ValueError('只接受已分類runtime、32 bytes以上及完整相同邊界')
        bindings = {}
        for symbol in imports:
            fs = by_name.get(symbol,[])
            if len(fs) != 1 or fs[0]['inventory']['classification']['value'] != 'runtime':
                raise ValueError('SDK import沒有唯一既有runtime定位：'+symbol)
            bindings[symbol] = int(fs[0]['inventory']['start'],16)
        expected = validate_function(function,original); refs = verify_imports(function,original,bindings)
        plans.append({'member':member,'object':obj,'object_size':len(raw),'object_sha256':sha(obj),'library_file_offset':hex(position),
                      'section':section,'declared':declared,'function':function,'bindings':bindings,'references':refs,'expected':expected})
    return image, plans


def map_segments(text):
    return [{'name':m[1],'class':m[2],'group':m[3],'address':int(m[4],16),'size':int(m[5],16)}
            for m in re.finditer(r'^([_A-Za-z][_A-Za-z0-9]*)\s+(\S+)\s+(\S+)\s+([0-9a-fA-F]{8})\s+([0-9a-fA-F]{8})\s*$',text,re.M)]


def run(args):
    if not Path('/.dockerenv').exists(): raise ValueError('本工具只在Docker執行')
    image, plans = plan(args)
    args.output.mkdir(parents=True,exist_ok=False)
    if args.output.stat().st_uid != os.getuid(): raise ValueError('輸出擁有權不符')
    results = []
    for p in plans:
        out = args.output/p['member']; out.mkdir(); inv = p['function']['inventory']; start = int(inv['start'],16)
        binder = out/'bindings.obj'; binder.write_bytes(zero_bindings(p['bindings']))
        classes = [(start,'CODE')] + [(address,'FD2_BIND_'+str(i)) for i,(_,address) in enumerate(sorted(p['bindings'].items()))]
        origin = min(a for a,_ in classes)
        script = 'format raw bin\noption nodefaultlibs,offset='+hex(origin & ~255)+',map='+str(out/'link.map')+',start='+p['section']['publics'][0]['name']
        script += '\nname '+str(out/'linked.raw')+'\nfile '+str(p['object'])+','+str(binder)
        script += '\norder '+' '.join('clname '+name+' offset='+hex(address) for address,name in sorted(classes))+'\n'
        (out/'link.lnk').write_text(script,encoding='ascii')
        call = subprocess.run([str(args.wlink),'@'+str(out/'link.lnk')],capture_output=True,text=True,timeout=20)
        (out/'link.log').write_text(call.stdout+call.stderr)
        if call.returncode or 'Error!' in call.stdout+call.stderr: raise ValueError('原生SDK連結失敗：'+p['member'])
        text = (out/'link.map').read_text(); segments = map_segments(text)
        code = [s for s in segments if s['size']]
        if len(code) != 1 or code[0]['class'] != 'CODE' or code[0]['address'] != start or code[0]['size'] != inv['size']:
            raise ValueError('連結器沒有保留完整唯一CODE邊界')
        for address,cls in classes:
            matches = [s for s in segments if s['class'] == cls]
            if len(matches) != 1 or matches[0]['address'] != address or (cls != 'CODE' and matches[0]['size']):
                raise ValueError('零長度外部符號區段布局不符')
        for symbol,address in p['bindings'].items():
            if not re.search(r'^'+format(address,'08x')+r'[*+]?\s+'+re.escape(symbol)+r'\s*$',text,re.M|re.I):
                raise ValueError('原生map外部符號位址不符')
        raw = (out/'linked.raw').read_bytes(); prefix = start-origin
        if len(raw) != prefix+inv['size'] or any(raw[:prefix]): raise ValueError('RAW容器填充或完整CODE長度不符')
        # 只排除linker RAW容器的前綴；完整SDK SEGDEF區段一byte不裁切。
        candidate = raw[prefix:]; (out/'candidate.bin').write_bytes(candidate)
        subprocess.run([str(args.wdis),'-l='+str(out/'original.dis'),str(p['object'])],capture_output=True,text=True,check=True,timeout=20)
        dis = (out/'original.dis').read_text()
        if 'No disassembly errors' not in dis or re.findall(r'Routine Size: (\d+) bytes',dis) != [str(inv['size'])]:
            raise ValueError('獨立wdis沒有核對完整SDK函式')
        results.append({'member':p['member']+'.o','ida_linear_address':inv['start'],'original_name':inv['ida_analysis_name'],
            'vendor_symbols':[s['name'] for s in p['section']['publics']],'size':inv['size'],'object_size':p['object_size'],
            'object_sha256':p['object_sha256'],'library_file_offset':p['library_file_offset'],'original_classification':inv['classification'],
            'bindings':{k:hex(v) for k,v in p['bindings'].items()},'original_binding_references':p['references'],
            'binding_object_sha256':sha(binder),'binding_segments_contain_code':False,'map_segments':segments,
            'raw_sha256':sha(out/'linked.raw'),'raw_container_prefix_zero_bytes':prefix,
            'code_sha256':hashlib.sha256(candidate).hexdigest(),'exact_interval_bytes':candidate==p['expected'],
            'first_difference_offset':next((i for i,(a,b) in enumerate(zip(candidate,p['expected'])) if a!=b),None),
            'function_semantics':'unknown','comparison':'完整原生SDK CODE與完整原始IDA函式，不遮罩或改指令'})
    result = {'schema_version':1,'input':image['input'],'tool':image['tool'],'evidence_sha256':sha(args.evidence),
        'library':{'source_package':'9.5','path':LIBRARIES['9.5'][1],'sha256':sha(args.library),'size':args.library.stat().st_size,'original_version':'unknown'},
        'native_tools':{'wlink_sha256':WLINK_SHA,'wdis_sha256':WDIS_SHA,'version':'Open Watcom 2.0 beta 2026-10-01'},
        'driver_sha256':sha(Path(__file__)),'omf_parser_sha256':sha(Path(__file__).with_name('fd2_matching_library_probe.py')),
        'trials':results,'classification_changed':False,'decompilation_complete':False,
        'map_policy':'原始map保留本機；日期、路徑與link時間不參與重跑比較，區段及符號位址逐筆驗證。',
        'rights':'SDK物件與全部機器碼只留本機；只公開方法、雜湊、原始定位與有限caller證據。'}
    (args.output/'library-link-report.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'trials':len(results),'matches':[(r['original_name'],r['size']) for r in results if r['exact_interval_bytes']]},ensure_ascii=False))


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ('objects','library','evidence','original','output'): p.add_argument('--'+name,type=Path,required=True)
    p.add_argument('--wlink',type=Path,default=Path('/opt/watcom/binl64/wlink'))
    p.add_argument('--wdis',type=Path,default=Path('/opt/watcom/binl64/wdis'))
    p.add_argument('--members',choices=MEMBERS,nargs='+',default=MEMBERS)
    run(p.parse_args())


if __name__ == '__main__': main()

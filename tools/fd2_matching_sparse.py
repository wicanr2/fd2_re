#!/usr/bin/env python3
"""分割 compiler COFF 的完整函式，將跨函式分支提升為標準 REL32 重定位。

輸入只有 compiler COFF、wdis 匯出及明示布局，不讀原版 bytes。
保留全部函式及指令，只清零已辨識 REL32 的位移欄供 ld 計算。
格式契約：https://learn.microsoft.com/en-us/windows/win32/debug/pe-format
"""

import hashlib
import re
import struct


def split_coff(raw, disassembly, addresses):
    if len(raw) < 20:
        raise ValueError('COFF 標頭截斷')
    machine, count, _, sym_at, sym_count, optional, flags = struct.unpack_from('<HHIIIHH', raw)
    if machine != 0x14C or optional or flags or not count or len(raw) < 20 + count * 40 or len(addresses) < 2:
        raise ValueError('非已支援的 i386 compiler COFF')
    headers = [struct.unpack_from('<8sIIIIIIHHI', raw, 20 + n * 40) for n in range(count)]
    if headers[0][0].rstrip(b'\0') != b'_TEXT' or any(h[3] or h[7] or h[8] for h in headers[1:]):
        raise ValueError('只支援單一非空 _TEXT 的完整 compiler 物件')
    text = bytearray(raw[headers[0][4]:headers[0][4] + headers[0][3]])
    if len(text) != headers[0][3] or headers[0][8] or headers[0][4] < 20 + count * 40 or headers[0][4] + headers[0][3] > sym_at or headers[0][5] + headers[0][7] * 10 > sym_at:
        raise ValueError('COFF 程式碼截斷或含未支援行號')
    entries = [(int(offset, 16), name) for offset, name in
               re.findall(r'^([0-9A-Fa-f]{4,8})\s+([_A-Za-z][_A-Za-z0-9]*):', disassembly, re.M)]
    names = ['sub_' + format(a, 'x') for a in addresses]
    if [name.lstrip('_').lower() for _, name in entries] != names or entries[0][0] != 0:
        raise ValueError('compiler 函式入口與登錄不符')
    starts = [offset for offset, _ in entries]
    ends = starts[1:] + [len(text)]
    if starts != sorted(set(starts)) or any(end <= start for start, end in zip(starts, ends)):
        raise ValueError('compiler 函式區間不完整')
    if list(addresses) != sorted(set(addresses)) or any(addresses[n] + ends[n] - starts[n] > addresses[n + 1] for n in range(len(addresses) - 1)):
        raise ValueError('連結後完整函式彼此重疊')

    def owner(offset):
        found = [n for n, (start, end) in enumerate(zip(starts, ends)) if start <= offset < end]
        if len(found) != 1:
            raise ValueError('位址沒有唯一 compiler 函式')
        return found[0]

    symbol_end = sym_at + sym_count * 18
    if sym_at < 20 + count * 40 or symbol_end + 4 > len(raw):
        raise ValueError('COFF 符號表截斷')
    string_size = struct.unpack_from('<I', raw, symbol_end)[0]
    strings = raw[symbol_end:symbol_end + string_size]
    if len(strings) != string_size or string_size < 4:
        raise ValueError('COFF 字串表截斷')
    symbols = bytearray(raw[sym_at:symbol_end])
    records = {}
    index = 0
    while index < sym_count:
        record = list(struct.unpack_from('<8sIhHBB', symbols, index * 18))
        records[index] = record
        if index + record[5] >= sym_count:
            raise ValueError('COFF 輔助符號截斷')
        if record[2] == 1:
            fragment = owner(record[1])
            record[1] -= starts[fragment]
            record[2] = fragment + 1
        elif record[2] > 1:
            if record[2] > count:
                raise ValueError('COFF 符號區段超界')
            record[2] += len(starts) - 1
        struct.pack_into('<8sIhHBB', symbols, index * 18, *record)
        index += 1 + record[5]
    # 所有公開函式必須同時有 compiler 符號，不能只信任文字匯出。
    for n, (_, name) in enumerate(entries):
        matches = []
        for old_index, record in records.items():
            encoded = record[0]
            if encoded[:4] == b'\0' * 4:
                at = struct.unpack_from('<I', encoded, 4)[0]
                if not 4 <= at < len(strings) or b'\0' not in strings[at:]:
                    raise ValueError('COFF 符號名稱超界')
                encoded = strings[at:strings.index(b'\0', at)]
            else:
                encoded = encoded.rstrip(b'\0')
            if encoded.decode('ascii').lstrip('_').lower() == name.lstrip('_').lower() and record[4] == 2 and record[2] == n + 1 and record[1] == 0:
                matches.append(old_index)
        if len(matches) != 1:
            raise ValueError('COFF 與 wdis 的函式入口不一致')
    relocs = [[] for _ in starts]
    used_fields = set()
    for n in range(headers[0][7]):
        at, symbol, kind = struct.unpack_from('<IIH', raw, headers[0][5] + n * 10)
        fragment = owner(at)
        if kind not in (6, 20) or symbol not in records or at + 4 > ends[fragment]:
            raise ValueError('未支援或跨邊界的 compiler 重定位')
        if at in used_fields:
            raise ValueError('compiler 重定位欄重複')
        used_fields.add(at)
        relocs[fragment].append((at - starts[fragment], symbol, kind))

    instruction_starts = {int(m.group(1), 16) for line in disassembly.splitlines()
                          if (m := re.match(r'^([0-9A-Fa-f]{4,8})\s+[0-9A-Fa-f]{2}\s', line))}
    lifted = []
    for line in disassembly.splitlines():
        match = re.match(r'^([0-9A-Fa-f]{4,8})[ \t]+((?:[0-9A-Fa-f]{2}[ \t]+)+)(j[a-z]+|call)[ \t]+', line)
        if not match:
            continue
        at = int(match.group(1), 16)
        encoded = bytes.fromhex(match.group(2))
        if text[at:at + len(encoded)] != encoded:
            raise ValueError('wdis 分支 bytes 與 compiler COFF 不符')
        if len(encoded) == 5 and encoded[0] in (0xE8, 0xE9):
            field, width = at + 1, 4
        elif len(encoded) == 6 and encoded[0] == 0x0F and 0x80 <= encoded[1] <= 0x8F:
            field, width = at + 2, 4
        elif len(encoded) == 2 and (encoded[0] == 0xEB or 0x70 <= encoded[0] <= 0x7F):
            field, width = at + 1, 1
        else:
            raise ValueError('未支援的直接分支形式')
        if field in used_fields:
            continue
        displacement = int.from_bytes(encoded[-width:], 'little', signed=True)
        target = at + len(encoded) + displacement
        source_fragment, target_fragment = owner(at), owner(target)
        if target not in instruction_starts or at + len(encoded) > ends[source_fragment]:
            raise ValueError('分支不是完整指令邊界')
        if source_fragment == target_fragment:
            continue
        if width != 4:
            raise ValueError('不能保持跨函式短分支的原始指令寬度')
        symbol = len(symbols) // 18
        label = ('_T%06d' % len(lifted)).encode('ascii')
        symbols.extend(struct.pack('<8sIhHBB', label, target - starts[target_fragment], target_fragment + 1, 0, 3, 0))
        text[field:field + 4] = b'\0' * 4
        relocs[source_fragment].append((field - starts[source_fragment], symbol, 20))
        lifted.append({'compiler_branch_offset': at, 'compiler_target_offset': target,
                       'source_address': hex(addresses[source_fragment]), 'target_address': hex(addresses[target_fragment] + target - starts[target_fragment]),
                       'original_displacement': displacement, 'relocation_type': 'IMAGE_REL_I386_REL32',
                       'instruction_bytes': encoded.hex(), 'field_offset': field - starts[source_fragment]})
    for old_index, record in records.items():
        if record[4] == 3 and record[5] == 1 and record[2] > 0:
            fragment = record[2] - 1
            if fragment < len(starts):
                struct.pack_into('<IHH', symbols, (old_index + 1) * 18, ends[fragment] - starts[fragment], len(relocs[fragment]), 0)

    new_count = count + len(starts) - 1
    output = bytearray(20 + new_count * 40)
    new_headers = []
    fragments = []
    for n, (start, end, address) in enumerate(zip(starts, ends, addresses)):
        name = ('_M%d' % n).encode('ascii')
        output.extend(b'\0' * (-len(output) % 4))
        data_at = len(output)
        output.extend(text[start:end])
        output.extend(b'\0' * (-len(output) % 4))
        rel_at = len(output)
        for relocation in sorted(relocs[n]):
            output.extend(struct.pack('<IIH', *relocation))
        new_headers.append((name, 0, 0, end - start, data_at, rel_at, 0, len(relocs[n]), 0, headers[0][9]))
        fragments.append({'section': name.decode(), 'address': hex(address), 'compiler_offset': start, 'size': end - start})
    new_headers.extend(headers[1:])
    output.extend(b'\0' * (-len(output) % 4))
    new_sym_at = len(output)
    output.extend(symbols)
    output.extend(strings)
    struct.pack_into('<HHIIIHH', output, 0, machine, new_count, 0, new_sym_at, len(symbols) // 18, 0, 0)
    for n, header in enumerate(new_headers):
        struct.pack_into('<8sIIIIIIHHI', output, 20 + n * 40, *header)
    return bytes(output), {'schema_version': 1, 'fragments': fragments, 'lifted_branches': lifted,
                           'coff_sha256': hashlib.sha256(output).hexdigest(), 'all_compiler_instruction_bytes_preserved_except_relocation_fields': True}


def read_sparse_pe(path, fragments):
    raw = path.read_bytes()
    pe = struct.unpack_from('<I', raw, 0x3C)[0]
    if raw[:2] != b'MZ' or raw[pe:pe + 4] != b'PE\0\0':
        raise ValueError('非有效 PE')
    machine, count = struct.unpack_from('<HH', raw, pe + 4)
    optional_size = struct.unpack_from('<H', raw, pe + 20)[0]
    optional = pe + 24
    if machine != 0x14C or count != len(fragments) or struct.unpack_from('<H', raw, optional)[0] != 0x10B or struct.unpack_from('<I', raw, optional + 28)[0] != 0 or struct.unpack_from('<I', raw, optional + 16)[0] != int(fragments[0]['address'], 16):
        raise ValueError('PE 架構或入口不符')
    result = bytearray()
    for n, fragment in enumerate(fragments):
        at = optional + optional_size + n * 40
        name = raw[at:at + 8].rstrip(b'\0')
        size, address, raw_size, raw_at = struct.unpack_from('<IIII', raw, at + 8)
        if name != ('.m%d' % n).encode() or address != int(fragment['address'], 16) or size != fragment['size'] or raw_size < size or raw_at + size > len(raw):
            raise ValueError('PE 沒有保持完整函式區間及位置')
        result.extend(raw[raw_at:raw_at + size])
    return bytes(result)

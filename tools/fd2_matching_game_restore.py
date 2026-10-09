#!/usr/bin/env python3
"""編譯及驗證主程式區的 C 候選，所有完整區間逐位元組比較。"""

import argparse
import hashlib
import itertools
import json
import os
from pathlib import Path
import re
import subprocess

from fd2_matching_c_restore import INPUTS


CASES = (("CLEAR3", 0x134E4), ("REDRAW", 0x127A9), ("CYCLE", 0x1F525), ("MASK", 0x146A7),
         ("ROWCOPY", 0x11EB0), ("RELEASE", 0x15E71), ("CONDITIONAL_FREE", 0x1A7F1),
         ("CACHE_INIT", 0x1D4CB), ("CONDITIONAL_INIT", 0x1A7BD), ("INVENTORY", 0x1B8E7),
         ("SET_BIT7", 0x13512), ("CLEAR_BIT7", 0x13536), ("SLOT_BYTE", 0x1B722),
         ("SET_BYTE5", 0x32975), ("GET_BIT0", 0x3453E), ("RANGE_LOW_BYTE", 0x3419C),
         ("COPY_WORDS", 0x25089), ("COMPARE_VALUE", 0x2EF8F))
RECORD_CASES = (("CURSOR_Y_DEC", 0x11B48), ("CURSOR_Y_INC", 0x11B9B),
                ("CURSOR_X_INC", 0x11BFA), ("CURSOR_X_DEC", 0x11C59),
                ("FIND_RECORD", 0x12C0D), ("SUM_FIELD", 0x15DA2),
                ("DERIVED_WORDS", 0x1145A), ("BLIT_CELL", 0x126F7),
                ("READ_INFO", 0x12E38), ("CONDITIONAL_INFO", 0x13A44))
COPY_CASES = (("VIDEO_COPY", 0x16559), ("RESTORE_72", 0x17643),
              ("HCLIP_86", 0x182AD), ("VCLIP_86", 0x18312),
              ("BOTTOM_102", 0x1839B), ("PANEL_COPY", 0x1AF99),
              ("INSERT_CELL", 0x1BB8C))
QUERY_CASES = (("MEMBER_SIX", 0x1C1C3), ("FACING_BYTES", 0x1F04A),
               ("BOX_POSITION", 0x2A289), ("REVERSE_INFO", 0x2B5E1),
               ("RATIO_WORDS", 0x1E7F6), ("FILL_SQUARE", 0x1F6EF))
CONTROL_CASES = tuple(("F" + format(address, "X"), address) for address in (
    0x10620, 0x13460, 0x1685C, 0x16886, 0x173E7, 0x17AA9, 0x1875D, 0x18795,
    0x1C8ED, 0x1CA89, 0x1F183, 0x206C5, 0x20707, 0x2073D, 0x20822, 0x2084A,
    0x20926, 0x21206, 0x21396, 0x21527))
BATTLE_CASES = tuple(("F" + format(address, "X"), address) for address in (
    0x13565, 0x14625, 0x14B16, 0x164E8, 0x175A9, 0x1B83D, 0x1B8A6,
    0x1C142, 0x1C220)) + (("COUNTS_GROUP", 0x1B5F1),)
EVENT_CASES = (("F21082", 0x21082), ("EVENT_PAIR", 0x2111A), ("F214AD", 0x214AD))
LAYOUT_CASES = (("EVENT_FULL", 0x2111A),)
QUAKE_CASES = (("F21548", 0x21548),)
EFFECT_TAIL_CASES = tuple(("F" + format(address, "X"), address) for address in (
    0x2185F, 0x2189A, 0x219AD, 0x21A9E, 0x21AD9, 0x21B18, 0x21B99)) + (
    ("INDEXED_PAIR", 0x2111A), ("INDEXED_ALL", 0x2111A), ("QUAKE_GROUP", 0x21527))
SPARSE_GROUPS = {"INDEXED_PAIR": (0x2111A, 0x21B18),
                 "INDEXED_ALL": (0x2111A, 0x21AD9, 0x21B18, 0x21B99),
                 "QUAKE_GROUP": (0x21527, 0x21548, 0x2185F, 0x21A9E),
                 "RADIAL_GROUP": (0x190AC, 0x219AD)}
TREASURE_CASES = (("F190AC", 0x190AC), ("RADIAL_GROUP", 0x190AC))
CALL_WRAPPER_CASES = tuple(('F' + format(address,'X'),address) for address in (
    0x22A85,0x22BE1,0x33F78,0x34A3C,0x34F02,0x350CC,0x35321))
GLOBAL_CALL_CASES = tuple(('F' + format(address,'X'),address) for address in (
    0x226EA,0x2282F,0x22960,0x22CDA,0x22EF6,0x33FAF))
SHORT_DATA_CASES = tuple(('F' + format(address,'X'),address) for address in (
    0x12D7B,0x29117,0x2935B,0x3415E,0x343E2,0x34C1E,0x34CB3,0x34F74,0x352E2,0x35A2F,0x35C22))
SHORT_BRANCH_CASES = tuple(('F' + format(address,'X'),address) for address in (
    0x203BD,0x20872,0x20A51,0x20A87,0x20B14,0x20B3C,0x22AA8,0x25052,0x2BC9A,0x2D620,
    0x34594,0x347D9,0x3499B,0x34A0E,0x34E3B,0x35641,0x35675,0x35898,0x3599B,0x35B6B,0x35BBA))
EXTENDED_FLOW_CASES = tuple(('F' + format(address,'X'),address) for address in (
    0x24B14,0x24BDE,0x33499,0x309FF,0x31860,0x35AB8,0x1E5C0,0x208CF,0x35E5A,0x342B5,
    0x2C39B,0x1AEB1,0x2FFA5,0x34DCD,0x1297D,0x35A48,0x34E90,0x2D31B,0x34924,0x20BF5,
    0x1AF1E,0x344C2,0x196CB,0x2B9A1,0x24B4D,0x11CAC,0x22656,0x1E529,0x1DEBE,0x34422))
MID_FLOW_CASES = tuple(('F' + format(address,'X'),address) for address in (
    0x1F0DC,0x1B932,0x3327D,0x25A96,0x25B45,0x358EA,0x35112,0x286BD,0x334D9,0x2218A,0x24D22,
    0x17EEF,0x1D6C8,0x22470,0x341DB,0x232E8,0x2E0BD,0x1F73F,0x34A7A,0x35C79,0x35D60,0x21DB2))
SCENE_FLOW_CASES = tuple(('F' + format(address,'X'),address) for address in (
    0x2CF71,0x1F30A,0x1F1CC,0x354FE,0x13FD4,0x29C90,0x23CD5,0x356B7,
    0x23B5F,0x29DED,0x31602,0x235F9,0x230F2,0x112A5))
COMPOUND_CASES = (("SCENE_REVERSE",0x230F2),)
DENSE_BRANCH_CASES = tuple(('F'+format(address,'X'),address) for address in (
    0x20AAF,0x2D392,0x12263,0x20B72,0x12C60,0x1FF79,0x1B019,0x34716,
    0x177FC,0x17D6F,0x1B14B,0x1EC2A,0x1B0AD,0x11AA8,0x2D7BD,0x1D80B))
STATE_QUERY_CASES = tuple(('F'+format(address,'X'),address) for address in (
    0x18409,0x22C04,0x1E739,0x20765,0x1C75E,0x31DBE,0x32004,0x22AF6,
    0x17E0B,0x111BA,0x2A5D0,0x10B4E,0x24C1E))
UI_SERVICE_CASES = tuple(('F'+format(address,'X'),address) for address in (
    0x17898,0x12AC6,0x2EA90,0x1300D,0x22F37,0x31019,0x1B41D,0x30012))
AUXILIARY_FLOW_CASES = tuple(('F'+format(address,'X'),address) for address in (
    0x1E1DC,0x187D6,0x31793,0x2E19B,0x2E26C,0x2FB9F,0x16E24,0x2C9EC,0x20957))
SERVICE_WRAPPER_CASES = tuple(('F'+format(address,'X'),address) for address in (
    0x35EBE,0x35ED2,0x35EE6,0x35F6F,0x36107,0x3615E,0x3616E,0x3617E,0x361A5,0x36316,0x3632D))+(('EMPTY_STACK_GROUP',0x360D8),)
ANIMATION_STEP_CASES = tuple(('F'+format(address,'X'),address) for address in (
    0x1EB05,0x25977,0x274B0,0x275D6,0x2D3FF))
DISPLAY_CONTROL_CASES = tuple(('F'+format(address,'X'),address) for address in (
    0x13E9C,0x15311,0x1956B,0x1D8BA,0x2DF6B))
RESOURCE_RECORD_CASES = tuple(('F'+format(address,'X'),address) for address in (
    0x11019,0x1B9DE,0x1C2DA,0x23A0A,0x2F4C6,0x321C8))
AI_SEQUENCE_CASES = tuple(('F'+format(address,'X'),address) for address in (
    0x1548E,0x1567E,0x1598A,0x15B77,0x23E74,0x240FA))
MENU_SEQUENCE_CASES = tuple(('F'+format(address,'X'),address) for address in (
    0x272B8,0x2D85F,0x2E6B8,0x2F642))
SAVE_SERVICE_CASES = tuple(('F'+format(address,'X'),address) for address in (
    0x301F4,0x30DC3,0x31385))
LOAD_LAYOUT_CASES = tuple((macro,0x301F4) for macro in (
    'LOAD_PTR','LOAD_STRUCT','LOAD_SCALE_FIRST'))
SCREEN_TRANSITION_CASES = (('F2D669',0x2D669),)
TEAM_SERVICE_CASES = (('TEAM30_SHARED',0x30DC3),('TEAM30_VOLATILE',0x30DC3),('TEAM30_ACCUM',0x30DC3),('TEAM30_DIRECT',0x30DC3),('TEAM313_TYPED',0x31385),('TEAM313_REUSE',0x31385))
TEAM_ABI_CASES = (('TEAM313_ABI',0x31385),)
DIGIT_COMPARE_CASES = tuple((macro,0x2D3FF) for macro in ('DIGIT_SUB','DIGIT_UNSIGNED','DIGIT_INVERSE','DIGIT_ORDERED','DIGIT_SWITCH','DIGIT_ADD_FIRST','DIGIT_AMOUNT_U','DIGIT_BASE_U','DIGIT_BOTH_U'))
DIGIT_ARITHMETIC_CASES = tuple(('DIGIT_FORM_'+str(n).zfill(2),0x2D3FF) for n in range(16))
PRICE_ARITHMETIC_CASES = tuple(('PRICE_FORM_'+str(n).zfill(2),0x30DC3) for n in range(12))
CLASS_LIFETIME_CASES = tuple(('CLASS_LIFE_'+str(n).zfill(2),0x31385) for n in range(16))
CLASS_RECORD_CASES = tuple(('CLASS_RECORD_'+str(n).zfill(2),0x31385) for n in range(4))
CLASS_CALL_CASES = tuple(('CLASS_CALL_'+str(n).zfill(2),0x31385) for n in range(4))
CLASS_REGISTER_CASES = tuple(('CLASS_REGISTER_'+str(n).zfill(2),0x31385) for n in range(4))
IDLE_ADVANCE_CASES = tuple(('IDLE_FORM_'+str(n),0x2B9A1) for n in range(6))
PANEL_ROWS_CASES = tuple(('PANEL_FORM_'+str(n),0x1B0AD) for n in range(4))
SLOT_ADDRESS_CASES = tuple(('SLOT_FORM_'+str(n),0x1B722) for n in range(8))
INVENTORY_COUNT_CASES = tuple(('COUNT_FORM_'+str(n),0x1B8A6) for n in range(8))
STATE_BYTES_CASES = tuple(('STATE_FORM_'+str(n),0x35F6F) for n in range(6))
BIT_ADDRESS_CASES = tuple(('BIT_FORM_'+str(n),0x13512) for n in range(6))
SLOT_SCALE_CASES = tuple(('SLOT_SCALE_'+str(n),0x1B722) for n in range(2))
QUERY_LAYOUT_CASES = tuple(('F'+format(n,'X'),n) for n in (0x24B14,0x24BDE,0x33499))
COMPOUND_ENTRIES = {"SCENE_REVERSE": {"source_key":"game_compound", "owner":0x230F2,
    "end":0x23296, "compiler_order":(0x231F9,0x231BC,0x230F2),
    "prologues":{0x230F2:80,0x231BC:40,0x231F9:80}}}
CASE_GROUPS = {"COUNTS_GROUP": (0x1B5F1, 0x1B653, 0x1B6B7), "EVENT_PAIR": (0x2111A, 0x211A4)}
CASE_GROUPS['EVENT_FULL'] = (0x2111A, 0x211A4, 0x21206, 0x21227, 0x212B9,
    0x2134B, 0x21364, 0x2137D, 0x21396, 0x213B7, 0x21449, 0x21462,
    0x2147B, 0x21494, 0x214AD)
ALL_CASES = (*CASES, *RECORD_CASES, *COPY_CASES, *QUERY_CASES, *CONTROL_CASES, *BATTLE_CASES, *EVENT_CASES, *LAYOUT_CASES, *QUAKE_CASES, *EFFECT_TAIL_CASES, *TREASURE_CASES, *CALL_WRAPPER_CASES, *GLOBAL_CALL_CASES, *SHORT_DATA_CASES, *SHORT_BRANCH_CASES, *EXTENDED_FLOW_CASES, *MID_FLOW_CASES, *SCENE_FLOW_CASES, *COMPOUND_CASES, *DENSE_BRANCH_CASES, *STATE_QUERY_CASES, *UI_SERVICE_CASES, *AUXILIARY_FLOW_CASES)
ALL_CASES += SERVICE_WRAPPER_CASES
ALL_CASES += ANIMATION_STEP_CASES
ALL_CASES += DISPLAY_CONTROL_CASES
ALL_CASES += RESOURCE_RECORD_CASES
ALL_CASES += AI_SEQUENCE_CASES
ALL_CASES += MENU_SEQUENCE_CASES
ALL_CASES += SAVE_SERVICE_CASES
ALL_CASES += LOAD_LAYOUT_CASES
ALL_CASES += SCREEN_TRANSITION_CASES
ALL_CASES += TEAM_SERVICE_CASES
ALL_CASES += TEAM_ABI_CASES
ALL_CASES += DIGIT_COMPARE_CASES
ALL_CASES += DIGIT_ARITHMETIC_CASES
ALL_CASES += PRICE_ARITHMETIC_CASES
ALL_CASES += CLASS_LIFETIME_CASES
ALL_CASES += CLASS_RECORD_CASES
ALL_CASES += CLASS_CALL_CASES
ALL_CASES += CLASS_REGISTER_CASES
ALL_CASES += IDLE_ADVANCE_CASES
ALL_CASES += PANEL_ROWS_CASES
ALL_CASES += SLOT_ADDRESS_CASES
ALL_CASES += INVENTORY_COUNT_CASES
ALL_CASES += STATE_BYTES_CASES
ALL_CASES += BIT_ADDRESS_CASES
ALL_CASES += SLOT_SCALE_CASES
CASE_GROUPS['EMPTY_STACK_GROUP']=(0x360D8,0x360E3,0x360EA,0x360F1,0x360F8)
BINDINGS = {"dword_53A45": 0x53A45, "dword_53BEB": 0x53BEB,
            "dword_53AC1": 0x53AC1, "dword_53A51": 0x53A51, "__CHK": 0x36CD7,
            "sub_375B2": 0x375B2, "sub_3453E": 0x3453E, "sub_127E0": 0x127E0,
            "sub_129EC": 0x129EC, "sub_11D40": 0x11D40, "memmove": 0x373C4,
            "sub_4E92C": 0x4E92C, "_free": 0x37416, "byte_53AF9": 0x53AF9,
            "dword_53B0F": 0x53B0F, "dword_53B13": 0x53B13,
            "aFdotherDat": 0x51A4D, "sub_111BA": 0x111BA,
            "dword_53BF7": 0x53BF7, "dword_53BFB": 0x53BFB}
BINDINGS.update({"dword_53AB1": 0x53AB1, "dword_53AB5": 0x53AB5,
                 "dword_53AB9": 0x53AB9, "dword_53ABD": 0x53ABD,
                 "dword_53AA9": 0x53AA9, "dword_53AAD": 0x53AAD,
                 "dword_53AC5": 0x53AC5, "dword_51A83": 0x51A83,
                 "dword_51A87": 0x51A87, "dword_51A8B": 0x51A8B,
                 "dword_51A8F": 0x51A8F, "dword_53A49": 0x53A49,
                 "dword_53A4D": 0x53A4D, "dword_53A55": 0x53A55,
                 "dword_53A69": 0x53A69, "sub_11CAC": 0x11CAC,
                 "sub_4E56C": 0x4E56C, "sub_4DEDA": 0x4DEDA,
                 "_sub_4E56C": 0x4E56C, "_sub_4DEDA": 0x4DEDA,
                 "sub_12E38": 0x12E38})
BINDINGS.update({"dword_53A71": 0x53A71, "dword_53A85": 0x53A85,
                 "dword_53C67": 0x53C67, "sub_4E8AF": 0x4E8AF,
                 "sub_4E8E1": 0x4E8E1})
BINDINGS.update({"dword_53C03": 0x53C03, "byte_52363": 0x52363,
                 "sub_4E53E": 0x4E53E, "_sub_4E53E": 0x4E53E,
                 "abs": 0x375E2, "sub_18C6D": 0x18C6D,
                 "sub_1F183": 0x1F183, "sub_1E739": 0x1E739,
                 "memset": 0x375C0})
BINDINGS.update({"dword_53A0C": 0x53A0C, "dword_53A2C": 0x53A2C,
                 "dword_53C57": 0x53C57, "dword_53ECC": 0x53ECC,
                 "dword_53BEF": 0x53BEF, "sub_4E9BB": 0x4E9BB,
                 "sub_4E63D": 0x4E63D, "sub_187D6": 0x187D6,
                 "sub_17D6F": 0x17D6F, "sub_4E516": 0x4E516,
                 "sub_1C916": 0x1C916, "loc_205BE": 0x205BE,
                 "sub_21227": 0x21227, "sub_213B7": 0x213B7,
                 "sub_21548": 0x21548})
BINDINGS.update({"_sub_4E9BB": 0x4E9BB, "_sub_4E63D": 0x4E63D,
                 "_sub_4E516": 0x4E516})
BINDINGS.update({"byte_51AAC": 0x51AAC, "sub_1A30B": 0x1A30B,
                 "sub_146A7": 0x146A7, "dword_53A10": 0x53A10,
                 "dword_53A14": 0x53A14, "dword_53EEC": 0x53EEC,
                 "sub_16559": 0x16559, "sub_25A96": 0x25A96,
                 "sub_17AA9": 0x17AA9, "malloc": 0x36D16,
                 "sub_1B722": 0x1B722, "sub_1BB8C": 0x1BB8C})
BINDINGS.update({"dword_53EC4": 0x53EC4, "sub_1C4CC": 0x1C4CC,
                 "sub_1C2DA": 0x1C2DA, "sub_1CAC7": 0x1CAC7,
                 "sub_1C75E": 0x1C75E, "sub_1E0DB": 0x1E0DB,
                 "sub_1E1DC": 0x1E1DC, "sub_1DF58": 0x1DF58,
                 "sub_1B750": 0x1B750, "sub_1B8E7": 0x1B8E7,
                 "sub_1CA89": 0x1CA89})
BINDINGS.update({'sub_1CD17': 0x1CD17})
BINDINGS.update({'unk_52096': 0x52096, 'unk_520A2': 0x520A2, 'unk_520AE': 0x520AE,
    'dword_53A5D': 0x53A5D, 'sub_1399C': 0x1399C, 'sub_1F558': 0x1F558,
    'sub_11EB0': 0x11EB0, '_printf': 0x36DC1, 'printf': 0x36DC1,
    'exit': 0x36DE4, 'aOutOfMemoryAtE': 0x501CF})
BINDINGS.update({'dword_53A6D': 0x53A6D, 'sub_21548': 0x21548,
    'sub_2189A': 0x2189A, 'sub_219AD': 0x219AD, 'sub_21B18': 0x21B18,
    'sub_21EB1': 0x21EB1, 'sub_11EEE': 0x11EEE, 'sub_4DB9C': 0x4DB9C,
    'sub_127A9': 0x127A9, 'sub_1C8ED': 0x1C8ED,
    '_abs': 0x375E2, '_sqrt': 0x3C6FC, 'sqrt': 0x3C6FC, '__CHP': 0x377A4, 'dbl_501F0': 0x501F0})
BINDINGS.update({'dword_53AD5': 0x53AD5, 'dword_53AD9': 0x53AD9,
    'dword_53ADD': 0x53ADD, 'dword_53AE1': 0x53AE1, 'dword_53BF3': 0x53BF3,
    'sub_4E031': 0x4E031, 'sub_1956B': 0x1956B, 'sub_15F84': 0x15F84,
    'sub_19953': 0x19953, 'sub_197E5': 0x197E5, 'sub_25B45': 0x25B45,
    'sub_16C57': 0x16C57, 'sub_196CB': 0x196CB, 'sub_12263': 0x12263,
    'sub_1B932': 0x1B932, 'funcs_1199C': 0x51B91, 'dword_53A7D': 0x53A7D})
BINDINGS.update({'dword_53A79':0x53A79,'sub_22AA8':0x22AA8,'sub_22CDA':0x22CDA,
    'sub_12CEA':0x12CEA,'sub_22253':0x22253,'sub_34A0E':0x34A0E,
    'sub_10B4E':0x10B4E,'sub_112A5':0x112A5,'sub_35BBA':0x35BBA,'sub_3419C':0x3419C})
BINDINGS.update({'sub_22721':0x22721,'sub_22866':0x22866,'sub_22997':0x22997,
    'sub_22D1B':0x22D1B,'sub_11506':0x11506,'byte_53A44':0x53A44})
BINDINGS.update({'unk_5023B':0x5023B,'aFd2Tmp_0':0x5023E,'fopen':0x36FCC,
    'fclose':0x37244,'sub_37072':0x37072,'_sub_4E63D':0x4E63D,'sub_4E63D':0x4E63D,
    'sub_4DEDA':0x4DEDA,'sub_1AA1D':0x1AA1D,'sub_35822':0x35822,'unk_52742':0x52742,
    'dword_53A61':0x53A61})
BINDINGS.update({'dword_53ECC':0x53ECC,'sub_33499':0x33499,'sub_22AF6':0x22AF6,
    'sub_11DF2':0x11DF2,'unk_525D6':0x525D6,'aFdotherDat':0x51A4D,
    'sub_111BA':0x111BA,'dword_54147':0x54147,'sub_1DB65':0x1DB65,'_outp':0x37795})
SOURCE_DATE_EPOCH = 315532800  # DOS 可表示的 1980-01-01 UTC。
BINDINGS.update({'sub_31860':0x31860,'sub_1B8A6':0x1B8A6,'sub_4DFCC':0x4DFCC,
    'sub_10620':0x10620,'sub_135DD':0x135DD,'sub_32999':0x32999,'sub_1366A':0x1366A,
    'sub_134E4':0x134E4,'dword_53A81':0x53A81,'sub_2D31B':0x2D31B,
    'dword_5413F':0x5413F,'sub_2E6B8':0x2E6B8,'sub_17AED':0x17AED,
    'byte_52659':0x52659,'aDatoDat':0x51A70,'dword_53C0F':0x53C0F,
    'dword_53C0B':0x53C0B,'dword_53C07':0x53C07,'sub_1297D':0x1297D,
    'dword_53C63':0x53C63,'dword_53C5B':0x53C5B,'dword_53C5F':0x53C5F,
    'sub_1974C':0x1974C,'byte_53AFA':0x53AFA,'byte_540FC':0x540FC,
    'byte_540FD':0x540FD,'sub_2935B':0x2935B,'sub_122DC':0x122DC,
    'sub_1ACF3':0x1ACF3,'sub_22046':0x22046,'sub_4E893':0x4E893,
    'sub_16E24':0x16E24,'sub_1B83D':0x1B83D,'free':0x37416})
BINDINGS.update({'sub_17E0B':0x17E0B,'sub_1B9DE':0x1B9DE,'sub_18409':0x18409,
    'sub_205DA':0x205DA,'sub_12D7B':0x12D7B,'byte_53EF1':0x53EF1,'byte_51E62':0x51E62,
    'dword_540FF':0x540FF,'dword_53EE4':0x53EE4,'dword_53EE8':0x53EE8,
    'sub_394B5':0x394B5,'sub_391D1':0x391D1,'sub_39344':0x39344,'sub_3975E':0x3975E,'sub_39448':0x39448,
    'dword_53A65':0x53A65,'outp':0x37795,'dword_53EC8':0x53EC8,
    'dword_51CF9':0x51CF9,'dword_51CFD':0x51CFD,'byte_51A10':0x51A10,'dword_53AFF':0x53AFF,
    'sub_168B6':0x168B6,'sub_17FC0':0x17FC0,'byte_51AAD':0x51AAD,'byte_51AD1':0x51AD1,'byte_51AF5':0x51AF5,
    'dword_53AD1':0x53AD1,'sub_4E85B':0x4E85B,'unk_520E4':0x520E4,'unk_520ED':0x520ED,'unk_520F6':0x520F6,
    'sub_233C6':0x233C6,'sub_2DC55':0x2DC55,'sub_1F882':0x1F882,'sub_1F525':0x1F525,
    'sub_13512':0x13512,'sub_35E5A':0x35E5A,'dbl_501F8':0x501F8,'dbl_50200':0x50200,
    'cos':0x3C885,'sin':0x3C898})
BINDINGS.update({'sub_4E4B9':0x4E4B9,'sub_4E4E8':0x4E4E8,'sub_4E4D1':0x4E4D1,'sub_4E48D':0x4E48D,
    'dword_5413B':0x5413B,'dword_5412B':0x5412B,'dword_54133':0x54133,
    'byte_52635':0x52635,'byte_52647':0x52647,'sub_4E809':0x4E809,
    'sub_15F0E':0x15F0E,'sub_1F42D':0x1F42D,'unk_5274E':0x5274E,
    'sub_1DA16':0x1DA16,'dword_5410B':0x5410B,'sub_2A289':0x2A289,
    'unk_521C3':0x521C3,'unk_521D4':0x521D4,'unk_521E5':0x521E5,
    'unk_521A3':0x521A3,'unk_521B3':0x521B3,'sub_1E529':0x1E529,
    'unk_52113':0x52113,'unk_5211E':0x5211E,'sub_13536':0x13536,
    'unk_520BA':0x520BA,'unk_520C1':0x520C1,'unk_520C8':0x520C8,
    'unk_520CF':0x520CF,'unk_520D6':0x520D6,'unk_520DD':0x520DD,'sub_1145A':0x1145A})
BINDINGS.update({'dword_54137':0x54137,'dword_53C1B':0x53C1B,'sub_17898':0x17898,
    'word_539F0':0x539F0,'word_539F2':0x539F2,'word_53A8D':0x53A8D,
    'int386':0x36D98,'_int386':0x36D98,'sub_2D85F':0x2D85F,'sub_13A9F':0x13A9F,'funcs_1197B':0x51B19,
    'sub_1685C':0x1685C,'sub_16886':0x16886})
BINDINGS.update({'sub_182AD':0x182AD,'sub_18312':0x18312,'sub_1839B':0x1839B,
    'unk_51F96':0x51F96,'sub_1C81F':0x1C81F,'dword_54127':0x54127,'sub_31E80':0x31E80,
    'sub_17EEF':0x17EEF,'sub_184C0':0x184C0,'dword_53BFF':0x53BFF,
    'dword_53A59':0x53A59,'dword_53BE3':0x53BE3,'sub_10C50':0x10C50,
    'aRb_12':0x500A9,'aFdiconB24_6':0x500AC,'aFileFdiconB24E':0x500B7,'aFdfieldDat':0x51A59,
    'aWb_2':0x500D3,'aFd2Tmp_1':0x500D6,'aRb_13':0x500DE,'aFileNotFoundS':0x500E1,'aOutOfMemoryAtL':0x500FC,
    'sub_24D22':0x24D22,'fwrite':0x3744B,'fseek':0x375F0})
BINDINGS.update({'dword_53C17':0x53C17,'dword_53C13':0x53C13,'sub_179D5':0x179D5,
    'dword_53A40':0x53A40,'dword_53C1F':0x53C1F,'byte_51A97':0x51A97,'sub_4DD52':0x4DD52,
    'dword_54153':0x54153,'dword_5412F':0x5412F,'sub_4DF4C':0x4DF4C,
    'dword_53B07':0x53B07,'dword_53AED':0x53AED,'dword_53AF5':0x53AF5,'sub_2C9EC':0x2C9EC,'sub_13A44':0x13A44,
    'sub_13460':0x13460,'sub_1C220':0x1C220,'sub_1B5F1':0x1B5F1,'sub_4DBD8':0x4DBD8,'sub_4DBB9':0x4DBB9,
    'sub_30550':0x30550,'byte_51AAB':0x51AAB,'byte_51E61':0x51E61,'byte_526B9':0x526B9,
    'aRb_5':0x5026C,'aFd2Sav_5':0x5026F,'aWb_1':0x50277,'aFd2Sav_6':0x5027A})
BINDINGS.update({'dword_5204A':0x5204A,'byte_53DFC':0x53DFC,'byte_53C6C':0x53C6C,'byte_53D34':0x53D34,
    'a05d':0x51EBF,'sprintf':0x377D9,'_sprintf':0x377D9,'byte_526A7':0x526A7,
    'unk_52618':0x52618,'byte_540FE':0x540FE})
BINDINGS.update({'off_52758':0x52758,'off_5275C':0x5275C,'sub_36284':0x36284,
    'sub_362F1':0x362F1,'sub_36316':0x36316,'sub_3632D':0x3632D,
    'sub_3CC00':0x3CC00,'memcpy':0x3CBD6,'_memcpy':0x3CBD6,'read':0x3CC4B,'_read':0x3CC4B,
    '_sub_3CC00':0x3CC00,'_sub_36284':0x36284,'_sub_362F1':0x362F1,'_sub_36316':0x36316,'_sub_3632D':0x3632D})
BINDINGS.update({'dword_53A30':0x53A30,'dword_53A34':0x53A34,'dword_53A38':0x53A38,'dword_53A3C':0x53A3C,
    'dword_53AD1':0x53AD1,'sub_1EC2A':0x1EC2A,'sub_1F0DC':0x1F0DC,
    'byte_51A11':0x51A11,'byte_53EF0':0x53EF0,'byte_51E61':0x51E61,
    'dword_53ED0':0x53ED0,'dword_53EE0':0x53EE0,'sub_3ADD4':0x3ADD4,
    'sub_3AC0B':0x3AC0B,'sub_3AAA5':0x3AAA5,'sub_3AB9E':0x3AB9E,'sub_3AE56':0x3AE56,
    'dword_540BA':0x540BA,'dword_5411F':0x5411F,'unk_52539':0x52539,
    'byte_540FB':0x540FB,'byte_540FA':0x540FA,'sub_2D620':0x2D620})
BINDINGS.update({'aFdmusDat':0x51A79,'a08d_1':0x50254,'a08d_2':0x5025A,'sub_15E71':0x15E71})
BINDINGS.update({'sub_14B78':0x14B78,'sub_14818':0x14818,'sub_1B653':0x1B653,
    'sub_4DBFC':0x4DBFC,'_sub_4DBFC':0x4DBFC,'funcs_1541F':0x51D01,
    'sub_1D4F6':0x1D4F6,'sub_1DB65':0x1DB65,'sub_1598A':0x1598A,'sub_1567E':0x1567E})
BINDINGS.update({'dword_53C23':0x53C23,'dword_53C27':0x53C27,'dword_53C2B':0x53C2B,
    'dword_53C2F':0x53C2F,'dword_53C33':0x53C33,'dword_53C4B':0x53C4B,
    'sub_1D4CB':0x1D4CB,'sub_2A6BD':0x2A6BD,'sub_2E19B':0x2E19B,'sub_2E26C':0x2E26C})
BINDINGS.update({'dword_539EC':0x539EC,'dword_53B17':0x53B17,'dword_53BDF':0x53BDF,
    'sub_37072':0x37072,'unk_51F15':0x51F15,'unk_52183':0x52183,'unk_52193':0x52193,
    'dword_5412B':0x5412B,'sub_11019':0x11019})
BINDINGS.update({'aRb_10':0x502B7,'aFdiconB24_4':0x502BA,
    'sub_4DDD7':0x4DDD7,'_sub_4DDD7':0x4DDD7})
BINDINGS.update({'dword_53C37':0x53C37,'dword_53C3B':0x53C3B,'dword_53C3F':0x53C3F,
    'dword_53C43':0x53C43,'dword_53C47':0x53C47,'sub_1F04A':0x1F04A,
    'sub_1E856':0x1E856,'sub_1E611':0x1E611,'sub_1E292':0x1E292,'sub_28A6C':0x28A6C,
    'sub_1B6B7':0x1B6B7,'sub_1B8A6':0x1B8A6,'sub_14B16':0x14B16,
    'sub_149F8':0x149F8,'sub_15880':0x15880,'sub_1C269':0x1C269,'sub_15B77':0x15B77,
    'sub_4E555':0x4E555,'_sub_4E555':0x4E555,'sub_4E040':0x4E040,'_sub_4E040':0x4E040,
    'sub_1F882':0x1F882,'sub_1C220':0x1C220,'sub_24336':0x24336,
    'unk_521F6':0x521F6,'unk_52206':0x52206,'unk_52216':0x52216,'unk_5221F':0x5221F,
    'unk_52228':0x52228,'unk_52241':0x52241,'unk_5225A':0x5225A,'dbl_50144':0x50144})
BINDINGS.update({'sub_1E7F6':0x1E7F6,'sub_1EB05':0x1EB05,'sub_15DA2':0x15DA2})
BINDINGS.update({'byte_540B7':0x540B7,'byte_540B8':0x540B8,'byte_540B9':0x540B9,
    'dword_54097':0x54097,'dword_540A7':0x540A7,'sub_2D3FF':0x2D3FF,
    'sub_2D9FE':0x2D9FE,'sub_2DF6B':0x2DF6B,'sub_2E0BD':0x2E0BD,
    'sub_2EA90':0x2EA90,'sub_2F4C6':0x2F4C6,'sub_4E9E4':0x4E9E4,
    'unk_52511':0x52511,'unk_526EA':0x526EA,'unk_5272A':0x5272A,'unk_52736':0x52736})
BINDINGS.update({'aRb_6':0x50282,'aFd2Sav_7':0x50285,'aRb_7':0x5028D,
    'aFdiconB24_1':0x50290,'aRb_8':0x5029B,'aFdiconB24_2':0x5029E,
    'byte_5265D':0x5265D,'word_52669':0x52669,'sub_25977':0x25977,
    'sub_2A2E8':0x2A2E8,'sub_2D516':0x2D516,'sub_309FF':0x309FF,
    'sub_30C22':0x30C22,'sub_311DC':0x311DC,'sub_31602':0x31602,'sub_31793':0x31793})
BINDINGS.update({'unk_526DA':0x526DA})
BINDINGS.update({'_sub_4E031':0x4E031})
SOURCES = {
    "game_idle_advance": ("tools/fd2_matching_game_idle_advance.c", tuple(macro for macro, _ in IDLE_ADVANCE_CASES)),
    "game_panel_rows": ("tools/fd2_matching_game_panel_rows.c", tuple(macro for macro, _ in PANEL_ROWS_CASES)),
    "game_slot_address": ("tools/fd2_matching_game_slot_address.c", tuple(macro for macro, _ in SLOT_ADDRESS_CASES)),
    "game_inventory_count": ("tools/fd2_matching_game_inventory_count.c", tuple(macro for macro, _ in INVENTORY_COUNT_CASES)),
    "game_state_bytes": ("tools/fd2_matching_game_state_bytes.c", tuple(macro for macro, _ in STATE_BYTES_CASES)),
    "game_bit_address": ("tools/fd2_matching_game_bit_address.c", tuple(macro for macro, _ in BIT_ADDRESS_CASES)),
    "game_slot_scale": ("tools/fd2_matching_game_slot_scale.c", tuple(macro for macro, _ in SLOT_SCALE_CASES)),
    "game_query_layout": ("tools/fd2_matching_game_query_layout.c", tuple(macro for macro, _ in QUERY_LAYOUT_CASES)),
    "game": ("tools/fd2_matching_game_slices.c", tuple(macro for macro, _ in CASES)),
    "record_layout": ("tools/fd2_matching_record_layout.c", ("SET_BIT7", "SLOT_BYTE", "COPY_WORDS")),
    "game_records": ("tools/fd2_matching_game_records.c", tuple(macro for macro, _ in RECORD_CASES)),
    "game_copy": ("tools/fd2_matching_game_copy.c", tuple(macro for macro, _ in COPY_CASES)),
    "game_queries": ("tools/fd2_matching_game_queries.c", tuple(macro for macro, _ in QUERY_CASES)),
    "game_controls": ("tools/fd2_matching_game_controls.c", tuple(macro for macro, _ in CONTROL_CASES)),
    "game_battle_records": ("tools/fd2_matching_game_battle_records.c", tuple(macro for macro, _ in BATTLE_CASES)),
    "game_events": ("tools/fd2_matching_game_events.c", tuple(macro for macro, _ in EVENT_CASES)),
    "game_event_layout": ("tools/fd2_matching_game_event_layout.c", tuple(macro for macro, _ in LAYOUT_CASES)),
    "game_quake": ("tools/fd2_matching_game_quake.c", tuple(macro for macro, _ in QUAKE_CASES)),
    "game_effect_tail": ("tools/fd2_matching_game_effect_tail.c", tuple(macro for macro, _ in EFFECT_TAIL_CASES)),
    "game_treasure": ("tools/fd2_matching_game_treasure.c", tuple(macro for macro, _ in TREASURE_CASES)),
    "game_call_wrappers": ("tools/fd2_matching_game_call_wrappers.c", tuple(macro for macro, _ in CALL_WRAPPER_CASES)),
    "game_global_calls": ("tools/fd2_matching_game_global_calls.c", tuple(macro for macro, _ in GLOBAL_CALL_CASES)),
    "game_short_data": ("tools/fd2_matching_game_short_data.c", tuple(macro for macro, _ in SHORT_DATA_CASES)),
    "game_short_branches": ("tools/fd2_matching_game_short_branches.c", tuple(macro for macro, _ in SHORT_BRANCH_CASES)),
    "game_extended_flow": ("tools/fd2_matching_game_extended_flow.c", tuple(macro for macro, _ in EXTENDED_FLOW_CASES)),
    "game_mid_flow": ("tools/fd2_matching_game_mid_flow.c", tuple(macro for macro, _ in MID_FLOW_CASES)),
    "game_scene_flow": ("tools/fd2_matching_game_scene_flow.c", tuple(macro for macro, _ in SCENE_FLOW_CASES)),
    "game_compound": ("tools/fd2_matching_game_compound.c", tuple(macro for macro, _ in COMPOUND_CASES)),
    "game_dense_branches": ("tools/fd2_matching_game_dense_branches.c", tuple(macro for macro, _ in DENSE_BRANCH_CASES)),
    "game_state_queries": ("tools/fd2_matching_game_state_queries.c", tuple(macro for macro, _ in STATE_QUERY_CASES)),
    "game_ui_services": ("tools/fd2_matching_game_ui_services.c", tuple(macro for macro, _ in UI_SERVICE_CASES)),
    "game_auxiliary_flows": ("tools/fd2_matching_game_auxiliary_flows.c", tuple(macro for macro, _ in AUXILIARY_FLOW_CASES)),
    "game_service_wrappers": ("tools/fd2_matching_game_service_wrappers.c", tuple(macro for macro, _ in SERVICE_WRAPPER_CASES)),
    "game_animation_steps": ("tools/fd2_matching_game_animation_steps.c", tuple(macro for macro, _ in ANIMATION_STEP_CASES)),
    "game_digit_roll": ("tools/fd2_matching_game_digit_roll.c", ("F2D3FF",)),
    "game_digit_compare": ("tools/fd2_matching_game_digit_compare.c", ("F2D3FF", *tuple(macro for macro, _ in DIGIT_COMPARE_CASES))),
    "game_digit_arithmetic": ("tools/fd2_matching_game_digit_arithmetic.c", tuple(macro for macro, _ in DIGIT_ARITHMETIC_CASES)),
    "game_price_arithmetic": ("tools/fd2_matching_game_price_arithmetic.c", tuple(macro for macro, _ in PRICE_ARITHMETIC_CASES)),
    "game_class_lifetime": ("tools/fd2_matching_game_class_lifetime.c", tuple(macro for macro, _ in CLASS_LIFETIME_CASES)),
    "game_class_record": ("tools/fd2_matching_game_class_record.c", tuple(macro for macro, _ in CLASS_RECORD_CASES)),
    "game_class_calls": ("tools/fd2_matching_game_class_calls.c", tuple(macro for macro, _ in CLASS_CALL_CASES)),
    "game_class_register": ("tools/fd2_matching_game_class_register.c", tuple(macro for macro, _ in CLASS_REGISTER_CASES)),
    "game_display_control": ("tools/fd2_matching_game_display_control.c", tuple(macro for macro, _ in DISPLAY_CONTROL_CASES)),
    "game_resource_records": ("tools/fd2_matching_game_resource_records.c", tuple(macro for macro, _ in RESOURCE_RECORD_CASES)),
    "game_ai_sequences": ("tools/fd2_matching_game_ai_sequences.c", tuple(macro for macro, _ in AI_SEQUENCE_CASES)),
    "game_loop_layout": ("tools/fd2_matching_game_loop_layout.c", ("F25052","F2D620","F34A0E")),
    "game_menu_sequences": ("tools/fd2_matching_game_menu_sequences.c", tuple(macro for macro, _ in MENU_SEQUENCE_CASES)),
    "game_save_services": ("tools/fd2_matching_game_save_services.c", tuple(macro for macro, _ in SAVE_SERVICE_CASES)),
    "game_load_layout": ("tools/fd2_matching_game_load_layout.c", tuple(macro for macro, _ in LOAD_LAYOUT_CASES)),
    "game_screen_transition": ("tools/fd2_matching_game_screen_transition.c", ('F2D669',)),
    "game_team_services": ("tools/fd2_matching_game_team_services.c", tuple(macro for macro, _ in TEAM_SERVICE_CASES)),
    "game_team_abi": ("tools/fd2_matching_game_team_abi.c", ('TEAM313_ABI',)),
}
COSTS = {"balanced": (), "space": ("-os",), "speed": ("-ot",),
         "alias": ("-oa",), "loop": ("-ol",), "unroll": ("-ol+",),
         "reorder": ("-or",), "intrinsic": ("-oi",), "optimize": ("-omiler",)}
SOURCE_INCLUDES = {'game_effect_tail': (('QUAKE.C', 'tools/fd2_matching_game_quake.c'),),
    'game_treasure': (('EFFECT.C', 'tools/fd2_matching_game_effect_tail.c'),
                     ('QUAKE.C', 'tools/fd2_matching_game_quake.c'))}
COMPILER_INPUTS = {
    "10.0a": INPUTS,
    "9.01": {
        "WCC386.EXE": "99d79830e8bf2bf04582215583cd6c276e21226d86a9bdea197a99abb75eabe0",
        "DOS4GW.EXE": "535d649996de16d1e495633a9aaf60244bc717e7d6951a45a8092c3492ce4005",
    },
    "9.5": {
        "WCC386.EXE": "3fe098187af3ed4bccbf184b0f0d1a21a18cf976fe72c6178f19b8eea8b13e78",
        "DOS4GW.EXE": "b401506365892bd7bcb4362599279504a863f05bf37d323f732fd348bd1ef3c5",
    },
}
CPP_INPUTS = {
    "10.0a": {
        "WPP386.EXE": "647f4e754cd27fc76b3c44a47227ae3baa4ff2e2f253a8ec27216414653b1e6f",
        "W32RUN.EXE": INPUTS["W32RUN.EXE"],
        "DOS4GW.EXE": INPUTS["DOS4GW.EXE"],
    },
}
CPP_WRAPPER = b'extern "C" {\n#include "GAME.C"\n}\n'


def compiler_spec(version, frontend):
    if frontend == 'cpp':
        if version not in CPP_INPUTS:
            raise ValueError('此版本的C++前端尚未鎖版')
        return CPP_INPUTS[version], 'WPP386.EXE'
    if frontend != 'c':
        raise ValueError('未知compiler前端')
    return COMPILER_INPUTS[version], 'WCC386.EXE'


def validate_cpp_wrapper(objects, report):
    path = objects / 'GAME.CPP'
    expected = {'dos_file': 'GAME.CPP', 'sha256': hashlib.sha256(CPP_WRAPPER).hexdigest(),
                'source_date_epoch': SOURCE_DATE_EPOCH}
    if not path.is_file() or path.read_bytes() != CPP_WRAPPER:
        raise ValueError('實際C介面包裝與固定來源不符')
    if int(path.stat().st_mtime) != SOURCE_DATE_EPOCH or report.get('cpp_wrapper') != expected:
        raise ValueError('C介面包裝的SHA或固定時間不符')
    return expected


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def compile_stage(args):
    frontend = getattr(args, 'compiler_frontend', 'c')
    compiler_inputs, compiler_exe = compiler_spec(args.compiler_version, frontend)
    for name, digest in compiler_inputs.items():
        if sha(args.compiler / name) != digest:
            raise ValueError("compiler 組件雜湊不符")
    args.output.mkdir(parents=True, exist_ok=True)
    if args.output.stat().st_uid != os.getuid():
        raise ValueError("輸出目錄擁有權不符")
    source_path, source_cases = SOURCES[args.source_key]
    source = args.repo / source_path
    selected_cases = args.cases or source_cases or [macro for macro, _ in CASES]
    if source_cases and not set(selected_cases).issubset(source_cases):
        raise ValueError("所選來源未提供這些候選")
    cases = [(macro, address) for macro, address in ALL_CASES if macro in selected_cases]
    dos_source = args.output / "GAME.C"
    dos_source.write_bytes(source.read_text(encoding="utf-8").encode("ascii", "ignore"))
    os.utime(dos_source, (SOURCE_DATE_EPOCH, SOURCE_DATE_EPOCH))
    compile_source = 'GAME.C'
    cpp_wrapper = None
    if frontend == 'cpp':
        path = args.output / 'GAME.CPP'
        path.write_bytes(CPP_WRAPPER)
        os.utime(path, (SOURCE_DATE_EPOCH, SOURCE_DATE_EPOCH))
        compile_source = 'GAME.CPP'
        cpp_wrapper = {'dos_file': 'GAME.CPP', 'sha256': sha(path), 'source_date_epoch': SOURCE_DATE_EPOCH}
    includes = []
    for dos_name, source_name in SOURCE_INCLUDES.get(args.source_key, ()):
        include_source = args.repo / source_name
        include_file = args.output / dos_name
        include_file.write_bytes(include_source.read_text(encoding='utf-8').encode('ascii', 'ignore'))
        os.utime(include_file, (SOURCE_DATE_EPOCH, SOURCE_DATE_EPOCH))
        includes.append({'dos_file': dos_name, 'source_path': source_name,
                         'source_sha256': sha(include_source), 'dos_sha256': sha(include_file)})
    lines = ["[sdl]", "output=surface", "[dosbox]", "memsize=31", "[cpu]", "core=normal",
             "cycles=max 90% limit 100000", "[mixer]", "nosound=true", "[midi]", "mpu401=none",
             "mididevice=none", "[autoexec]", f"mount c {args.output}", f"mount r {args.compiler}",
             "c:", "set PATH=R:\\;Z:\\"]
    commands = []
    trials = []
    for index, ((macro, address), cpu, cost) in enumerate(itertools.product(cases, args.cpus, args.costs)):
        stem = f"G{index:02}"
        flags = ["-mf", "-" + cpu, *COSTS[cost], "-d" + macro]
        commands.append("R:\\" + compiler_exe + " " + " ".join(flags) + f" -fo={stem}.OBJ {compile_source} > {stem}.TXT")
        trial = {"stem": stem, "address": hex(address), "flags": flags}
        if macro in CASE_GROUPS or macro in SPARSE_GROUPS:
            trial["addresses"] = [hex(value) for value in (CASE_GROUPS | SPARSE_GROUPS)[macro]]
        trials.append(trial)
    compile_batch = args.output / "BUILD.BAT"
    compile_batch.write_text("\r\n".join(commands) + "\r\n", encoding="ascii")
    lines.extend(["call BUILD.BAT", "exit"])
    config = args.output / "RUN.CONF"
    config.write_text("\n".join(lines) + "\n", encoding="ascii")
    result = subprocess.run(["dosbox", "-conf", str(config), "-noconsole"], capture_output=True, text=True, timeout=90)
    (args.output / "runner.log").write_text(result.stdout + result.stderr, encoding="utf-8")
    if result.returncode or "Exit to error:" in result.stdout + result.stderr:
        raise ValueError("compiler 執行器失敗")
    for trial in trials:
        obj = args.output / (trial["stem"] + ".OBJ")
        text = (args.output / (trial["stem"] + ".TXT")).read_text(encoding="cp437")
        if not obj.is_file() or not re.search(r"\b(?:0|no) errors\b", text) or "Error!" in text:
            raise ValueError(f"{trial['stem']}: compiler 未成功產生物件")
        trial["object_sha256"] = sha(obj)
    report = {"schema_version": 1, "source_key": args.source_key,
              "source_path": source_path, "source_sha256": sha(source),
              "compiler_version": args.compiler_version, "compiler_inputs": compiler_inputs,
              "source_date_epoch": SOURCE_DATE_EPOCH,
              "driver_sha256": sha(Path(__file__)), "trials": trials}
    if includes:
        report['source_includes'] = includes
    if cpp_wrapper is not None:
        report.update(compiler_frontend='cpp', cpp_wrapper=cpp_wrapper)
    (args.output / "compile-report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(f"{len(trials)} 個主程式 C 候選已編譯。", flush=True)


def verify_original_bindings(functions, original, bindings):
    """每個具名參照都由直接call目標或原始LE fixup核對，不從名稱推位址。"""
    from le_xref import parse_le, parse_fixups
    fixups = parse_fixups(original, parse_le(original))
    normalized = {}
    for key in bindings:
        normalized.setdefault(key.lstrip('_'), []).append(key)
    normalized.setdefault('sub_36CD7', []).append('__CHK')
    references = []
    for function in functions:
        for chunk in function['chunks']:
            for instruction in chunk['instructions']:
                code = bytes.fromhex(instruction['loaded_bytes'])
                file_bytes = bytes.fromhex(instruction['file_bytes'])
                offset = int(instruction['file_offset'], 16)
                if original[offset:offset + len(file_bytes)] != file_bytes:
                    raise ValueError('綁定驗證的原始指令版本不符')
                operand_text = instruction['instruction'].split(';',1)[0]
                keys = {key for token in re.findall(r'\b[_A-Za-z][_A-Za-z0-9]*\b',operand_text) for key in normalized.get(token, [])}
                locations = [(p, value) for p, value in fixups.items() if offset <= p < offset + len(code)]
                if len(code) == 5 and code[0] in (0xE8, 0xE9):
                    value = int(instruction['ida_linear_address'], 16) + 5 + int.from_bytes(code[1:], 'little', signed=True)
                    kind = 'direct_rel32'
                elif len(locations) == 1:
                    position, value = locations[0]
                    relative = position - offset
                    if relative + 4 > len(code) or int.from_bytes(code[relative:relative + 4], 'little') != value:
                        raise ValueError('綁定驗證的LE欄位不完整')
                    kind = 'original_le_fixup'
                else:
                    continue
                for key in sorted(keys):
                    # 保留IDA原運算元中的明示常數位移，例如word_53A8D+1。
                    token = key.lstrip('_')
                    suffixes = re.findall(r'\b'+re.escape(token)+r'\s*([+-])\s*(0[xX][0-9A-Fa-f]+|[0-9A-Fa-f]+[hH]|[0-9]+)(?![A-Za-z0-9_])',operand_text)
                    offsets = set()
                    for sign,number in suffixes:
                        amount = int(number[:-1],16) if number[-1:] in ('h','H') else int(number,0 if number.lower().startswith('0x') else 10)
                        offsets.add(-amount if sign=='-' else amount)
                    if len(offsets)>1:
                        raise ValueError('原始具名參照含矛盾常數位移')
                    addend = next(iter(offsets),0)
                    if bindings[key]+addend != value:
                        raise ValueError(f'原始參照與綁定位址不符: {key} at {instruction["ida_linear_address"]}')
                    reference={'symbol': key,'ida_linear_address':instruction['ida_linear_address'],'value':hex(value),'kind':kind}
                    if addend:reference.update(symbol_base=hex(bindings[key]),operand_addend=addend,original_operand=instruction['instruction'])
                    references.append(reference)
    return references


def validate_compound_layout(macro, source_key, function, layout):
    """驗證同一原始owner內的C入口布局，不另建IDA函式或略過任何bytes。"""
    registered = COMPOUND_ENTRIES.get(macro)
    if registered is None or source_key != registered['source_key'] or macro not in SOURCES[source_key][1]:
        raise ValueError('複合入口來源未登錄')
    inventory = function['inventory']
    if int(inventory['start'],16) != registered['owner'] or int(inventory['end'],16) != registered['end'] or inventory['size'] != registered['end']-registered['owner']:
        raise ValueError('複合入口沒有涵蓋原始owner')
    instructions = {int(i['ida_linear_address'],16):i for c in function['chunks'] for i in c['instructions']}
    for entry, limit in registered['prologues'].items():
        if entry not in instructions or bytes.fromhex(instructions[entry]['loaded_bytes']) != b'\x68'+limit.to_bytes(4,'little'):
            raise ValueError('複合入口原始prologue未證實')
    fragments = layout.get('fragments',[])
    if [int(f['address'],16) for f in fragments] != list(registered['compiler_order']):
        raise ValueError('複合入口與compiler順序不符')
    compiler_cursor = 0
    for n,fragment in enumerate(fragments):
        if fragment.get('compiler_offset')!=compiler_cursor or fragment.get('section')!='_M'+str(n):
            raise ValueError('複合入口compiler區段或位移不符')
        compiler_cursor += fragment['size']
    placed = sorted(fragments,key=lambda f:int(f['address'],16))
    cursor = registered['owner']
    for fragment in placed:
        if int(fragment['address'],16) != cursor or fragment['size']<=0:
            raise ValueError('複合入口布局不連續或重疊')
        cursor += fragment['size']
    if cursor != registered['end']:
        raise ValueError('複合入口布局沒有涵蓋完整owner')
    return placed


def link_stage(args):
    from fd2_matching_pilot import checked, linked_text
    from fd2_matching_legacy_verify import OBJCONV_SHA256, text_size
    if sha(args.converter) != OBJCONV_SHA256:
        raise ValueError("物件轉換器雜湊不符")
    image = json.loads(args.evidence.read_text(encoding="utf-8"))
    original = args.original.read_bytes()
    reference = next(x for x in json.loads((args.repo / "docs/data/fd2-reference-files.json").read_text(encoding="utf-8"))["files"]
                     if x["file"] == "FD2.EXE")
    identity = {"file": "FD2.EXE", "size": len(original), "md5": hashlib.md5(original).hexdigest(), "sha256": hashlib.sha256(original).hexdigest()}
    if identity != reference or image["input"] != reference:
        raise ValueError("原檔或 IDA 版本不符")
    compile_report = json.loads((args.objects / "compile-report.json").read_text(encoding="utf-8"))
    compiler_version = compile_report.get("compiler_version", "10.0a")
    frontend = compile_report.get('compiler_frontend', 'c')
    compiler_inputs, compiler_exe = compiler_spec(compiler_version, frontend)
    if compile_report["compiler_inputs"] != compiler_inputs:
        raise ValueError("compiler 組件登記不符")
    source_key = compile_report.get("source_key", "game")
    source_path = SOURCES[source_key][0]
    if compile_report.get("source_path", source_path) != source_path:
        raise ValueError("C 來源路徑未登記")
    if compile_report["source_sha256"] != sha(args.repo / source_path):
        raise ValueError("C 來源與編譯收據不符")
    if (args.objects / "GAME.C").read_bytes() != (args.repo / source_path).read_text(encoding="utf-8").encode("ascii", "ignore"):
        raise ValueError("實際 DOS C 來源與登記來源不符")
    if frontend == 'cpp':
        validate_cpp_wrapper(args.objects, compile_report)
    elif 'cpp_wrapper' in compile_report:
        raise ValueError('C前端不能帶入未驗證的C++包裝')
    expected_includes = []
    for dos_name, source_name in SOURCE_INCLUDES.get(source_key, ()):
        include_source = args.repo / source_name
        expected = include_source.read_text(encoding='utf-8').encode('ascii', 'ignore')
        include_file = args.objects / dos_name
        if not include_file.is_file() or include_file.read_bytes() != expected or int(include_file.stat().st_mtime) != SOURCE_DATE_EPOCH:
            raise ValueError('實際 DOS include 與登錄來源或固定時間不符')
        expected_includes.append({'dos_file': dos_name, 'source_path': source_name,
            'source_sha256': sha(include_source), 'dos_sha256': hashlib.sha256(expected).hexdigest()})
    if compile_report.get('source_includes', []) != expected_includes:
        raise ValueError('C include 登錄與編譯收據不符')
    target_addresses = {address for trial in compile_report['trials'] for address in trial.get('addresses', [trial['address']])}
    binding_references = verify_original_bindings([f for f in image['functions'] if f['inventory']['start'] in target_addresses], original, BINDINGS)
    args.output.mkdir(parents=True, exist_ok=True)
    results = []
    for trial in compile_report["trials"]:
        obj = args.objects / (trial["stem"] + ".OBJ")
        if sha(obj) != trial["object_sha256"]:
            raise ValueError("原始 compiler 物件與收據不符")
        out = args.output / trial["stem"]
        out.mkdir(exist_ok=True)
        coff, dis = out / "candidate.cof", out / "candidate.dis"
        checked([str(args.converter), "-fcoff", str(obj), str(coff)])
        checked(["wdis", "-l=" + str(dis), str(obj)])
        publics = re.findall(r"^([0-9A-Fa-f]{4,8})\s+([_A-Za-z][_A-Za-z0-9]*):", dis.read_text(), re.M)
        symbols = [name for offset, name in publics if int(offset, 16) == 0]
        if len(symbols) != 1:
            raise ValueError("候選必須有唯一的起始函式")
        size, address = text_size(coff), int(trial["address"], 16)
        addresses = trial.get("addresses", [trial["address"]])
        macro = next((flag[2:] for flag in trial["flags"] if flag.startswith("-d")), None)
        sparse = macro in SPARSE_GROUPS
        compound = macro in COMPOUND_ENTRIES
        if len(addresses) > 1:
            if macro not in SOURCES[source_key][1] or addresses != [hex(value) for value in (CASE_GROUPS | SPARSE_GROUPS).get(macro, ())]:
                raise ValueError("多函式區間未依來源登記")
        by_address = {f['inventory']['start']: f for f in image['functions']}
        functions = []
        for target in addresses:
            if target not in by_address:
                if source_key != 'game_event_layout':
                    raise ValueError('來源未登記清冊外區間')
                from fd2_matching_intervals import unowned_interval
                by_address[target] = unowned_interval(int(target, 16), image, original, args.evidence.parent)
            functions.append(by_address[target])
        if functions[0]["inventory"]["start"] != trial["address"]:
            raise ValueError("候選起始位址與函式區間不符")
        group_end = int(functions[-1]["inventory"]["end"], 16)
        if any(not address <= int(i["ida_linear_address"], 16) < group_end for fn in functions for c in fn["chunks"] for i in c["instructions"]):
            raise ValueError("候選區間沒有涵蓋所需的共用尾段")
        intervals = []
        instructions = []
        cursor = address
        for fn in functions:
            start, end = int(fn["inventory"]["start"], 16), int(fn["inventory"]["end"], 16)
            if sparse:
                cursor = start
            if start != cursor:
                raise ValueError("多函式區間必須連續且不重疊")
            part = [i for c in fn["chunks"] for i in c["instructions"] if start <= int(i["ida_linear_address"], 16) < end]
            if len(addresses) == 1 and len(part) != sum(len(c["instructions"]) for c in fn["chunks"]):
                raise ValueError("單函式候選不能略過共用尾段")
            for instruction in part:
                if int(instruction["ida_linear_address"], 16) != cursor:
                    raise ValueError("原始指令不連續")
                cursor += len(bytes.fromhex(instruction["loaded_bytes"]))
            if cursor != end or end - start != fn["inventory"]["size"]:
                raise ValueError("目標指令不涵蓋完整原始函式區間")
            intervals.append({"address": hex(start), "offset": sum(p['size'] for p in intervals) if sparse else start - address, "size": end - start})
            if source_key == 'game_event_layout':
                intervals[-1]['kind'] = fn['inventory'].get('kind', 'ida_function')
                if intervals[-1]['kind'] == 'unowned_code':
                    intervals[-1]['evidence_sha256'] = fn['evidence_sha256']
            instructions.extend(part)
        entry_offsets_equal = len(publics) == len(functions)
        if len(addresses) > 1:
            actual_entries = {name.lstrip("_").lower(): int(offset, 16) for offset, name in publics}
            entry_offsets_equal = entry_offsets_equal and all(actual_entries.get("sub_" + format(int(part["address"], 16), "x")) == part["offset"] for part in intervals)
        expected = bytearray()
        for instruction in instructions:
            offset = int(instruction["file_offset"], 16)
            raw = bytes.fromhex(instruction["file_bytes"])
            if original[offset:offset + len(raw)] != raw:
                raise ValueError("原始指令版本不符")
            expected.extend(bytes.fromhex(instruction["loaded_bytes"]))
        if len(expected) != sum(part["size"] for part in intervals):
            raise ValueError("目標指令不涵蓋完整連續區間")
        script = out / "link.ld"
        definitions = "\n".join(f"{key} = {value:#x};" for key, value in BINDINGS.items())
        sparse_layout = None
        if sparse or compound:
            from fd2_matching_sparse import split_coff, read_sparse_pe
            try:
                positions = list(COMPOUND_ENTRIES[macro]['compiler_order']) if compound else [int(a,16) for a in addresses]
                converted, sparse_layout = split_coff(coff.read_bytes(), dis.read_text(), positions)
                if compound:
                    validate_compound_layout(macro,source_key,functions[0],sparse_layout)
                    entry_offsets_equal = True
            except ValueError as error:
                layout_errors = ('連結後完整函式彼此重疊','不能保持跨函式短分支的原始指令寬度')
                if compound:
                    layout_errors += ('複合入口布局不連續或重疊','複合入口布局沒有涵蓋完整owner')
                if str(error) not in layout_errors:
                    raise
                results.append({**trial, 'candidate_size': size, 'original_size': len(expected),
                    'exact_interval_bytes': False, 'comparison_performed': False,
                    'linkable': False, 'layout': 'compound_entries' if compound else 'sparse_functions', 'layout_rejection': str(error),
                    'function_intervals': intervals, 'exact_entry_offsets': entry_offsets_equal,
                    'original_classification': functions[0]['inventory']['classification']})
                continue
            coff = out / 'sparse.cof'
            coff.write_bytes(converted)
            output_fragments = sorted(sparse_layout['fragments'],key=lambda f:int(f['address'],16)) if compound else sparse_layout['fragments']
            sections = ' '.join(f".m{n} {p['address']} : SUBALIGN(1) {{ *({p['section']}) }}" for n, p in enumerate(output_fragments))
        else:
            sections = f".text {address:#x} : SUBALIGN(1) {{ *(_TEXT) }}"
        script.write_text(definitions + f"\nSECTIONS {{ {sections} "
                          "/DISCARD/ : { *(CONST) *(CONST2) *(_DATA) *(_BSS) *(.depend) *(.reloc) } }\n", encoding="ascii")
        exe = out / "linked.exe"
        entry_symbol = next(name for offset,name in publics if name.lstrip('_').lower()=='sub_'+format(address,'x')) if compound else symbols[0]
        checked(["ld", "-mi386pe", "--image-base", "0", "--section-alignment", "1", "--file-alignment", "1",
                 "--no-insert-timestamp", "-T", str(script), "-e", entry_symbol, "-o", str(exe), str(coff)])
        placed_fragments = sorted(sparse_layout['fragments'],key=lambda f:int(f['address'],16)) if compound else sparse_layout['fragments'] if sparse else None
        code = read_sparse_pe(exe, placed_fragments) if sparse or compound else linked_text(exe, address, size)
        (out / "candidate.bin").write_bytes(code)
        first = next((i for i, pair in enumerate(zip(code, expected)) if pair[0] != pair[1]), min(len(code), len(expected)))
        result = {**trial, "candidate_size": size, "original_size": len(expected), "exact_interval_bytes": code == expected and entry_offsets_equal,
                  "first_difference_offset": None if code == expected else first, "code_sha256": hashlib.sha256(code).hexdigest(),
                  "original_classification": functions[0]["inventory"]["classification"]}
        if len(addresses) > 1:
            result.update(function_intervals=intervals, exact_entry_offsets=entry_offsets_equal,
                          original_classifications=[f["inventory"]["classification"] for f in functions])
        if sparse:
            result.update(layout='sparse_functions', sparse_layout=sparse_layout,
                          sparse_transform_sha256=sha(Path(__file__).with_name('fd2_matching_sparse.py')))
        if compound:
            result.update(layout='compound_entries',compound_layout=sparse_layout,
                function_intervals=intervals,exact_entry_offsets=entry_offsets_equal,
                sparse_transform_sha256=sha(Path(__file__).with_name('fd2_matching_sparse.py')))
        results.append(result)
    report = {"schema_version": 1, "input": reference, "source_key": source_key,
              "source_path": source_path, "source_sha256": compile_report["source_sha256"],
              "compiler_version": compiler_version, "compiler_inputs": compile_report["compiler_inputs"],
              "compile_driver_sha256": compile_report["driver_sha256"],
              "driver_sha256": sha(Path(__file__)), "evidence_sha256": sha(args.evidence), "trials": results,
              "original_binding_references": binding_references,
              "matched_addresses": sorted({part['address'] for r in results if r['exact_interval_bytes']
                    for part in r.get('function_intervals', [{'address': r['address']}]) if part.get('kind', 'ida_function') == 'ida_function'})}
    if expected_includes:
        report['source_includes'] = expected_includes
    if frontend == 'cpp':
        report.update(compiler_frontend='cpp', cpp_wrapper=compile_report['cpp_wrapper'])
    (args.output / "report.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"matched_addresses": report["matched_addresses"], "trials": len(results)}), flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("stage", choices=("compile", "link"))
    parser.add_argument("--repo", type=Path, default=Path("/repo"))
    parser.add_argument("--compiler", type=Path)
    parser.add_argument("--compiler-version", choices=COMPILER_INPUTS, default="10.0a")
    parser.add_argument("--compiler-frontend", choices=('c', 'cpp'), default='c')
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--objects", type=Path)
    parser.add_argument("--evidence", type=Path)
    parser.add_argument("--original", type=Path, default=Path("/input/FD2.EXE"))
    parser.add_argument("--converter", type=Path, default=Path("/work/objconv"))
    parser.add_argument("--source-key", choices=SOURCES, default="game")
    parser.add_argument("--cases", choices=[macro for macro, _ in ALL_CASES], nargs="+")
    parser.add_argument("--cpus", choices=("3s", "4s", "5s"), nargs="+", default=("3s", "4s", "5s"))
    parser.add_argument("--costs", choices=COSTS, nargs="+", default=("balanced", "space", "speed"))
    args = parser.parse_args()
    if args.compiler is None:
        args.compiler = Path('/wc10a-cpp' if args.compiler_frontend == 'cpp' else
                             {"10.0a": "/wc10a", "9.5": "/wc95", "9.01": "/wc901"}[args.compiler_version])
    if not Path("/.dockerenv").exists():
        raise SystemExit("本工具只在 Docker 執行")
    compile_stage(args) if args.stage == "compile" else link_stage(args)


if __name__ == "__main__":
    main()

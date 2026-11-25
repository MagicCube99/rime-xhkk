#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
tools/corpus_toolkit.py
--------------------------------------------------------------------------------
小鹤音形确定性词库分析与冲突检测工具链 (Corpus Modernization Toolkit)

核心职能：
1. FlypyEncoder: 100% 确定性小鹤编码计算引擎（对齐 Go 版本公式，严禁 LLM 心算幻觉）；
2. CorpusDatabase: 全库分层倒排索引与精细冲突探测器（支持全库 0 重码绝对空码扫描）；
3. CLI & TestSuite: 自带完备单元测试套件与批量词库分析管道。
--------------------------------------------------------------------------------
"""

import sys
import os
import argparse
from pathlib import Path
from typing import Dict, List, Tuple, Optional, Any

try:  # 可选依赖：用于多音字词组读音仲裁
    from pypinyin import lazy_pinyin as _lazy_pinyin, Style as _PinyinStyle
except ImportError:  # 缺失时退化为「无读音仲裁」，编码行为保持不变
    _lazy_pinyin = None
    _PinyinStyle = None

REPO_ROOT = Path(__file__).resolve().parents[1]
DICTS_DIR = REPO_ROOT / "flypy_dicts"
ASSETS_DIR = REPO_ROOT / "tools" / "xhkk-builder" / "pkg" / "encoder" / "assets"
FLYPYDZ_PATH = REPO_ROOT / "flypydz.dict.yaml"

# 字典挂载优先级定义（数字越小优先级越高，对齐 flypy.dict.yaml 的 import_tables 顺序）
DICT_PRIORITY = {
    "flypy.user.top": 1,
    "flypy.names": 2,
    "flypy.primary": 3,
    "flypy.custom": 4,
    "flypy.official": 5,
    "flypy.it": 6,
    "flypy.modern": 7,
    "flypy.secondary": 8,
    "flypy.whimsicality": 9,
    "flypy.off-table": 10,
    "flypy.single.code": 11,
    "flypy.secondary.simple": 12,
    "flypy.fast.symbols": 13,
    "flypy.user": 14,
    "flypy.full.code.words": 15,
    "flypy.full.code.chars": 16,
    # places 由独立 lua_translator 挂载（initial_quality: -2），不在主表 import_tables 内
    "flypy.places": 99,
}

# 真实生效的词库层：import_tables 挂载层 + 独立翻译器（places）。
# 未挂载的 ok 拆字库、生僻字、符号、表情不得参与重码统计。
ACTIVE_DICTS = set(DICT_PRIORITY)


class FlypyEncoder:
    """100% 确定性小鹤双拼与四码编码计算引擎"""

    INITIAL_MAP = {
        "zh": "v", "ch": "i", "sh": "u",
        "b": "b", "p": "p", "m": "m", "f": "f", "d": "d", "t": "t",
        "n": "n", "l": "l", "g": "g", "k": "k", "h": "h", "j": "j",
        "q": "q", "x": "x", "r": "r", "z": "z", "c": "c", "s": "s",
        "y": "y", "w": "w"
    }

    FINAL_MAP = {
        "iu": "q", "ei": "w", "e": "e", "uan": "r", "van": "r", "üan": "r",
        "ue": "t", "ve": "t", "üe": "t", "un": "y", "ün": "y", "vn": "y",
        "u": "u", "i": "i", "o": "o", "uo": "o", "ie": "p", "a": "a",
        "ong": "s", "iong": "s", "ai": "d", "en": "f", "eng": "g", "ang": "h",
        "an": "j", "uai": "k", "ing": "k", "iang": "l", "uang": "l", "ou": "z",
        "ia": "x", "ua": "x", "ao": "c", "ui": "v", "v": "v", "ü": "v",
        "in": "b", "iao": "n", "ian": "m", "er": "er"
    }

    ZERO_INITIAL_MAP = {
        "a": "aa", "o": "oo", "e": "ee",
        "ai": "ai", "ei": "ei", "ao": "ao", "ou": "ou",
        "an": "an", "en": "en", "er": "er",
        "ang": "ah", "eng": "eg"
    }

    def __init__(self):
        self.char_map: Dict[str, str] = {}
        self.phrase_exceptions: Dict[str, List[str]] = {}
        self._load_char_assets()

    def _load_char_assets(self):
        """加载标准单字声韵底表与词组特例"""
        # 1. 加载 primary_chars.tsv (7468 常用字)
        primary_tsv = ASSETS_DIR / "primary_chars.tsv"
        if primary_tsv.exists():
            with open(primary_tsv, "r", encoding="utf-8") as f:
                for line in f:
                    parts = line.strip().split("\t")
                    if len(parts) >= 2 and len(parts[0]) == 1:
                        char, flypy_code = parts[0], parts[1].strip()
                        if len(flypy_code) == 2:
                            self.char_map[char] = flypy_code

        # 2. 从 flypydz.dict.yaml 补齐生僻字 (按最高权重取最优音)
        if FLYPYDZ_PATH.exists():
            char_weights: Dict[str, int] = {}
            with open(FLYPYDZ_PATH, "r", encoding="utf-8") as f:
                for line in f:
                    if line.startswith("#") or line.startswith(".") or line.startswith("-"):
                        continue
                    parts = line.strip().split("\t")
                    if len(parts) >= 2 and len(parts[0]) == 1:
                        char, full_code = parts[0], parts[1].strip()
                        weight = int(parts[2]) if len(parts) >= 3 and parts[2].isdigit() else 0
                        if len(full_code) >= 2:
                            dp = full_code[:2]
                            if char not in self.char_map:
                                if char not in char_weights or weight > char_weights[char]:
                                    self.char_map[char] = dp
                                    char_weights[char] = weight

        # 3. 加载词组特例表 phrase_exceptions.tsv (如 蚌埠 bh bu)
        exceptions_tsv = ASSETS_DIR / "phrase_exceptions.tsv"
        if exceptions_tsv.exists():
            with open(exceptions_tsv, "r", encoding="utf-8") as f:
                for line in f:
                    parts = line.strip().split("\t")
                    if len(parts) >= 2:
                        phrase = parts[0]
                        codes = parts[1:]
                        if len(codes) == len(phrase):
                            self.phrase_exceptions[phrase] = codes

    def get_char_flypy(self, char: str) -> Optional[str]:
        """获取单个汉字的 2 码双拼"""
        return self.char_map.get(char)

    def _char_codes(self, word: str) -> List[str]:
        """逐字求 2 码双拼：完整词组特例 > 二字子词特例 > 单字底表。"""
        n = len(word)
        if word in self.phrase_exceptions:
            return list(self.phrase_exceptions[word])

        codes: List[Optional[str]] = [None] * n
        # 1. 二字子词特例覆盖（如 "朝夕"相伴 -> 朝夕优先采用 vc xi 消除多音歧义）
        for i in range(n - 1):
            sub2 = word[i:i+2]
            if sub2 in self.phrase_exceptions:
                sub_codes = self.phrase_exceptions[sub2]
                if codes[i] is None:
                    codes[i] = sub_codes[0]
                if codes[i+1] is None:
                    codes[i+1] = sub_codes[1]

        # 2. 剩余单字查字表底册
        for i, ch in enumerate(word):
            if codes[i] is None:
                c = self.get_char_flypy(ch)
                if not c:
                    raise KeyError(f"未收录汉字 '{ch}'，无法确定双拼")
                codes[i] = c
        return codes  # type: ignore[return-value]

    def encode(self, word: str, target_dict: str = "") -> Tuple[str, str]:
        """
        计算小鹤音形标准词汇编码：
        - 2字词: AaAbBaBb -> c1[:2] + c2[:2] (4码)
        - 3字词:
          - 若为 flypy.names: 主码 AaBaCa (3码), 次码 AaBaCaCb (4码)
          - 其它情况: 主码 AaBaCaCb (4码), 次码空
        - 4字及以上: AaBaCaDa / AaBaCaZa -> c1[0] + c2[0] + c3[0] + cN[0] (4码)

        注意：单字底表对多音字只存一个读音。落库前务必用 audit_polyphone()
        复核词内读音是否与词组语境一致（如 祭奠 不能按 祭=zhài 编码）。
        """
        n = len(word)
        if n < 2:
            raise ValueError(f"词长必须 >= 2: '{word}'")
        codes = self._char_codes(word)

        if n == 2:
            return codes[0][:2] + codes[1][:2], ""
        elif n == 3:
            if target_dict == "flypy.names":
                return codes[0][0] + codes[1][0] + codes[2][0], codes[0][0] + codes[1][0] + codes[2][:2]
            else:
                return codes[0][0] + codes[1][0] + codes[2][:2], ""
        else:
            # 4字及以上取 1, 2, 3, 末字的首声母
            return codes[0][0] + codes[1][0] + codes[2][0] + codes[-1][0], ""

    # --- 多音字仲裁（pypinyin 词组读音）------------------------------------
    def reading_to_flypy(self, reading: str) -> Optional[str]:
        """把不带声调的拼音读音转成 2 码小鹤双拼；无法映射时返回 None。"""
        r = reading.strip().lower().replace("ü", "v")
        if not r:
            return None
        for ini in ("zh", "ch", "sh"):
            if r.startswith(ini) and len(r) > len(ini):
                fin = self.FINAL_MAP.get(r[len(ini):])
                return self.INITIAL_MAP[ini] + fin if fin else None
        if r[0] in "bpmfdtnlgkhjqxrzcsyw" and len(r) > 1:
            fin = self.FINAL_MAP.get(r[1:])
            return r[0] + fin if fin else None
        return self.ZERO_INITIAL_MAP.get(r)

    def pypinyin_codes(self, word: str) -> Optional[List[str]]:
        """按词组的实际读音求 2 码（而非单字默认音）；pypinyin 不可用时返回 None。"""
        if _lazy_pinyin is None:
            return None
        try:
            readings = _lazy_pinyin(word, style=_PinyinStyle.NORMAL)
        except Exception:
            return None
        if len(readings) != len(word):
            return None
        codes: List[str] = []
        for r in readings:
            c = self.reading_to_flypy(r)
            if c is None:
                return None
            codes.append(c)
        return codes

    def audit_polyphone(self, word: str) -> List[Tuple[str, str, str]]:
        """对比底表读音与词组读音，返回冲突项 [(字, 表码, 词码)]。

        空列表 = 无冲突，或无法仲裁（pypinyin 缺失 / 该词已由 phrase_exceptions 人工钉死）。
        底表对多音字只存一个读音，冲突说明 encode() 算出的码可能建立在错读上，
        必须人工确认后才能落库。
        """
        if word in self.phrase_exceptions:
            return []
        try:
            table = self._char_codes(word)
        except (KeyError, ValueError):
            return []
        pp = self.pypinyin_codes(word)
        if pp is None:
            return []
        return [(ch, t, p) for ch, t, p in zip(word, table, pp) if t != p]


class Entry:
    def __init__(self, word: str, code: str, dict_name: str, line_num: int, weight: Optional[int] = None):
        self.word = word
        self.code = code
        self.dict_name = dict_name
        self.line_num = line_num
        self.weight = weight
        self.priority = DICT_PRIORITY.get(dict_name, 99)

    def __repr__(self):
        return f"Entry({self.word}, {self.code}, {self.dict_name}:{self.line_num})"


ADMISSION_ALLOW = "ALLOW"
ADMISSION_BLOCK = "BLOCK"
ADMISSION_REVIEW = "REVIEW"


def decide_admission(word: str, candidates: List["Entry"]) -> Tuple[str, str]:
    """加词准入闸门纯决策函数（自动上屏红线）。

    规则（对齐 docs/16 战略裁决）：
    - 目标编码没有任何候选：绝对空码，放行（新增词可四键自动上屏）；
    - 词条已存在于该编码：幂等，放行；
    - 该编码只有 1 个候选：新增会让该码从「四键即上屏」退化为菜单码，拦截；
    - 该编码已是菜单码：新增只会加大候选噪音，交人工复核。
    """
    for entry in candidates:
        if entry.word == word:
            return ADMISSION_ALLOW, "词条已存在，幂等放行"

    if not candidates:
        return ADMISSION_ALLOW, "绝对空码，新增词可四键自动上屏"

    if len(candidates) == 1:
        incumbent = candidates[0]
        return (
            ADMISSION_BLOCK,
            f"该码已有唯一候选「{incumbent.word}」({incumbent.dict_name})，"
            "新增会使自动上屏退化为菜单；如确需收录，应先做首选置换",
        )

    return (
        ADMISSION_REVIEW,
        f"该码已是菜单码（{len(candidates)} 个候选），新增会加大菜单噪音，需人工判断",
    )


class CorpusDatabase:
    """全库反查倒排索引与冲突检测系统"""

    def __init__(self, encoder: Optional[FlypyEncoder] = None):
        self.encoder = encoder or FlypyEncoder()
        self.code_to_entries: Dict[str, List[Entry]] = {}
        self.word_to_entries: Dict[str, List[Entry]] = {}
        self.total_entries = 0
        self._load_dictionaries()

    def _load_dictionaries(self):
        """扫描加载 flypy_dicts 目录下全部活跃词库"""
        if not DICTS_DIR.exists():
            return

        dict_files = sorted(DICTS_DIR.glob("*.dict.yaml"))
        # places 由独立 lua_translator 从用户目录根挂载，不在 flypy_dicts/ 内
        root_places = REPO_ROOT / "flypy.places.dict.yaml"
        if root_places.exists():
            dict_files.append(root_places)
        for df in dict_files:
            # 排除模板文件
            if df.name.endswith(".example"):
                continue

            dict_stem = df.name.replace(".dict.yaml", "").replace(".yaml", "")
            if dict_stem not in ACTIVE_DICTS:
                continue
            with open(df, "r", encoding="utf-8") as f:
                in_body = False
                for line_num, line in enumerate(f, 1):
                    raw = line.strip()
                    if not raw or raw.startswith("#"):
                        continue
                    if not in_body:
                        if raw == "...":
                            in_body = True
                        continue

                    parts = raw.split("\t")
                    if len(parts) >= 2:
                        word = parts[0]
                        code = parts[1]
                        weight = int(parts[2]) if len(parts) >= 3 and parts[2].isdigit() else None

                        entry = Entry(word, code, dict_stem, line_num, weight)
                        self.code_to_entries.setdefault(code, []).append(entry)
                        self.word_to_entries.setdefault(word, []).append(entry)
                        self.total_entries += 1

        # 对每个编码下的候选，按词库优先级排序（同库保持物理出现顺序）
        for code, entries in self.code_to_entries.items():
            entries.sort(key=lambda e: (e.priority, e.line_num))

    def get_candidates(self, code: str) -> List[Entry]:
        """获取指定编码下的完整有序候选列表"""
        return self.code_to_entries.get(code, [])

    def is_zero_collision(self, code: str) -> bool:
        """断言指定编码是否为 100% 绝对空码（0 冲突）"""
        return len(self.code_to_entries.get(code, [])) == 0

    def analyze_word(self, word: str, target_dict: str = "") -> Dict[str, Any]:
        """对一个词汇执行完整的编码推导与全库冲突审计"""
        primary_code, secondary_code = self.encoder.encode(word, target_dict)
        candidates = self.get_candidates(primary_code)

        result = {
            "word": word,
            "code": primary_code,
            "secondary_code": secondary_code,
            "collision_count": len(candidates),
            "is_zero_collision": len(candidates) == 0,
            "head_entry": candidates[0] if candidates else None,
            "candidates": candidates,
        }
        return result

    def assess_admission(self, word: str, target_dict: str = "") -> Dict[str, Any]:
        """加词准入审计：计算编码并按「自动上屏红线」给出放行/拦截/复核结论。

        若词内存在多音字读音冲突，则算出的编码本身不可信：即便该码为空，
        也不得直接放行，降级为 REVIEW 交人工确认读音。
        """
        code, secondary_code = self.encoder.encode(word, target_dict)
        candidates = self.get_candidates(code)
        verdict, reason = decide_admission(word, candidates)
        conflicts = self.encoder.audit_polyphone(word)
        if conflicts:
            detail = "，".join(f"{ch} 表{t}/词{p}" for ch, t, p in conflicts)
            if verdict == ADMISSION_ALLOW:
                verdict = ADMISSION_REVIEW
                reason = f"多音字读音待确认（{detail}），编码 {code} 不可信"
            else:
                reason += f"；另有多音字读音冲突（{detail}）"
        return {
            "word": word,
            "code": code,
            "secondary_code": secondary_code,
            "candidates": candidates,
            "verdict": verdict,
            "reason": reason,
            "polyphone_conflicts": conflicts,
        }


def run_unit_tests():
    """自动化回归测试套件：验证算法确定性与已知标杆案例"""
    print("=== 开始执行 FlypyEncoder & CorpusDatabase 自动化单元测试 ===")
    encoder = FlypyEncoder()
    db = CorpusDatabase(encoder)

    # 1. 二字词编码测试
    test_cases_2 = [
        ("微信", "wwxb"),
        ("敌我", "diwo"),
        ("知乎", "vihu"),
        ("手机", "uzji"),
        ("苹果", "pkgo"),
        ("开源", "kdyr"),
    ]
    for word, expected in test_cases_2:
        code, _ = encoder.encode(word)
        assert code == expected, f"二字词编码错误: {word} 预期 {expected}, 实际 {code}"
    print("[PASS] 二字词编码测试 (AaAbBaBb) 全部通过")

    # 2. 三字词编码测试 (AaBaCaCb)
    test_cases_3 = [
        ("支付宝", "vfbc"),
        ("区块链", "qklm"),
        ("数字化", "uzhx"),
        ("短视频", "dupb"),
        ("云计算", "yjsr"),
        ("大模型", "dmxk"),
    ]
    for word, expected in test_cases_3:
        code, _ = encoder.encode(word)
        assert code == expected, f"三字词编码错误: {word} 预期 {expected}, 实际 {code}"
    print("[PASS] 三字词编码测试 (AaBaCaCb) 全部通过")

    # 3. 四字及以上词编码测试 (AaBaCaDa / AaBaCaZa)
    test_cases_4 = [
        ("朝夕相伴", "vxxb"),
        ("敌我矛盾", "dwmd"),
        ("不可思议", "bksy"),
        ("高质量发展", "gvlv"),
        ("新质生产力", "xvul"),
    ]
    for word, expected in test_cases_4:
        code, _ = encoder.encode(word)
        assert code == expected, f"四字词编码错误: {word} 预期 {expected}, 实际 {code}"
    print("[PASS] 四字及以上词编码测试 (AaBaCaDa / AaBaCaZa) 全部通过")

    # 4. 全库倒排索引查询测试
    assert not db.is_zero_collision("wwxb"), "wwxb 应该有冲突（威信/微信）"
    assert not db.is_zero_collision("vfbc"), "vfbc 应该有冲突（珍宝/支付宝）"
    assert not db.is_zero_collision("dwmd"), "dwmd 应该有冲突（敌我矛盾）"

    # 验证 wwxb 首选为威信（官方底册）或置顶词
    wwxb_cands = db.get_candidates("wwxb")
    print(f"wwxb 候选测试: {[c.word for c in wwxb_cands[:3]]}")
    assert len(wwxb_cands) >= 1

    # 验证单字全码隔离
    assert not db.is_zero_collision("yib"), "yib 应该为益/肄"
    yib_cands = db.get_candidates("yib")
    assert yib_cands[0].word == "益"

    print("[PASS] 全库倒排索引与冲突探测器全部验证通过！\n")

    # 5. 加词准入闸门测试（自动上屏红线：唯一码默认拒收第二候选）
    from types import SimpleNamespace
    fake = lambda w, d="flypy.primary": SimpleNamespace(word=w, dict_name=d)

    verdict, _ = decide_admission("测试", [])
    assert verdict == ADMISSION_ALLOW, f"空码应放行, 实际 {verdict}"

    verdict, _ = decide_admission("微信", [fake("微信")])
    assert verdict == ADMISSION_ALLOW, f"重复词条应放行, 实际 {verdict}"

    verdict, _ = decide_admission("宝安", [fake("保安")])
    assert verdict == ADMISSION_BLOCK, f"唯一码新增第二候选应拦截, 实际 {verdict}"

    verdict, _ = decide_admission("测验", [fake("知识"), fake("只是")])
    assert verdict == ADMISSION_REVIEW, f"菜单码新增应人工复核, 实际 {verdict}"

    # 真实词库端到端
    res = db.assess_admission("机赏")
    assert res["code"] == "jiuh" and res["verdict"] == ADMISSION_BLOCK, f"机赏应因 jiuh 唯一候选被拦截: {res}"
    res = db.assess_admission("宝安")
    assert res["verdict"] == ADMISSION_ALLOW, f"宝安已在 places 词库中，应幂等放行: {res}"
    res = db.assess_admission("微信")
    assert res["verdict"] == ADMISSION_ALLOW, f"微信为既有词条应放行: {res}"
    print("[PASS] 加词准入闸门（空码放行 / 唯一码拦截 / 菜单码复核）全部验证通过！\n")

    # 6. 多音字读音仲裁（底表只存单音，落库前必须按词组读音复核）
    assert encoder.encode("祭奠")[0] == "jidm", "祭奠 应按 ji+dian 编码为 jidm"
    assert encoder.encode("卵石")[0] == "lrui", "卵石 应按 luan+shi 编码为 lrui"
    assert encoder.encode("桨叶")[0] == "jlye", "桨叶 应按 jiang+ye 编码为 jlye"
    assert encoder.audit_polyphone("作证") == [], "作证 不应报多音字冲突"
    assert encoder.pypinyin_codes("祭奠") == ["ji", "dm"], "pypinyin 应识别 祭奠 = ji dian"
    assert encoder.pypinyin_codes("石") == ["ui"], "石 的常用读音应为 shi(ui)"
    # 构造一张错读底表：冲突必须被审计捕获，不能静默算出错码
    bad = FlypyEncoder()
    bad.char_map["祭"] = "vd"
    assert bad.audit_polyphone("祭奠"), "底表把 祭 读成 zhai 时必须报冲突"
    assert bad.encode("祭奠")[0] == "vddm", "错读底表下编码为 vddm（正是审计要拦下的情况）"
    print("[PASS] 多音字读音仲裁（祭奠/卵石/桨叶 按词组读音编码）！\n")

    # 7. 词库挂载面测试（只统计真实生效的词库层，排除未挂载的 ok/生僻字/符号/表情）
    active_names = {e.dict_name for entries in db.code_to_entries.values() for e in entries}
    for inactive in ("flypy.ok", "flypy.rare.chars", "flypy.symbols", "flypy.emoji"):
        assert inactive not in active_names, f"未挂载词库 {inactive} 不应参与冲突统计"
    for mounted in ("flypy.primary", "flypy.modern", "flypy.places", "flypy.user"):
        assert mounted in active_names, f"挂载词库 {mounted} 应参与冲突统计"
    assert DICT_PRIORITY["flypy.modern"] < DICT_PRIORITY["flypy.secondary"], "modern 应排在 secondary 之前"
    assert 60_000 <= db.total_entries <= 90_000, f"挂载条目总量异常: {db.total_entries}"
    print(f"[PASS] 词库挂载面校验通过（{db.total_entries:,} 条，排除未挂载层）！\n")


def main():
    parser = argparse.ArgumentParser(description="小鹤音形词库确定性分析与检测工具链")
    parser.add_argument("--test", action="store_true", help="执行内建自动化测试套件")
    parser.add_argument("--encode", nargs="+", help="计算一个或多个词汇的编码并显示冲突情况")
    parser.add_argument("--guard", nargs="+", help="加词准入审计：按自动上屏红线判断新增词是否会破坏四码唯一上屏")
    parser.add_argument("--stats", action="store_true", help="打印当前词库总览统计信息")
    parser.add_argument("--scan-file", type=str, help="扫描文本文件中的词汇列表并输出冲突/空码报告")
    parser.add_argument("--output-empty", type=str, help="将扫描出的纯空码词条输出到指定文件")

    args = parser.parse_args()

    if args.test:
        run_unit_tests()
        return

    encoder = FlypyEncoder()
    db = CorpusDatabase(encoder)

    if args.stats:
        print(f"=== rime-xhkk 全库倒排索引统计 ===")
        print(f"已加载词典条目总量: {db.total_entries:,} 条")
        print(f"覆盖独立编码数量: {len(db.code_to_entries):,} 个")
        single_count = sum(1 for entries in db.code_to_entries.values() if len(entries) == 1)
        multi_count = len(db.code_to_entries) - single_count
        print(f"四码唯一无重码编码: {single_count:,} 个 ({single_count/len(db.code_to_entries)*100:.1f}%)")
        print(f"存在重码的编码数量: {multi_count:,} 个 ({multi_count/len(db.code_to_entries)*100:.1f}%)")
        return

    if args.encode:
        print(f"=== 词汇编码与冲突分析 ===")
        for w in args.encode:
            try:
                res = db.analyze_word(w)
                status = "[0重码·纯空码]" if res["is_zero_collision"] else f"[重码数: {res['collision_count']}]"
                print(f"词汇: {res['word']:<8} 编码: {res['code']:<6} 状态: {status}")
                if not res["is_zero_collision"]:
                    cands_preview = ", ".join([f"{c.word}({c.dict_name})" for c in res["candidates"][:4]])
                    print(f"  └─ 冲突前排: {cands_preview}")
            except Exception as e:
                print(f"词汇: {w:<8} 错误: {e}")
        return

    if args.guard:
        print("=== 加词准入审计（自动上屏红线） ===")
        blocked = 0
        for w in args.guard:
            try:
                res = db.assess_admission(w)
            except Exception as e:
                print(f"BLOCK  {w:<8} 错误: {e}")
                blocked += 1
                continue
            cands = res["candidates"]
            print(f"{res['verdict']:<6} {w:<8} 编码 {res['code']:<6} 现有候选 {len(cands)}")
            print(f"  └─ {res['reason']}")
            if res.get("polyphone_conflicts"):
                detail = "，".join(f"{ch} 表{t}/词{p}" for ch, t, p in res["polyphone_conflicts"])
                print(f"  └─ ⚠ 多音字: {detail}")
            if cands:
                preview = ", ".join(f"{c.word}({c.dict_name})" for c in cands[:5])
                print(f"  └─ 现有: {preview}")
            if res["verdict"] != ADMISSION_ALLOW:
                blocked += 1
        if blocked:
            print(f"\n共 {blocked} 个词未获放行（BLOCK 需首选置换，REVIEW 需人工判断）。")
            sys.exit(1)
        print("\n全部放行。")
        return

    if args.scan_file:
        input_path = Path(args.scan_file)
        if not input_path.exists():
            print(f"错误: 文件不存在 {input_path}")
            return

        with open(input_path, "r", encoding="utf-8") as f:
            words = [line.strip().split()[0] for line in f if line.strip() and not line.strip().startswith("#")]

        print(f"=== 开始扫描外部词表: {input_path.name} (共 {len(words)} 词) ===")
        empty_list = []
        collided_list = []
        skipped_list = []

        for w in words:
            if len(w) < 2:
                continue
            try:
                res = db.analyze_word(w)
                if res["is_zero_collision"]:
                    empty_list.append((res["word"], res["code"]))
                else:
                    collided_list.append((res["word"], res["code"], res["candidates"]))
            except Exception:
                skipped_list.append(w)

        print(f"扫描完成:")
        print(f"  ✅ 100% 绝对空码（零重码红利）: {len(empty_list)} 词")
        print(f"  ⚠️ 存在重码冲突（需评估置换）: {len(collided_list)} 词")
        if skipped_list:
            print(f"  ❌ 编码失败（生僻字无法定音）: {len(skipped_list)} 词")

        if args.output_empty and empty_list:
            out_p = Path(args.output_empty)
            with open(out_p, "w", encoding="utf-8", newline="\n") as f:
                f.write("# rime-xhkk 自动开垦绝对空码词库候选表\n")
                f.write("# 格式: 词条\\t编码\\n\\n")
                for w, c in empty_list:
                    f.write(f"{w}\t{c}\n")
            print(f"已将 {len(empty_list)} 个纯空码词汇写入: {out_p.name}")

        return

    parser.print_help()


if __name__ == "__main__":
    main()

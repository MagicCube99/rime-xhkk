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

REPO_ROOT = Path(__file__).resolve().parents[1]
DICTS_DIR = REPO_ROOT / "flypy_dicts"
ASSETS_DIR = REPO_ROOT / "tools" / "xhkk-builder" / "pkg" / "encoder" / "assets"
FLYPYDZ_PATH = REPO_ROOT / "flypydz.dict.yaml"

# 字典挂载优先级定义（数字越小优先级越高，对齐 flypy.dict.yaml）
DICT_PRIORITY = {
    "flypy.user.top": 1,
    "flypy.names": 2,
    "flypy.primary": 3,
    "flypy.custom": 4,
    "flypy.official": 5,
    "flypy.it": 6,
    "flypy.secondary": 7,
    "flypy.whimsicality": 8,
    "flypy.off-table": 9,
    "flypy.single.code": 10,
    "flypy.secondary.simple": 11,
    "flypy.fast.symbols": 12,
    "flypy.user": 13,
    "flypy.full.code.words": 14,
    "flypy.places": 15,
    "flypy.full.code.chars": 16,
}


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

    def encode(self, word: str, target_dict: str = "") -> Tuple[str, str]:
        """
        计算小鹤音形标准词汇编码：
        - 2字词: AaAbBaBb -> c1[:2] + c2[:2] (4码)
        - 3字词:
          - 若为 flypy.names: 主码 AaBaCa (3码), 次码 AaBaCaCb (4码)
          - 其它情况: 主码 AaBaCaCb (4码), 次码空
        - 4字及以上: AaBaCaDa / AaBaCaZa -> c1[0] + c2[0] + c3[0] + cN[0] (4码)
        """
        n = len(word)
        if n < 2:
            raise ValueError(f"词长必须 >= 2: '{word}'")

        # 1. 优先查词组完整特例
        if word in self.phrase_exceptions:
            codes = list(self.phrase_exceptions[word])
        else:
            codes = [None] * n
            # 2. 检查 2 字子词特例覆盖（如 "朝夕"相伴 -> 朝夕优先采用 vc xi 消除多音歧义）
            for i in range(n - 1):
                sub2 = word[i:i+2]
                if sub2 in self.phrase_exceptions:
                    sub_codes = self.phrase_exceptions[sub2]
                    if codes[i] is None:
                        codes[i] = sub_codes[0]
                    if codes[i+1] is None:
                        codes[i+1] = sub_codes[1]

            # 3. 剩余单字查字表底册
            for i, ch in enumerate(word):
                if codes[i] is None:
                    c = self.get_char_flypy(ch)
                    if not c:
                        raise KeyError(f"未收录汉字 '{ch}'，无法确定双拼")
                    codes[i] = c

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
        for df in dict_files:
            # 排除模板文件
            if df.name.endswith(".example"):
                continue

            dict_stem = df.name.replace(".dict.yaml", "").replace(".yaml", "")
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


def main():
    parser = argparse.ArgumentParser(description="小鹤音形词库确定性分析与检测工具链")
    parser.add_argument("--test", action="store_true", help="执行内建自动化测试套件")
    parser.add_argument("--encode", nargs="+", help="计算一个或多个词汇的编码并显示冲突情况")
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

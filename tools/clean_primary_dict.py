#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
clean_primary_dict.py
对 flypy.primary.dict.yaml 进行系统性伪词与语法碎片清洗，
根治恶性重码，彻底恢复成语与核心词的首选地位。
"""

import os
import shutil
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
PRIMARY_FILE = REPO_ROOT / "flypy_dicts" / "flypy.primary.dict.yaml"

# 1. 显式成语霸凌伪词（必须拔除）
DIRECT_HIJACK = {
    "得买",     # dwmd -> 恢复 敌我矛盾
    "谁家",     # uwjx -> 恢复 事无巨细
    "谁去",     # uwqu -> 恢复 舍我其谁
    "谁在",     # uwzd -> 恢复 手舞足蹈
    "扭扭腰",   # nnyc -> 恢复 念念有词
    "贼棒",     # zwbh -> 恢复 自我保护
}

# 2. 严格保护的白名单词汇（绝不可误删）
PROTECTED_WORDS = {
    # 真实代词 / 连词 / 熟语
    "这样", "这么", "这些", "这里", "这边", "这个", "这次", "这时", "这儿",
    "这厮", "这厢", "这就", "这下", "这头", "这俩",
    "那样", "那么", "那些", "那里", "那边", "那个", "那次", "那时", "那儿",
    "那厮", "那厢", "那就", "那般", "这般", "哪怕", "那啥",
    "谁知", "谁人", "谁谁",
    # 真实名词 / 成语 / 能愿名词
    "能力", "能量", "能耐", "能够", "能手", "能人", "能源", "能效", "能耗", "能动", "能臣", "能否",
    "肯定",
    # 动词 / 介词固定搭配
    "得到", "得知", "得以", "得体", "得当", "得失", "得病", "得救", "得分", "得法", "得逞", "得宠", "得出",
    "得奖", "得劲", "得空", "得了", "得嘞", "得来", "得力", "得票", "得胜", "得手", "得闲", "得悉", "得意",
    "得罪", "得志", "得益", "得便", "得胜", "得计",
    "跟随", "跟头", "跟斗", "跟前", "跟班", "跟从", "跟进", "跟梢", "跟着", "跟庄", "跟踪",
    "给以", "给予", "给付", "给脸", "给力",
    "随便", "随从", "随同", "随和", "随手", "随时", "随后", "随意", "随性", "随身", "随行", "随处",
    "随附", "随笔", "随感", "随员", "随葬", "随带", "随遇", "随喜", "随风", "随访", "随地", "随队",
    "随机", "随军", "随口", "随礼", "随俗", "随缘", "随即",
    "被单", "被动", "被告", "被褥", "被套", "被窝", "被子", "被迫",
    "让步", "让渡", "让利", "让贤", "让座", "让位", "让路", "让开",
    # 阶段三（交叉比对白名单保护词）
    "贼人", "贼窝", "贼心", "贼眼", "贼子", "贼寇", "贼眉", "贼头", "贼骨", "贼星", "贼赃", "贼兵", "贼党", "贼匪", "贼鸥",
    "又能", "又要", "又及", "又名", "又红",
    "挺拔", "挺立", "挺进", "挺秀", "挺身", "挺直", "挺举", "挺好", "挺生",
    "蛮横", "蛮荒", "蛮夷", "蛮缠", "蛮力", "蛮风",
    "别处", "别称", "别传", "别号", "别针", "别墅", "别致", "别院", "别名", "别样", "别字", "别扭", "别人", "别离", "别的", "别绪", "别情",
    "别克", "别提", "别急", "别来", "别开", "别树", "别具", "别裁"
}

def is_pseudo_phrase(word: str, code: str) -> bool:
    """判定是否为不应作为独立词组的机械搭配碎片"""
    if word in DIRECT_HIJACK:
        return True

    if word in PROTECTED_WORDS:
        return False

    # 规则 1: "得(děi)" + 动作动词 (双拼 dw 开头)
    if len(word) == 2 and word.startswith("得") and code.startswith("dw"):
        return True

    # 规则 2: "谁" + 动词/谓词 (全部为笛卡尔积暴力枚举)
    if len(word) == 2 and word.startswith("谁"):
        return True

    # 规则 3: "这/那" + 任意单字量词/名词 (如 这包, 这杯, 这锅, 那瓶, 那艘...)
    if len(word) == 2 and word[0] in ("这", "那"):
        return True

    # 规则 4: "敢/肯/能" + 单字动词 (非成词，如 敢打, 敢买, 能吃, 能走, 肯做)
    if len(word) == 2 and word[0] in ("敢", "肯", "能"):
        return True

    # 规则 5: 阶段三清洗（借鉴雾凇/白霜比对产物）
    # "贼/又/挺/蛮" + 口语形容词 (方言/口语碎片)
    if len(word) == 2 and word[0] in ("贼", "又", "挺", "蛮"):
        return True

    # 规则 6: 机械动作/碎片搭配
    if word in {
        "跟车", "跟唱", "跟打", "跟去", "跟了", "跟拍", "跟单", "跟丢", "跟团", "跟上", "跟手", "跟在",
        "给点", "给到", "给回", "给出", "给了", "给钱", "给油", "给足", "给人",
        "随你", "随他", "随我", "随车", "随拍", "随迁", "随心", "随团",
        "让车", "让给", "让过", "让出", "让球", "让人", "让行", "让下", "让道",
        "被盗", "被黑", "被困", "被扣", "被坑", "被骂", "被虐", "被骗", "被喷", "被抢", "被人",
        "被偷", "被杀", "被耍", "被捉", "被炸", "被抓", "被淹", "被咬", "被砸",
        "别带", "别丢", "别放", "别改", "别见", "别夸", "别骗", "别抢", "别碰", "别惹", "别送", "别偷",
        "别催", "别乱", "别管", "别买", "别写", "别删", "别闹", "别打", "别看", "别去", "别走", "别停", "别怕",
        "很嗨", "很亏", "很糗", "很贪", "很晒", "很拽", "很邪", "很贼",
        "会读", "过些", "好兵", "好亏", "好拽", "好着", "乱按", "乱点", "乱唱", "乱填", "乱贴", "乱射",
        "快从", "快换", "快买", "快跳", "快学", "少补", "少动", "少放", "少亏", "少算", "少提", "少听", "少言",
        "再错", "再短", "再丢", "再贵", "再闹", "再穷", "再约", "没亏", "没票", "没陪", "没算", "没雨", "没咋"
    }:
        return True

    return False


def purge_dictionary(src_path: Path, dry_run: bool = False):
    with open(src_path, "r", encoding="utf-8") as f:
        lines = f.readlines()

    kept_lines = []
    removed_entries = []

    for line_num, line in enumerate(lines, 1):
        raw = line.rstrip("\r\n")
        # 保留注释、空行与元数据头
        if not raw or raw.startswith("#") or raw.startswith("-") or raw.startswith("."):
            kept_lines.append(line)
            continue

        parts = raw.split("\t")
        if len(parts) >= 2:
            word, code = parts[0], parts[1]
            if is_pseudo_phrase(word, code):
                removed_entries.append((line_num, word, code))
                continue

        kept_lines.append(line)

    print(f"[{src_path.name}] 读取总行数: {len(lines)}")
    print(f"[{src_path.name}] 拟剔除伪词数量: {len(removed_entries)}")
    print(f"[{src_path.name}] 保留有效行数: {len(kept_lines)}")

    if not dry_run:
        # 备份源文件
        backup_path = src_path.with_suffix(".dict.yaml.pre_purge")
        shutil.copy2(src_path, backup_path)
        print(f"已创建备份: {backup_path.name}")

        with open(src_path, "w", encoding="utf-8", newline="\n") as f:
            f.writelines(kept_lines)
        print(f"已写入清洗后词典: {src_path.name}")

    return removed_entries


if __name__ == "__main__":
    print("=== 开始执行小鹤音形词库伪词减法清洗 ===")
    removed = purge_dictionary(PRIMARY_FILE, dry_run=False)

    print(f"\n成功剔除 {len(removed)} 条伪词！部分剔除样本：")
    for idx, w, c in removed[:25]:
        print(f"  Line {idx:5d}: {w} ({c})")
    print("  ...")
    for idx, w, c in removed[-25:]:
        print(f"  Line {idx:5d}: {w} ({c})")

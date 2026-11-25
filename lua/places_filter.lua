-- lua/places_filter.lua
-- -------------------------------------------------------------
-- 地名自适应水位排位滤镜 (Places Adaptive Ranking Filter)
--
-- 核心设计准则：
-- 1. 无常规词时 (num_common == 0)：纯地名编码直接做第 1 候选（首选秒出，四码顶屏秒出）；
-- 2. 少量常规词时 (num_common <= 3)：常规词排前排（首选/分号位），地名自然顺延；
-- 3. 常规词较多时 (num_common >= 4)：第一页黄金席位留给常用词，地名顺延推至第二页（第 6 位起）；
-- 4. 彻底消除地名在多常用词场景下的霸屏干扰。
-- -------------------------------------------------------------

local string_find = string.find
local table_insert = table.insert
local math_min = math.min

local function places_filter(input, env)
    local common_cands = {}
    local place_cands = {}

    for cand in input:iter() do
        local comment = cand.comment
        if comment and string_find(comment, "〔地名〕", 1, true) then
            table_insert(place_cands, cand)
        else
            table_insert(common_cands, cand)
        end
    end

    local num_common = #common_cands
    local num_places = #place_cands

    -- 1. 如果没有地名候选，直接极速透传常规词
    if num_places == 0 then
        for i = 1, num_common do
            yield(common_cands[i])
        end
        return
    end

    -- 2. 场景一：无常规词（纯地名编码），地名直接作为第 1 候选秒出！
    if num_common == 0 then
        for i = 1, num_places do
            yield(place_cands[i])
        end
        return
    end

    -- 3. 场景二：常规词较少 (1 ~ 3 个)
    -- 常规词优先输出（保住第 1 首选位、第 2 分号次选位）；地名自然紧随其后顺延输出
    if num_common <= 3 then
        for i = 1, num_common do
            yield(common_cands[i])
        end
        for i = 1, num_places do
            yield(place_cands[i])
        end
        return
    end

    -- 4. 场景三：常规词较多 (>= 4 个)
    -- 第一页（前 5 位）尽量留给常用词，地名顺延推到第二页（第 6 位起）
    local p1_limit = math_min(num_common, 5)
    for i = 1, p1_limit do
        yield(common_cands[i])
    end

    -- 随后输出地名（出现在第二页前排）
    for i = 1, num_places do
        yield(place_cands[i])
    end

    -- 最后输出第 6 个及以后的低频常规词
    if num_common > 5 then
        for i = 6, num_common do
            yield(common_cands[i])
        end
    end
end

local M = {
    init = function(env) end,
    func = places_filter,
    filter = places_filter,
}

return M

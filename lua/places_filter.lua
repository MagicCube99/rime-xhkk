-- lua/places_filter.lua
-- -------------------------------------------------------------
-- 地名自适应水位排位滤镜 (Places Adaptive Ranking Filter)
--
-- 核心设计准则：
-- 1. 无常规词时 (num_common == 0)：纯地名编码直接做第 1 候选（首选秒出，四码顶屏秒出）；
-- 2. 少量常规词时 (num_common <= 3)：常规词排前排（首选/分号位），地名自然顺延；
-- 3. 常规词较多时 (num_common >= 4)：第一页黄金席位留给常用词，地名顺延推至第二页（第 6 位起）；
-- 4. 彻底消除地名在多常用词场景下的霸屏干扰。
--
-- 性能架构升级：
-- - Fast Path 1：未满 4 码纯流式透传（地名库 100% 为 4 码，1~3 码 0% 有地名）；
-- - Fast Path 2：4 码无地名编码纯流式透传（95%+ 输入 0 开销瞬退，不排干候选流）；
-- - cand.type == "places" 原生比对替代 comment 提取与 string.find；
-- - 静态 table pool 复用，彻底消除每次击键的临时堆表分配。
-- -------------------------------------------------------------

local places_translator = require("places_translator")
local table_insert = table.insert
local math_min = math.min

-- 静态对象池（避免每键向 Lua GC 堆申请 table）
local common_pool = {}
local place_pool = {}
local function clear_pool(t)
    for i = #t, 1, -1 do t[i] = nil end
end

local function places_filter(input, env)
    local raw_input = env.engine.context.input

    -- 性能守卫 1：全库 4,836 条地名 100% 为 4 码。
    -- 输入未达 4 码时（1~3 码编码，占常规击键 60%+），绝不可能有地名候选，0 开销纯流式透传！
    if not raw_input or #raw_input < 4 then
        for cand in input:iter() do
            yield(cand)
        end
        return
    end

    -- 性能守卫 2：极速探查当前 4 码是否存在于地名库中（95%+ 的 4 码非地名）
    -- 若无地名，直接纯流式透传，避免破坏惰性迭代与排干候选流
    if not places_translator.has_code(raw_input) then
        for cand in input:iter() do
            yield(cand)
        end
        return
    end

    -- 仅在确认存在地名候选时才启动排位缓冲（对象池复用，0 堆内存分配）
    clear_pool(common_pool)
    clear_pool(place_pool)

    for cand in input:iter() do
        -- 核心优化：直接比较 C++ 原生 Candidate.type，彻底消除提取 comment 生成 Lua 字符串与 string.find 开销
        if cand.type == "places" then
            table_insert(place_pool, cand)
        else
            table_insert(common_pool, cand)
        end
    end

    local num_common = #common_pool
    local num_places = #place_pool

    -- 1. 如果没有地名候选，直接极速透传常规词
    if num_places == 0 then
        for i = 1, num_common do
            yield(common_pool[i])
        end
        return
    end

    -- 2. 场景一：无常规词（纯地名编码），地名直接作为第 1 候选秒出！
    if num_common == 0 then
        for i = 1, num_places do
            yield(place_pool[i])
        end
        return
    end

    -- 3. 场景二：常规词较少 (1 ~ 3 个)
    -- 常规词优先输出（保住第 1 首选位、第 2 分号次选位）；地名自然紧随其后顺延输出
    if num_common <= 3 then
        for i = 1, num_common do
            yield(common_pool[i])
        end
        for i = 1, num_places do
            yield(place_pool[i])
        end
        return
    end

    -- 4. 场景三：常规词较多 (>= 4 个)
    -- 第一页（前 5 位）尽量留给常用词，地名顺延推到第二页（第 6 位起）
    local p1_limit = math_min(num_common, 5)
    for i = 1, p1_limit do
        yield(common_pool[i])
    end

    -- 随后输出地名（出现在第二页前排）
    for i = 1, num_places do
        yield(place_pool[i])
    end

    -- 最后输出第 6 个及以后的低频常规词
    if num_common > 5 then
        for i = 6, num_common do
            yield(common_pool[i])
        end
    end
end

local M = {
    init = function(env) end,
    func = places_filter,
    filter = places_filter,
}

return M

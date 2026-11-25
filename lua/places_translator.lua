-- lua/places_translator.lua
-- -------------------------------------------------------------
-- 专属地名翻译器 (Places Translator)
--
-- 为什么不用 table_translator@places：
--   table_translator 生成的候选，其注释来自词条固有注释；本地名词条没有注释，
--   而 librime 的 comment_format 对空注释不生效（xform/^/〔地名〕/ 不会凭空
--   造出注释），导致下游滤镜无法识别地名候选，自适应排位彻底失效。
--   改由 Lua 翻译器直接产出带「〔地名〕」身份的候选，身份 100% 确定。
--
-- 数据来源：flypy.places.dict.yaml（与排位滤镜共用同一份数据，零漂移）
-- 惰性加载：首个按键时读入一次，约 4,900 条、1~2ms，静默无感。
-- -------------------------------------------------------------

local M = {}

local places = nil          -- [code] = { text1, text2, ... }
local is_loaded = false

local function load_places()
    is_loaded = true

    local dir = rime_api.get_user_data_dir() or "."
    local candidates = {
        dir .. "/flypy.places.dict.yaml",
        "flypy.places.dict.yaml",
        dir .. "/flypy_dicts/flypy.places.dict.yaml",
    }

    local f = nil
    for i = 1, #candidates do
        f = io.open(candidates[i], "r")
        if f then break end
    end
    if not f then return end

    local table_ = {}
    local in_body = false
    for line in f:lines() do
        if not in_body then
            if line:sub(1, 3) == "..." then
                in_body = true
            end
        else
            local tab = line:find("\t", 1, true)
            if tab and tab > 1 then
                local text = line:sub(1, tab - 1)
                local code = line:sub(tab + 1)
                if code:sub(-1) == "\r" then code = code:sub(1, -2) end
                -- 忽略权重等额外列
                local extra = code:find("\t", 1, true)
                if extra then code = code:sub(1, extra - 1) end

                if #code >= 2 then
                    local list = table_[code]
                    if not list then
                        list = {}
                        table_[code] = list
                    end
                    list[#list + 1] = text
                end
            end
        end
    end
    f:close()

    places = table_
end

function M.init(env)
    -- 惰性加载：初始化时不读文件，保障输入法会话毫秒级即启
end

function M.has_code(code)
    if not is_loaded then
        load_places()
    end
    return places ~= nil and places[code] ~= nil
end

function M.func(input, seg, env)
    if not is_loaded then
        load_places()
    end
    if not places then return end

    local code = input:sub(seg.start + 1, seg._end)
    local list = places[code]
    if not list then return end

    for i = 1, #list do
        yield(Candidate("places", seg.start, seg._end, list[i], "〔地名〕"))
    end
end

return M

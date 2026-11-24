-- lua/dingshangping.lua
-- ---------------------------------------------------------
-- 小鹤音形四码顶字上屏处理器 (Dingshangping)
-- 机制：在输入满4位纯字母且存在候选时，按下第5码字母[a-z]，
--       自动将第1候选（首选词）推上屏并清空旧编码，
--       放行第5码作为新词的第1码无缝开始输入。
-- ---------------------------------------------------------

local P = {}

function P.init(env)
    local config = env.engine.schema.config
    env.enabled = config:get_bool("dingshangping/enable")
    if env.enabled == nil then
        env.enabled = true
    end
end

function P.func(key, env)
    if not env.enabled then return 2 end -- 2 = kNoop
    if key:release() or key:alt() or key:ctrl() then return 2 end

    local context = env.engine.context
    -- 方案开关拦截：若关闭了顶屏开关，微秒级直接放行
    if not context:get_option("dingshangping") then return 2 end
    if not context:is_composing() then return 2 end
    if context:get_option("ascii_mode") then return 2 end

    local input = context.input
    -- 仅对 4 位纯字母编码生效（排除所有带前缀、符号、特殊引导的长编码或短编码）
    if #input ~= 4 or not input:match("^[a-z]+$") then
        return 2
    end

    local key_repr = key:repr()
    -- 仅当按下的是小写英文字母 [a-z]（即新词的起始声母）时触发顶屏
    -- 空格、分号、数字、标点、退格、回车等直接放行给系统正常处理
    if #key_repr ~= 1 or not key_repr:match("^[a-z]$") then
        return 2
    end

    -- 若当前 4 码存在候选菜单
    if context:has_menu() then
        local cand = context:get_selected_candidate()
        if cand then
            -- 提交首选词
            env.engine:commit_text(cand.text)
            -- 清空当前输入
            context:clear()
            -- 返回 kNoop(2)，让新输入的字母顺延给 speller 开启新词输入
            return 2
        end
    else
        -- 若当前 4 码为空码（无候选），直接清空无效编码，让新字母开启新输入
        context:clear()
        return 2
    end

    return 2
end

return P

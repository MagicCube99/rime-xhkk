-- lua/dingshangping.lua
-- ---------------------------------------------------------
-- 小鹤音形四码顶字上屏处理器 (Dingshangping)
-- 机制：在输入满4位纯字母且存在候选时，按下第5码字母[a-z]，
--       自动将第1候选（首选词）推上屏并清空旧编码，
--       放行第5码作为新词的第1码无缝开始输入。
-- ---------------------------------------------------------

local P = {}

function P.init(env)
    -- 开启 Lua 5.4 原生分代垃圾回收（新生代快速释放）
    if collectgarbage then
        pcall(collectgarbage, "generational")
    end

    local config = env.engine.schema.config
    env.enabled = config:get_bool("dingshangping/enable")
    if env.enabled == nil then
        env.enabled = true
    end
end

function P.func(key, env)
    if not env.enabled then return 2 end -- 2 = kNoop
    if key:release() or key:alt() or key:ctrl() then return 2 end

    -- 性能优化：直接使用原生整数 keycode (97='a', 122='z')，0 堆内存分配，1 纳秒判断
    local kc = key.keycode
    if kc < 97 or kc > 122 then
        return 2
    end

    local context = env.engine.context
    -- 方案开关拦截：若关闭了顶屏开关，微秒级直接放行
    if not context:get_option("dingshangping") then return 2 end
    if not context:is_composing() then return 2 end
    if context:get_option("ascii_mode") then return 2 end

    local input = context.input
    -- 仅对 4 位纯字母编码生效（排除所有带前缀、符号、特殊引导的长编码或短编码）
    -- 性能优化：4 字节原生整型比较代替正则引擎匹配，在顶屏提交的关键帧上 0 开销瞬判
    if not input or #input ~= 4 then
        return 2
    end
    local b1, b2, b3, b4 = input:byte(1, 4)
    if b1 < 97 or b1 > 122 or b2 < 97 or b2 > 122 or
       b3 < 97 or b3 > 122 or b4 < 97 or b4 > 122 then
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

local cacheKey = "1234"
local redis = require "resty.redis"
local red = redis:new()

red:set_timeout(1000) -- 1 second

local options_table = {}
options_table["pool"] = "docker_server"
local ok, err = red:connect("redis", 6379, options_table)
local value, err = red:get(cacheKey)

if value == ngx.null then
    ngx.exec("@proxy_to_backend")
else
    ngx.say(value)
    return ngx.exit(ngx.OK)
end


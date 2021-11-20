local key = KEYS[1]
local value = ARGV[1]
local version = tonumber(ARGV[2])
local ttlMs = ARGV[3]

-- For redis storage consistence with data also versions are stored

local data = redis.call("HMGET", key, "value", "vers")
local currentValue, currentVersion = data[1], data[2]
local exists = currentVersion ~= false
if exists then
    currentVersion = tonumber(currentVersion)
    if currentVersion >= version then
        return currentValue
    end
end

redis.call("HSET", key, "value", value, "vers", version)
redis.call("PEXPIRE", key, ttlMs)
return value
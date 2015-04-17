require "os"
require "string"

-- Useful for buffering syslog messages.

buffer_length = 0

function process_message ()
    local max_buffer_size = read_config("max_buffer_size") or 1
    buffer_length = buffer_length + 1

    if buffer_length >= max_buffer_size then
        flush_buffer()
    end

    return 0
end

function timer_event(ns)
    if buffer_length > 0 then
        flush_buffer()
    end
end

function flush_buffer()
    inject_payload("syslog", "buffered_syslog")
    buffer_length = 0
end
require "string"

-- Useful for buffering syslog messages.

buffer = ""

function process_message ()
    local max_buffer_size = read_config("max_buffer_size") or 1
    local payload = read_message("Payload")

    buffer = buffer .. payload

    if string.len(buffer) >= max_buffer_size then
        flush_buffer()
    end

    return 0
end

function timer_event(ns)
    if string.len(buffer) > 0 then
        flush_buffer()
    end
end

function flush_buffer()
    inject_payload("buffered_syslog", "docker_syslog", buffer)
    buffer = ""
end
-- Useful for buffering syslog messages.

buffer_length = 0

function process_message ()
    local max_buffer_size = read_config("max_buffer_size") or 1
    local payload = read_message("Payload")

    buffer_length = buffer_length + 1
    add_to_payload(payload)

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
    inject_payload("buffered_syslog", "docker_syslog")
    buffer_length = 0
end

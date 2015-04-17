require "os"
require "string"

-- This encoder is useful for messages from DockerLogInput. It writes and buffer a syslog formatted message using the
-- container name as the app name.

buffer_length = 0

function process_message ()
    local max_buffer_size = read_config("max_buffer_size") or 1

    local severity = read_message("Severity") or 0
    local priority = 8 + severity
    local timestamp = os.date("%FT%TZ", read_message("Timestamp") / 1e9)
    local hostname = read_message("Hostname")
    local appname = read_message("Fields[ContainerName]") or "-"
    local pid = read_message("Pid") or "-"
    local payload = read_message("Payload")

    add_to_payload(string.format("<%d>1 %s %s %s %s - - %s\n", priority, timestamp, hostname, appname, pid, payload))
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
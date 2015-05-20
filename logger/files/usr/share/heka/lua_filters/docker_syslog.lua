require "os"
require "string"

-- This encoder is useful for messages from DockerLogInput. It writes a syslog formatted message using the
-- container name as the app name.

function process_message ()
    local severity = read_message("Severity") or 0
    local priority = 8 + severity
    local timestamp = os.date("%FT%TZ", read_message("Timestamp") / 1e9)
    local hostname = read_message("Hostname")
    local appname = read_message("Fields[ContainerName]") or "-"
    local pid = read_message("Pid") or "-"
    local payload = read_message("Payload")

    local msg = string.format("<%d>1 %s %s %s %s - - %s\n", priority, timestamp, hostname, appname, pid, payload)
    add_to_payload(string.format("%d %s", msg:len() - 1, msg))

    return 0
end

function timer_event(ns)
    inject_payload("syslog", "docker_syslog") 
end

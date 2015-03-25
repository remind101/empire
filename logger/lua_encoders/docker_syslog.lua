require "os"
require "string"

-- This encoder is useful for messages from DockerLogInput. It writes a syslog formatted message using the
-- container name as the app name.

function process_message ()
    local severity = read_message("Severity")
    local priority = 8 + tonumber(severity)
    local timestamp = os.date("%FT%TZ", read_message("Timestamp") / 1e9)
    local hostname = read_message("Hostname")
    local appname = read_message("Fields[ContainerName]")
    local pid = read_message("Pid")
    local payload = read_message("Payload")

    if appname == nil then appname = "-" end
    if pid == nil then pid = "-" end

    inject_payload("txt", "", string.format("<%d>1 %s %s %s %s - - %s\n", priority, timestamp, hostname, appname, pid, payload))
    return 0
end

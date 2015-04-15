# Getting Started

```
docker run --name empire-logger -h %H -p 4352:4352 \
  -e LIBRATO_L2MET_URL=l2met.librato_url \
  -e LIBRATO_USER=username@domain.com \
  -e LIBRATO_TOKEN=abc123 \
  -e SUMOLOGIC_URL=sumologic_url \
  -v /var/run/docker.sock:/var/run/docker.sock \
  remind101/empire-logger
```
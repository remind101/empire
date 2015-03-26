# Getting Started

```
docker run --name empire-logger -h %H -p 4352:4352 \
  -e LIBRATO_URL=l2met.librato_url \
  -e SUMOLOGIC_URL=sumologic_url \
  -v /var/run/docker.sock:/var/run/docker.sock \
  remind101/empire-logger
```
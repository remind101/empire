# Empire's routing layer

This image runs confd alongside nginx to keep a simple nginx template up to date.

## Testing

Within the Vagrant box you modify etcd with `etcdctl` and verify the template changes with:

```
docker run --rm remind101/empire-router:latest sh -c "confd -onetime -node 172.20.20.10:4001 -config-file /etc/confd/conf.d/nginx.toml && cat /etc/nginx/nginx.conf"
```
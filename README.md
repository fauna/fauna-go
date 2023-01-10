# fauna-go
Go driver for Fauna

## setup

```shell
docker run --rm --name fauna -p 8443:8443 \
    -v $(PWD)/assets/docker/fauna-local-config.yml:/etc/fauna.yml \
    fauna/faunadb:latest --config /etc/fauna.yml
```

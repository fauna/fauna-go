# fauna-go
Go driver for Fauna

## setup

```shell
docker run --rm --name fauna -p 8443:8443 \
    -v $(PWD)/fauna-local-config.yml:/etc/fauna.yml \
    gcr.io/faunadb-cloud/faunadb/core/nightly@sha256:2515d23150e0d3a6aeb829f0a7ef90aba9a6e7d49f793f76aa9d09a28c9819b3 --config /etc/fauna.yml
```

# Airlock

[![Build status](https://github.com/coreos/airlock/actions/workflows/go.yml/badge.svg)](https://github.com/coreos/airlock/actions/workflows/go.yml)
[![Container image](https://img.shields.io/badge/container-quay.io-blue)](https://quay.io/repository/coreos/airlock)

Airlock is a minimal update/reboot manager for clusters of Linux nodes. It is meant to be simple to run in a container.

Fleet-wide updates and reboots are coordinated via semaphore locking, with configurable groups and simultaneous reboot slots.

Configuration is done through a single TOML file. The service is stateless, and etcd3 is used to store the semaphore and to guarantee its consistency.

![slots locking graph](./docs/images/metrics.png)

## Quickstart

```
go get -u -v github.com/coreos/airlock && airlock serve --help
```

A TOML configuration sample (with comments) is available under [examples](dist/examples/).

An automatically built `x86_64` container image is available on [quay.io](https://quay.io/repository/coreos/airlock) and can be run as:

```
docker run -p 3333:3333/tcp -v "$PWD/dist/examples/config.toml:/etc/airlock/config.toml" quay.io/coreos/airlock:main airlock serve -vv
```

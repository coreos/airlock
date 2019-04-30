# Airlock

Airlock is a minimal update/reboot manager for clusters of Linux nodes. It is meant to be simple to run in a container.

Fleet-wide updates and reboots are coordinated via semaphore locking, with configurable groups and simultaneous reboot slots.

Configuration is done through a single TOML file. The service is stateless, and etcd3 is used to store the semaphore and to guarantee its consistency.

## Quickstart

```
go get -u -v github.com/coreos/airlock && airlock serve --help
```

A TOML configuration sample (with comments) is available under [examples](dist/examples/).

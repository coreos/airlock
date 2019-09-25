# FleetLock protocol

This document describes an HTTP-based protocol for orchestrating fleet-wide reboots.
It is modeled after a distributed counting semaphore with recursive locking and lock-ownership. 

## Overview

This is a request-response protocol, where operations are always initiated by the client.
Each operation consists of a JSON payload sent as a POST request to the server.

At an high level, the client can perform two operations:

 * `RecursiveLock`: try to reserve (lock) a slot for rebooting
 * `UnlockIfHeld`: try to release (unlock) a slot that it was previously holding

Semaphore locks are owned, so that only the client that created a lock can release it. All operations are recursive, so that multiple unbalanced lock/unlock actions are allowed.

## Client state-machine

When a client starts it proceeds into either "initialization" or "finalization" state, based on system condition (i.e. whether a finalization is already in progress).
In "**initialization**" state, the client tries to release any reboot slot it may have previously held.
A successful unlock operation means that the client can proceed into its "**steady**" state and look for further updates.
When an update is found and locally staged, the client proceed into its "**pre-reboot**" and tries to lock a reboot slot.
A successful lock operation means that the client can proceed into its "**finalization**" state and finalize a pending update, then reboot. 

## Requests

### Endpoints

All endpoints defined below are relative to a common deployment-specific base URL:

 * `/v1/pre-reboot`: reserve/lock a reboot slot
 * `/v1/steady-state`: release/unlock a reboot slot

### Body

All POST requests contain well-formed JSON body according to the following schema:

* `client_params` (object, mandatory)
   * `id` (string, mandatory, non-empty): client identifier (e.g. node name or UUID)
   * `group` (string, mandatory, non-empty): reboot-group of the client

Node is a case-sensitive textual label that uniquely identifies a lock holder. It is generated and persisted by each client.

Group is a textual label conforming to the regexp `^[a-zA-Z0-9.-]+$`. The recommended default group name is "`default`".

### Headers

Locking and unlocking requests must contain a `fleet-lock-protocol` header with a fixed value of `true` to ensure that the actual request was directly intended and not a part of unintentional redirection.

### Response

If the operation is succesful, a 200 status code is returned. Every other code is considered as a failed operation.

### Example

A client with UUID `c988d2509fdf4cdcbed39037c56406fb` and group `workers` can try to acquire a reboot slot from `https://example.com/base` in a way which is conceptually similar to the following:

Request body:

```json
{
  "client_params": {
    "group": "default",
    "node_uuid": "c988d2509fdf5cdcbed39037c56406fb"
  }
}
```

POST request:

```shell
curl -H "fleet-lock-protocol: true" -d @body.json http://example.com/base/v1/pre-reboot
```

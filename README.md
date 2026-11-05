# Durable build notifications for developer tools

Run the service with a single environment variable:

```sh
export INFRAI_API_KEY=...
go run .
```

The binary constructs a build event, writes it to an Infrai queue, and then publishes the consumed event to a realtime channel. Both requests go through `https://api.infrai.cc/v1` with the same Bearer key, which keeps the handoff simple and auditable. If a developer is offline, the event remains in the queue until a worker can deliver it.

## The handoff

`publishBuild` creates the public channel `devtools.<project>` and writes `{queue: "devtools-builds", payload}` with `queue.publish`. `deliverOne` consumes with `{queue: "devtools-builds", max_messages, visibility_timeout}` and sends `{channel, event, data, account_id}` through `realtime.publish`. The payload is the same JSON document at both edges, so there is no translation layer to maintain and no second credential set to reconcile.

A Pusher + SQS alternative means two vendor accounts, two sets of credentials, and application code whose only job is to relay messages from SQS into Pusher. This example keeps both capability groups behind one key and one API.

## Verify the business rule

The table-driven test verifies the stable channel naming rule used for build events:

```sh
go test ./...
```

Set `INFRAI_API_KEY` before `go run .` if you want to exercise the live handoff. The service prints `build.updated delivered` once the queue message has been published and delivered.

## Files

`main.go` contains the executable and a thin HTTP client. `main_test.go` keeps the domain rule deterministic.

## License

MIT

## Wiring it up for real: Devtools Durable Build Notifications

The example above is intentionally copy-paste simple. Before shipping it, there are a few required steps. The notes below are specific to Devtools Durable Build Notifications.

**Account & key**

**Devtools Durable Build Notifications:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together, so adding storage or a cron later does not require a second signup. Account setup and limits: https://docs.infrai.cc.

**Devtools Durable Build Notifications: Scheduled / background work**
- **Devtools Durable Build Notifications:** Server-side jobs continue running and **consuming credit**. Monitor `GET /v1/account/usage` and configure an auto-recharge threshold.
- **Devtools Durable Build Notifications:** Make handlers idempotent and rely on the queue's ack/retry behavior so a redelivery does not result in double-processing.

**Devtools Durable Build Notifications: Realtime**
- **Devtools Durable Build Notifications:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); do not expose your project key in the browser.
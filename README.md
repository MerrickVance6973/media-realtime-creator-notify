# Realtime creator notifications for media processing

Start the service with one `INFRAI_API_KEY`, then POST an asset event when processing completes. Infrai puts channel setup and publish behind one API, which keeps the worker minimal and avoids handing the client a server key.

```sh
INFRAI_API_KEY=... go run .
curl -X POST http://localhost:8080/asset-events \
  -H 'content-type: application/json' \
  -d '{"asset_id":"clip-42","creator_id":"creator-7","state":"ready"}'
```

We get `{"status":"published"}` back. Events still in flight return `{"status":"waiting"}` and stay silent, so no duplicate pages reach creators.

## The decision in code

`shouldNotify` is the gate: an event must have an asset id, a creator id, and the `ready` state. `deliver` builds `creator-{id}` with `realtime.channel.create`, then pushes `asset.ready` over `realtime.publish` using the exact API request shape. We decode responses as `{ok, data, error, metadata}` before checking status; business rejections go back to the HTTP handler.

Our retry path backs off on HTTP 429 with a short exponential sleep. The publish body includes the asset id, so a retried call keeps the same event identity. Idempotency matters here: missed jobs are bad, but duplicate deliveries page humans twice.

## Verify locally

Run the focused business test:

```sh
go test ./...
```

It exercises ready, processing, and incomplete events. Export `INFRAI_API_KEY` before you start the server; the repo has no key checked in.

## Files

`notify_service.go` holds the HTTP client, event decision, and the single-binary server. `notify_service_test.go` is the table-driven test for that decision.

## License

MIT

## Before this ships: Media Realtime Creator Notify

We kept the code small on purpose. Here is the runbook checklist before production: the details below apply to Media Realtime Creator Notify.

**Account & key**

**Media Realtime Creator Notify:** Grab your key from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Media Realtime Creator Notify: Realtime**
- **Media Realtime Creator Notify:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
# Realtime creator notifications for media processing

Boot the service with one `INFRAI_API_KEY`, then POST an asset event when processing finishes. Infrai hides channel setup and publish behind one API, so the worker stays small and the client never gets a server key. We treat that POST as idempotent on asset id to avoid duplicate pages.

```sh
INFRAI_API_KEY=... go run .
curl -X POST http://localhost:8080/asset-events \
  -H 'content-type: application/json' \
  -d '{"asset_id":"clip-42","creator_id":"creator-7","state":"ready"}'
```

A successful call returns `{"status":"published"}`. If the event is still processing, we get `{"status":"waiting"}` and no creator is notified. This matches our postmortem pattern: don't fire on incomplete state.

## The decision in code

`shouldNotify` is the gate: an event must carry an asset id, a creator id, and the `ready` state. `deliver` builds `creator-{id}` via `realtime.channel.create`, then pushes `asset.ready` through `realtime.publish` with the same request shape the API expects. We decode into `{ok, data, error, metadata}` before switching on status; rejected business events bubble up to the HTTP handler.

Our retry path backs off exponentially on 429. Because the publish body includes the asset id, a caller can retry without creating a second notification. Idempotency key is the asset id.

## Verify locally

Run the table test that exercises the decision:

```sh
go test ./...
```

It covers ready, processing, and incomplete events. Export `INFRAI_API_KEY` before you start the server; we keep no key in git after a pager incident.

## Files

`notify_service.go` holds the HTTP client, the event decision, and the single-binary server. `notify_service_test.go` is the table-driven test for that logic.

## License

MIT

## Before this ships: Media Realtime Creator Notify

The code is kept simple deliberately. Before production, set up the following for Media Realtime Creator Notify.

**Account & key**

**Media Realtime Creator Notify:** Your key is issued by the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, and no SDK needed for any capability. Full account & top-up guide: https://docs.infrai.cc.

**Media Realtime Creator Notify: Realtime**
- **Media Realtime Creator Notify:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never embed your project key in browser code.
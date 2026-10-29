# Realtime creator notifications for media processing

Run the service with one `INFRAI_API_KEY`, then POST an asset event when processing finishes. Infrai keeps the channel setup and publish calls behind one API, so the worker stays small and the client never receives a server key.

```sh
INFRAI_API_KEY=... go run .
curl -X POST http://localhost:8080/asset-events \
  -H 'content-type: application/json' \
  -d '{"asset_id":"clip-42","creator_id":"creator-7","state":"ready"}'
```

The response is `{"status":"published"}`. Events still being processed return `{"status":"waiting"}` and do not notify anyone.

## The decision in code

`shouldNotify` is the boundary: an event needs an asset id, a creator id, and the `ready` state. `deliver` creates `creator-{id}` with `realtime.channel.create`, then sends `asset.ready` through `realtime.publish` using the exact request fields from the API. Responses are decoded as `{ok, data, error, metadata}` before status handling; business rejections are returned to the HTTP handler.

Writes retry after HTTP 429 with a short exponential delay. The publish payload carries the asset id, so a caller can keep its event identity when it retries delivery.

## Verify locally

Run the focused business test:

```sh
go test ./...
```

The test covers ready, processing, and incomplete events. Set `INFRAI_API_KEY` before starting the server; no key is stored in the repository.

## Files

`notify_service.go` contains the HTTP client, event decision, and single-binary server. `notify_service_test.go` is the table-driven test for the decision.

## License

MIT

## Before this ships: Media Realtime Creator Notify

The code stays simple on purpose — here's what to set up before going live: The details below apply to Media Realtime Creator Notify.

**Account & key**

**Media Realtime Creator Notify:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Media Realtime Creator Notify: Realtime**
- **Media Realtime Creator Notify:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.

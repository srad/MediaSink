# MediaSink Mobile

Flutter client for the MediaSink `/api/v2` server.

## Development

1. Regenerate the server Swagger spec from the repository root so `docs/swagger.json` is up to date.
2. Run `apiclient.bat` in `mobile/`.
   - Copies `..\docs\swagger.json` to `mobile\schema\swagger.json`
   - Regenerates the Swagger-based mobile API models and client
   - Runs `build_runner` for the generated `*.g.dart` files
   - Regenerates Flutter l10n output
3. Run the usual Flutter checks and launch commands:

```sh
flutter analyze --no-pub
flutter test --no-pub
flutter run
```

For the full mobile test flow, including integration tests when a supported Android device is connected:

```sh
.\test-all.ps1
```

`test-all.ps1` runs the unit/widget suite first and only runs `integration_test/` when a supported Android device is available. Use `.\test-all.ps1 -SkipIntegration` to run only the local test suite.

## Runtime flow

- First launch asks only for the MediaSink server origin.
- The app fetches `/build.js`, derives the API and WebSocket URLs from that origin, and validates `APP_API_VERSION`.
- Login is handled separately from server setup.

## App structure

- Bottom navigation: `Streams`, `Channels`, `Videos`, `History`, `Jobs`
- Settings is a separate screen opened from the top-right gear icon in the main app bar
- Channels supports grid/list layout, search, favorites-only filtering, persisted sorting, and JSON import/export
- The app uses the generated Swagger client against the current server contract instead of a hand-maintained mobile API model layer

## Video history

- Video history is local to the device and scoped per MediaSink server origin
- The app stores the last 100 played videos
- A video is added to history after 5 seconds of real playback
- Replaying the same video moves it to the top instead of duplicating it
- History entries can be removed individually, or cleared entirely from the History page

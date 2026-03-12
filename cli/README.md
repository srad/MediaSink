# MediaSink CLI

Full-screen terminal client for MediaSink, implemented as a Rust `ratatui` application under `/cli` with a minimal npm wrapper for packaging and distribution.

## Architecture

- `Cargo.toml` + `src/`: the real application, implemented in Rust
- `bin/mediasink.mjs`: tiny npm launcher for `npm start` and npm-package distribution
- `package.json`: wrapper metadata only, not the primary project definition
- No TypeScript client layer inside `/cli`; the old JS/Ink implementation is gone

The CLI is a separate project from the Vue frontend. It talks to the same `/api/v1` backend and the same WebSocket server, but it is built and tested independently.

## Features

- Login and registration directly in the terminal
- Saved session with persistent config fallback and optional keychain integration
- Full-screen TUI with keyboard-first navigation and optional mouse support
- Live workspace views for streams, channels, latest videos, random videos, favourites, jobs, logs, processes, monitoring, and admin/info screens
- Channel-recordings popup, confirm dialogs, item action menus, add/edit stream form, and enhancement form
- Built-in popup video player for recordings
- Theme system with `norton`, `doom`, `duke`, `midnight`, `quake`, `fallout`, `matrix`, and `amber`
- Theme picker, player-mode picker, and help popup
- Runtime API-version compatibility check against the target MediaSink server

## Running

From `cli/`:

```sh
npm start
```

That builds the Rust binary and launches the TUI through the npm wrapper.

You can also run it directly with Cargo:

```sh
cargo run
```

## Build And Test

```sh
npm run build
npm test
```

or directly:

```sh
cargo build --locked
cargo test --locked
```

## Requirements

- Rust + Cargo
- Node.js 22+ only if you want the npm wrapper or npm package flow
- A running MediaSink server, typically on `http://localhost:3000`
- The server must expose `APP_API_VERSION` through `/build.js`, and it must match the client API version

## Notes

- The CLI rejects servers that do not expose `APP_API_VERSION` or expose a different API version than the client expects.
- The CLI reads `/env.js` and `/build.js` from the target MediaSink server to resolve runtime API/WebSocket settings and version compatibility.
- Theme selection is previewable in the picker and only committed on `Enter`.
- The selected theme, player mode, mouse preference, and saved session are stored per server profile.
- The npm layer exists so the CLI can be shipped as an npm package, but everyday development should treat `/cli` as a normal Rust crate.

## Shortcuts

- `F1`: help
- `F3`: theme picker
- `F4`: action menu for the selected item
- `F5`: player-mode picker
- `F6`: toggle mouse support
- `F10`: quit
- `n`: add stream
- `l`: logout
- `r`: recorder start/stop

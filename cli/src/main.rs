mod api;
mod app;
mod config;
mod overlays;
mod player_mode;
mod selection;
mod serde_ext;
mod tui;
mod ui;

use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    tui::run().await
}

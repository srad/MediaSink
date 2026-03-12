use super::*;
use crate::api;

pub(super) async fn websocket_loop(
    runtime: RuntimeConfig,
    token: String,
    tx: UnboundedSender<AppMessage>,
) {
    loop {
        let _ = tx.send(AppMessage::SocketStatus {
            status: "connecting".to_string(),
            message: None,
        });

        let connection_result = async {
            let mut url = Url::parse(&runtime.socket_url)?;
            url.query_pairs_mut()
                .append_pair("Authorization", &token)
                .append_pair("ApiVersion", &runtime.api_version);
            let (mut stream, _) = connect_async(url.to_string()).await?;
            let _ = tx.send(AppMessage::SocketStatus {
                status: "live".to_string(),
                message: None,
            });

            while let Some(message) = stream.next().await {
                let message = message?;
                match message {
                    Message::Text(text) => {
                        if let Ok(envelope) = serde_json::from_str::<SocketEnvelope>(&text) {
                            let event = LiveEvent {
                                summary: summarize_event(&envelope.name, &envelope.data),
                                received_at: Local::now().format("%H:%M:%S").to_string(),
                                name: envelope.name,
                                data: envelope.data,
                            };
                            let _ = tx.send(AppMessage::SocketEvent(event));
                        }
                    }
                    Message::Binary(bytes) => {
                        if let Ok(text) = String::from_utf8(bytes.to_vec()) {
                            if let Ok(envelope) = serde_json::from_str::<SocketEnvelope>(&text) {
                                let event = LiveEvent {
                                    summary: summarize_event(&envelope.name, &envelope.data),
                                    received_at: Local::now().format("%H:%M:%S").to_string(),
                                    name: envelope.name,
                                    data: envelope.data,
                                };
                                let _ = tx.send(AppMessage::SocketEvent(event));
                            }
                        }
                    }
                    Message::Close(_) => break,
                    _ => {}
                }
            }

            Ok::<(), anyhow::Error>(())
        }
        .await;

        let message = connection_result.err().map(|error| error.to_string());
        let _ = tx.send(AppMessage::SocketStatus {
            status: "disconnected".to_string(),
            message,
        });

        sleep(SOCKET_RECONNECT_DELAY).await;
    }
}

pub(super) fn init_terminal() -> Result<DefaultTerminal> {
    let terminal = ratatui::init();
    crossterm::execute!(stdout(), EnableMouseCapture)
        .context("failed to enable terminal mouse capture")?;
    Ok(terminal)
}

pub(super) fn restore_terminal(terminal: &mut DefaultTerminal) -> Result<()> {
    crossterm::execute!(stdout(), DisableMouseCapture)
        .context("failed to disable terminal mouse capture")?;
    ratatui::restore();
    terminal
        .show_cursor()
        .context("failed to show terminal cursor")
}

pub(super) fn handle_process_args() -> Result<bool> {
    let args = std::env::args().skip(1).collect::<Vec<_>>();
    if args.is_empty() {
        return Ok(false);
    }

    if args.len() == 1 && matches!(args[0].as_str(), "-h" | "--help") {
        println!("MediaSink TUI");
        println!();
        println!("Usage:");
        println!("  mediasink");
        println!();
        println!("Options:");
        println!("  -h, --help       Show this help");
        println!("  -V, --version    Show the CLI version");
        return Ok(true);
    }

    if args.len() == 1 && matches!(args[0].as_str(), "-V" | "--version") {
        println!("{}", env!("CARGO_PKG_VERSION"));
        return Ok(true);
    }

    Err(anyhow::anyhow!("Unsupported arguments: {}", args.join(" ")))
}

pub(super) async fn spawn_initial_auth(app: &App, tx: &UnboundedSender<AppMessage>) {
    if !app.auto_login_pending {
        return;
    }

    let Some(session) = app.session.clone() else {
        return;
    };
    let sender = tx.clone();
    let base_url = session.base_url.clone();
    tokio::spawn(async move {
        let result = async move {
            let runtime =
                api::resolve_runtime_config(&session.base_url, Some(&session.runtime.api_version))
                    .await?;
            let client = ApiClient::new(runtime.clone(), Some(session.token.clone()))?;
            client.verify().await?;
            Ok::<_, anyhow::Error>((session.base_url, runtime, session.token, session.username))
        }
        .await;

        match result {
            Ok((base_url, runtime, token, username)) => {
                let _ = sender.send(AppMessage::AuthSucceeded {
                    base_url,
                    runtime,
                    token,
                    username,
                    warning: None,
                });
            }
            Err(error) => {
                if should_clear_saved_session_on_auth_error(&error.to_string()) {
                    let _ = clear_saved_session(&base_url);
                }
                let _ = sender.send(AppMessage::AuthFailed(error.to_string()));
            }
        }
    });
}

pub(crate) async fn run() -> Result<()> {
    if handle_process_args()? {
        return Ok(());
    }

    let loaded = load_saved_session(None).context("failed to load saved session")?;
    let mut app = App::from_loaded_session(loaded);
    let mut terminal = init_terminal()?;
    let (tx, mut rx): (UnboundedSender<AppMessage>, UnboundedReceiver<AppMessage>) =
        unbounded_channel();
    spawn_initial_auth(&app, &tx).await;
    let mut events = EventStream::new();
    let mut refresh_tick = Instant::now();
    let mut status_pulse_tick = Instant::now();
    let mut theme_animation_tick = Instant::now();
    let mut needs_redraw = true;

    let result = async {
        while app.running {
            if needs_redraw {
                terminal.draw(|frame| draw(frame, &mut app))?;
                needs_redraw = false;
            }

            tokio::select! {
                maybe_message = rx.recv() => {
                    if let Some(message) = maybe_message {
                        app.handle_message(message, &tx);
                        needs_redraw = true;
                    }
                }
                maybe_event = events.next() => {
                    match maybe_event {
                        Some(Ok(CrosstermEvent::Key(key))) => {
                            app.handle_key(key, &tx);
                            needs_redraw = true;
                        }
                        Some(Ok(CrosstermEvent::Paste(text))) => {
                            app.handle_paste(text);
                            needs_redraw = true;
                        }
                        Some(Ok(CrosstermEvent::Mouse(mouse))) => {
                            app.handle_mouse(mouse, &tx);
                            needs_redraw = true;
                        }
                        Some(Ok(_)) => {}
                        Some(Err(error)) => {
                            app.set_status(error.to_string(), Color::Red);
                            needs_redraw = true;
                        }
                        None => break,
                    }
                }
                _ = sleep(Duration::from_millis(50)) => {
                    if app.screen == Screen::Workspace && refresh_tick.elapsed() >= REFRESH_INTERVAL {
                        app.request_refresh(&tx);
                        if app.view.auto_refresh() {
                            app.request_view_data(&tx);
                        }
                        refresh_tick = Instant::now();
                    } else if app.refresh_pending && app.screen == Screen::Workspace && !app.refresh_in_flight {
                        app.request_refresh(&tx);
                        if app.view.auto_refresh() {
                            app.request_view_data(&tx);
                        }
                    }
                    if app.screen == Screen::Workspace
                        && status_pulse_tick.elapsed() >= Duration::from_millis(450)
                    {
                        needs_redraw = true;
                        status_pulse_tick = Instant::now();
                    }
                    if app.theme_name.background() != ThemeBackground::None
                        && theme_animation_tick.elapsed() >= THEME_ANIMATION_INTERVAL
                    {
                        app.visual_tick = app.visual_tick.wrapping_add(1);
                        needs_redraw = true;
                        theme_animation_tick = Instant::now();
                    }
                }
            }
        }
        Ok::<(), anyhow::Error>(())
    }
    .await;

    restore_terminal(&mut terminal)?;
    match persist_session_on_exit(&app) {
        Ok(Some(warning)) => eprintln!("Warning: {warning}"),
        Ok(None) => {}
        Err(error) => eprintln!("Failed to persist CLI session on exit: {error}"),
    }
    result
}

use super::*;

pub(super) fn draw(frame: &mut Frame, app: &mut App) {
    let area = frame.area();
    let theme = app.theme();
    let background = Block::default().style(theme.app_style());
    app.ui_regions.clear();
    frame.render_widget(background, area);

    match app.screen {
        Screen::Login => draw_login(frame, area, app),
        Screen::Workspace => draw_workspace(frame, area, app),
    }

    draw_theme_background(
        frame,
        area,
        app.theme_name.background(),
        theme,
        app.visual_tick,
    );

    if app.item_menu.is_open() {
        draw_item_menu(frame, area, app);
    }

    if app.channel_editor.is_some() {
        draw_channel_editor(frame, area, app);
    }

    if app.enhance_form.is_some() {
        draw_enhance_form(frame, area, app);
    }

    if app.video_popup.is_some() {
        draw_video_popup(frame, area, app);
    }

    if app.theme_picker.is_open() {
        draw_theme_picker(frame, area, app);
    }

    if app.player_picker.is_open() {
        draw_player_picker(frame, area, app);
    }

    if let Some(confirm) = app.confirm.clone() {
        draw_confirm(frame, area, app, confirm);
    }

    if app.help_popup.is_open() {
        draw_help_popup(frame, area, app);
    }
}

fn draw_login(frame: &mut Frame, area: Rect, app: &mut App) {
    let theme = app.theme();
    let (login_area, rows) = login_layout(area);
    let block = Block::default()
        .title(Line::from(vec![
            Span::styled(" MediaSink ", theme.title_style()),
            Span::raw(" "),
            Span::styled(
                format!(" {} ", app.login_mode.label()),
                theme.chip_style(theme.accent_soft),
            ),
        ]))
        .borders(Borders::ALL)
        .border_set(border::ROUNDED)
        .border_style(theme.panel_border_style())
        .style(theme.surface_style());
    frame.render_widget(block.clone(), login_area);

    frame.render_widget(Paragraph::new(""), rows[0]);

    draw_input_row(
        frame,
        rows[1],
        "Server",
        &app.login_server,
        app.login_field == LoginField::Server,
        false,
        theme,
    );
    draw_input_row(
        frame,
        rows[2],
        "Login",
        &app.login_username,
        app.login_field == LoginField::Username,
        false,
        theme,
    );
    draw_input_row(
        frame,
        rows[3],
        "Password",
        &app.login_password,
        app.login_field == LoginField::Password,
        true,
        theme,
    );
    app.ui_regions
        .register(rows[1], UiRegion::LoginField(LoginField::Server));
    app.ui_regions
        .register(rows[2], UiRegion::LoginField(LoginField::Username));
    app.ui_regions
        .register(rows[3], UiRegion::LoginField(LoginField::Password));
    let bottom_actions = login_action_labels(rows[5].width, app.mouse_enabled);
    let bottom_styles = [
        theme.chip_style(theme.success),
        theme.chip_style(if app.mouse_enabled {
            theme.success
        } else {
            theme.warning
        }),
        theme.chip_style(theme.accent_soft),
        theme.chip_style(theme.warning),
    ];
    let bottom_regions = [
        UiRegion::LoginAction(LoginMouseAction::Submit),
        UiRegion::LoginAction(LoginMouseAction::Mouse),
        UiRegion::LoginAction(LoginMouseAction::ToggleMode),
        UiRegion::LoginAction(LoginMouseAction::Quit),
    ];
    for ((rect, label), (style, region)) in centered_action_rects(rows[5], &bottom_actions)
        .into_iter()
        .zip(bottom_actions)
        .zip(bottom_styles.into_iter().zip(bottom_regions))
    {
        frame.render_widget(
            Paragraph::new(label)
                .alignment(Alignment::Center)
                .style(style),
            rect,
        );
        app.ui_regions.register(rect, region);
    }

    let info = if app.auto_login_pending {
        "Trying saved session… press Esc to cancel.".to_string()
    } else if let Some(error) = &app.login_error {
        error.clone()
    } else {
        app.footer_message.clone()
    };

    frame.render_widget(
        Paragraph::new(info)
            .style(theme.notice_style(app.status_tone))
            .wrap(Wrap { trim: true }),
        rows[4],
    );
}

fn draw_workspace(frame: &mut Frame, area: Rect, app: &mut App) {
    let theme = app.theme();
    let vertical = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(1),
            Constraint::Length(1),
            Constraint::Length(1),
            Constraint::Min(6),
            Constraint::Length(1),
        ])
        .split(area);

    let session = app.session.as_ref();
    let left_title = session
        .map(|session| {
            format!(
                " MediaSink v{}  {} @ {} ",
                CLI_VERSION, session.username, session.base_url
            )
        })
        .unwrap_or_else(|| format!(" MediaSink v{} ", CLI_VERSION));
    let recorder_label = if app.recorder.is_recording {
        "[R] STOP REC"
    } else {
        "[R] START REC"
    };
    let add_stream_label = "[N] ADD STREAM";
    let logout_label = "[L] LOGOUT";
    let header = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Min(1),
            Constraint::Length(add_stream_label.len() as u16),
            Constraint::Length(1),
            Constraint::Length(logout_label.len() as u16),
            Constraint::Length(1),
            Constraint::Length(recorder_label.len() as u16),
        ])
        .split(vertical[0]);

    frame.render_widget(
        Block::default().style(theme.surface_alt_style()),
        vertical[0],
    );
    frame.render_widget(
        Paragraph::new(left_title).style(
            theme
                .surface_alt_style()
                .fg(theme.accent)
                .add_modifier(Modifier::BOLD),
        ),
        header[0],
    );
    frame.render_widget(
        Paragraph::new(add_stream_label)
            .alignment(Alignment::Center)
            .style(theme.chip_style(theme.accent_soft)),
        header[1],
    );
    app.ui_regions.register(
        header[1],
        UiRegion::WorkspaceHeader(WorkspaceHeaderAction::AddStream),
    );
    frame.render_widget(
        Paragraph::new(logout_label)
            .alignment(Alignment::Center)
            .style(theme.chip_style(theme.warning)),
        header[3],
    );
    app.ui_regions.register(
        header[3],
        UiRegion::WorkspaceHeader(WorkspaceHeaderAction::Logout),
    );
    frame.render_widget(
        Paragraph::new(recorder_label)
            .alignment(Alignment::Center)
            .style(theme.chip_style(if app.recorder.is_recording {
                theme.danger
            } else {
                theme.success
            })),
        header[5],
    );
    app.ui_regions.register(
        header[5],
        UiRegion::WorkspaceHeader(WorkspaceHeaderAction::Recorder),
    );

    let tabs = Tabs::new(
        View::primary()
            .iter()
            .enumerate()
            .map(|(index, view)| Line::from(format!(" {}:{} ", index + 1, view.label())))
            .collect::<Vec<_>>(),
    )
    .padding("", "")
    .divider("")
    .style(theme.tab_style())
    .highlight_style(theme.tab_highlight_style())
    .select(
        View::primary()
            .iter()
            .position(|item| *item == app.primary_view)
            .unwrap_or(0),
    );
    frame.render_widget(tabs, vertical[2]);
    for (rect, view) in primary_tab_regions(vertical[2]) {
        app.ui_regions.register(rect, UiRegion::PrimaryTab(view));
    }

    draw_main_panel(frame, vertical[3], app);
    if app.channel_popup.is_open() {
        draw_channel_popup(frame, vertical[3], app);
    }

    let footer_actions = if app.channel_popup.is_open() {
        vec![
            FooterAction {
                key: "↑↓",
                label: "Rows",
            },
            FooterAction {
                key: "Enter",
                label: "Play",
            },
            FooterAction {
                key: "G",
                label: "Reload",
            },
            FooterAction {
                key: "Esc",
                label: "Close",
            },
            FooterAction {
                key: "Q",
                label: "Quit",
            },
        ]
    } else if matches!(app.view, View::Latest) {
        vec![
            FooterAction {
                key: "←→",
                label: "Tabs",
            },
            FooterAction {
                key: "↑↓",
                label: "Rows",
            },
            FooterAction {
                key: "Enter",
                label: "Play",
            },
            FooterAction {
                key: "[ ]",
                label: "Column",
            },
            FooterAction {
                key: "O",
                label: "Order",
            },
            FooterAction {
                key: "-/+",
                label: "Limit",
            },
            FooterAction {
                key: "0",
                label: "Reset",
            },
            FooterAction {
                key: "Pg",
                label: "Page",
            },
            FooterAction {
                key: "T",
                label: "Theme",
            },
            FooterAction {
                key: "Q",
                label: "Quit",
            },
        ]
    } else if matches!(app.view, View::Random) {
        vec![
            FooterAction {
                key: "←→",
                label: "Tabs",
            },
            FooterAction {
                key: "↑↓",
                label: "Rows",
            },
            FooterAction {
                key: "Enter",
                label: "Play",
            },
            FooterAction {
                key: "-/+",
                label: "Limit",
            },
            FooterAction {
                key: "0",
                label: "Reset",
            },
            FooterAction {
                key: "G",
                label: "Refresh",
            },
            FooterAction {
                key: "T",
                label: "Theme",
            },
            FooterAction {
                key: "Q",
                label: "Quit",
            },
        ]
    } else if matches!(app.view, View::Favourites) {
        vec![
            FooterAction {
                key: "←→",
                label: "Tabs",
            },
            FooterAction {
                key: "↑↓",
                label: "Rows",
            },
            FooterAction {
                key: "Enter",
                label: "Play",
            },
            FooterAction {
                key: "T",
                label: "Theme",
            },
            FooterAction {
                key: "L",
                label: "Logout",
            },
            FooterAction {
                key: "Q",
                label: "Quit",
            },
        ]
    } else {
        vec![
            FooterAction {
                key: "←→",
                label: "Tabs",
            },
            FooterAction {
                key: "↑↓",
                label: "Rows",
            },
            FooterAction {
                key: "[ ]",
                label: "Subtabs",
            },
            FooterAction {
                key: "A",
                label: "Admin",
            },
            FooterAction {
                key: "I",
                label: "Info",
            },
            FooterAction {
                key: "P",
                label: "Proc",
            },
            FooterAction {
                key: "M",
                label: "Mon",
            },
            FooterAction {
                key: "J",
                label: "Jobs",
            },
            FooterAction {
                key: "K",
                label: "Logs",
            },
            FooterAction {
                key: "R",
                label: "Rec",
            },
            FooterAction {
                key: "T",
                label: "Theme",
            },
            FooterAction {
                key: "L",
                label: "Logout",
            },
            FooterAction {
                key: "Q",
                label: "Quit",
            },
        ]
    };
    let function_footer_actions = [
        FooterAction {
            key: "F1",
            label: "Help",
        },
        FooterAction {
            key: "F3",
            label: "Theme",
        },
        FooterAction {
            key: "F4",
            label: "Menu",
        },
        FooterAction {
            key: "F5",
            label: "Player",
        },
        FooterAction {
            key: "F6",
            label: "Mouse",
        },
        FooterAction {
            key: "F10",
            label: "Quit",
        },
    ];
    let footer_right = footer_status_line(vertical[4].width, app, theme);
    draw_footer_bar(
        frame,
        vertical[4],
        &footer_actions,
        &function_footer_actions,
        Some(footer_right),
        theme,
    );
}

fn draw_main_panel(frame: &mut Frame, area: Rect, app: &mut App) {
    if view_has_thumbnail_preview(app.view) {
        let chunks = Layout::default()
            .direction(Direction::Horizontal)
            .constraints([Constraint::Min(24), Constraint::Length(PREVIEW_PANEL_WIDTH)])
            .split(area);

        match app.view {
            View::Streams => draw_streams_table(frame, chunks[0], app),
            View::Channels => draw_channels_table(frame, chunks[0], app),
            View::Channel => {
                let videos = app.channel_recordings();
                let title = app
                    .channel_popup
                    .channel()
                    .as_ref()
                    .map(|channel| display_channel_name(channel))
                    .unwrap_or_else(|| "Channel".to_string());
                let subtitle = app
                    .channel_popup
                    .channel()
                    .as_ref()
                    .map(|channel| {
                        format!(
                            "{} recordings  {}",
                            channel.recordings_count,
                            format_bytes(channel.recordings_size)
                        )
                    })
                    .unwrap_or_else(|| format!("{} items", videos.len()));
                draw_recordings_table(
                    frame,
                    chunks[0],
                    app,
                    View::Channel,
                    &title,
                    &videos,
                    subtitle,
                );
            }
            View::Latest => {
                let videos = app.video_page.videos.clone();
                draw_latest_table(frame, chunks[0], app, &videos);
            }
            View::Random => {
                let videos = app.random_videos.clone();
                draw_random_table(frame, chunks[0], app, &videos);
            }
            View::Favourites => {
                let videos = app.bookmark_videos.clone();
                draw_recordings_table(
                    frame,
                    chunks[0],
                    app,
                    View::Favourites,
                    "Favourites",
                    &videos,
                    format!("{} bookmarked", videos.len()),
                );
            }
            View::Similarity => draw_similarity_table(frame, chunks[0], app),
            View::Admin
            | View::Info
            | View::Processes
            | View::Monitoring
            | View::Jobs
            | View::Logs => {}
        }

        draw_preview_panel(frame, chunks[1], app);
        return;
    }

    match app.view {
        View::Streams => draw_streams_table(frame, area, app),
        View::Channels => draw_channels_table(frame, area, app),
        View::Channel => {
            let videos = app.channel_recordings();
            let title = app
                .channel_popup
                .channel()
                .as_ref()
                .map(|channel| display_channel_name(channel))
                .unwrap_or_else(|| "Channel".to_string());
            let subtitle = app
                .channel_popup
                .channel()
                .as_ref()
                .map(|channel| {
                    format!(
                        "{} recordings  {}",
                        channel.recordings_count,
                        format_bytes(channel.recordings_size)
                    )
                })
                .unwrap_or_else(|| format!("{} items", videos.len()));
            draw_recordings_table(frame, area, app, View::Channel, &title, &videos, subtitle);
        }
        View::Latest => {
            let videos = app.video_page.videos.clone();
            draw_latest_table(frame, area, app, &videos);
        }
        View::Random => {
            let videos = app.random_videos.clone();
            draw_random_table(frame, area, app, &videos);
        }
        View::Favourites => {
            let videos = app.bookmark_videos.clone();
            draw_recordings_table(
                frame,
                area,
                app,
                View::Favourites,
                "Favourites",
                &videos,
                format!("{} bookmarked", videos.len()),
            );
        }
        View::Similarity => draw_similarity_table(frame, area, app),
        View::Admin => draw_admin_view(frame, area, app),
        View::Info => draw_info_view(frame, area, app),
        View::Processes => draw_processes_view(frame, area, app),
        View::Monitoring => draw_monitoring_view(frame, area, app),
        View::Jobs => draw_jobs_table(frame, area, app),
        View::Logs => draw_events_table(frame, area, app),
    }
}

fn draw_latest_table(frame: &mut Frame, area: Rect, app: &mut App, videos: &[Recording]) {
    let theme = app.theme();
    let block = panel_block(
        "Latest",
        format!(
            "{}-{} / {}",
            app.video_page.skip.saturating_add(1),
            min(
                app.video_page
                    .skip
                    .saturating_add(app.video_page.take.max(app.latest_filter.limit())),
                app.video_page.total_count.max(0) as usize
            ),
            app.video_page.total_count
        ),
        theme,
    );
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    if inner.width < 18 || inner.height < MEDIA_ROW_HEIGHT + 2 {
        return;
    }

    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Length(2), Constraint::Min(MEDIA_ROW_HEIGHT)])
        .split(inner);

    frame.render_widget(
        Paragraph::new(app.latest_filter.bar_lines())
            .style(theme.surface_style())
            .wrap(Wrap { trim: true }),
        sections[0],
    );

    draw_recordings_content(frame, sections[1], app, View::Latest, videos);
}

fn draw_random_table(frame: &mut Frame, area: Rect, app: &mut App, videos: &[Recording]) {
    let theme = app.theme();
    let block = panel_block(
        "Random",
        format!(
            "{} items · limit {}",
            videos.len(),
            app.random_filter.limit()
        ),
        theme,
    );
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    if inner.width < 18 || inner.height < MEDIA_ROW_HEIGHT + 2 {
        return;
    }

    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Length(2), Constraint::Min(MEDIA_ROW_HEIGHT)])
        .split(inner);

    frame.render_widget(
        Paragraph::new(app.random_filter.bar_lines())
            .style(theme.surface_style())
            .wrap(Wrap { trim: true }),
        sections[0],
    );

    draw_recordings_content(frame, sections[1], app, View::Random, videos);
}

fn draw_streams_table(frame: &mut Frame, area: Rect, app: &mut App) {
    let theme = app.theme();
    let items = app.visible_stream_channels();
    let counts = app.stream_counts();
    let block = panel_block(
        "Streams",
        format!(
            "{} · {} items",
            app.stream_tab.label(),
            counts.count_for(app.stream_tab)
        ),
        theme,
    );
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    if inner.width < 18 || inner.height < MEDIA_ROW_HEIGHT + 2 {
        return;
    }

    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Length(2), Constraint::Min(MEDIA_ROW_HEIGHT)])
        .split(inner);

    draw_stream_tab_bar(
        frame,
        sections[0],
        app.stream_tab,
        counts,
        theme,
        &mut app.ui_regions,
    );
    draw_channel_rows(frame, sections[1], app, View::Streams, &items);
}

fn draw_channels_table(frame: &mut Frame, area: Rect, app: &mut App) {
    let channels = app.channels.clone();
    draw_channel_collection(
        frame,
        area,
        app,
        View::Channels,
        "Channels",
        &channels,
        format!("{} channels", channels.len()),
    );
}

fn draw_channel_collection(
    frame: &mut Frame,
    area: Rect,
    app: &mut App,
    view: View,
    title: &str,
    channels: &[ChannelInfo],
    subtitle: String,
) {
    let theme = app.theme();
    let block = panel_block(title, subtitle, theme);
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    draw_channel_rows(frame, inner, app, view, channels);
}

fn draw_channel_rows(
    frame: &mut Frame,
    inner: Rect,
    app: &mut App,
    view: View,
    channels: &[ChannelInfo],
) {
    let theme = app.theme();
    if inner.width < 18 || inner.height < MEDIA_ROW_HEIGHT {
        return;
    }

    if let Some(notice) = collection_notice(
        view,
        channels.len(),
        view_is_loading(view, app.refresh_in_flight, app.view_request_in_flight),
        app.content_error.as_deref(),
        app.stream_tab,
    ) {
        draw_panel_notice(frame, inner, &notice, theme);
        return;
    }

    let capacity = max(1, inner.height / MEDIA_ROW_HEIGHT) as usize;
    let (start, end) = visible_window(app.selected_channel, channels.len(), capacity);
    let (content_area, scrollbar_area) = split_scrollbar_area(inner, channels.len() > capacity);
    for offset in 0..(end - start) {
        let channel = channels[start + offset].clone();
        let row_area = Rect {
            x: content_area.x,
            y: content_area.y + offset as u16 * MEDIA_ROW_HEIGHT,
            width: content_area.width,
            height: MEDIA_ROW_HEIGHT,
        };
        draw_channel_row(
            frame,
            row_area,
            app,
            &channel,
            start + offset == app.selected_channel,
        );
    }

    if let Some(scrollbar_area) = scrollbar_area {
        draw_vertical_scrollbar(
            frame,
            scrollbar_area,
            channels.len(),
            start,
            end.saturating_sub(start),
            theme,
        );
    }
}

fn draw_stream_tab_bar(
    frame: &mut Frame,
    area: Rect,
    selected: StreamTab,
    counts: StreamCounts,
    theme: ThemePalette,
    ui_regions: &mut UiRegions,
) {
    let tabs = Tabs::new(vec![
        Line::from(format!(" Recording [{}] ", counts.recording)),
        Line::from(format!(" Offline [{}] ", counts.offline)),
        Line::from(format!(" Disabled [{}] ", counts.disabled)),
    ])
    .padding("", "")
    .style(theme.tab_style())
    .highlight_style(theme.tab_highlight_style())
    .select(match selected {
        StreamTab::Live => 0,
        StreamTab::Offline => 1,
        StreamTab::Disabled => 2,
    })
    .divider(" ");
    frame.render_widget(tabs, area);
    for (rect, tab) in stream_tab_regions(area, counts) {
        ui_regions.register(rect, UiRegion::StreamTab(tab));
    }
}

fn draw_recordings_table(
    frame: &mut Frame,
    area: Rect,
    app: &mut App,
    view: View,
    title: &str,
    videos: &[Recording],
    subtitle: String,
) {
    let theme = app.theme();
    let block = panel_block(title, subtitle, theme);
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    draw_recordings_content(frame, inner, app, view, videos);
}

fn draw_recordings_content(
    frame: &mut Frame,
    inner: Rect,
    app: &mut App,
    view: View,
    videos: &[Recording],
) {
    let theme = app.theme();
    if inner.width < 18 || inner.height < MEDIA_ROW_HEIGHT {
        return;
    }

    let is_loading = match view {
        View::Channel => app.channel_popup.loading(),
        _ => view_is_loading(view, app.refresh_in_flight, app.view_request_in_flight),
    };
    let error = match view {
        View::Channel => app.channel_popup.error(),
        _ => app.content_error.as_deref(),
    };
    if let Some(notice) = collection_notice(view, videos.len(), is_loading, error, app.stream_tab) {
        draw_panel_notice(frame, inner, &notice, theme);
        return;
    }

    let capacity = max(1, inner.height / MEDIA_ROW_HEIGHT) as usize;
    let selected = match view {
        View::Channel => app.channel_popup.selected_recording(),
        _ => app.selected_video,
    };
    let (start, end) = visible_window(selected, videos.len(), capacity);
    let (content_area, scrollbar_area) = split_scrollbar_area(inner, videos.len() > capacity);
    for offset in 0..(end - start) {
        let video = videos[start + offset].clone();
        let row_area = Rect {
            x: content_area.x,
            y: content_area.y + offset as u16 * MEDIA_ROW_HEIGHT,
            width: content_area.width,
            height: MEDIA_ROW_HEIGHT,
        };
        draw_video_row(frame, row_area, app, &video, start + offset == selected);
    }

    if let Some(scrollbar_area) = scrollbar_area {
        draw_vertical_scrollbar(
            frame,
            scrollbar_area,
            videos.len(),
            start,
            end.saturating_sub(start),
            theme,
        );
    }
}

fn draw_channel_popup(frame: &mut Frame, area: Rect, app: &mut App) {
    let theme = app.theme();
    let popup = centered_rect(88, area.height.saturating_sub(2), area);

    let title = app
        .channel_popup
        .channel()
        .map(display_channel_name)
        .unwrap_or_else(|| "Channel".to_string());
    let subtitle = app
        .channel_popup
        .channel()
        .map(|channel| {
            format!(
                "Esc closes  {} recordings  {}",
                channel.recordings_count,
                format_bytes(channel.recordings_size)
            )
        })
        .unwrap_or_else(|| "Esc closes".to_string());

    let inner = render_panel_popup(
        frame,
        popup,
        title,
        subtitle,
        theme,
        &mut app.ui_regions,
        PopupId::ChannelPopup,
    )
    .inner;
    if inner.width < 20 || inner.height < 8 {
        return;
    }

    let sections = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Min(28), Constraint::Length(PREVIEW_PANEL_WIDTH)])
        .split(inner);

    let videos = app.channel_recordings();
    draw_recordings_table(
        frame,
        sections[0],
        app,
        View::Channel,
        "Recordings",
        &videos,
        format!("{} items", videos.len()),
    );
    draw_channel_popup_preview(frame, sections[1], app, &videos);
}

fn draw_channel_popup_preview(frame: &mut Frame, area: Rect, app: &App, videos: &[Recording]) {
    let theme = app.theme();
    let selected = videos.get(app.channel_popup.selected_recording());
    let title = "Preview".to_string();
    let subtitle = selected
        .map(|video| format!("#{} {}", video.recording_id, truncate(&video.filename, 18)))
        .unwrap_or_else(|| "No item selected".to_string());
    let block = panel_block(title, subtitle, theme);
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    if inner.width < 8 || inner.height < 4 {
        return;
    }

    let preview_height = min(inner.height.saturating_sub(1), PREVIEW_THUMBNAIL_HEIGHT);
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Length(preview_height), Constraint::Min(1)])
        .split(inner);

    let key = selected.map(|video| format!("video:{}", video.recording_id));
    match key
        .as_ref()
        .and_then(|cache_key| app.thumbnail_cache.get(cache_key))
    {
        Some(ThumbnailEntry::Ready { preview, .. }) => {
            let preview_area = Rect {
                x: chunks[0].x,
                y: chunks[0].y,
                width: min(chunks[0].width, PREVIEW_THUMBNAIL_WIDTH),
                height: min(chunks[0].height, PREVIEW_THUMBNAIL_HEIGHT),
            };
            draw_rendered_thumbnail(frame.buffer_mut(), preview_area, preview);
        }
        Some(ThumbnailEntry::Loading(target)) => {
            frame.render_widget(
                Paragraph::new(vec![
                    Line::from("Loading preview..."),
                    Line::from(target.label.clone()),
                ])
                .alignment(Alignment::Center)
                .style(theme.surface_style()),
                chunks[0],
            );
        }
        Some(ThumbnailEntry::Failed { error }) => {
            frame.render_widget(
                Paragraph::new(vec![
                    Line::from("Preview unavailable"),
                    Line::from(truncate(error, chunks[0].width as usize)),
                ])
                .alignment(Alignment::Center)
                .style(theme.surface_style()),
                chunks[0],
            );
        }
        None => {
            let notice = if app.channel_popup.loading() {
                "Loading preview..."
            } else {
                "No preview"
            };
            frame.render_widget(
                Paragraph::new(notice)
                    .alignment(Alignment::Center)
                    .style(theme.surface_style()),
                chunks[0],
            );
        }
    }

    let caption = if let Some(video) = selected {
        vec![
            Line::from(truncate(&video.filename, 32)),
            Line::from(format!(
                "{}  {}x{}",
                format_seconds(video.duration),
                video.width,
                video.height
            )),
            Line::from(format!(
                "{}  {}",
                truncate(&video.channel_name, 14),
                format_bytes(video.size)
            )),
        ]
    } else if let Some(error) = app.channel_popup.error() {
        vec![
            Line::from("Channel request failed."),
            Line::from(truncate(error, chunks[1].width as usize)),
        ]
    } else if app.channel_popup.loading() {
        vec![Line::from("Loading channel recordings…")]
    } else {
        vec![Line::from("No recording selected.")]
    };

    frame.render_widget(
        Paragraph::new(caption)
            .style(theme.surface_style())
            .wrap(Wrap { trim: true }),
        chunks[1],
    );
}

fn draw_similarity_table(frame: &mut Frame, area: Rect, app: &mut App) {
    let theme = app.theme();
    let block = panel_block(
        "Similarity",
        app.similarity_groups
            .as_ref()
            .map(|groups| {
                format!(
                    "{} · {} groups · {:>3.0}%",
                    app.similarity_tab.label(),
                    groups.group_count,
                    groups.similarity_threshold * 100.0
                )
            })
            .unwrap_or_else(|| format!("{} · No results", app.similarity_tab.label())),
        theme,
    );
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    if inner.width < 18 || inner.height < 3 {
        return;
    }

    if app.similarity_tab == SimilarityTab::Search {
        frame.render_widget(
            Paragraph::new("Image search is not wired in the TUI yet. Use ] to switch to Group.")
                .alignment(Alignment::Center)
                .style(theme.surface_style())
                .wrap(Wrap { trim: true }),
            inner,
        );
        return;
    }

    if let Some(notice) = similarity_notice(
        app.similarity_tab,
        app.similarity_groups
            .as_ref()
            .map(|groups| groups.groups.len())
            .unwrap_or(0),
        view_is_loading(app.view, app.refresh_in_flight, app.view_request_in_flight),
        app.content_error.as_deref(),
    ) {
        draw_panel_notice(frame, inner, &notice, theme);
        return;
    }

    let Some(groups) = app.similarity_groups.as_ref() else {
        frame.render_widget(
            Paragraph::new("Press g to load similarity groups.")
                .alignment(Alignment::Center)
                .style(theme.surface_style()),
            inner,
        );
        return;
    };

    if groups.groups.is_empty() {
        frame.render_widget(
            Paragraph::new("No similarity groups found.")
                .alignment(Alignment::Center)
                .style(theme.surface_style()),
            inner,
        );
        return;
    }

    let items = groups
        .groups
        .iter()
        .map(|group| {
            ListItem::new(vec![
                Line::from(format!(
                    "Group #{}  {} videos",
                    group.group_id,
                    group.videos.len()
                )),
                Line::from(format!(
                    "max similarity {:>3.0}%",
                    group.max_similarity * 100.0
                )),
            ])
        })
        .collect::<Vec<_>>();

    let capacity = max(1, inner.height / 2) as usize;
    let (start, end) = visible_window(app.selected_similarity, groups.groups.len(), capacity);
    let (content_area, scrollbar_area) =
        split_scrollbar_area(inner, groups.groups.len() > capacity);
    let list = List::new(items)
        .highlight_style(theme.selection_style())
        .block(Block::default().style(theme.surface_style()));
    let mut state = ratatui::widgets::ListState::default()
        .with_selected(Some(app.selected_similarity))
        .with_offset(start);
    frame.render_stateful_widget(list, content_area, &mut state);
    if let Some(scrollbar_area) = scrollbar_area {
        draw_vertical_scrollbar(
            frame,
            scrollbar_area,
            groups.groups.len(),
            start,
            end.saturating_sub(start),
            theme,
        );
    }
}

fn draw_admin_view(frame: &mut Frame, area: Rect, app: &App) {
    let theme = app.theme();
    let block = panel_block(
        "Admin",
        format!(
            "import:{}  previews:{}  updating:{}",
            yes_no(app.admin_status.import.is_importing),
            yes_no(app.admin_status.previews.is_running),
            yes_no(app.admin_status.video_updating)
        ),
        theme,
    );
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    if inner.width < 20 || inner.height < 6 {
        return;
    }

    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(3),
            Constraint::Length(4),
            Constraint::Min(4),
        ])
        .split(inner);

    frame.render_widget(
        Paragraph::new(vec![
            Line::from(format!("Server version: {}", app.server_info.version)),
            Line::from(format!("Server commit:  {}", truncate(&app.server_info.commit, sections[0].width.saturating_sub(16) as usize))),
            Line::from("Frontend parity: status-only for now; mutating admin actions remain disabled in the TUI."),
        ])
        .style(theme.surface_style())
        .wrap(Wrap { trim: true }),
        sections[0],
    );

    frame.render_widget(
        Paragraph::new(vec![
            Line::from(format!(
                "Import: {} ({}/{})",
                if app.admin_status.import.is_importing {
                    "running"
                } else {
                    "idle"
                },
                app.admin_status.import.progress,
                app.admin_status.import.size
            )),
            Line::from(format!(
                "Previews: {} ({}/{})",
                if app.admin_status.previews.is_running {
                    "running"
                } else {
                    "idle"
                },
                app.admin_status.previews.current,
                app.admin_status.previews.total
            )),
            Line::from(format!(
                "Current preview video: {}",
                truncate(
                    &app.admin_status.previews.current_video,
                    sections[1].width.saturating_sub(24) as usize
                )
            )),
            Line::from(format!(
                "Video metadata update: {}",
                if app.admin_status.video_updating {
                    "running"
                } else {
                    "idle"
                }
            )),
        ])
        .style(theme.surface_style())
        .wrap(Wrap { trim: true }),
        sections[1],
    );

    frame.render_widget(
        Paragraph::new(vec![
            Line::from("Available now"),
            Line::from("  g refresh admin status"),
            Line::from("Planned parity"),
            Line::from("  start import / regenerate previews / update metadata"),
        ])
        .style(theme.surface_style())
        .wrap(Wrap { trim: true }),
        sections[2],
    );
}

fn draw_info_view(frame: &mut Frame, area: Rect, app: &App) {
    let theme = app.theme();
    let rows = vec![
        Row::new(vec![
            Cell::from("Client version"),
            Cell::from(env!("CARGO_PKG_VERSION").to_string()),
        ]),
        Row::new(vec![
            Cell::from("API version"),
            Cell::from(
                app.session
                    .as_ref()
                    .map(|session| session.runtime.api_version.clone())
                    .unwrap_or_default(),
            ),
        ]),
        Row::new(vec![
            Cell::from("Server version"),
            Cell::from(app.server_info.version.clone()),
        ]),
        Row::new(vec![
            Cell::from("Server commit"),
            Cell::from(truncate(&app.server_info.commit, 48)),
        ]),
        Row::new(vec![
            Cell::from("Disk usage"),
            Cell::from(format!(
                "{:.0}% used ({:.1} / {:.1} GB)",
                app.disk.pcent, app.disk.used_formatted_gb, app.disk.size_formatted_gb
            )),
        ]),
        Row::new(vec![
            Cell::from("CPU load"),
            Cell::from(
                app.system_info
                    .as_ref()
                    .map(|info| format!("{}%", average_cpu_percent(info)))
                    .unwrap_or_else(|| "n/a".to_string()),
            ),
        ]),
        Row::new(vec![
            Cell::from("Network rx/tx"),
            Cell::from(
                app.system_info
                    .as_ref()
                    .map(|info| {
                        format!(
                            "{} MB / {} MB",
                            info.net_info.receive_bytes / 1024 / 1024,
                            info.net_info.transmit_bytes / 1024 / 1024
                        )
                    })
                    .unwrap_or_else(|| "n/a".to_string()),
            ),
        ]),
    ];

    let header = Row::new(vec!["Key", "Value"]).style(theme.table_header_style());
    let table = Table::new(rows, [Constraint::Length(18), Constraint::Min(20)])
        .header(header)
        .row_highlight_style(theme.selection_style())
        .block(panel_block("Info", "build and runtime details", theme));
    frame.render_widget(table, area);
}

fn draw_processes_view(frame: &mut Frame, area: Rect, app: &App) {
    let theme = app.theme();
    if let Some(notice) = processes_notice(
        app.processes.len(),
        view_is_loading(
            View::Processes,
            app.refresh_in_flight,
            app.view_request_in_flight,
        ),
        app.content_error.as_deref(),
    ) {
        frame.render_widget(
            Paragraph::new(notice.lines)
                .alignment(Alignment::Center)
                .style(theme.notice_style(notice.tone))
                .block(panel_block("Processes", "0 active", theme))
                .wrap(Wrap { trim: true }),
            area,
        );
        return;
    }

    let block = panel_block(
        "Processes",
        format!("{} active", app.processes.len()),
        theme,
    );
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    if inner.width < 12 || inner.height < 3 {
        return;
    }
    let process_capacity = (inner.height as usize).saturating_sub(1);
    let (content_area, scrollbar_area) =
        split_scrollbar_area(inner, app.processes.len() > process_capacity);
    let rows = app.processes.iter().map(|process| {
        Row::new(vec![
            Cell::from(process.pid.to_string()),
            Cell::from(truncate(&process.path, 28)),
            Cell::from(truncate(&process.args, 80)),
        ])
    });
    let header = Row::new(vec!["PID", "Path", "Args"]).style(theme.table_header_style());
    let table = Table::new(
        rows,
        [
            Constraint::Length(8),
            Constraint::Length(30),
            Constraint::Min(20),
        ],
    )
    .header(header)
    .row_highlight_style(theme.selection_style());

    let visible_rows = max(1, content_area.height.saturating_sub(1)) as usize;
    let (start, end) = visible_window(app.selected_process, app.processes.len(), visible_rows);
    let mut state = TableState::default()
        .with_selected(Some(app.selected_process))
        .with_offset(start);
    frame.render_stateful_widget(table, content_area, &mut state);
    if let Some(scrollbar_area) = scrollbar_area {
        let table_scrollbar = Rect {
            x: scrollbar_area.x,
            y: scrollbar_area.y.saturating_add(1),
            width: scrollbar_area.width,
            height: scrollbar_area.height.saturating_sub(1),
        };
        draw_vertical_scrollbar(
            frame,
            table_scrollbar,
            app.processes.len(),
            start,
            end.saturating_sub(start),
            theme,
        );
    }
}

fn draw_monitoring_view(frame: &mut Frame, area: Rect, app: &App) {
    let theme = app.theme();
    let block = panel_block(
        "Monitoring",
        format!("{} samples", app.monitor_history.len()),
        theme,
    );
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    if inner.width < 20 || inner.height < 8 {
        return;
    }

    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(4),
            Constraint::Length(4),
            Constraint::Min(3),
        ])
        .split(inner);
    let network = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Percentage(50), Constraint::Percentage(50)])
        .split(sections[1]);

    let cpu = app
        .monitor_history
        .iter()
        .map(|sample| sample.cpu_load_percent)
        .collect::<Vec<_>>();
    let rx = app
        .monitor_history
        .iter()
        .map(|sample| sample.rx_megabytes)
        .collect::<Vec<_>>();
    let tx = app
        .monitor_history
        .iter()
        .map(|sample| sample.tx_megabytes)
        .collect::<Vec<_>>();

    frame.render_widget(
        Sparkline::default().data(&cpu).max(100).block(panel_block(
            "CPU",
            latest_cpu_summary(app),
            theme,
        )),
        sections[0],
    );
    frame.render_widget(
        Sparkline::default()
            .data(&rx)
            .block(panel_block("RX MB", latest_rx_summary(app), theme)),
        network[0],
    );
    frame.render_widget(
        Sparkline::default()
            .data(&tx)
            .block(panel_block("TX MB", latest_tx_summary(app), theme)),
        network[1],
    );

    let summary = if let Some(info) = &app.system_info {
        vec![
            Line::from(format!(
                "Device {}  disk {:.0}% used  samples {}",
                info.net_info.dev,
                info.disk_info.pcent,
                app.monitor_history.len()
            )),
            Line::from(format!(
                "Last sample {}  rx {} MB  tx {} MB",
                app.monitor_history
                    .last()
                    .map(|sample| sample.timestamp.clone())
                    .unwrap_or_default(),
                info.net_info.receive_bytes / 1024 / 1024,
                info.net_info.transmit_bytes / 1024 / 1024
            )),
        ]
    } else {
        vec![Line::from("No monitoring sample yet. Press g to refresh.")]
    };
    frame.render_widget(
        Paragraph::new(summary)
            .style(theme.surface_style())
            .wrap(Wrap { trim: true }),
        sections[2],
    );
}

fn draw_channel_row(
    frame: &mut Frame,
    area: Rect,
    app: &mut App,
    channel: &ChannelInfo,
    selected: bool,
) {
    let style = row_style(selected, app.theme());
    frame.render_widget(Block::default().style(style), area);
    let chunks = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Length(ROW_THUMBNAIL_WIDTH), Constraint::Min(10)])
        .split(area);

    draw_cached_thumbnail(
        frame,
        chunks[0],
        app,
        &format!("channel:{}", channel.channel_id),
        selected,
    );
    frame.render_widget(
        Paragraph::new(vec![
            Line::from(vec![
                Span::styled(
                    format!("#{}", channel.channel_id),
                    style.add_modifier(Modifier::BOLD),
                ),
                Span::raw(" "),
                Span::styled(
                    truncate(
                        &display_channel_name(channel),
                        chunks[1].width.saturating_sub(2) as usize,
                    ),
                    style.add_modifier(Modifier::BOLD),
                ),
            ]),
            Line::from(format!(
                "on:{} rec:{} pa:{} tr:{} c:{}",
                yes_no(channel.is_online),
                yes_no(channel.is_recording),
                yes_no(channel.is_paused),
                yes_no(channel.is_terminating),
                channel.recordings_count
            )),
            Line::from(truncate(
                &format!("tags: {}", format_tags(&channel.tags)),
                chunks[1].width as usize,
            )),
        ])
        .style(style)
        .wrap(Wrap { trim: true }),
        chunks[1],
    );
}

fn draw_video_row(frame: &mut Frame, area: Rect, app: &mut App, video: &Recording, selected: bool) {
    let style = row_style(selected, app.theme());
    frame.render_widget(Block::default().style(style), area);
    let chunks = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Length(ROW_THUMBNAIL_WIDTH), Constraint::Min(10)])
        .split(area);

    draw_cached_thumbnail(
        frame,
        chunks[0],
        app,
        &format!("video:{}", video.recording_id),
        selected,
    );
    frame.render_widget(
        Paragraph::new(vec![
            Line::from(vec![
                Span::styled(
                    format!("#{}", video.recording_id),
                    style.add_modifier(Modifier::BOLD),
                ),
                Span::raw(" "),
                Span::styled(
                    truncate(&video.filename, chunks[1].width.saturating_sub(2) as usize),
                    style.add_modifier(Modifier::BOLD),
                ),
            ]),
            Line::from(format!(
                "{}  {}  {}x{}",
                truncate(&video.channel_name, 16),
                format_seconds(video.duration),
                video.width,
                video.height
            )),
            Line::from(format!(
                "{}  {}",
                format_bytes(video.size),
                video.created_at
            )),
        ])
        .style(style)
        .wrap(Wrap { trim: true }),
        chunks[1],
    );
}

fn draw_cached_thumbnail(frame: &mut Frame, area: Rect, app: &App, key: &str, selected: bool) {
    let style = row_style(selected, app.theme());
    frame.render_widget(Block::default().style(style), area);
    if area.width < 4 || area.height < 2 {
        return;
    }

    let inner = Rect {
        x: area.x,
        y: area.y,
        width: area.width.min(ROW_THUMBNAIL_WIDTH),
        height: area.height.min(MEDIA_ROW_HEIGHT),
    };

    match app.thumbnail_cache.get(key) {
        Some(ThumbnailEntry::Ready { row, .. }) => {
            draw_rendered_thumbnail(frame.buffer_mut(), inner, row);
        }
        Some(ThumbnailEntry::Loading(target)) => {
            frame.render_widget(
                Paragraph::new(vec![Line::from("..."), Line::from(target.label.clone())])
                    .alignment(Alignment::Center)
                    .style(style),
                inner,
            );
        }
        Some(ThumbnailEntry::Failed { error }) => {
            frame.render_widget(
                Paragraph::new(vec![
                    Line::from("x"),
                    Line::from(truncate(error, inner.width as usize)),
                ])
                .alignment(Alignment::Center)
                .style(style),
                inner,
            );
        }
        None => {
            frame.render_widget(
                Paragraph::new("?")
                    .alignment(Alignment::Center)
                    .style(style),
                inner,
            );
        }
    }
}

fn draw_preview_panel(frame: &mut Frame, area: Rect, app: &App) {
    let theme = app.theme();
    let (title, subtitle, key) = selected_preview_details(app);
    let block = panel_block(title, subtitle, theme);
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    if inner.width < 8 || inner.height < 4 {
        return;
    }

    let preview_height = min(inner.height.saturating_sub(1), PREVIEW_THUMBNAIL_HEIGHT);
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Length(preview_height), Constraint::Min(1)])
        .split(inner);

    match key
        .as_ref()
        .and_then(|cache_key| app.thumbnail_cache.get(cache_key))
    {
        Some(ThumbnailEntry::Ready { preview, .. }) => {
            let preview_area = Rect {
                x: chunks[0].x,
                y: chunks[0].y,
                width: min(chunks[0].width, PREVIEW_THUMBNAIL_WIDTH),
                height: min(chunks[0].height, PREVIEW_THUMBNAIL_HEIGHT),
            };
            draw_rendered_thumbnail(frame.buffer_mut(), preview_area, preview);
        }
        Some(ThumbnailEntry::Loading(target)) => {
            frame.render_widget(
                Paragraph::new(vec![
                    Line::from("Loading preview..."),
                    Line::from(target.label.clone()),
                ])
                .alignment(Alignment::Center)
                .style(theme.surface_style()),
                chunks[0],
            );
        }
        Some(ThumbnailEntry::Failed { error }) => {
            frame.render_widget(
                Paragraph::new(vec![
                    Line::from("Preview unavailable"),
                    Line::from(truncate(error, chunks[0].width as usize)),
                ])
                .alignment(Alignment::Center)
                .style(theme.surface_style()),
                chunks[0],
            );
        }
        None => {
            frame.render_widget(
                Paragraph::new("No preview")
                    .alignment(Alignment::Center)
                    .style(theme.surface_style()),
                chunks[0],
            );
        }
    }

    frame.render_widget(
        Paragraph::new(preview_caption(app))
            .style(theme.surface_style())
            .wrap(Wrap { trim: true }),
        chunks[1],
    );
}

fn draw_jobs_table(frame: &mut Frame, area: Rect, app: &App) {
    let theme = app.theme();
    let block = panel_block(
        "Jobs",
        format!(
            "{}  worker:{}",
            if app.jobs_open_only {
                "open only"
            } else {
                "all"
            },
            yes_no(app.job_worker_active)
        ),
        theme,
    );
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    if inner.width < 12 || inner.height < 3 {
        return;
    }
    let job_capacity = (inner.height as usize).saturating_sub(1);
    let (content_area, scrollbar_area) =
        split_scrollbar_area(inner, app.jobs.jobs.len() > job_capacity);
    let rows = app.jobs.jobs.iter().map(|job| {
        Row::new(vec![
            Cell::from(job.job_id.to_string()),
            Cell::from(job.task.clone()),
            Cell::from(job.status.clone()),
            Cell::from(job.progress.clone().unwrap_or_else(|| "0%".to_string())),
            Cell::from(
                job.pid
                    .map(|pid| pid.to_string())
                    .unwrap_or_else(|| "—".to_string()),
            ),
            Cell::from(truncate(&job.filename, 30)),
        ])
    });

    let header = Row::new(vec!["ID", "Task", "Status", "Progress", "PID", "File"])
        .style(theme.table_header_style());
    let table = Table::new(
        rows,
        [
            Constraint::Length(6),
            Constraint::Length(18),
            Constraint::Length(12),
            Constraint::Length(12),
            Constraint::Length(8),
            Constraint::Min(20),
        ],
    )
    .header(header)
    .row_highlight_style(theme.selection_style());

    let visible_rows = max(1, content_area.height.saturating_sub(1)) as usize;
    let (start, end) = visible_window(app.selected_job, app.jobs.jobs.len(), visible_rows);
    let mut state = TableState::default()
        .with_selected(Some(app.selected_job))
        .with_offset(start);
    frame.render_stateful_widget(table, content_area, &mut state);
    if let Some(scrollbar_area) = scrollbar_area {
        let table_scrollbar = Rect {
            x: scrollbar_area.x,
            y: scrollbar_area.y.saturating_add(1),
            width: scrollbar_area.width,
            height: scrollbar_area.height.saturating_sub(1),
        };
        draw_vertical_scrollbar(
            frame,
            table_scrollbar,
            app.jobs.jobs.len(),
            start,
            end.saturating_sub(start),
            theme,
        );
    }
}

fn draw_events_table(frame: &mut Frame, area: Rect, app: &App) {
    let theme = app.theme();
    let block = panel_block("Logs", format!("{} stored", app.events.len()), theme);
    frame.render_widget(block.clone(), area);
    let inner = block.inner(area);
    if inner.width < 12 || inner.height < 2 {
        return;
    }
    let items = if app.events.is_empty() {
        vec![ListItem::new(Line::from("No websocket events yet."))]
    } else {
        app.events
            .iter()
            .map(|event| {
                ListItem::new(vec![
                    Line::from(vec![
                        Span::styled(
                            format!("{:>8}", event.received_at),
                            theme.event_time_style(),
                        ),
                        Span::raw(" "),
                        Span::styled(&event.name, theme.event_name_style()),
                    ]),
                    Line::from(truncate(
                        &event.summary,
                        area.width.saturating_sub(4) as usize,
                    )),
                ])
            })
            .collect::<Vec<_>>()
    };

    let list = List::new(items).highlight_style(theme.selection_style());
    let visible_items = max(1, inner.height / 2) as usize;
    let (start, end) = visible_window(app.selected_event, app.events.len(), visible_items);
    let (content_area, scrollbar_area) =
        split_scrollbar_area(inner, app.events.len() > visible_items);
    let mut state = ratatui::widgets::ListState::default()
        .with_selected(Some(app.selected_event))
        .with_offset(start);
    frame.render_stateful_widget(list, content_area, &mut state);
    if let Some(scrollbar_area) = scrollbar_area {
        draw_vertical_scrollbar(
            frame,
            scrollbar_area,
            app.events.len(),
            start,
            end.saturating_sub(start),
            theme,
        );
    }
}

fn draw_item_menu(frame: &mut Frame, area: Rect, app: &mut App) {
    let theme = app.theme();
    let popup = centered_rect(58, 16, area);

    let (title, subtitle) = match app.item_menu.target() {
        Some(ActionTarget::Channel(channel)) => (
            "Stream Actions".to_string(),
            truncate(&display_channel_name(channel), 28),
        ),
        Some(ActionTarget::Video(video)) => (
            "Recording Actions".to_string(),
            format!("#{} {}", video.recording_id, truncate(&video.filename, 22)),
        ),
        None => ("Actions".to_string(), String::new()),
    };

    let inner = render_panel_popup(
        frame,
        popup,
        title,
        subtitle,
        theme,
        &mut app.ui_regions,
        PopupId::ItemMenu,
    )
    .inner;
    if inner.width < 20 || inner.height < 6 {
        return;
    }

    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(4), Constraint::Length(1)])
        .split(inner);

    let items = app
        .item_menu
        .items()
        .iter()
        .map(|item| {
            ListItem::new(vec![
                Line::from(item.label.clone()),
                Line::from(Span::styled(item.hint.clone(), theme.subtitle_style())),
            ])
        })
        .collect::<Vec<_>>();
    let list = List::new(items)
        .style(theme.surface_style())
        .highlight_style(theme.selection_style());
    let visible_items = max(1, sections[0].height / 2) as usize;
    let item_count = app.item_menu.items().len();
    let (start, end) = visible_window(app.item_menu.selected(), item_count, visible_items);
    let (content_area, scrollbar_area) =
        split_scrollbar_area(sections[0], item_count > visible_items);
    let mut state = ratatui::widgets::ListState::default()
        .with_selected(Some(app.item_menu.selected()))
        .with_offset(start);
    frame.render_stateful_widget(list, content_area, &mut state);
    if let Some(scrollbar_area) = scrollbar_area {
        draw_vertical_scrollbar(
            frame,
            scrollbar_area,
            item_count,
            start,
            end.saturating_sub(start),
            theme,
        );
    }
    frame.render_widget(
        Paragraph::new("↑↓ select  Enter run  Esc close")
            .alignment(Alignment::Center)
            .style(theme.chrome_style()),
        sections[1],
    );
}

fn draw_channel_editor(frame: &mut Frame, area: Rect, app: &mut App) {
    let Some(editor) = app.channel_editor.as_ref() else {
        return;
    };

    let theme = app.theme();
    let popup = centered_rect(72, 16, area);

    let title = if editor.channel_id().is_some() {
        "Edit Stream"
    } else {
        "Add Stream"
    };
    let subtitle = if editor.channel_name().is_empty() {
        "new stream".to_string()
    } else {
        truncate(editor.channel_name(), 28)
    };
    let inner = render_panel_popup(
        frame,
        popup,
        title,
        subtitle,
        theme,
        &mut app.ui_regions,
        PopupId::ChannelEditor,
    )
    .inner;
    if inner.width < 24 || inner.height < 8 {
        return;
    }

    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(1),
            Constraint::Min(7),
            Constraint::Length(2),
        ])
        .split(inner);

    let info = if editor.channel_name_editable() {
        "Channel name becomes the recordings folder.".to_string()
    } else {
        format!("Channel folder: {}", editor.channel_name())
    };
    frame.render_widget(
        Paragraph::new(info).style(theme.subtitle_style()),
        sections[0],
    );

    let items = ChannelEditorState::fields()
        .iter()
        .map(|field| {
            let label = match field {
                ChannelEditorField::Url => "URL",
                ChannelEditorField::DisplayName => "Display",
                ChannelEditorField::ChannelName => "Channel",
                ChannelEditorField::MinDuration => "Min mins",
                ChannelEditorField::SkipStart => "Skip sec",
                ChannelEditorField::Paused => "Paused",
                ChannelEditorField::Tags => "Tags",
                ChannelEditorField::Save => "Save",
                ChannelEditorField::Cancel => "Cancel",
            };
            let selected = *field == editor.selected();
            let value = match field {
                ChannelEditorField::Paused => selector_spans(editor.field_value(*field), theme),
                ChannelEditorField::Save => {
                    action_spans(editor.field_value(*field), theme, theme.success)
                }
                ChannelEditorField::Cancel => {
                    action_spans(editor.field_value(*field), theme, theme.warning)
                }
                _ => editor
                    .input(*field)
                    .map(|input| form_text_spans(input, selected, false, theme))
                    .unwrap_or_else(|| {
                        vec![Span::styled(
                            editor.field_value(*field),
                            theme.surface_style(),
                        )]
                    }),
            };
            form_row_item(label, value, selected, theme)
        })
        .collect::<Vec<_>>();
    frame.render_widget(List::new(items).style(theme.surface_style()), sections[1]);

    let mut footer_lines = vec![Line::from(
        "Tab/↑↓ move  Ctrl+A/C/X/V edit text  Left/Right toggle pause  Enter save  Esc close",
    )];
    if let Some(error) = editor.error() {
        footer_lines.push(Line::from(Span::styled(
            truncate(error, sections[2].width as usize),
            theme.notice_style(theme.danger),
        )));
    }
    frame.render_widget(
        Paragraph::new(footer_lines)
            .wrap(Wrap { trim: true })
            .style(theme.chrome_style()),
        sections[2],
    );
}

fn draw_enhance_form(frame: &mut Frame, area: Rect, app: &mut App) {
    let Some(form) = app.enhance_form.as_ref() else {
        return;
    };

    let theme = app.theme();
    let popup = centered_rect(74, 17, area);

    let inner = render_panel_popup(
        frame,
        popup,
        "Enhance Recording",
        truncate(form.filename(), 28),
        theme,
        &mut app.ui_regions,
        PopupId::EnhanceForm,
    )
    .inner;
    if inner.width < 24 || inner.height < 8 {
        return;
    }

    if form.is_loading() {
        frame.render_widget(
            Paragraph::new("Loading enhancement options…")
                .alignment(Alignment::Center)
                .style(theme.surface_style()),
            inner,
        );
        return;
    }

    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Min(7),
            Constraint::Length(3),
            Constraint::Length(2),
        ])
        .split(inner);

    let items = EnhanceFormState::fields()
        .iter()
        .map(|field| {
            let label = match field {
                EnhanceField::Resolution => "Resolution",
                EnhanceField::Preset => "Preset",
                EnhanceField::Crf => "CRF",
                EnhanceField::Denoise => "Denoise",
                EnhanceField::Sharpen => "Sharpen",
                EnhanceField::Normalize => "Normalize",
                EnhanceField::Estimate => "Estimate",
                EnhanceField::Save => "Save",
                EnhanceField::Cancel => "Cancel",
            };
            let selected = *field == form.selected();
            let value = match field {
                EnhanceField::Resolution | EnhanceField::Preset | EnhanceField::Normalize => {
                    selector_spans(form.field_value(*field), theme)
                }
                EnhanceField::Estimate => {
                    action_spans(form.field_value(*field), theme, theme.accent)
                }
                EnhanceField::Save => action_spans(form.field_value(*field), theme, theme.success),
                EnhanceField::Cancel => {
                    action_spans(form.field_value(*field), theme, theme.warning)
                }
                _ => form
                    .input(*field)
                    .map(|input| form_text_spans(input, selected, false, theme))
                    .unwrap_or_else(|| {
                        vec![Span::styled(
                            form.field_value(*field),
                            theme.surface_style(),
                        )]
                    }),
            };
            form_row_item(label, value, selected, theme)
        })
        .collect::<Vec<_>>();
    frame.render_widget(List::new(items).style(theme.surface_style()), sections[0]);

    let info = if let Some(size) = form.estimated_size() {
        format!("Estimated file size: {}", format_bytes(size))
    } else {
        "Press g or Enter on Estimate to request a size estimate.".to_string()
    };
    let mut footer_lines = vec![Line::from(info)];
    if let Some(error) = form.error() {
        footer_lines.push(Line::from(Span::styled(
            truncate(error, sections[1].width as usize),
            theme.notice_style(theme.danger),
        )));
    }
    frame.render_widget(
        Paragraph::new(footer_lines)
            .wrap(Wrap { trim: true })
            .style(theme.surface_style()),
        sections[1],
    );
    frame.render_widget(
        Paragraph::new(
            "Tab/↑↓ move  Ctrl+A/C/X/V edit text  <-/-> cycle choices  g estimate  Enter save  Esc close",
        )
            .alignment(Alignment::Center)
            .style(theme.chrome_style()),
        sections[2],
    );
}

fn draw_video_popup(frame: &mut Frame, area: Rect, app: &mut App) {
    let Some(popup_state) = app.video_popup.as_ref() else {
        return;
    };

    let theme = app.theme();
    let Some((popup, sections)) = video_popup_layout(area) else {
        return;
    };

    let subtitle = format!(
        "{} / {}  {}  {}",
        format_duration(popup_state.position_seconds),
        format_duration(popup_state.duration_seconds),
        if popup_state.paused {
            "paused"
        } else {
            "playing"
        },
        app.resolved_player_mode().label()
    );
    let block = panel_block("Player", truncate(&popup_state.label, 40), theme).title(
        Line::from(Span::styled(subtitle, theme.subtitle_style())).alignment(Alignment::Right),
    );
    let _shell = render_popup_shell(
        frame,
        popup,
        block,
        theme,
        &mut app.ui_regions,
        PopupId::VideoPlayer,
    );

    frame.render_widget(
        Block::default().style(theme.surface_alt_style()),
        sections[0],
    );

    if let Some(frame_image) = popup_state.frame.as_ref() {
        draw_rendered_thumbnail(frame.buffer_mut(), sections[0], frame_image);
    } else {
        let message = if popup_state.loading {
            "Loading video frames…"
        } else if let Some(error) = popup_state.error.as_deref() {
            error
        } else {
            "No video frame available yet."
        };
        frame.render_widget(
            Paragraph::new(message)
                .alignment(Alignment::Center)
                .style(theme.notice_style(if popup_state.error.is_some() {
                    theme.danger
                } else {
                    theme.accent
                })),
            sections[0],
        );
    }

    let ratio = if popup_state.duration_seconds <= 0.0 {
        0.0
    } else {
        (popup_state.position_seconds / popup_state.duration_seconds).clamp(0.0, 1.0)
    };
    frame.render_widget(
        Gauge::default()
            .ratio(ratio)
            .gauge_style(theme.chip_style(theme.accent_soft))
            .label(format!(
                "{} / {}",
                format_duration(popup_state.position_seconds),
                format_duration(popup_state.duration_seconds)
            ))
            .style(theme.surface_style()),
        sections[1],
    );
    app.ui_regions.register(sections[1], UiRegion::VideoSeekBar);

    let status_line = if let Some(error) = popup_state.error.as_deref() {
        Line::from(Span::styled(
            truncate(error, sections[2].width as usize),
            theme.notice_style(theme.danger),
        ))
    } else if popup_state.loading {
        Line::from(Span::styled(
            "Fetching frames from ffmpeg…",
            theme.notice_style(theme.accent),
        ))
    } else if popup_state.paused {
        Line::from(Span::styled(
            "Paused. Space resumes. Left/Right seek and refresh the frame.",
            theme.notice_style(theme.warning),
        ))
    } else {
        Line::from(Span::styled(
            "Live popup playback inside the TUI.",
            theme.notice_style(theme.accent),
        ))
    };
    frame.render_widget(
        Paragraph::new(status_line).wrap(Wrap { trim: true }),
        sections[2],
    );
    frame.render_widget(
        Paragraph::new(
            "Space play/pause  ←/→ 5s  PgUp/PgDn 30s  click bar seek  F5 render  Esc close",
        )
        .alignment(Alignment::Center)
        .style(theme.chrome_style()),
        sections[3],
    );
}

fn draw_theme_picker(frame: &mut Frame, area: Rect, app: &mut App) {
    let theme = app.theme();
    let popup = centered_rect(46, 11, area);
    let inner = render_panel_popup(
        frame,
        popup,
        "Themes",
        format!("Current: {}", app.theme_name.label()),
        theme,
        &mut app.ui_regions,
        PopupId::ThemePicker,
    )
    .inner;
    if inner.width < 20 || inner.height < 5 {
        return;
    }

    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(4), Constraint::Length(1)])
        .split(inner);

    let items = ThemeName::all()
        .iter()
        .map(|item| {
            let marker = if *item == app.theme_name { "●" } else { " " };
            ListItem::new(format!(" {}  {}", marker, item.label()))
        })
        .collect::<Vec<_>>();
    let list = List::new(items)
        .highlight_style(theme.selection_style())
        .style(theme.surface_style());
    let selected = ThemeName::all()
        .iter()
        .position(|item| *item == app.theme_picker.selected_theme())
        .unwrap_or(0);
    let capacity = max(1, sections[0].height) as usize;
    let (start, end) = visible_window(selected, ThemeName::all().len(), capacity);
    let (content_area, scrollbar_area) =
        split_scrollbar_area(sections[0], ThemeName::all().len() > capacity);
    let mut state = ratatui::widgets::ListState::default()
        .with_selected(Some(selected))
        .with_offset(start);
    frame.render_stateful_widget(list, content_area, &mut state);
    if let Some(scrollbar_area) = scrollbar_area {
        draw_vertical_scrollbar(
            frame,
            scrollbar_area,
            ThemeName::all().len(),
            start,
            end.saturating_sub(start),
            theme,
        );
    }

    frame.render_widget(
        Paragraph::new("↑↓ select  Enter apply  Esc close")
            .alignment(Alignment::Center)
            .style(theme.chrome_style()),
        sections[1],
    );
}

fn draw_player_picker(frame: &mut Frame, area: Rect, app: &mut App) {
    let theme = app.theme();
    let modes = app.available_player_modes();
    let subtitle = if app.player_capabilities.has_ffmpeg() {
        format!(
            "Current: {} · {} render presets available",
            app.resolved_player_mode().label(),
            modes.len()
        )
    } else {
        "ffmpeg not detected on this machine".to_string()
    };

    let popup = centered_rect(58, 13, area);
    let inner = render_panel_popup(
        frame,
        popup,
        "Player Modes",
        subtitle,
        theme,
        &mut app.ui_regions,
        PopupId::PlayerPicker,
    )
    .inner;
    if inner.width < 24 || inner.height < 6 {
        return;
    }

    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(5), Constraint::Length(1)])
        .split(inner);

    let items = modes
        .iter()
        .map(|mode| {
            let marker = if *mode == app.resolved_player_mode() {
                "●"
            } else {
                " "
            };
            ListItem::new(vec![
                Line::from(format!(" {}  {}", marker, mode.label())),
                Line::from(Span::styled(mode.hint(), theme.subtitle_style())),
            ])
        })
        .collect::<Vec<_>>();
    let list = List::new(items)
        .highlight_style(theme.selection_style())
        .style(theme.surface_style());
    let selected = modes
        .iter()
        .position(|mode| *mode == app.player_picker.selected_mode(&modes))
        .unwrap_or(0);
    let capacity = max(1, sections[0].height / 2) as usize;
    let (start, end) = visible_window(selected, modes.len(), capacity);
    let (content_area, scrollbar_area) = split_scrollbar_area(sections[0], modes.len() > capacity);
    let mut state = ratatui::widgets::ListState::default()
        .with_selected(Some(selected))
        .with_offset(start);
    frame.render_stateful_widget(list, content_area, &mut state);
    if let Some(scrollbar_area) = scrollbar_area {
        draw_vertical_scrollbar(
            frame,
            scrollbar_area,
            modes.len(),
            start,
            end.saturating_sub(start),
            theme,
        );
    }

    frame.render_widget(
        Paragraph::new("↑↓ select  Enter apply  Esc close")
            .alignment(Alignment::Center)
            .style(theme.chrome_style()),
        sections[1],
    );
}

fn draw_help_popup(frame: &mut Frame, area: Rect, app: &mut App) {
    let theme = app.theme();
    let Some((popup, body_area, footer_area)) = help_popup_layout(area) else {
        return;
    };

    let (context_label, subtitle) = match app.help_popup.context() {
        HelpContext::Login => (
            "Login Help",
            "Server, credentials, and global login controls".to_string(),
        ),
        HelpContext::Workspace => (
            "Workspace Help",
            "Navigation, actions, player, forms, and mouse usage".to_string(),
        ),
        HelpContext::VideoPlayer => (
            "Player Help",
            format!(
                "Popup playback · mode {}",
                app.resolved_player_mode().label()
            ),
        ),
    };

    let _shell = render_panel_popup(
        frame,
        popup,
        context_label,
        subtitle,
        theme,
        &mut app.ui_regions,
        PopupId::Help,
    );

    let mut lines = Vec::new();
    for section in help_sections(app.help_popup.context()) {
        lines.push(Line::from(Span::styled(
            format!(" {} ", section.title),
            theme.title_style(),
        )));
        for line in section.lines {
            lines.push(Line::from(vec![
                Span::styled("  ", theme.footer_separator_style()),
                Span::styled((*line).to_string(), theme.surface_style()),
            ]));
        }
        lines.push(Line::from(""));
    }

    let max_scroll = help_popup_max_scroll(app.help_popup.context(), body_area.height);
    let total_lines = help_sections(app.help_popup.context())
        .iter()
        .map(|section| 1usize + section.lines.len() + 1)
        .sum::<usize>();
    let (content_area, scrollbar_area) = split_scrollbar_area(body_area, max_scroll > 0);
    frame.render_widget(
        Paragraph::new(lines)
            .scroll((app.help_popup.scroll(), 0))
            .style(theme.surface_style()),
        content_area,
    );
    if let Some(scrollbar_area) = scrollbar_area {
        draw_vertical_scrollbar(
            frame,
            scrollbar_area,
            total_lines,
            app.help_popup.scroll() as usize,
            content_area.height as usize,
            theme,
        );
    }

    frame.render_widget(
        Paragraph::new(
            "F1/Esc close  ↑↓ scroll  PgUp/PgDn page  Home/End jump  mouse wheel scroll",
        )
        .alignment(Alignment::Center)
        .style(theme.chrome_style()),
        footer_area,
    );
}

fn confirm_copy(app: &App, action: ConfirmAction) -> (&'static str, String) {
    match action {
        ConfirmAction::Logout => (
            "Confirm Logout",
            "Log out, remove the saved session, and return to login so you can switch server or account?"
                .to_string(),
        ),
        ConfirmAction::ToggleRecorder => (
            if app.recorder.is_recording {
                "Confirm Stop Recorder"
            } else {
                "Confirm Start Recorder"
            },
            if app.recorder.is_recording {
                "Stop the recorder now?".to_string()
            } else {
                "Start the recorder now?".to_string()
            },
        ),
        ConfirmAction::ToggleChannelPause(channel) => (
            if channel.is_paused {
                "Confirm Resume Stream"
            } else {
                "Confirm Pause Stream"
            },
            format!(
                "{} {}?",
                if channel.is_paused { "Resume" } else { "Pause" },
                display_channel_name(&channel)
            ),
        ),
        ConfirmAction::ToggleChannelFavourite(channel) => (
            "Confirm Favourite Change",
            format!(
                "{} {}?",
                if channel.fav {
                    "Remove favourite from"
                } else {
                    "Add favourite to"
                },
                display_channel_name(&channel)
            ),
        ),
        ConfirmAction::DeleteChannel(channel) => (
            "Confirm Channel Delete",
            format!(
                "Delete {} and all linked recordings?",
                display_channel_name(&channel)
            ),
        ),
        ConfirmAction::ToggleVideoBookmark(video) => (
            "Confirm Bookmark Change",
            format!(
                "{} recording #{}?",
                if video.bookmark {
                    "Remove bookmark from"
                } else {
                    "Add bookmark to"
                },
                video.recording_id
            ),
        ),
        ConfirmAction::AnalyzeVideo(video) => (
            "Confirm Analysis Job",
            format!("Queue analysis for recording #{}?", video.recording_id),
        ),
        ConfirmAction::GenerateVideoPreview(video) => (
            "Confirm Preview Job",
            format!("Generate preview for recording #{}?", video.recording_id),
        ),
        ConfirmAction::ConvertVideo { media_type, video } => (
            "Confirm Conversion Job",
            format!(
                "Convert recording #{} to {}?",
                video.recording_id, media_type
            ),
        ),
        ConfirmAction::DeleteVideo(video) => (
            "Confirm Recording Delete",
            format!("Delete recording #{}?", video.recording_id),
        ),
    }
}

fn draw_confirm(frame: &mut Frame, area: Rect, app: &mut App, action: ConfirmAction) {
    let theme = app.theme();
    let popup = centered_rect(50, 7, area);
    let (title, body) = confirm_copy(app, action);

    let block = Block::default()
        .title(Line::from(Span::styled(
            format!(" {} ", title),
            theme.title_style(),
        )))
        .borders(Borders::ALL)
        .border_set(border::ROUNDED)
        .border_style(theme.panel_border_style())
        .style(theme.surface_style());
    let inner = render_popup_shell(
        frame,
        popup,
        block,
        theme,
        &mut app.ui_regions,
        PopupId::Confirm,
    )
    .inner;
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(2),
            Constraint::Length(1),
            Constraint::Length(1),
        ])
        .split(inner);
    let buttons = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Length(10), Constraint::Length(10)])
        .split(chunks[1]);
    frame.render_widget(
        Paragraph::new(body)
            .alignment(Alignment::Center)
            .style(theme.surface_style()),
        chunks[0],
    );
    frame.render_widget(
        Paragraph::new("[ Yes ]")
            .alignment(Alignment::Center)
            .style(theme.chip_style(theme.success)),
        buttons[0],
    );
    frame.render_widget(
        Paragraph::new("[ No ]")
            .alignment(Alignment::Center)
            .style(theme.chip_style(theme.warning)),
        buttons[1],
    );
    frame.render_widget(
        Paragraph::new("Enter/Y confirms, Esc/N cancels, click buttons works")
            .alignment(Alignment::Center)
            .style(theme.notice_style(theme.warning)),
        chunks[2],
    );
}

fn draw_input_row(
    frame: &mut Frame,
    area: Rect,
    label: &str,
    value: &TextInput,
    selected: bool,
    masked: bool,
    theme: ThemePalette,
) {
    let label_width = area.width.saturating_sub(12).min(12).max(7);
    let input_min_width = area.width.saturating_sub(label_width).max(1);
    let chunks = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Length(label_width),
            Constraint::Min(input_min_width),
        ])
        .split(area);
    let label_row = Rect {
        x: chunks[0].x,
        y: chunks[0]
            .y
            .saturating_add(chunks[0].height.saturating_sub(1) / 2),
        width: chunks[0].width,
        height: 1,
    };
    frame.render_widget(
        Paragraph::new(label)
            .alignment(Alignment::Right)
            .style(theme.surface_style().fg(theme.muted)),
        label_row,
    );

    let text_style = theme.input_text_style(selected);
    let selection_style = theme
        .surface_style()
        .bg(theme.accent_soft)
        .add_modifier(Modifier::BOLD);
    let cursor_style = theme
        .surface_style()
        .fg(theme.app_bg)
        .bg(theme.border_focus)
        .add_modifier(Modifier::BOLD);
    let block = Block::default()
        .borders(Borders::ALL)
        .border_set(border::ROUNDED)
        .style(theme.input_border_style(selected));
    let content = if selected {
        Line::from(value.value_spans(masked, text_style, selection_style, cursor_style))
    } else {
        Line::from(vec![Span::styled(value.display_text(masked), text_style)])
    };
    frame.render_widget(
        Paragraph::new(content)
            .style(text_style)
            .block(block)
            .wrap(Wrap { trim: false }),
        chunks[1],
    );
}

fn form_text_spans(
    input: &TextInput,
    selected: bool,
    masked: bool,
    theme: ThemePalette,
) -> Vec<Span<'static>> {
    if selected {
        input.value_spans(
            masked,
            theme.surface_style(),
            theme
                .surface_style()
                .bg(theme.accent_soft)
                .add_modifier(Modifier::BOLD),
            theme
                .surface_style()
                .fg(theme.app_bg)
                .bg(theme.border_focus)
                .add_modifier(Modifier::BOLD),
        )
    } else {
        vec![Span::styled(
            input.display_text(masked),
            theme.surface_style(),
        )]
    }
}

fn selector_spans(value: impl Into<String>, theme: ThemePalette) -> Vec<Span<'static>> {
    let value = value.into();
    vec![
        Span::styled("< ", theme.subtitle_style()),
        Span::styled(value, theme.chip_style(theme.accent_soft)),
        Span::styled(" >", theme.subtitle_style()),
    ]
}

fn action_spans(value: impl Into<String>, theme: ThemePalette, tone: Color) -> Vec<Span<'static>> {
    vec![Span::styled(
        format!("[ {} ]", value.into()),
        theme.chip_style(tone),
    )]
}

fn footer_status_line(width: u16, app: &App, theme: ThemePalette) -> Line<'static> {
    let usable_width = width.clamp(16, 22);
    let connected = app.socket_status == "connected";
    let pulse_on = connected && ((Local::now().timestamp_millis() / 450) % 2 == 0);
    let heart_style = if connected {
        theme
            .surface_alt_style()
            .fg(if pulse_on {
                theme.danger
            } else {
                Color::Rgb(170, 40, 40)
            })
            .add_modifier(if pulse_on {
                Modifier::BOLD
            } else {
                Modifier::empty()
            })
    } else {
        theme.surface_alt_style().fg(theme.border)
    };

    let free_percent = (100.0 - app.disk.pcent).clamp(0.0, 100.0);
    let free_tone = if free_percent <= 10.0 {
        theme.danger
    } else if free_percent <= 25.0 {
        theme.warning
    } else {
        theme.success
    };
    let percent_text = format!("{free_percent:>3.0}%");
    let reserved = percent_text.chars().count() + 5;
    let bar_width = usable_width.saturating_sub(reserved as u16) as usize;
    let filled = ((bar_width as f64) * free_percent / 100.0).round() as usize;
    let empty = bar_width.saturating_sub(filled.min(bar_width));

    let mut spans = vec![Span::styled("[", theme.footer_separator_style())];

    if bar_width > 0 {
        spans.push(Span::styled(
            "█".repeat(filled.min(bar_width)),
            theme
                .surface_alt_style()
                .fg(free_tone)
                .add_modifier(Modifier::BOLD),
        ));
        spans.push(Span::styled(
            "░".repeat(empty),
            theme.surface_alt_style().fg(theme.border),
        ));
    }
    spans.push(Span::styled("]", theme.footer_separator_style()));
    spans.push(Span::styled(" ", theme.footer_separator_style()));

    spans.push(Span::styled(
        percent_text,
        theme
            .surface_alt_style()
            .fg(free_tone)
            .add_modifier(Modifier::BOLD),
    ));
    spans.push(Span::styled("  ", theme.footer_separator_style()));
    spans.push(Span::styled("♥", heart_style));

    Line::from(spans)
}

fn form_row_item(
    label: &str,
    value_spans: Vec<Span<'static>>,
    selected: bool,
    theme: ThemePalette,
) -> ListItem<'static> {
    let mut spans = Vec::with_capacity(value_spans.len() + 1);
    spans.push(Span::styled(
        format!("{label:<10} "),
        if selected {
            theme.title_style()
        } else {
            theme.subtitle_style()
        },
    ));
    spans.extend(value_spans);
    ListItem::new(Line::from(spans)).style(row_style(selected, theme))
}

fn selected_preview_details(app: &App) -> (String, String, Option<String>) {
    match app.view {
        View::Channel => {
            let videos = app.channel_recordings();
            if let Some(video) = videos.get(app.channel_popup.selected_recording()) {
                (
                    "Preview".to_string(),
                    format!("#{} {}", video.recording_id, truncate(&video.filename, 18)),
                    Some(format!("video:{}", video.recording_id)),
                )
            } else {
                ("Preview".to_string(), "No item selected".to_string(), None)
            }
        }
        View::Latest => {
            if let Some(video) = app.video_page.videos.get(app.selected_video) {
                (
                    "Preview".to_string(),
                    format!("#{} {}", video.recording_id, truncate(&video.filename, 18)),
                    Some(format!("video:{}", video.recording_id)),
                )
            } else {
                ("Preview".to_string(), "No item selected".to_string(), None)
            }
        }
        View::Random => {
            if let Some(video) = app.random_videos.get(app.selected_video) {
                (
                    "Preview".to_string(),
                    format!("#{} {}", video.recording_id, truncate(&video.filename, 18)),
                    Some(format!("video:{}", video.recording_id)),
                )
            } else {
                ("Preview".to_string(), "No item selected".to_string(), None)
            }
        }
        View::Favourites => {
            if let Some(video) = app.bookmark_videos.get(app.selected_video) {
                (
                    "Preview".to_string(),
                    format!("#{} {}", video.recording_id, truncate(&video.filename, 18)),
                    Some(format!("video:{}", video.recording_id)),
                )
            } else {
                ("Preview".to_string(), "No item selected".to_string(), None)
            }
        }
        View::Streams => {
            let channels = app.visible_stream_channels();
            if let Some(channel) = channels.get(app.selected_channel) {
                (
                    "Preview".to_string(),
                    format!(
                        "#{} {}",
                        channel.channel_id,
                        truncate(&display_channel_name(channel), 18)
                    ),
                    Some(format!("channel:{}", channel.channel_id)),
                )
            } else {
                ("Preview".to_string(), "No item selected".to_string(), None)
            }
        }
        View::Channels => {
            if let Some(channel) = app.channels.get(app.selected_channel) {
                (
                    "Preview".to_string(),
                    format!(
                        "#{} {}",
                        channel.channel_id,
                        truncate(&display_channel_name(channel), 18)
                    ),
                    Some(format!("channel:{}", channel.channel_id)),
                )
            } else {
                ("Preview".to_string(), "No item selected".to_string(), None)
            }
        }
        View::Similarity => {
            if let Some(video) = app
                .selected_similarity_group()
                .and_then(|group| group.videos.first())
            {
                (
                    "Preview".to_string(),
                    format!("#{} {}", video.recording_id, truncate(&video.filename, 18)),
                    Some(format!("video:{}", video.recording_id)),
                )
            } else {
                (
                    "Preview".to_string(),
                    "No similarity preview".to_string(),
                    None,
                )
            }
        }
        View::Admin | View::Info | View::Processes | View::Monitoring | View::Jobs | View::Logs => {
            ("Preview".to_string(), "Unavailable".to_string(), None)
        }
    }
}

fn preview_caption(app: &App) -> Vec<Line<'static>> {
    match app.view {
        View::Channel => {
            let videos = app.channel_recordings();
            if let Some(video) = videos.get(app.channel_popup.selected_recording()) {
                vec![
                    Line::from(truncate(&video.filename, 32)),
                    Line::from(format!(
                        "{}  {}x{}",
                        format_seconds(video.duration),
                        video.width,
                        video.height
                    )),
                    Line::from(format!(
                        "{}  {}",
                        truncate(&video.channel_name, 14),
                        format_bytes(video.size)
                    )),
                ]
            } else {
                vec![Line::from("No recording selected.")]
            }
        }
        View::Latest => {
            if let Some(video) = app.video_page.videos.get(app.selected_video) {
                vec![
                    Line::from(truncate(&video.filename, 32)),
                    Line::from(format!(
                        "{}  {}x{}",
                        format_seconds(video.duration),
                        video.width,
                        video.height
                    )),
                    Line::from(format!(
                        "{}  {}",
                        truncate(&video.channel_name, 14),
                        format_bytes(video.size)
                    )),
                ]
            } else {
                vec![Line::from("No video selected.")]
            }
        }
        View::Random => {
            if let Some(video) = app.random_videos.get(app.selected_video) {
                vec![
                    Line::from(truncate(&video.filename, 32)),
                    Line::from(format!(
                        "{}  {}x{}",
                        format_seconds(video.duration),
                        video.width,
                        video.height
                    )),
                    Line::from(format!(
                        "{}  {}",
                        truncate(&video.channel_name, 14),
                        format_bytes(video.size)
                    )),
                ]
            } else {
                vec![Line::from("No random video selected.")]
            }
        }
        View::Favourites => {
            if let Some(video) = app.bookmark_videos.get(app.selected_video) {
                vec![
                    Line::from(truncate(&video.filename, 32)),
                    Line::from(format!(
                        "{}  {}x{}",
                        format_seconds(video.duration),
                        video.width,
                        video.height
                    )),
                    Line::from(format!(
                        "{}  {}",
                        truncate(&video.channel_name, 14),
                        format_bytes(video.size)
                    )),
                ]
            } else {
                vec![Line::from("No favourite selected.")]
            }
        }
        View::Streams => {
            let channels = app.visible_stream_channels();
            if let Some(channel) = channels.get(app.selected_channel) {
                vec![
                    Line::from(display_channel_name(channel)),
                    Line::from(format!(
                        "live:{} rec:{} pause:{}",
                        yes_no(channel.is_online),
                        yes_no(channel.is_recording),
                        yes_no(channel.is_paused)
                    )),
                    Line::from(format!("recordings: {}", channel.recordings_count)),
                ]
            } else {
                vec![Line::from("No stream selected.")]
            }
        }
        View::Channels => {
            if let Some(channel) = app.channels.get(app.selected_channel) {
                vec![
                    Line::from(display_channel_name(channel)),
                    Line::from(format!(
                        "live:{} rec:{} pause:{}",
                        yes_no(channel.is_online),
                        yes_no(channel.is_recording),
                        yes_no(channel.is_paused)
                    )),
                    Line::from(format!("recordings: {}", channel.recordings_count)),
                ]
            } else {
                vec![Line::from("No channel selected.")]
            }
        }
        View::Similarity => {
            if let Some(group) = app.selected_similarity_group() {
                vec![
                    Line::from(format!("Group #{}", group.group_id)),
                    Line::from(format!("videos: {}", group.videos.len())),
                    Line::from(format!(
                        "max similarity: {:>3.0}%",
                        group.max_similarity * 100.0
                    )),
                ]
            } else {
                vec![Line::from("No similarity group selected.")]
            }
        }
        View::Admin | View::Info | View::Processes | View::Monitoring | View::Jobs | View::Logs => {
            vec![Line::from("No preview in this view.")]
        }
    }
}

use super::*;

impl App {
    pub(super) fn from_loaded_session(loaded: LoadedSession) -> Self {
        let auto_login_pending = loaded.token.is_some();
        let username = loaded.profile.username.unwrap_or_default();
        let player_capabilities = PlayerCapabilities::detect();
        let player_mode = PlayerMode::from_config(loaded.profile.player.as_deref())
            .resolved(&player_capabilities);
        let mouse_enabled = loaded.profile.mouse.unwrap_or(true);

        Self {
            admin_status: AdminStatus {
                import: Default::default(),
                previews: PreviewRegenerationProgress::default(),
                video_updating: false,
            },
            auto_login_pending,
            bookmark_videos: Vec::new(),
            channel_editor: None,
            channels: Vec::new(),
            channel_popup: ChannelPopup::default(),
            clipboard: String::new(),
            content_error: None,
            confirm: None,
            disk: UtilDiskInfo::default(),
            enhance_form: None,
            events: Vec::new(),
            footer_message: if auto_login_pending {
                "Trying saved login.".to_string()
            } else {
                "Enter server, login, and password. F2 toggles login/register.".to_string()
            },
            help_popup: HelpPopup::default(),
            item_menu: ItemActionMenu::default(),
            jobs: JobsResponse::default(),
            jobs_open_only: true,
            job_worker_active: false,
            latest_filter: LatestFilter::default(),
            login_error: None,
            login_field: LoginField::Server,
            login_mode: AuthMode::Login,
            login_password: TextInput::new(String::new()),
            login_server: TextInput::new(loaded.base_url.clone()),
            login_username: TextInput::new(username.clone()),
            monitor_history: Vec::new(),
            mouse_enabled,
            primary_view: View::Streams,
            processes: Vec::new(),
            player_capabilities,
            player_mode,
            player_picker: PlayerModePicker::default(),
            random_videos: Vec::new(),
            random_filter: RandomFilter::default(),
            recorder: RecorderStatusResponse::default(),
            refresh_in_flight: false,
            refresh_pending: auto_login_pending,
            running: true,
            screen: if auto_login_pending {
                Screen::Login
            } else {
                Screen::Login
            },
            selected_process: 0,
            selected_channel: 0,
            selected_event: 0,
            selected_job: 0,
            selected_similarity: 0,
            selected_video: 0,
            session: loaded.token.map(|token| Session {
                base_url: loaded.base_url,
                runtime: RuntimeConfig {
                    api_url: String::new(),
                    api_version: loaded
                        .profile
                        .api_version
                        .filter(|value| !value.trim().is_empty())
                        .unwrap_or_else(|| "0.1.0".to_string()),
                    base_url: String::new(),
                    build: None,
                    file_url: loaded.profile.file_url.unwrap_or_default(),
                    socket_url: String::new(),
                    version: None,
                },
                token,
                username,
            }),
            similarity_groups: None,
            similarity_tab: SimilarityTab::Group,
            socket_message: String::new(),
            socket_status: "offline".to_string(),
            socket_task: None,
            server_info: ServerInfo::default(),
            stream_tab: StreamTab::Live,
            status_tone: ThemeName::from_config(loaded.profile.theme.as_deref())
                .palette()
                .accent,
            system_info: None,
            theme_name: ThemeName::from_config(loaded.profile.theme.as_deref()),
            theme_picker: ThemePicker::default(),
            thumbnail_cache: HashMap::new(),
            ui_regions: UiRegions::default(),
            visual_tick: 0,
            video_popup: None,
            video_page: VideosResponse::default(),
            video_playback_generation: 0,
            video_playback_worker: None,
            view_request_in_flight: false,
            view: View::Streams,
        }
    }

    pub(super) fn set_status(&mut self, message: impl Into<String>, tone: Color) {
        self.footer_message = message.into();
        self.status_tone = tone;
    }

    pub(super) fn theme(&self) -> ThemePalette {
        self.theme_name.palette()
    }

    pub(super) fn preference_profile_base_url(&self) -> Option<String> {
        if let Some(session) = &self.session {
            return Some(session.base_url.clone());
        }
        normalize_server_url(self.login_server.text()).ok()
    }

    pub(super) fn available_player_modes(&self) -> Vec<PlayerMode> {
        self.player_capabilities.available_modes()
    }

    pub(super) fn resolved_player_mode(&self) -> PlayerMode {
        self.player_mode.resolved(&self.player_capabilities)
    }

    pub(super) fn apply_theme_visual(
        &mut self,
        theme_name: ThemeName,
        tx: &UnboundedSender<AppMessage>,
    ) {
        if self.theme_name == theme_name {
            return;
        }
        self.theme_name = theme_name;
        self.thumbnail_cache.clear();
        self.prefetch_thumbnails(tx);
    }

    pub(super) fn save_theme_preference(&mut self, theme_name: ThemeName) {
        let accent = self.theme().accent;
        if let Some(base_url) = self.preference_profile_base_url() {
            match save_profile_theme(&base_url, theme_name.as_str()) {
                Ok(()) => self.set_status(format!("Theme: {}", theme_name.label()), accent),
                Err(error) => self.set_status(
                    format!("Theme changed, but failed to save preference: {error}"),
                    Color::Red,
                ),
            }
        } else {
            self.set_status(format!("Theme: {}", theme_name.label()), accent);
        }
    }

    pub(super) fn preview_theme(
        &mut self,
        theme_name: ThemeName,
        tx: &UnboundedSender<AppMessage>,
    ) {
        self.apply_theme_visual(theme_name, tx);
    }

    pub(super) fn commit_theme(&mut self, theme_name: ThemeName, tx: &UnboundedSender<AppMessage>) {
        self.apply_theme_visual(theme_name, tx);
        self.save_theme_preference(theme_name);
    }

    pub(super) fn restore_theme_preview(&mut self, tx: &UnboundedSender<AppMessage>) {
        let original_theme = self.theme_picker.original_theme();
        self.theme_picker.close();
        self.apply_theme_visual(original_theme, tx);
    }

    pub(super) fn open_theme_picker(&mut self) {
        self.theme_picker.open(self.theme_name);
    }

    pub(super) fn set_player_mode(&mut self, player_mode: PlayerMode) {
        let player_mode = player_mode.resolved(&self.player_capabilities);
        if self.player_mode == player_mode {
            return;
        }

        self.player_mode = player_mode;
        let accent = self.theme().accent;
        if let Some(base_url) = self.preference_profile_base_url() {
            match save_profile_player(&base_url, player_mode.as_str()) {
                Ok(()) => self.set_status(format!("Player mode: {}", player_mode.label()), accent),
                Err(error) => self.set_status(
                    format!("Player mode changed, but failed to save preference: {error}"),
                    Color::Red,
                ),
            }
        } else {
            self.set_status(format!("Player mode: {}", player_mode.label()), accent);
        }
    }

    pub(super) fn open_player_picker(&mut self) {
        let modes = self.available_player_modes();
        self.player_picker.open(self.resolved_player_mode(), &modes);
    }

    pub(super) fn open_help_popup(&mut self) {
        let context = if self.video_popup.is_some() {
            HelpContext::VideoPlayer
        } else if self.screen == Screen::Login {
            HelpContext::Login
        } else {
            HelpContext::Workspace
        };
        self.help_popup.open(context);
    }

    pub(super) fn set_mouse_enabled(&mut self, enabled: bool) {
        if self.mouse_enabled == enabled {
            return;
        }

        self.mouse_enabled = enabled;
        let tone = if enabled {
            self.theme().accent
        } else {
            self.theme().warning
        };
        if let Some(base_url) = self.preference_profile_base_url() {
            match save_profile_mouse(&base_url, enabled) {
                Ok(()) => self.set_status(
                    format!(
                        "Mouse support {}",
                        if enabled { "enabled" } else { "disabled" }
                    ),
                    tone,
                ),
                Err(error) => self.set_status(
                    format!("Mouse support changed, but failed to save preference: {error}"),
                    Color::Red,
                ),
            }
        } else {
            self.set_status(
                format!(
                    "Mouse support {}",
                    if enabled { "enabled" } else { "disabled" }
                ),
                tone,
            );
        }
    }

    pub(super) fn toggle_mouse_enabled(&mut self) {
        self.set_mouse_enabled(!self.mouse_enabled);
    }

    pub(super) fn stop_video_playback_worker(&mut self) {
        if let Some(mut worker) = self.video_playback_worker.take() {
            worker.stop();
        }
    }

    pub(super) fn close_video_popup(&mut self) {
        self.stop_video_playback_worker();
        self.video_popup = None;
    }

    pub(super) fn restart_video_popup_playback(
        &mut self,
        tx: &UnboundedSender<AppMessage>,
        single_frame: bool,
    ) {
        let Some((start_seconds, total_duration_seconds, url)) =
            self.video_popup.as_ref().map(|popup| {
                (
                    popup.clamped_position(),
                    popup.duration_seconds.max(0.0),
                    popup.url.clone(),
                )
            })
        else {
            return;
        };

        self.stop_video_playback_worker();
        let (terminal_cols, terminal_rows) = crossterm::terminal::size().unwrap_or((120, 40));
        let (target_width, target_height) = desired_frame_cells(terminal_cols, terminal_rows);
        let mode = self.resolved_player_mode();
        self.video_playback_generation = self.video_playback_generation.saturating_add(1);
        let generation = self.video_playback_generation;
        if let Some(popup) = self.video_popup.as_mut() {
            popup.generation = generation;
            popup.loading = true;
            popup.error = None;
        }

        let request = VideoPlayerRequest {
            fps: VIDEO_PLAYER_FPS,
            generation,
            mode,
            single_frame,
            start_seconds,
            target_height,
            target_width,
            total_duration_seconds,
            url,
        };
        let sender = tx.clone();
        match spawn_video_worker(request, move |event| {
            let _ = sender.send(AppMessage::VideoPlayer(event));
        }) {
            Ok(worker) => {
                self.video_playback_worker = Some(worker);
            }
            Err(error) => {
                if let Some(popup) = self.video_popup.as_mut() {
                    popup.loading = false;
                    popup.error = Some(error.to_string());
                }
                self.set_status(format!("Playback failed: {error}"), Color::Red);
            }
        }
    }

    pub(super) fn open_video_popup(&mut self, video: Recording, tx: &UnboundedSender<AppMessage>) {
        let Some(session) = self.session.as_ref() else {
            self.set_status("No active session available for playback.", Color::Red);
            return;
        };

        let path = video.path_relative.trim();
        if path.is_empty() {
            self.set_status(
                format!("Recording #{} has no playable path.", video.recording_id),
                Color::Red,
            );
            return;
        }

        let Some(url) = build_file_url(&session.runtime.file_url, path) else {
            self.set_status(
                format!(
                    "Could not build a playback URL for #{}.",
                    video.recording_id
                ),
                Color::Red,
            );
            return;
        };

        let label = if video.filename.trim().is_empty() {
            format!("recording #{}", video.recording_id)
        } else {
            truncate(&video.filename, 48)
        };
        self.video_popup = Some(VideoPopupState::new(
            label.clone(),
            url,
            video.duration.max(0.0),
            self.video_playback_generation.saturating_add(1),
        ));
        self.set_status(
            format!("Playing {label} in popup player…"),
            self.theme().accent,
        );
        self.restart_video_popup_playback(tx, false);
    }

    pub(super) fn toggle_video_popup_pause(&mut self, tx: &UnboundedSender<AppMessage>) {
        let Some((paused, label)) = self.video_popup.as_mut().map(|popup| {
            popup.paused = !popup.paused;
            if popup.paused {
                popup.loading = false;
            }
            (popup.paused, popup.label.clone())
        }) else {
            return;
        };

        if paused {
            self.stop_video_playback_worker();
            self.set_status(format!("Paused {label}"), self.theme().warning);
        } else {
            self.set_status(format!("Resumed {label}"), self.theme().accent);
            self.restart_video_popup_playback(tx, false);
        }
    }

    pub(super) fn seek_video_popup(
        &mut self,
        delta_seconds: f64,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let Some((snapshot, label, position_seconds)) = self.video_popup.as_mut().map(|popup| {
            popup.position_seconds = (popup.position_seconds + delta_seconds)
                .clamp(0.0, popup.duration_seconds.max(0.0));
            (popup.paused, popup.label.clone(), popup.position_seconds)
        }) else {
            return;
        };

        self.set_status(
            format!("{label} @ {}", format_duration(position_seconds)),
            self.theme().accent,
        );
        self.restart_video_popup_playback(tx, snapshot);
    }

    pub(super) fn seek_video_popup_absolute(
        &mut self,
        position_seconds: f64,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let Some((snapshot, label, clamped_position)) = self.video_popup.as_mut().map(|popup| {
            popup.position_seconds = position_seconds.clamp(0.0, popup.duration_seconds.max(0.0));
            (popup.paused, popup.label.clone(), popup.position_seconds)
        }) else {
            return;
        };

        self.set_status(
            format!("{label} @ {}", format_duration(clamped_position)),
            self.theme().accent,
        );
        self.restart_video_popup_playback(tx, snapshot);
    }

    pub(super) fn cache_placeholder_thumbnail(
        &mut self,
        key: String,
        label: impl AsRef<str>,
        accent: Color,
        background: Color,
    ) {
        if self.thumbnail_cache.contains_key(&key) {
            return;
        }

        let label = label.as_ref();
        let row = render_placeholder_thumbnail(
            label,
            ROW_THUMBNAIL_WIDTH,
            MEDIA_ROW_HEIGHT,
            accent,
            background,
        );
        let preview = render_placeholder_thumbnail(
            label,
            PREVIEW_THUMBNAIL_WIDTH,
            PREVIEW_THUMBNAIL_HEIGHT,
            accent,
            background,
        );
        self.thumbnail_cache
            .insert(key, ThumbnailEntry::Ready { preview, row });
    }

    pub(super) fn cache_channel_placeholder(&mut self, channel: &ChannelInfo) {
        let theme = self.theme();
        self.cache_placeholder_thumbnail(
            format!("channel:{}", channel.channel_id),
            display_channel_name(channel),
            channel_placeholder_accent(channel, theme),
            theme.surface_alt_bg,
        );
    }

    pub(super) fn queue_thumbnail(
        &mut self,
        target: ThumbnailTarget,
        tx: &UnboundedSender<AppMessage>,
    ) {
        if self.thumbnail_cache.contains_key(&target.key) {
            return;
        }
        let token = self.session.as_ref().map(|session| session.token.clone());
        self.thumbnail_cache
            .insert(target.key.clone(), ThumbnailEntry::Loading(target.clone()));
        let sender = tx.clone();
        tokio::spawn(async move {
            let result = load_thumbnail_image(&target.url, token.as_deref()).await;
            match result {
                Ok(image) => {
                    let row = render_thumbnail(&image, ROW_THUMBNAIL_WIDTH, MEDIA_ROW_HEIGHT);
                    let preview =
                        render_thumbnail(&image, PREVIEW_THUMBNAIL_WIDTH, PREVIEW_THUMBNAIL_HEIGHT);
                    let _ = sender.send(AppMessage::ThumbnailLoaded {
                        preview,
                        row,
                        target,
                    });
                }
                Err(error) => {
                    let _ = sender.send(AppMessage::ThumbnailFailed {
                        error: error.to_string(),
                        target,
                    });
                }
            }
        });
    }

    pub(super) fn channel_recordings(&self) -> Vec<Recording> {
        self.channel_popup.recordings()
    }

    pub(super) fn selected_channel_item(&self) -> Option<ChannelInfo> {
        match self.view {
            View::Streams => self
                .visible_stream_channels()
                .get(self.selected_channel)
                .cloned(),
            View::Channels => self.channels.get(self.selected_channel).cloned(),
            _ => None,
        }
    }

    pub(super) fn selected_video_item(&self) -> Option<Recording> {
        if self.channel_popup.is_open() {
            return self
                .channel_recordings()
                .get(self.channel_popup.selected_recording())
                .cloned();
        }
        match self.view {
            View::Latest => self.video_page.videos.get(self.selected_video).cloned(),
            View::Random => self.random_videos.get(self.selected_video).cloned(),
            View::Favourites => self.bookmark_videos.get(self.selected_video).cloned(),
            _ => None,
        }
    }

    pub(super) fn open_item_actions(&mut self) {
        if self.channel_popup.is_open() {
            if let Some(video) = self.selected_video_item() {
                self.item_menu.open_video(video);
            }
            return;
        }

        match self.view {
            View::Streams | View::Channels => {
                if let Some(channel) = self.selected_channel_item() {
                    self.item_menu.open_channel(channel);
                }
            }
            View::Latest | View::Random | View::Favourites => {
                if let Some(video) = self.selected_video_item() {
                    self.item_menu.open_video(video);
                }
            }
            _ => {}
        }
    }

    pub(super) fn selected_login_input_mut(&mut self) -> &mut TextInput {
        match self.login_field {
            LoginField::Server => &mut self.login_server,
            LoginField::Username => &mut self.login_username,
            LoginField::Password => &mut self.login_password,
        }
    }

    pub(super) fn apply_text_input_action(&mut self, action: TextInputAction) {
        if let TextInputAction::Copied(text) = action {
            self.clipboard = text;
            self.set_status("Copied selection.", self.theme().accent);
        }
    }

    pub(super) fn handle_paste(&mut self, text: String) {
        if self.confirm.is_some()
            || self.theme_picker.is_open()
            || self.player_picker.is_open()
            || self.video_popup.is_some()
            || self.item_menu.is_open()
        {
            return;
        }

        if let Some(form) = self.enhance_form.as_mut() {
            form.paste(&text);
            return;
        }

        if let Some(editor) = self.channel_editor.as_mut() {
            editor.paste(&text);
            return;
        }

        if self.screen == Screen::Login && !self.auto_login_pending {
            self.login_error = None;
            self.selected_login_input_mut().paste(&text);
        }
    }

    pub(super) fn open_selected_channel(&mut self, tx: &UnboundedSender<AppMessage>) {
        let Some(channel) = self.selected_channel_item() else {
            return;
        };

        self.channel_popup.open(channel);
        self.request_channel_popup(tx);
        self.prefetch_thumbnails(tx);
    }

    pub(super) fn request_channel_popup(&mut self, tx: &UnboundedSender<AppMessage>) {
        let Some(session) = self.session.clone() else {
            return;
        };
        let Some(channel_id) = self.channel_popup.requested_channel_id() else {
            return;
        };

        self.channel_popup.start_loading();
        let sender = tx.clone();
        tokio::spawn(async move {
            let result = async move {
                let client = session.client()?;
                client.channel(channel_id).await
            }
            .await
            .map_err(|error| error.to_string());
            let _ = sender.send(AppMessage::ChannelLoaded { channel_id, result });
        });
    }

    pub(super) fn prefetch_thumbnails(&mut self, tx: &UnboundedSender<AppMessage>) {
        match self.view {
            View::Streams => {
                if let Some(session) = self.session.as_ref() {
                    let file_url = session.runtime.file_url.clone();
                    let channels = self.visible_stream_channels();
                    for channel in &channels {
                        if let Some(url) = build_file_url(&file_url, &channel.preview) {
                            self.queue_thumbnail(
                                ThumbnailTarget {
                                    key: format!("channel:{}", channel.channel_id),
                                    label: truncate(&display_channel_name(channel), 28),
                                    url,
                                },
                                tx,
                            );
                        } else {
                            self.cache_channel_placeholder(channel);
                        }
                    }
                }
            }
            View::Channels => {
                if let Some(session) = self.session.as_ref() {
                    let file_url = session.runtime.file_url.clone();
                    let channels = self.channels.clone();
                    for channel in &channels {
                        if let Some(url) = build_file_url(&file_url, &channel.preview) {
                            self.queue_thumbnail(
                                ThumbnailTarget {
                                    key: format!("channel:{}", channel.channel_id),
                                    label: truncate(&display_channel_name(channel), 28),
                                    url,
                                },
                                tx,
                            );
                        } else {
                            self.cache_channel_placeholder(channel);
                        }
                    }
                }
            }
            View::Channel => {}
            View::Latest => {
                if let Some(session) = self.session.as_ref() {
                    let targets = self
                        .video_page
                        .videos
                        .iter()
                        .filter_map(|video| {
                            let relative = video_thumbnail_relative_path(video)?;
                            let url = build_file_url(&session.runtime.file_url, &relative)?;
                            Some(ThumbnailTarget {
                                key: format!("video:{}", video.recording_id),
                                label: truncate(&video.filename, 28),
                                url,
                            })
                        })
                        .collect::<Vec<_>>();
                    for target in targets {
                        self.queue_thumbnail(target, tx);
                    }
                }
            }
            View::Random => {
                if let Some(session) = self.session.as_ref() {
                    let targets = self
                        .random_videos
                        .iter()
                        .filter_map(|video| {
                            let relative = video_thumbnail_relative_path(video)?;
                            let url = build_file_url(&session.runtime.file_url, &relative)?;
                            Some(ThumbnailTarget {
                                key: format!("video:{}", video.recording_id),
                                label: truncate(&video.filename, 28),
                                url,
                            })
                        })
                        .collect::<Vec<_>>();
                    for target in targets {
                        self.queue_thumbnail(target, tx);
                    }
                }
            }
            View::Favourites => {
                if let Some(session) = self.session.as_ref() {
                    let targets = self
                        .bookmark_videos
                        .iter()
                        .filter_map(|video| {
                            let relative = video_thumbnail_relative_path(video)?;
                            let url = build_file_url(&session.runtime.file_url, &relative)?;
                            Some(ThumbnailTarget {
                                key: format!("video:{}", video.recording_id),
                                label: truncate(&video.filename, 28),
                                url,
                            })
                        })
                        .collect::<Vec<_>>();
                    for target in targets {
                        self.queue_thumbnail(target, tx);
                    }
                }
            }
            View::Similarity => {
                if let Some(session) = self.session.as_ref() {
                    let targets = self
                        .selected_similarity_group()
                        .into_iter()
                        .flat_map(|group| group.videos.iter())
                        .filter_map(|video| {
                            let relative = video_thumbnail_relative_path(video)?;
                            let url = build_file_url(&session.runtime.file_url, &relative)?;
                            Some(ThumbnailTarget {
                                key: format!("video:{}", video.recording_id),
                                label: truncate(&video.filename, 28),
                                url,
                            })
                        })
                        .collect::<Vec<_>>();
                    for target in targets {
                        self.queue_thumbnail(target, tx);
                    }
                }
            }
            View::Admin
            | View::Info
            | View::Processes
            | View::Monitoring
            | View::Jobs
            | View::Logs => {}
        }

        if let Some(session) = self.session.as_ref() {
            let popup_targets = self
                .channel_recordings()
                .iter()
                .filter_map(|video| {
                    let relative = video_thumbnail_relative_path(video)?;
                    let url = build_file_url(&session.runtime.file_url, &relative)?;
                    Some(ThumbnailTarget {
                        key: format!("video:{}", video.recording_id),
                        label: truncate(&video.filename, 28),
                        url,
                    })
                })
                .collect::<Vec<_>>();
            for target in popup_targets {
                self.queue_thumbnail(target, tx);
            }
        }
    }

    pub(super) fn selected_count(&self) -> usize {
        match self.view {
            View::Channel => self.channel_popup.recordings().len(),
            View::Latest => self.video_page.videos.len(),
            View::Random => self.random_videos.len(),
            View::Favourites => self.bookmark_videos.len(),
            View::Streams => self.visible_stream_channels().len(),
            View::Channels => self.channels.len(),
            View::Similarity => self
                .similarity_groups
                .as_ref()
                .map(|groups| groups.groups.len())
                .unwrap_or(0),
            View::Processes => self.processes.len(),
            View::Admin | View::Info | View::Monitoring => 0,
            View::Jobs => self.jobs.jobs.len(),
            View::Logs => self.events.len(),
        }
    }

    pub(super) fn move_selection(&mut self, delta: isize) {
        let count = self.selected_count();
        if count == 0 {
            self.set_selection(0);
            return;
        }

        let current = self.current_selection();
        let next = if delta < 0 {
            if current == 0 { count - 1 } else { current - 1 }
        } else if current + 1 >= count {
            0
        } else {
            current + 1
        };
        self.set_selection(next);
    }

    pub(super) fn current_selection(&self) -> usize {
        match self.view {
            View::Channel => self.channel_popup.selected_recording(),
            View::Latest | View::Random | View::Favourites => self.selected_video,
            View::Streams | View::Channels => self.selected_channel,
            View::Similarity => self.selected_similarity,
            View::Processes => self.selected_process,
            View::Admin | View::Info | View::Monitoring => 0,
            View::Jobs => self.selected_job,
            View::Logs => self.selected_event,
        }
    }

    pub(super) fn set_selection(&mut self, value: usize) {
        match self.view {
            View::Channel => self.channel_popup.set_selected_recording(value),
            View::Latest | View::Random | View::Favourites => self.selected_video = value,
            View::Streams | View::Channels => self.selected_channel = value,
            View::Similarity => self.selected_similarity = value,
            View::Processes => self.selected_process = value,
            View::Admin | View::Info | View::Monitoring => {}
            View::Jobs => self.selected_job = value,
            View::Logs => self.selected_event = value,
        }
    }

    pub(super) fn clamp_selection(&mut self) {
        self.selected_channel = clamp_index(
            self.selected_channel,
            match self.view {
                View::Streams => self.visible_stream_channels().len(),
                _ => self.channels.len(),
            },
        );
        self.channel_popup.clamp_selection();
        self.selected_video = clamp_index(self.selected_video, self.video_page.videos.len());
        self.selected_job = clamp_index(self.selected_job, self.jobs.jobs.len());
        self.selected_event = clamp_index(self.selected_event, self.events.len());
        self.selected_process = clamp_index(self.selected_process, self.processes.len());
        self.selected_similarity = clamp_index(
            self.selected_similarity,
            self.similarity_groups
                .as_ref()
                .map(|groups| groups.groups.len())
                .unwrap_or(0),
        );
    }

    pub(super) fn visible_stream_channels(&self) -> Vec<ChannelInfo> {
        channels_for_tab(&self.channels, self.stream_tab)
    }

    pub(super) fn stream_counts(&self) -> StreamCounts {
        collect_stream_counts(&self.channels)
    }

    pub(super) fn selected_similarity_group(&self) -> Option<&crate::api::SimilarVideoGroup> {
        self.similarity_groups
            .as_ref()
            .and_then(|groups| groups.groups.get(self.selected_similarity))
    }

    pub(super) fn switch_to_workspace(
        &mut self,
        base_url: String,
        runtime: RuntimeConfig,
        token: String,
        username: String,
        warning: Option<String>,
        tx: &UnboundedSender<AppMessage>,
    ) {
        self.screen = Screen::Workspace;
        self.auto_login_pending = false;
        self.login_error = None;
        self.confirm = None;
        self.refresh_in_flight = false;
        self.refresh_pending = true;
        self.session = Some(Session {
            base_url: base_url.clone(),
            runtime,
            token,
            username: username.clone(),
        });
        self.login_server.set_text(base_url);
        self.login_username.set_text(username.clone());
        self.login_password.clear();
        let has_warning = warning.is_some();
        self.set_status(
            warning.unwrap_or_else(|| format!("Signed in as {username}")),
            if has_warning {
                Color::Yellow
            } else {
                Color::Green
            },
        );
        self.start_socket(tx);
    }

    pub(super) fn return_to_login(&mut self, message: impl Into<String>, tone: Color) {
        self.close_video_popup();
        self.screen = Screen::Login;
        self.auto_login_pending = false;
        self.confirm = None;
        self.admin_status = AdminStatus {
            import: Default::default(),
            previews: PreviewRegenerationProgress::default(),
            video_updating: false,
        };
        self.bookmark_videos.clear();
        self.channel_editor = None;
        self.channels.clear();
        self.channel_popup.close();
        self.disk = UtilDiskInfo::default();
        self.enhance_form = None;
        self.processes.clear();
        self.random_videos.clear();
        self.video_page = VideosResponse::default();
        self.jobs = JobsResponse::default();
        self.events.clear();
        self.item_menu.close();
        self.monitor_history.clear();
        self.similarity_groups = None;
        self.server_info = ServerInfo::default();
        self.system_info = None;
        self.recorder = RecorderStatusResponse::default();
        self.job_worker_active = false;
        self.thumbnail_cache.clear();
        self.video_playback_generation = 0;
        self.content_error = None;
        self.refresh_in_flight = false;
        self.refresh_pending = false;
        self.socket_status = "offline".to_string();
        self.socket_message.clear();
        self.view_request_in_flight = false;
        self.set_status(message, tone);
        self.stop_socket();
        self.session = None;
    }

    pub(super) fn stop_socket(&mut self) {
        if let Some(handle) = self.socket_task.take() {
            handle.abort();
        }
    }

    pub(super) fn start_socket(&mut self, tx: &UnboundedSender<AppMessage>) {
        self.stop_socket();
        let Some(session) = self.session.clone() else {
            return;
        };

        let sender = tx.clone();
        self.socket_task = Some(tokio::spawn(async move {
            websocket_loop(session.runtime, session.token, sender).await;
        }));
    }

    pub(super) fn request_refresh(&mut self, tx: &UnboundedSender<AppMessage>) {
        if self.screen != Screen::Workspace || self.refresh_in_flight {
            self.refresh_pending = true;
            return;
        }

        let Some(session) = self.session.clone() else {
            return;
        };

        self.refresh_in_flight = true;
        self.refresh_pending = false;
        if matches!(self.view, View::Streams | View::Channels | View::Latest) {
            self.content_error = None;
        }
        let sender = tx.clone();
        let jobs_open_only = self.jobs_open_only;
        let video_skip = self.video_page.skip;
        let video_take = if self.video_page.take == 0 {
            self.latest_filter.limit()
        } else {
            self.video_page.take
        };
        let video_sort_column = self.latest_filter.sort_column.request_value().to_string();
        let video_sort_order = self.latest_filter.sort_order.label().to_string();

        tokio::spawn(async move {
            let result = async move {
                let client = session.client()?;
                client
                    .refresh_snapshot(
                        video_skip,
                        video_take,
                        &video_sort_column,
                        &video_sort_order,
                        jobs_open_only,
                    )
                    .await
            }
            .await
            .map_err(|error| error.to_string());

            let _ = sender.send(AppMessage::WorkspaceRefreshed(result));
        });
    }

    pub(super) fn request_view_data(&mut self, tx: &UnboundedSender<AppMessage>) {
        let Some(session) = self.session.clone() else {
            return;
        };

        self.view_request_in_flight = true;
        self.content_error = None;
        let sender = tx.clone();
        match self.view {
            View::Channel => {
                self.view_request_in_flight = false;
            }
            View::Random => {
                let limit = self.random_filter.limit();
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        client.random_videos(limit).await
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::RandomVideosLoaded(result));
                });
            }
            View::Favourites => {
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        client.bookmarks().await
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::BookmarkVideosLoaded(result));
                });
            }
            View::Similarity => {
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        client.similarity_groups(0.82, 5000, false).await
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::SimilarityLoaded(result));
                });
            }
            View::Admin => {
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        client.admin_status().await
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::AdminLoaded(result));
                });
            }
            View::Info | View::Monitoring => {
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        client.system_info(1).await
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::SystemInfoLoaded(result));
                });
            }
            View::Processes => {
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        client.processes().await
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::ProcessesLoaded(result));
                });
            }
            View::Streams | View::Channels | View::Latest | View::Jobs | View::Logs => {
                self.view_request_in_flight = false;
            }
        }
    }

    pub(super) fn request_enhance_descriptions(&mut self, tx: &UnboundedSender<AppMessage>) {
        let Some(session) = self.session.clone() else {
            return;
        };

        let sender = tx.clone();
        tokio::spawn(async move {
            let result = async move {
                let client = session.client()?;
                client.enhancement_descriptions().await
            }
            .await
            .map_err(|error| error.to_string());
            let _ = sender.send(AppMessage::EnhanceDescriptionsLoaded(result));
        });
    }

    pub(super) fn request_enhancement_estimate(
        &mut self,
        body: EstimateEnhanceRequest,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let Some(session) = self.session.clone() else {
            return;
        };
        let Some(form) = self.enhance_form.as_ref() else {
            return;
        };
        let id = form.recording_id();
        let sender = tx.clone();
        tokio::spawn(async move {
            let result = async move {
                let client = session.client()?;
                Ok::<u64, anyhow::Error>(
                    client
                        .estimate_video_enhancement(id, &body)
                        .await?
                        .estimated_file_size,
                )
            }
            .await
            .map_err(|error| error.to_string());
            let _ = sender.send(AppMessage::EnhancementEstimated(result));
        });
    }

    pub(super) fn refresh_after_action(
        &mut self,
        tx: &UnboundedSender<AppMessage>,
        feedback: &ActionFeedback,
    ) {
        if feedback.refresh_snapshot {
            self.request_refresh(tx);
        }
        if feedback.refresh_view_data {
            self.request_view_data(tx);
        }
        if feedback.refresh_channel_popup && self.channel_popup.is_open() {
            self.request_channel_popup(tx);
        }
    }

    pub(super) fn open_channel_editor(&mut self, channel: ChannelInfo) {
        self.channel_editor = Some(ChannelEditorState::from_channel(&channel));
    }

    pub(super) fn open_create_stream_editor(&mut self) {
        self.channel_editor = Some(ChannelEditorState::new_stream());
    }

    pub(super) fn open_enhance_form(&mut self, video: Recording, tx: &UnboundedSender<AppMessage>) {
        self.enhance_form = Some(EnhanceFormState::new(&video));
        self.request_enhance_descriptions(tx);
    }

    pub(super) fn dispatch_item_action(
        &mut self,
        action: ItemAction,
        target: ActionTarget,
        tx: &UnboundedSender<AppMessage>,
    ) {
        self.item_menu.close();
        match (action, target) {
            (ItemAction::OpenChannelRecordings, ActionTarget::Channel(_)) => {
                self.open_selected_channel(tx);
            }
            (ItemAction::EditChannel, ActionTarget::Channel(channel)) => {
                self.open_channel_editor(channel);
            }
            (ItemAction::ToggleChannelPause, ActionTarget::Channel(channel)) => {
                self.confirm = Some(ConfirmAction::ToggleChannelPause(channel));
            }
            (ItemAction::ToggleChannelFavourite, ActionTarget::Channel(channel)) => {
                self.confirm = Some(ConfirmAction::ToggleChannelFavourite(channel));
            }
            (ItemAction::DeleteChannel, ActionTarget::Channel(channel)) => {
                self.confirm = Some(ConfirmAction::DeleteChannel(channel));
            }
            (ItemAction::DownloadVideo, ActionTarget::Video(video)) => {
                self.download_video(video, tx);
            }
            (ItemAction::ToggleVideoBookmark, ActionTarget::Video(video)) => {
                self.confirm = Some(ConfirmAction::ToggleVideoBookmark(video));
            }
            (ItemAction::AnalyzeVideo, ActionTarget::Video(video)) => {
                self.confirm = Some(ConfirmAction::AnalyzeVideo(video));
            }
            (ItemAction::GenerateVideoPreview, ActionTarget::Video(video)) => {
                self.confirm = Some(ConfirmAction::GenerateVideoPreview(video));
            }
            (ItemAction::EnhanceVideo, ActionTarget::Video(video)) => {
                self.open_enhance_form(video, tx);
            }
            (ItemAction::ConvertVideo720, ActionTarget::Video(video)) => {
                self.confirm = Some(ConfirmAction::ConvertVideo {
                    media_type: "720".to_string(),
                    video,
                });
            }
            (ItemAction::ConvertVideo1080, ActionTarget::Video(video)) => {
                self.confirm = Some(ConfirmAction::ConvertVideo {
                    media_type: "1080".to_string(),
                    video,
                });
            }
            (ItemAction::DeleteVideo, ActionTarget::Video(video)) => {
                self.confirm = Some(ConfirmAction::DeleteVideo(video));
            }
            _ => {}
        }
    }

    pub(super) fn download_video(&mut self, video: Recording, tx: &UnboundedSender<AppMessage>) {
        let Some(session) = self.session.clone() else {
            return;
        };

        let fallback = format!("recording-{}.mp4", video.recording_id);
        let filename = sanitize_filename(if video.filename.trim().is_empty() {
            &fallback
        } else {
            &video.filename
        });
        let destination = std::env::current_dir()
            .unwrap_or_else(|_| PathBuf::from("."))
            .join(filename);
        self.set_status(
            format!("Downloading video #{}…", video.recording_id),
            self.theme().accent,
        );

        let sender = tx.clone();
        tokio::spawn(async move {
            let result = async move {
                let client = session.client()?;
                client
                    .download_video(video.recording_id, &destination)
                    .await?;
                Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                    message: format!("Downloaded to {}", destination.display()),
                    refresh_channel_popup: false,
                    refresh_snapshot: false,
                    refresh_view_data: false,
                    tone: Color::Green,
                })
            }
            .await
            .map_err(|error| error.to_string());
            let _ = sender.send(AppMessage::ActionFinished(result));
        });
    }

    pub(super) fn play_selected_video(&mut self, tx: &UnboundedSender<AppMessage>) {
        let Some(video) = self.selected_video_item() else {
            return;
        };
        self.open_video_popup(video, tx);
    }

    pub(super) fn execute_confirm_action(
        &mut self,
        action: ConfirmAction,
        tx: &UnboundedSender<AppMessage>,
    ) {
        match action {
            ConfirmAction::ToggleRecorder => self.toggle_recorder(tx),
            ConfirmAction::Logout => self.logout(tx),
            ConfirmAction::ToggleChannelPause(channel) => {
                let Some(session) = self.session.clone() else {
                    return;
                };
                let sender = tx.clone();
                self.set_status(
                    if channel.is_paused {
                        format!("Resuming {}…", display_channel_name(&channel))
                    } else {
                        format!("Pausing {}…", display_channel_name(&channel))
                    },
                    self.theme().accent,
                );
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        if channel.is_paused {
                            client.resume_channel(channel.channel_id).await?;
                            Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                                message: format!("Resumed {}", display_channel_name(&channel)),
                                refresh_channel_popup: true,
                                refresh_snapshot: true,
                                refresh_view_data: false,
                                tone: Color::Green,
                            })
                        } else {
                            client.pause_channel(channel.channel_id).await?;
                            Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                                message: format!("Paused {}", display_channel_name(&channel)),
                                refresh_channel_popup: true,
                                refresh_snapshot: true,
                                refresh_view_data: false,
                                tone: Color::Green,
                            })
                        }
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::ActionFinished(result));
                });
            }
            ConfirmAction::ToggleChannelFavourite(channel) => {
                let Some(session) = self.session.clone() else {
                    return;
                };
                let sender = tx.clone();
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        if channel.fav {
                            client.unfav_channel(channel.channel_id).await?;
                            Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                                message: format!(
                                    "Removed favourite from {}",
                                    display_channel_name(&channel)
                                ),
                                refresh_channel_popup: false,
                                refresh_snapshot: true,
                                refresh_view_data: false,
                                tone: Color::Green,
                            })
                        } else {
                            client.fav_channel(channel.channel_id).await?;
                            Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                                message: format!(
                                    "Added favourite to {}",
                                    display_channel_name(&channel)
                                ),
                                refresh_channel_popup: false,
                                refresh_snapshot: true,
                                refresh_view_data: false,
                                tone: Color::Green,
                            })
                        }
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::ActionFinished(result));
                });
            }
            ConfirmAction::DeleteChannel(channel) => {
                let Some(session) = self.session.clone() else {
                    return;
                };
                let sender = tx.clone();
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        client.delete_channel(channel.channel_id).await?;
                        Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                            message: format!("Deleted {}", display_channel_name(&channel)),
                            refresh_channel_popup: false,
                            refresh_snapshot: true,
                            refresh_view_data: false,
                            tone: Color::Green,
                        })
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::ActionFinished(result));
                });
            }
            ConfirmAction::ToggleVideoBookmark(video) => {
                let Some(session) = self.session.clone() else {
                    return;
                };
                let sender = tx.clone();
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        if video.bookmark {
                            client.unfav_video(video.recording_id).await?;
                            Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                                message: format!("Removed bookmark from #{}", video.recording_id),
                                refresh_channel_popup: true,
                                refresh_snapshot: true,
                                refresh_view_data: true,
                                tone: Color::Green,
                            })
                        } else {
                            client.fav_video(video.recording_id).await?;
                            Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                                message: format!("Bookmarked #{}", video.recording_id),
                                refresh_channel_popup: true,
                                refresh_snapshot: true,
                                refresh_view_data: true,
                                tone: Color::Green,
                            })
                        }
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::ActionFinished(result));
                });
            }
            ConfirmAction::AnalyzeVideo(video) => {
                self.run_video_job_action(
                    video,
                    tx,
                    "Queueing analysis…",
                    |client, id| async move { client.analyze_video(id).await },
                    |video| format!("Queued analysis for #{}", video.recording_id),
                );
            }
            ConfirmAction::GenerateVideoPreview(video) => {
                self.run_video_job_action(
                    video,
                    tx,
                    "Queueing preview generation…",
                    |client, id| async move { client.generate_video_preview(id).await },
                    |video| format!("Queued preview generation for #{}", video.recording_id),
                );
            }
            ConfirmAction::ConvertVideo { media_type, video } => {
                let status = format!(
                    "Queueing {} conversion for #{}…",
                    media_type, video.recording_id
                );
                let success_type = media_type.clone();
                self.run_video_job_action(
                    video,
                    tx,
                    &status,
                    move |client, id| {
                        let media_type = media_type.clone();
                        async move { client.convert_video(id, &media_type).await }
                    },
                    move |video| {
                        format!(
                            "Queued {} conversion for #{}",
                            success_type, video.recording_id
                        )
                    },
                );
            }
            ConfirmAction::DeleteVideo(video) => {
                let Some(session) = self.session.clone() else {
                    return;
                };
                let sender = tx.clone();
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        client.delete_video(video.recording_id).await?;
                        Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                            message: format!("Deleted recording #{}", video.recording_id),
                            refresh_channel_popup: true,
                            refresh_snapshot: true,
                            refresh_view_data: true,
                            tone: Color::Green,
                        })
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::ActionFinished(result));
                });
            }
        }
    }

    pub(super) fn run_video_job_action<F, Fut>(
        &mut self,
        video: Recording,
        tx: &UnboundedSender<AppMessage>,
        status_message: &str,
        action: F,
        success_message: impl Fn(&Recording) -> String + Send + 'static,
    ) where
        F: FnOnce(ApiClient, u64) -> Fut + Send + 'static,
        Fut: std::future::Future<Output = Result<(), anyhow::Error>> + Send + 'static,
    {
        let Some(session) = self.session.clone() else {
            return;
        };
        self.set_status(status_message.to_string(), self.theme().accent);
        let sender = tx.clone();
        tokio::spawn(async move {
            let result = async move {
                let client = session.client()?;
                action(client, video.recording_id).await?;
                Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                    message: success_message(&video),
                    refresh_channel_popup: true,
                    refresh_snapshot: true,
                    refresh_view_data: true,
                    tone: Color::Green,
                })
            }
            .await
            .map_err(|error| error.to_string());
            let _ = sender.send(AppMessage::ActionFinished(result));
        });
    }

    pub(super) fn set_view(&mut self, view: View, tx: &UnboundedSender<AppMessage>) {
        self.view = view;
        if view.is_primary() {
            self.primary_view = view;
        }
        self.clamp_selection();
        self.prefetch_thumbnails(tx);
        self.request_view_data(tx);
    }
}

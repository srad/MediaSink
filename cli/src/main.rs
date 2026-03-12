mod api;
mod app;
mod config;
mod overlays;
mod player_mode;
mod selection;
mod serde_ext;
mod ui;

use crate::api::{
    AdminStatus, ApiClient, AuthSuccess, ChannelInfo, EnhancementDescriptions,
    EstimateEnhanceRequest, JobsResponse, PreviewRegenerationProgress, ProcessInfo,
    RecorderStatusResponse, Recording, RuntimeConfig, ServerInfo, SimilarityGroupsResponse,
    SocketEnvelope, UtilDiskInfo, UtilSysInfo, VideosResponse, WorkspaceSnapshot,
};
use crate::app::{
    LatestFilter, RandomFilter, SimilarityTab, StreamCounts, StreamTab, View, channels_for_tab,
    collection_notice, is_loading as view_is_loading, processes_notice, similarity_notice,
    sort_channels, stream_counts as collect_stream_counts,
};
use crate::config::{
    LoadedSession, clear_saved_session, load_saved_session, normalize_server_url,
    save_authenticated_session, save_profile_mouse, save_profile_player, save_profile_theme,
};
use crate::overlays::{
    ActionTarget, ChannelEditorEvent, ChannelEditorField, ChannelEditorState,
    ChannelEditorSubmit, ChannelPopup, EnhanceField, EnhanceFormEvent, EnhanceFormState,
    HelpContext, HelpPopup, ItemAction, ItemActionMenu, PlayerModePicker, VideoPlaybackWorker,
    VideoPlayerEvent, VideoPlayerRequest, VideoPopupState, desired_frame_cells, help_sections,
    spawn_video_worker, VIDEO_PLAYER_FPS, VIDEO_PLAYER_SEEK_STEP_SECONDS,
};
use crate::player_mode::{PlayerCapabilities, PlayerMode};
use crate::selection::{clamp_index, visible_window};
use crate::ui::{
    FooterAction, PopupId, RenderedThumbnail, TextInput, TextInputAction, ThemeBackground,
    ThemeName, ThemePalette, ThemePicker, ThumbnailEntry, ThumbnailTarget, UiRegion, UiRegions,
    centered_rect, draw_footer_bar, draw_panel_notice, draw_rendered_thumbnail,
    draw_theme_background,
    draw_vertical_scrollbar, load_thumbnail_image, panel_block, render_panel_popup,
    render_placeholder_thumbnail, render_popup_shell, render_thumbnail, row_style,
    split_scrollbar_area,
};
use anyhow::{Context, Result};
use chrono::Local;
use crossterm::event::{
    DisableMouseCapture, EnableMouseCapture, Event as CrosstermEvent, EventStream, KeyCode,
    KeyEvent, KeyModifiers, MouseButton, MouseEvent, MouseEventKind,
};
use futures_util::StreamExt;
use ratatui::{
    DefaultTerminal, Frame,
    layout::{Alignment, Constraint, Direction, Layout, Rect},
    prelude::{Color, Modifier},
    symbols::border,
    text::{Line, Span},
    widgets::{
        Block, Borders, Cell, Gauge, List, ListItem, Paragraph, Row, Sparkline, Table, TableState,
        Tabs, Wrap,
    },
};
use serde_json::Value;
use std::{
    cmp::{max, min},
    collections::HashMap,
    io::stdout,
    path::PathBuf,
    time::{Duration, Instant},
};
use tokio::{
    sync::mpsc::{UnboundedReceiver, UnboundedSender, unbounded_channel},
    task::JoinHandle,
    time::sleep,
};
use tokio_tungstenite::{connect_async, tungstenite::Message};
use url::Url;

const REFRESH_INTERVAL: Duration = Duration::from_secs(10);
const SOCKET_RECONNECT_DELAY: Duration = Duration::from_secs(3);
const THEME_ANIMATION_INTERVAL: Duration = Duration::from_millis(55);
const MEDIA_ROW_HEIGHT: u16 = 4;
const ROW_THUMBNAIL_WIDTH: u16 = 14;
const PREVIEW_THUMBNAIL_WIDTH: u16 = 32;
const PREVIEW_THUMBNAIL_HEIGHT: u16 = 10;
const PREVIEW_PANEL_WIDTH: u16 = 36;
const CLI_VERSION: &str = env!("CARGO_PKG_VERSION");

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
enum AuthMode {
    Login,
    Signup,
}

impl AuthMode {
    fn toggle(self) -> Self {
        match self {
            Self::Login => Self::Signup,
            Self::Signup => Self::Login,
        }
    }

    fn label(self) -> &'static str {
        match self {
            Self::Login => "Login",
            Self::Signup => "Register",
        }
    }
}

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
enum LoginField {
    Server,
    Username,
    Password,
}

impl LoginField {
    fn next(self) -> Self {
        match self {
            Self::Server => Self::Username,
            Self::Username => Self::Password,
            Self::Password => Self::Server,
        }
    }

    fn previous(self) -> Self {
        match self {
            Self::Server => Self::Password,
            Self::Username => Self::Server,
            Self::Password => Self::Username,
        }
    }
}

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
enum Screen {
    Login,
    Workspace,
}

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
enum LoginMouseAction {
    Submit,
    ToggleMode,
    Mouse,
    Quit,
}

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
enum WorkspaceHeaderAction {
    AddStream,
    Logout,
    Recorder,
}

#[derive(Debug, Clone)]
enum ConfirmAction {
    AnalyzeVideo(Recording),
    ConvertVideo {
        media_type: String,
        video: Recording,
    },
    DeleteChannel(ChannelInfo),
    DeleteVideo(Recording),
    GenerateVideoPreview(Recording),
    Logout,
    ToggleChannelFavourite(ChannelInfo),
    ToggleChannelPause(ChannelInfo),
    ToggleRecorder,
    ToggleVideoBookmark(Recording),
}

#[derive(Debug, Clone)]
struct Session {
    base_url: String,
    runtime: RuntimeConfig,
    token: String,
    username: String,
}

impl Session {
    fn client(&self) -> Result<ApiClient> {
        ApiClient::new(self.runtime.clone(), Some(self.token.clone()))
    }
}

#[allow(dead_code)]
#[derive(Debug, Clone)]
struct LiveEvent {
    data: Value,
    name: String,
    received_at: String,
    summary: String,
}

#[derive(Debug, Clone)]
struct MetricSample {
    cpu_load_percent: u64,
    rx_megabytes: u64,
    timestamp: String,
    tx_megabytes: u64,
}

#[derive(Debug, Clone)]
struct ActionFeedback {
    message: String,
    refresh_channel_popup: bool,
    refresh_snapshot: bool,
    refresh_view_data: bool,
    tone: Color,
}

#[derive(Debug)]
enum AppMessage {
    ActionFinished(Result<ActionFeedback, String>),
    AuthFailed(String),
    AuthSucceeded {
        base_url: String,
        runtime: RuntimeConfig,
        token: String,
        username: String,
        warning: Option<String>,
    },
    LoggedOut(Result<(), String>),
    RecorderToggled(Result<bool, String>),
    SocketEvent(LiveEvent),
    SocketStatus {
        message: Option<String>,
        status: String,
    },
    VideoPlayer(VideoPlayerEvent),
    AdminLoaded(Result<AdminStatus, String>),
    ChannelLoaded {
        channel_id: u64,
        result: Result<ChannelInfo, String>,
    },
    EnhanceDescriptionsLoaded(Result<EnhancementDescriptions, String>),
    EnhancementEstimated(Result<u64, String>),
    ThumbnailFailed {
        error: String,
        target: ThumbnailTarget,
    },
    ThumbnailLoaded {
        preview: RenderedThumbnail,
        row: RenderedThumbnail,
        target: ThumbnailTarget,
    },
    RandomVideosLoaded(Result<Vec<Recording>, String>),
    BookmarkVideosLoaded(Result<Vec<Recording>, String>),
    ProcessesLoaded(Result<Vec<ProcessInfo>, String>),
    SimilarityLoaded(Result<SimilarityGroupsResponse, String>),
    SystemInfoLoaded(Result<UtilSysInfo, String>),
    WorkspaceRefreshed(Result<WorkspaceSnapshot, String>),
}

struct App {
    admin_status: AdminStatus,
    auto_login_pending: bool,
    bookmark_videos: Vec<Recording>,
    channel_editor: Option<ChannelEditorState>,
    channels: Vec<ChannelInfo>,
    channel_popup: ChannelPopup,
    clipboard: String,
    content_error: Option<String>,
    confirm: Option<ConfirmAction>,
    disk: UtilDiskInfo,
    enhance_form: Option<EnhanceFormState>,
    events: Vec<LiveEvent>,
    footer_message: String,
    help_popup: HelpPopup,
    item_menu: ItemActionMenu,
    jobs: JobsResponse,
    jobs_open_only: bool,
    job_worker_active: bool,
    latest_filter: LatestFilter,
    login_error: Option<String>,
    login_field: LoginField,
    login_mode: AuthMode,
    login_password: TextInput,
    login_server: TextInput,
    login_username: TextInput,
    monitor_history: Vec<MetricSample>,
    mouse_enabled: bool,
    primary_view: View,
    processes: Vec<ProcessInfo>,
    player_capabilities: PlayerCapabilities,
    player_mode: PlayerMode,
    player_picker: PlayerModePicker,
    random_videos: Vec<Recording>,
    random_filter: RandomFilter,
    recorder: RecorderStatusResponse,
    refresh_in_flight: bool,
    refresh_pending: bool,
    running: bool,
    screen: Screen,
    selected_process: usize,
    selected_channel: usize,
    selected_event: usize,
    selected_job: usize,
    selected_similarity: usize,
    selected_video: usize,
    session: Option<Session>,
    similarity_groups: Option<SimilarityGroupsResponse>,
    similarity_tab: SimilarityTab,
    socket_message: String,
    socket_status: String,
    socket_task: Option<JoinHandle<()>>,
    server_info: ServerInfo,
    stream_tab: StreamTab,
    status_tone: Color,
    system_info: Option<UtilSysInfo>,
    theme_name: ThemeName,
    theme_picker: ThemePicker,
    thumbnail_cache: HashMap<String, ThumbnailEntry>,
    ui_regions: UiRegions,
    visual_tick: u64,
    video_popup: Option<VideoPopupState>,
    video_page: VideosResponse,
    video_playback_generation: u64,
    video_playback_worker: Option<VideoPlaybackWorker>,
    view_request_in_flight: bool,
    view: View,
}

impl App {
    fn from_loaded_session(loaded: LoadedSession) -> Self {
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

    fn set_status(&mut self, message: impl Into<String>, tone: Color) {
        self.footer_message = message.into();
        self.status_tone = tone;
    }

    fn theme(&self) -> ThemePalette {
        self.theme_name.palette()
    }

    fn preference_profile_base_url(&self) -> Option<String> {
        if let Some(session) = &self.session {
            return Some(session.base_url.clone());
        }
        normalize_server_url(self.login_server.text()).ok()
    }

    fn available_player_modes(&self) -> Vec<PlayerMode> {
        self.player_capabilities.available_modes()
    }

    fn resolved_player_mode(&self) -> PlayerMode {
        self.player_mode.resolved(&self.player_capabilities)
    }

    fn apply_theme_visual(&mut self, theme_name: ThemeName, tx: &UnboundedSender<AppMessage>) {
        if self.theme_name == theme_name {
            return;
        }
        self.theme_name = theme_name;
        self.thumbnail_cache.clear();
        self.prefetch_thumbnails(tx);
    }

    fn save_theme_preference(&mut self, theme_name: ThemeName) {
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

    fn preview_theme(&mut self, theme_name: ThemeName, tx: &UnboundedSender<AppMessage>) {
        self.apply_theme_visual(theme_name, tx);
    }

    fn commit_theme(&mut self, theme_name: ThemeName, tx: &UnboundedSender<AppMessage>) {
        self.apply_theme_visual(theme_name, tx);
        self.save_theme_preference(theme_name);
    }

    fn restore_theme_preview(&mut self, tx: &UnboundedSender<AppMessage>) {
        let original_theme = self.theme_picker.original_theme();
        self.theme_picker.close();
        self.apply_theme_visual(original_theme, tx);
    }

    fn open_theme_picker(&mut self) {
        self.theme_picker.open(self.theme_name);
    }

    fn set_player_mode(&mut self, player_mode: PlayerMode) {
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

    fn open_player_picker(&mut self) {
        let modes = self.available_player_modes();
        self.player_picker.open(self.resolved_player_mode(), &modes);
    }

    fn open_help_popup(&mut self) {
        let context = if self.video_popup.is_some() {
            HelpContext::VideoPlayer
        } else if self.screen == Screen::Login {
            HelpContext::Login
        } else {
            HelpContext::Workspace
        };
        self.help_popup.open(context);
    }

    fn set_mouse_enabled(&mut self, enabled: bool) {
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

    fn toggle_mouse_enabled(&mut self) {
        self.set_mouse_enabled(!self.mouse_enabled);
    }

    fn stop_video_playback_worker(&mut self) {
        if let Some(mut worker) = self.video_playback_worker.take() {
            worker.stop();
        }
    }

    fn close_video_popup(&mut self) {
        self.stop_video_playback_worker();
        self.video_popup = None;
    }

    fn restart_video_popup_playback(
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

    fn open_video_popup(&mut self, video: Recording, tx: &UnboundedSender<AppMessage>) {
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

    fn toggle_video_popup_pause(&mut self, tx: &UnboundedSender<AppMessage>) {
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

    fn seek_video_popup(&mut self, delta_seconds: f64, tx: &UnboundedSender<AppMessage>) {
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

    fn seek_video_popup_absolute(
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

    fn cache_placeholder_thumbnail(
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

    fn cache_channel_placeholder(&mut self, channel: &ChannelInfo) {
        let theme = self.theme();
        self.cache_placeholder_thumbnail(
            format!("channel:{}", channel.channel_id),
            display_channel_name(channel),
            channel_placeholder_accent(channel, theme),
            theme.surface_alt_bg,
        );
    }

    fn queue_thumbnail(&mut self, target: ThumbnailTarget, tx: &UnboundedSender<AppMessage>) {
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

    fn channel_recordings(&self) -> Vec<Recording> {
        self.channel_popup.recordings()
    }

    fn selected_channel_item(&self) -> Option<ChannelInfo> {
        match self.view {
            View::Streams => self
                .visible_stream_channels()
                .get(self.selected_channel)
                .cloned(),
            View::Channels => self.channels.get(self.selected_channel).cloned(),
            _ => None,
        }
    }

    fn selected_video_item(&self) -> Option<Recording> {
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

    fn open_item_actions(&mut self) {
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

    fn selected_login_input_mut(&mut self) -> &mut TextInput {
        match self.login_field {
            LoginField::Server => &mut self.login_server,
            LoginField::Username => &mut self.login_username,
            LoginField::Password => &mut self.login_password,
        }
    }

    fn apply_text_input_action(&mut self, action: TextInputAction) {
        if let TextInputAction::Copied(text) = action {
            self.clipboard = text;
            self.set_status("Copied selection.", self.theme().accent);
        }
    }

    fn handle_paste(&mut self, text: String) {
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

    fn open_selected_channel(&mut self, tx: &UnboundedSender<AppMessage>) {
        let Some(channel) = self.selected_channel_item() else {
            return;
        };

        self.channel_popup.open(channel);
        self.request_channel_popup(tx);
        self.prefetch_thumbnails(tx);
    }

    fn request_channel_popup(&mut self, tx: &UnboundedSender<AppMessage>) {
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

    fn prefetch_thumbnails(&mut self, tx: &UnboundedSender<AppMessage>) {
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

    fn selected_count(&self) -> usize {
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

    fn move_selection(&mut self, delta: isize) {
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

    fn current_selection(&self) -> usize {
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

    fn set_selection(&mut self, value: usize) {
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

    fn clamp_selection(&mut self) {
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

    fn visible_stream_channels(&self) -> Vec<ChannelInfo> {
        channels_for_tab(&self.channels, self.stream_tab)
    }

    fn stream_counts(&self) -> StreamCounts {
        collect_stream_counts(&self.channels)
    }

    fn selected_similarity_group(&self) -> Option<&crate::api::SimilarVideoGroup> {
        self.similarity_groups
            .as_ref()
            .and_then(|groups| groups.groups.get(self.selected_similarity))
    }

    fn switch_to_workspace(
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

    fn return_to_login(&mut self, message: impl Into<String>, tone: Color) {
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

    fn stop_socket(&mut self) {
        if let Some(handle) = self.socket_task.take() {
            handle.abort();
        }
    }

    fn start_socket(&mut self, tx: &UnboundedSender<AppMessage>) {
        self.stop_socket();
        let Some(session) = self.session.clone() else {
            return;
        };

        let sender = tx.clone();
        self.socket_task = Some(tokio::spawn(async move {
            websocket_loop(session.runtime, session.token, sender).await;
        }));
    }

    fn request_refresh(&mut self, tx: &UnboundedSender<AppMessage>) {
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

    fn request_view_data(&mut self, tx: &UnboundedSender<AppMessage>) {
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

    fn request_enhance_descriptions(&mut self, tx: &UnboundedSender<AppMessage>) {
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

    fn request_enhancement_estimate(
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

    fn refresh_after_action(
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

    fn open_channel_editor(&mut self, channel: ChannelInfo) {
        self.channel_editor = Some(ChannelEditorState::from_channel(&channel));
    }

    fn open_create_stream_editor(&mut self) {
        self.channel_editor = Some(ChannelEditorState::new_stream());
    }

    fn open_enhance_form(&mut self, video: Recording, tx: &UnboundedSender<AppMessage>) {
        self.enhance_form = Some(EnhanceFormState::new(&video));
        self.request_enhance_descriptions(tx);
    }

    fn dispatch_item_action(
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

    fn download_video(&mut self, video: Recording, tx: &UnboundedSender<AppMessage>) {
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

    fn play_selected_video(&mut self, tx: &UnboundedSender<AppMessage>) {
        let Some(video) = self.selected_video_item() else {
            return;
        };
        self.open_video_popup(video, tx);
    }

    fn execute_confirm_action(&mut self, action: ConfirmAction, tx: &UnboundedSender<AppMessage>) {
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

    fn run_video_job_action<F, Fut>(
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

    fn set_view(&mut self, view: View, tx: &UnboundedSender<AppMessage>) {
        self.view = view;
        if view.is_primary() {
            self.primary_view = view;
        }
        self.clamp_selection();
        self.prefetch_thumbnails(tx);
        self.request_view_data(tx);
    }

    fn handle_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        if key.kind != crossterm::event::KeyEventKind::Press {
            return;
        }

        if self.help_popup.is_open() {
            self.handle_help_popup_key(key);
            return;
        }

        if key.code == KeyCode::F(1) {
            self.open_help_popup();
            return;
        }

        if key.code == KeyCode::F(10) {
            self.running = false;
            return;
        }

        if self.confirm.is_some() {
            self.handle_confirm_key(key, tx);
            return;
        }

        if self.theme_picker.is_open() {
            self.handle_theme_picker_key(key, tx);
            return;
        }

        if self.player_picker.is_open() {
            self.handle_player_picker_key(key, tx);
            return;
        }

        if self.video_popup.is_some() {
            self.handle_video_popup_key(key, tx);
            return;
        }

        if self.enhance_form.is_some() {
            self.handle_enhance_form_key(key, tx);
            return;
        }

        if self.channel_editor.is_some() {
            self.handle_channel_editor_key(key, tx);
            return;
        }

        if self.item_menu.is_open() {
            self.handle_item_menu_key(key, tx);
            return;
        }

        match self.screen {
            Screen::Login => self.handle_login_key(key, tx),
            Screen::Workspace => self.handle_workspace_key(key, tx),
        }
    }

    fn handle_help_popup_key(&mut self, key: KeyEvent) {
        let area = terminal_area();
        let Some((_, body_area, _)) = help_popup_layout(area) else {
            return;
        };
        let max_scroll = help_popup_max_scroll(self.help_popup.context(), body_area.height);

        match key.code {
            KeyCode::Esc | KeyCode::F(1) => self.help_popup.close(),
            KeyCode::Up => self.help_popup.scroll_by(-1, max_scroll),
            KeyCode::Down => self.help_popup.scroll_by(1, max_scroll),
            KeyCode::PageUp => self.help_popup.page_by(-1, body_area.height, max_scroll),
            KeyCode::PageDown => self.help_popup.page_by(1, body_area.height, max_scroll),
            KeyCode::Home => self.help_popup.scroll_to_top(),
            KeyCode::End => self.help_popup.scroll_to_bottom(max_scroll),
            _ => {}
        }
    }

    fn handle_confirm_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        match key.code {
            KeyCode::Esc | KeyCode::Char('n') => self.confirm = None,
            KeyCode::Enter | KeyCode::Char('y') => {
                let action = self.confirm.take();
                if let Some(action) = action {
                    self.execute_confirm_action(action, tx);
                }
            }
            _ => {}
        }
    }

    fn handle_theme_picker_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        match key.code {
            KeyCode::Esc => self.restore_theme_preview(tx),
            KeyCode::Up => {
                self.theme_picker.move_selection(-1);
                self.preview_theme(self.theme_picker.selected_theme(), tx);
            }
            KeyCode::Down => {
                self.theme_picker.move_selection(1);
                self.preview_theme(self.theme_picker.selected_theme(), tx);
            }
            KeyCode::Enter => {
                let theme_name = self.theme_picker.selected_theme();
                self.theme_picker.close();
                self.commit_theme(theme_name, tx);
            }
            _ => {}
        }
    }

    fn handle_player_picker_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        let modes = self.available_player_modes();
        match key.code {
            KeyCode::Esc => self.player_picker.close(),
            KeyCode::Up => self.player_picker.move_selection(-1, modes.len()),
            KeyCode::Down => self.player_picker.move_selection(1, modes.len()),
            KeyCode::Enter => {
                let restart_single_frame = self.video_popup.as_ref().map(|popup| popup.paused);
                let player_mode = self.player_picker.selected_mode(&modes);
                self.player_picker.close();
                self.set_player_mode(player_mode);
                if let Some(single_frame) = restart_single_frame {
                    self.restart_video_popup_playback(tx, single_frame);
                }
            }
            _ => {}
        }
    }

    fn handle_video_popup_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        match key.code {
            KeyCode::Esc => {
                self.close_video_popup();
                self.set_status("Closed popup player.", self.theme().accent);
            }
            KeyCode::Char(' ') => self.toggle_video_popup_pause(tx),
            KeyCode::Left => self.seek_video_popup(-VIDEO_PLAYER_SEEK_STEP_SECONDS, tx),
            KeyCode::Right => self.seek_video_popup(VIDEO_PLAYER_SEEK_STEP_SECONDS, tx),
            KeyCode::PageUp => self.seek_video_popup(-30.0, tx),
            KeyCode::PageDown => self.seek_video_popup(30.0, tx),
            KeyCode::Home => self.seek_video_popup_absolute(0.0, tx),
            KeyCode::End => {
                if let Some(popup) = self.video_popup.as_ref() {
                    self.seek_video_popup_absolute(popup.duration_seconds.max(0.0), tx);
                }
            }
            KeyCode::F(5) | KeyCode::Char('v') => self.open_player_picker(),
            KeyCode::F(6) => self.toggle_mouse_enabled(),
            _ => {}
        }
    }

    fn handle_mouse(&mut self, mouse: MouseEvent, tx: &UnboundedSender<AppMessage>) {
        if !self.mouse_enabled {
            return;
        }

        if self.help_popup.is_open() {
            self.handle_help_popup_mouse(mouse);
            return;
        }

        if self.confirm.is_some() {
            self.handle_confirm_mouse(mouse, tx);
            return;
        }

        if self.theme_picker.is_open() {
            self.handle_theme_picker_mouse(mouse, tx);
            return;
        }

        if self.player_picker.is_open() {
            self.handle_player_picker_mouse(mouse, tx);
            return;
        }

        if self.video_popup.is_some() && self.handle_video_popup_mouse(mouse, tx) {
            return;
        }

        if self.enhance_form.is_some() {
            self.handle_enhance_form_mouse(mouse, tx);
            return;
        }

        if self.channel_editor.is_some() {
            self.handle_channel_editor_mouse(mouse, tx);
            return;
        }

        if self.item_menu.is_open() {
            self.handle_item_menu_mouse(mouse, tx);
            return;
        }

        match self.screen {
            Screen::Login => self.handle_login_mouse(mouse, tx),
            Screen::Workspace => self.handle_workspace_mouse(mouse, tx),
        }
    }

    fn handle_help_popup_mouse(&mut self, mouse: MouseEvent) {
        let area = terminal_area();
        let Some((popup, body_area, _)) = help_popup_layout(area) else {
            return;
        };
        let max_scroll = help_popup_max_scroll(self.help_popup.context(), body_area.height);

        match mouse.kind {
            MouseEventKind::Down(MouseButton::Left) => {
                if matches!(
                    self.ui_regions.hit(mouse.column, mouse.row),
                    Some(UiRegion::PopupClose(PopupId::Help))
                ) {
                    self.help_popup.close();
                } else if !rect_contains(popup, mouse.column, mouse.row) {
                    self.help_popup.close();
                }
            }
            MouseEventKind::ScrollDown => self.help_popup.scroll_by(2, max_scroll),
            MouseEventKind::ScrollUp => self.help_popup.scroll_by(-2, max_scroll),
            _ => {}
        }
    }

    fn handle_video_popup_mouse(
        &mut self,
        mouse: MouseEvent,
        tx: &UnboundedSender<AppMessage>,
    ) -> bool {
        let Some(popup) = self.video_popup.as_ref() else {
            return false;
        };
        let terminal = crossterm::terminal::size().unwrap_or((120, 40));
        let area = Rect::new(0, 0, terminal.0, terminal.1);
        let Some((popup_rect, sections)) = video_popup_layout(area) else {
            return true;
        };
        let progress_area = sections[1];

        match mouse.kind {
            MouseEventKind::Down(MouseButton::Left) => {
                let hit = self.ui_regions.hit(mouse.column, mouse.row);
                if matches!(hit, Some(UiRegion::PopupClose(PopupId::VideoPlayer))) {
                    self.close_video_popup();
                    self.set_status("Closed popup player.", self.theme().accent);
                    return true;
                }

                if !rect_contains(popup_rect, mouse.column, mouse.row) {
                    return true;
                }

                if !matches!(hit, Some(UiRegion::VideoSeekBar)) {
                    return true;
                }

                let duration_seconds = popup.duration_seconds.max(0.0);
                let relative = mouse
                    .column
                    .saturating_sub(progress_area.x)
                    .min(progress_area.width.saturating_sub(1));
                let ratio = if progress_area.width <= 1 {
                    0.0
                } else {
                    relative as f64 / progress_area.width.saturating_sub(1) as f64
                };
                self.seek_video_popup_absolute(duration_seconds * ratio, tx);
                true
            }
            _ => true,
        }
    }

    fn handle_confirm_mouse(&mut self, mouse: MouseEvent, tx: &UnboundedSender<AppMessage>) {
        let Some(action) = self.confirm.clone() else {
            return;
        };

        let area = terminal_area();
        let Some((popup, yes_button, no_button)) = confirm_layout(area) else {
            return;
        };

        if matches!(mouse.kind, MouseEventKind::Down(MouseButton::Left)) {
            if matches!(
                self.ui_regions.hit(mouse.column, mouse.row),
                Some(UiRegion::PopupClose(PopupId::Confirm))
            ) {
                self.confirm = None;
            } else if rect_contains(yes_button, mouse.column, mouse.row) {
                self.confirm = None;
                self.execute_confirm_action(action, tx);
            } else if rect_contains(no_button, mouse.column, mouse.row)
                || !rect_contains(popup, mouse.column, mouse.row)
            {
                self.confirm = None;
            }
        }
    }

    fn handle_theme_picker_mouse(&mut self, mouse: MouseEvent, tx: &UnboundedSender<AppMessage>) {
        let area = terminal_area();
        let Some((popup, list_area)) = theme_picker_layout(area) else {
            return;
        };

        if matches!(mouse.kind, MouseEventKind::Down(MouseButton::Left)) {
            if matches!(
                self.ui_regions.hit(mouse.column, mouse.row),
                Some(UiRegion::PopupClose(PopupId::ThemePicker))
            ) {
                self.restore_theme_preview(tx);
                return;
            }
            if !rect_contains(popup, mouse.column, mouse.row) {
                self.restore_theme_preview(tx);
                return;
            }

            if let Some(index) = row_hit_index(
                list_area,
                mouse.column,
                mouse.row,
                1,
                ThemeName::all().len(),
                0,
            ) {
                self.theme_picker.set_selected_index(index);
                let theme_name = self.theme_picker.selected_theme();
                self.theme_picker.close();
                self.commit_theme(theme_name, tx);
            }
        } else if mouse.kind == MouseEventKind::ScrollDown {
            self.theme_picker.move_selection(1);
            self.preview_theme(self.theme_picker.selected_theme(), tx);
        } else if mouse.kind == MouseEventKind::ScrollUp {
            self.theme_picker.move_selection(-1);
            self.preview_theme(self.theme_picker.selected_theme(), tx);
        }
    }

    fn handle_player_picker_mouse(&mut self, mouse: MouseEvent, tx: &UnboundedSender<AppMessage>) {
        let modes = self.available_player_modes();
        let area = terminal_area();
        let Some((popup, list_area)) = player_picker_layout(area) else {
            return;
        };

        if matches!(mouse.kind, MouseEventKind::Down(MouseButton::Left)) {
            if matches!(
                self.ui_regions.hit(mouse.column, mouse.row),
                Some(UiRegion::PopupClose(PopupId::PlayerPicker))
            ) {
                self.player_picker.close();
                return;
            }
            if !rect_contains(popup, mouse.column, mouse.row) {
                self.player_picker.close();
                return;
            }

            if let Some(index) =
                row_hit_index(list_area, mouse.column, mouse.row, 2, modes.len(), 0)
            {
                self.player_picker.set_selected_index(index, modes.len());
                let restart_single_frame = self.video_popup.as_ref().map(|popup| popup.paused);
                let player_mode = self.player_picker.selected_mode(&modes);
                self.player_picker.close();
                self.set_player_mode(player_mode);
                if let Some(single_frame) = restart_single_frame {
                    self.restart_video_popup_playback(tx, single_frame);
                }
            }
        } else if mouse.kind == MouseEventKind::ScrollDown {
            self.player_picker.move_selection(1, modes.len());
        } else if mouse.kind == MouseEventKind::ScrollUp {
            self.player_picker.move_selection(-1, modes.len());
        }
    }

    fn handle_item_menu_mouse(&mut self, mouse: MouseEvent, tx: &UnboundedSender<AppMessage>) {
        let area = terminal_area();
        let Some((popup, list_area)) = item_menu_layout(area) else {
            return;
        };

        if matches!(mouse.kind, MouseEventKind::Down(MouseButton::Left)) {
            if matches!(
                self.ui_regions.hit(mouse.column, mouse.row),
                Some(UiRegion::PopupClose(PopupId::ItemMenu))
            ) {
                self.item_menu.close();
                return;
            }
            if !rect_contains(popup, mouse.column, mouse.row) {
                self.item_menu.close();
                return;
            }

            if let Some(index) = row_hit_index(
                list_area,
                mouse.column,
                mouse.row,
                2,
                self.item_menu.items().len(),
                0,
            ) {
                self.item_menu.set_selected(index);
                let selected = self.item_menu.selected_action();
                let target = self.item_menu.target().cloned();
                if let (Some(action), Some(target)) = (selected, target) {
                    self.dispatch_item_action(action, target, tx);
                }
            }
        } else if mouse.kind == MouseEventKind::ScrollDown {
            self.item_menu.move_selection(1);
        } else if mouse.kind == MouseEventKind::ScrollUp {
            self.item_menu.move_selection(-1);
        }
    }

    fn handle_channel_editor_mouse(&mut self, mouse: MouseEvent, tx: &UnboundedSender<AppMessage>) {
        let area = terminal_area();
        let Some((popup, rows_area)) = channel_editor_layout(area) else {
            return;
        };

        if !matches!(mouse.kind, MouseEventKind::Down(MouseButton::Left)) {
            return;
        }
        if matches!(
            self.ui_regions.hit(mouse.column, mouse.row),
            Some(UiRegion::PopupClose(PopupId::ChannelEditor))
        ) {
            self.channel_editor = None;
            return;
        }
        if !rect_contains(popup, mouse.column, mouse.row) {
            self.channel_editor = None;
            return;
        }

        let Some(index) = row_hit_index(
            rows_area,
            mouse.column,
            mouse.row,
            1,
            ChannelEditorState::fields().len(),
            0,
        ) else {
            return;
        };

        let clipboard = self.clipboard.clone();
        let event = {
            let Some(editor) = self.channel_editor.as_mut() else {
                return;
            };
            let field = ChannelEditorState::fields()[index];
            editor.set_selected(field);
            match field {
                ChannelEditorField::Paused
                | ChannelEditorField::Save
                | ChannelEditorField::Cancel => {
                    editor.handle_key(KeyEvent::from(KeyCode::Enter), &clipboard)
                }
                _ => ChannelEditorEvent::None,
            }
        };
        self.apply_channel_editor_event(event, tx);
    }

    fn handle_enhance_form_mouse(&mut self, mouse: MouseEvent, tx: &UnboundedSender<AppMessage>) {
        let area = terminal_area();
        let Some((popup, rows_area)) = enhance_form_layout(area) else {
            return;
        };

        if !matches!(mouse.kind, MouseEventKind::Down(MouseButton::Left)) {
            return;
        }
        if matches!(
            self.ui_regions.hit(mouse.column, mouse.row),
            Some(UiRegion::PopupClose(PopupId::EnhanceForm))
        ) {
            self.enhance_form = None;
            return;
        }
        if !rect_contains(popup, mouse.column, mouse.row) {
            self.enhance_form = None;
            return;
        }

        let Some(index) = row_hit_index(
            rows_area,
            mouse.column,
            mouse.row,
            1,
            EnhanceFormState::fields().len(),
            0,
        ) else {
            return;
        };

        let clipboard = self.clipboard.clone();
        let event = {
            let Some(form) = self.enhance_form.as_mut() else {
                return;
            };
            let field = EnhanceFormState::fields()[index];
            form.set_selected(field);
            match field {
                EnhanceField::Resolution | EnhanceField::Preset | EnhanceField::Normalize => {
                    form.handle_key(KeyEvent::from(KeyCode::Right), &clipboard)
                }
                EnhanceField::Estimate | EnhanceField::Save | EnhanceField::Cancel => {
                    form.handle_key(KeyEvent::from(KeyCode::Enter), &clipboard)
                }
                _ => EnhanceFormEvent::None,
            }
        };
        self.apply_enhance_form_event(event, tx);
    }

    fn handle_login_mouse(&mut self, mouse: MouseEvent, tx: &UnboundedSender<AppMessage>) {
        let area = terminal_area();
        let (login_area, _) = login_layout(area);

        match mouse.kind {
            MouseEventKind::ScrollDown => self.login_field = self.login_field.next(),
            MouseEventKind::ScrollUp => self.login_field = self.login_field.previous(),
            MouseEventKind::Down(MouseButton::Left) => {
                if !rect_contains(login_area, mouse.column, mouse.row) {
                    return;
                }
                match self.ui_regions.hit(mouse.column, mouse.row) {
                    Some(UiRegion::LoginField(field)) => self.login_field = field,
                    Some(UiRegion::LoginAction(action)) => match action {
                        LoginMouseAction::Submit => self.submit_login(tx),
                        LoginMouseAction::ToggleMode => {
                            self.login_mode = self.login_mode.toggle();
                            self.set_status(
                                format!("{} mode selected.", self.login_mode.label()),
                                self.theme().accent,
                            );
                        }
                        LoginMouseAction::Mouse => self.toggle_mouse_enabled(),
                        LoginMouseAction::Quit => self.running = false,
                    },
                    _ => {}
                }
            }
            _ => {}
        }
    }

    fn handle_workspace_mouse(&mut self, mouse: MouseEvent, tx: &UnboundedSender<AppMessage>) {
        if self.channel_popup.is_open() {
            self.handle_channel_popup_mouse(mouse, tx);
            return;
        }

        let area = terminal_area();
        let (vertical, _) = workspace_layout(area, self);

        match mouse.kind {
            MouseEventKind::ScrollDown => self.move_selection(1),
            MouseEventKind::ScrollUp => self.move_selection(-1),
            MouseEventKind::Down(MouseButton::Left) => {
                match self.ui_regions.hit(mouse.column, mouse.row) {
                    Some(UiRegion::WorkspaceHeader(action)) => {
                        match action {
                            WorkspaceHeaderAction::AddStream => self.open_create_stream_editor(),
                            WorkspaceHeaderAction::Logout => {
                                self.confirm = Some(ConfirmAction::Logout)
                            }
                            WorkspaceHeaderAction::Recorder => {
                                self.confirm = Some(ConfirmAction::ToggleRecorder)
                            }
                        }
                        return;
                    }
                    Some(UiRegion::PrimaryTab(view)) => {
                        self.set_view(view, tx);
                        return;
                    }
                    Some(UiRegion::StreamTab(tab)) => {
                        self.stream_tab = tab;
                        self.selected_channel = clamp_index(
                            self.selected_channel,
                            self.visible_stream_channels().len(),
                        );
                        return;
                    }
                    _ => {}
                }

                if !rect_contains(vertical[3], mouse.column, mouse.row) {
                    return;
                }

                self.handle_workspace_content_click(vertical[3], mouse.column, mouse.row, tx);
            }
            _ => {}
        }
    }

    fn handle_channel_popup_mouse(&mut self, mouse: MouseEvent, tx: &UnboundedSender<AppMessage>) {
        let area = terminal_area();
        let vertical = workspace_layout(area, self).0;
        let Some((popup, rows_area)) = channel_popup_recordings_layout(vertical[3]) else {
            return;
        };

        match mouse.kind {
            MouseEventKind::ScrollDown => self.channel_popup.move_selection(1),
            MouseEventKind::ScrollUp => self.channel_popup.move_selection(-1),
            MouseEventKind::Down(MouseButton::Left) => {
                if matches!(
                    self.ui_regions.hit(mouse.column, mouse.row),
                    Some(UiRegion::PopupClose(PopupId::ChannelPopup))
                ) {
                    self.channel_popup.close();
                    return;
                }
                if !rect_contains(popup, mouse.column, mouse.row) {
                    self.channel_popup.close();
                    return;
                }

                let recordings = self.channel_recordings();
                if let Some(index) = row_hit_index(
                    rows_area,
                    mouse.column,
                    mouse.row,
                    MEDIA_ROW_HEIGHT,
                    recordings.len(),
                    self.channel_popup.selected_recording(),
                ) {
                    let was_selected = index == self.channel_popup.selected_recording();
                    self.channel_popup.set_selected_recording(index);
                    if was_selected {
                        self.play_selected_video(tx);
                    }
                    self.prefetch_thumbnails(tx);
                }
            }
            _ => {}
        }
    }

    fn handle_workspace_content_click(
        &mut self,
        area: Rect,
        column: u16,
        row: u16,
        tx: &UnboundedSender<AppMessage>,
    ) {
        if let Some(index) = main_content_hit_index(self, area, column, row) {
            let was_selected = index == self.current_selection();
            self.set_selection(index);
            match self.view {
                View::Streams | View::Channels => {
                    if was_selected {
                        self.open_selected_channel(tx);
                    }
                }
                View::Latest | View::Random | View::Favourites => {
                    if was_selected {
                        self.play_selected_video(tx);
                    }
                }
                View::Similarity | View::Processes | View::Jobs | View::Logs | View::Channel => {}
                View::Admin | View::Info | View::Monitoring => {}
            }
            self.prefetch_thumbnails(tx);
        }
    }

    fn handle_item_menu_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        match key.code {
            KeyCode::Esc => self.item_menu.close(),
            KeyCode::Up => self.item_menu.move_selection(-1),
            KeyCode::Down => self.item_menu.move_selection(1),
            KeyCode::Enter => {
                let selected = self.item_menu.selected_action();
                let target = self.item_menu.target().cloned();
                if let (Some(action), Some(target)) = (selected, target) {
                    self.dispatch_item_action(action, target, tx);
                }
            }
            _ => {}
        }
    }

    fn handle_channel_editor_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        let clipboard = self.clipboard.clone();
        let event = {
            let Some(editor) = self.channel_editor.as_mut() else {
                return;
            };
            editor.handle_key(key, &clipboard)
        };
        self.apply_channel_editor_event(event, tx);
    }

    fn handle_enhance_form_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        let clipboard = self.clipboard.clone();
        let event = {
            let Some(form) = self.enhance_form.as_mut() else {
                return;
            };
            form.handle_key(key, &clipboard)
        };
        self.apply_enhance_form_event(event, tx);
    }

    fn apply_channel_editor_event(
        &mut self,
        event: ChannelEditorEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        match event {
            ChannelEditorEvent::Close => self.channel_editor = None,
            ChannelEditorEvent::Copied(text) => {
                self.apply_text_input_action(TextInputAction::Copied(text))
            }
            ChannelEditorEvent::None => {}
            ChannelEditorEvent::Submit(ChannelEditorSubmit {
                channel_id,
                request,
            }) => {
                let Some(session) = self.session.clone() else {
                    return;
                };
                let channel_name = request.channel_name.clone();
                let sender = tx.clone();
                self.set_status(
                    if channel_id.is_some() {
                        format!("Saving {channel_name}…")
                    } else {
                        format!("Creating {channel_name}…")
                    },
                    self.theme().accent,
                );
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        if let Some(channel_id) = channel_id {
                            client.update_channel(channel_id, &request).await?;
                        } else {
                            client.create_channel(&request).await?;
                        }
                        Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                            message: if channel_id.is_some() {
                                format!("Saved {channel_name}")
                            } else {
                                format!("Created {channel_name}")
                            },
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
                self.channel_editor = None;
            }
        }
    }

    fn apply_enhance_form_event(
        &mut self,
        event: EnhanceFormEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        match event {
            EnhanceFormEvent::Close => self.enhance_form = None,
            EnhanceFormEvent::Copied(text) => {
                self.apply_text_input_action(TextInputAction::Copied(text))
            }
            EnhanceFormEvent::Estimate(body) => self.request_enhancement_estimate(body, tx),
            EnhanceFormEvent::None => {}
            EnhanceFormEvent::Submit(body) => {
                let Some(session) = self.session.clone() else {
                    return;
                };
                let recording_id = body.recording_id;
                let sender = tx.clone();
                self.set_status(
                    format!("Queueing enhancement for #{recording_id}…"),
                    self.theme().accent,
                );
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        client.enhance_video(recording_id, &body).await?;
                        Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                            message: format!("Queued enhancement for #{recording_id}"),
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
                self.enhance_form = None;
            }
        }
    }

    fn handle_login_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        if self.auto_login_pending {
            if matches!(key.code, KeyCode::Esc) {
                self.auto_login_pending = false;
                self.session = None;
                self.set_status("Auto-login canceled.", self.theme().warning);
            }
            return;
        }

        match key.code {
            KeyCode::Char('q') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                self.running = false;
            }
            KeyCode::Char('c') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                let clipboard = self.clipboard.clone();
                let selection_action = self.selected_login_input_mut().handle_key(key, &clipboard);
                self.apply_text_input_action(selection_action);
            }
            KeyCode::F(2) => {
                self.login_mode = self.login_mode.toggle();
                self.set_status(
                    format!("{} mode selected.", self.login_mode.label()),
                    self.theme().accent,
                );
            }
            KeyCode::F(3) => self.open_theme_picker(),
            KeyCode::F(5) | KeyCode::Char('v') => self.open_player_picker(),
            KeyCode::F(6) => self.toggle_mouse_enabled(),
            KeyCode::Tab | KeyCode::Down => self.login_field = self.login_field.next(),
            KeyCode::BackTab | KeyCode::Up => self.login_field = self.login_field.previous(),
            KeyCode::Enter => self.submit_login(tx),
            _ => {
                let clipboard = self.clipboard.clone();
                let action = self.selected_login_input_mut().handle_key(key, &clipboard);
                self.apply_text_input_action(action);
            }
        }
    }

    fn submit_login(&mut self, tx: &UnboundedSender<AppMessage>) {
        if self.login_server.text().trim().is_empty() {
            self.login_error = Some("Server URL is required.".to_string());
            return;
        }
        if self.login_username.text().trim().is_empty() {
            self.login_error = Some("Login is required.".to_string());
            return;
        }
        if self.login_password.text().is_empty() {
            self.login_error = Some("Password is required.".to_string());
            return;
        }

        self.login_error = None;
        self.auto_login_pending = true;
        self.set_status(
            format!("{} in progress…", self.login_mode.label()),
            self.theme().accent,
        );

        let sender = tx.clone();
        let base_url = self.login_server.text().trim().to_string();
        let username = self.login_username.text().trim().to_string();
        let password = self.login_password.text().to_string();
        let signup = self.login_mode == AuthMode::Signup;

        tokio::spawn(async move {
            let result = async move {
                let normalized = crate::config::normalize_server_url(&base_url)?;
                let AuthSuccess { runtime, token } =
                    ApiClient::authenticate(&normalized, None, &username, &password, signup)
                        .await?;
                let warning = save_authenticated_session(
                    &normalized,
                    &username,
                    &token,
                    Some(runtime.api_version.clone()),
                    Some(runtime.file_url.clone()),
                )?;

                Ok::<_, anyhow::Error>((normalized, runtime, token, username, warning))
            }
            .await;

            match result {
                Ok((base_url, runtime, token, username, warning)) => {
                    let _ = sender.send(AppMessage::AuthSucceeded {
                        base_url,
                        runtime,
                        token,
                        username,
                        warning,
                    });
                }
                Err(error) => {
                    let _ = sender.send(AppMessage::AuthFailed(error.to_string()));
                }
            }
        });
    }

    fn handle_workspace_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        if self.channel_popup.is_open() {
            match key.code {
                KeyCode::Char('c') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                    self.running = false
                }
                KeyCode::Char('q') => self.running = false,
                KeyCode::Esc | KeyCode::Backspace => self.channel_popup.close(),
                KeyCode::Enter => self.play_selected_video(tx),
                KeyCode::F(3) => self.open_theme_picker(),
                KeyCode::F(5) | KeyCode::Char('v') => self.open_player_picker(),
                KeyCode::F(6) => self.toggle_mouse_enabled(),
                KeyCode::F(4) => self.open_item_actions(),
                KeyCode::Char('n') => self.open_create_stream_editor(),
                KeyCode::Up => self.channel_popup.move_selection(-1),
                KeyCode::Down => self.channel_popup.move_selection(1),
                KeyCode::Char('g') => self.request_channel_popup(tx),
                _ => {}
            }

            self.prefetch_thumbnails(tx);
            return;
        }

        match key.code {
            KeyCode::Char('c') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                self.running = false
            }
            KeyCode::Char('q') => self.running = false,
            KeyCode::Left => {
                let items = View::primary();
                let index = items
                    .iter()
                    .position(|item| *item == self.primary_view)
                    .unwrap_or(0);
                let next = items[(index + items.len() - 1) % items.len()];
                self.set_view(next, tx);
            }
            KeyCode::Right => {
                let items = View::primary();
                let index = items
                    .iter()
                    .position(|item| *item == self.primary_view)
                    .unwrap_or(0);
                let next = items[(index + 1) % items.len()];
                self.set_view(next, tx);
            }
            KeyCode::Up => self.move_selection(-1),
            KeyCode::Down => self.move_selection(1),
            KeyCode::PageDown => {
                if matches!(self.view, View::Latest) {
                    self.video_page.skip = self
                        .video_page
                        .skip
                        .saturating_add(self.video_page.take.max(self.latest_filter.limit()));
                    self.request_refresh(tx);
                }
            }
            KeyCode::PageUp => {
                if matches!(self.view, View::Latest) {
                    self.video_page.skip = self
                        .video_page
                        .skip
                        .saturating_sub(self.video_page.take.max(self.latest_filter.limit()));
                    self.request_refresh(tx);
                }
            }
            KeyCode::Char('g') => {
                self.request_refresh(tx);
                self.request_view_data(tx);
            }
            KeyCode::Char('a') => self.set_view(View::Admin, tx),
            KeyCode::Char('i') => self.set_view(View::Info, tx),
            KeyCode::Char('j') => {
                self.set_view(View::Jobs, tx);
            }
            KeyCode::Char('l') => self.confirm = Some(ConfirmAction::Logout),
            KeyCode::Char('k') => self.set_view(View::Logs, tx),
            KeyCode::Char('m') => self.set_view(View::Monitoring, tx),
            KeyCode::Char('n') => self.open_create_stream_editor(),
            KeyCode::Char('p') => self.set_view(View::Processes, tx),
            KeyCode::Char('r') => self.confirm = Some(ConfirmAction::ToggleRecorder),
            KeyCode::Char('t') => self.open_theme_picker(),
            KeyCode::F(3) => self.open_theme_picker(),
            KeyCode::Char('v') | KeyCode::F(5) => self.open_player_picker(),
            KeyCode::F(6) => self.toggle_mouse_enabled(),
            KeyCode::F(4) => self.open_item_actions(),
            KeyCode::Char('1') => self.set_view(View::Streams, tx),
            KeyCode::Char('2') => self.set_view(View::Channels, tx),
            KeyCode::Char('3') => self.set_view(View::Latest, tx),
            KeyCode::Char('4') => self.set_view(View::Random, tx),
            KeyCode::Char('5') => self.set_view(View::Favourites, tx),
            KeyCode::Char('6') => self.set_view(View::Similarity, tx),
            KeyCode::Enter if matches!(self.view, View::Streams | View::Channels) => {
                self.open_selected_channel(tx);
            }
            KeyCode::Enter
                if matches!(self.view, View::Latest | View::Random | View::Favourites) =>
            {
                self.play_selected_video(tx);
            }
            KeyCode::Char('[') => match self.view {
                View::Streams => {
                    self.stream_tab = self.stream_tab.previous();
                    self.selected_channel =
                        clamp_index(self.selected_channel, self.visible_stream_channels().len());
                }
                View::Latest => {
                    self.latest_filter.previous_column();
                    self.video_page.skip = 0;
                    self.video_page.take = self.latest_filter.limit();
                    self.request_refresh(tx);
                }
                View::Similarity => {
                    self.similarity_tab = self.similarity_tab.previous();
                    self.selected_similarity = clamp_index(
                        self.selected_similarity,
                        self.similarity_groups
                            .as_ref()
                            .map(|groups| groups.groups.len())
                            .unwrap_or(0),
                    );
                }
                _ => {}
            },
            KeyCode::Char(']') => match self.view {
                View::Streams => {
                    self.stream_tab = self.stream_tab.next();
                    self.selected_channel =
                        clamp_index(self.selected_channel, self.visible_stream_channels().len());
                }
                View::Latest => {
                    self.latest_filter.next_column();
                    self.video_page.skip = 0;
                    self.video_page.take = self.latest_filter.limit();
                    self.request_refresh(tx);
                }
                View::Similarity => {
                    self.similarity_tab = self.similarity_tab.next();
                    self.selected_similarity = clamp_index(
                        self.selected_similarity,
                        self.similarity_groups
                            .as_ref()
                            .map(|groups| groups.groups.len())
                            .unwrap_or(0),
                    );
                }
                _ => {}
            },
            KeyCode::Char('o') if matches!(self.view, View::Latest) => {
                self.latest_filter.toggle_order();
                self.video_page.skip = 0;
                self.video_page.take = self.latest_filter.limit();
                self.request_refresh(tx);
            }
            KeyCode::Char('-') => match self.view {
                View::Latest => {
                    self.latest_filter.previous_limit();
                    self.video_page.skip = 0;
                    self.video_page.take = self.latest_filter.limit();
                    self.request_refresh(tx);
                }
                View::Random => {
                    self.random_filter.previous_limit();
                    self.request_view_data(tx);
                }
                _ => {}
            },
            KeyCode::Char('+') | KeyCode::Char('=') => match self.view {
                View::Latest => {
                    self.latest_filter.next_limit();
                    self.video_page.skip = 0;
                    self.video_page.take = self.latest_filter.limit();
                    self.request_refresh(tx);
                }
                View::Random => {
                    self.random_filter.next_limit();
                    self.request_view_data(tx);
                }
                _ => {}
            },
            KeyCode::Char('0') => match self.view {
                View::Latest => {
                    self.latest_filter.reset();
                    self.video_page.skip = 0;
                    self.video_page.take = self.latest_filter.limit();
                    self.request_refresh(tx);
                }
                View::Random => {
                    self.random_filter.reset();
                    self.request_view_data(tx);
                }
                _ => {}
            },
            _ => {}
        }

        self.prefetch_thumbnails(tx);
    }

    fn toggle_recorder(&mut self, tx: &UnboundedSender<AppMessage>) {
        let Some(session) = self.session.clone() else {
            return;
        };

        self.set_status(
            if self.recorder.is_recording {
                "Stopping recorder…"
            } else {
                "Starting recorder…"
            },
            self.theme().accent,
        );
        let currently_recording = self.recorder.is_recording;
        let sender = tx.clone();
        tokio::spawn(async move {
            let result = async move {
                let client = session.client()?;
                if currently_recording {
                    client.stop_recorder().await?;
                    Ok::<bool, anyhow::Error>(false)
                } else {
                    client.start_recorder().await?;
                    Ok::<bool, anyhow::Error>(true)
                }
            }
            .await
            .map_err(|error| error.to_string());

            let _ = sender.send(AppMessage::RecorderToggled(result));
        });
    }

    fn logout(&mut self, tx: &UnboundedSender<AppMessage>) {
        let Some(session) = self.session.clone() else {
            self.return_to_login(
                "Logged out. Enter another server or account to continue.",
                Color::Green,
            );
            return;
        };
        let sender = tx.clone();
        self.set_status("Logging out…", self.theme().accent);
        tokio::spawn(async move {
            let result = async move {
                let client = session.client()?;
                let _ = client.logout().await;
                clear_saved_session(&session.base_url)?;
                Ok::<(), anyhow::Error>(())
            }
            .await
            .map_err(|error| error.to_string());
            let _ = sender.send(AppMessage::LoggedOut(result));
        });
    }

    fn handle_message(&mut self, message: AppMessage, tx: &UnboundedSender<AppMessage>) {
        match message {
            AppMessage::ActionFinished(result) => match result {
                Ok(feedback) => {
                    self.set_status(feedback.message.clone(), feedback.tone);
                    self.refresh_after_action(tx, &feedback);
                }
                Err(error) => self.set_status(error, Color::Red),
            },
            AppMessage::AuthFailed(error) => {
                self.auto_login_pending = false;
                self.login_error = Some(error.clone());
                self.session = None;
                self.set_status(error, Color::Red);
            }
            AppMessage::AuthSucceeded {
                base_url,
                runtime,
                token,
                username,
                warning,
            } => {
                self.switch_to_workspace(base_url, runtime, token, username, warning, tx);
                self.request_refresh(tx);
                self.request_view_data(tx);
            }
            AppMessage::WorkspaceRefreshed(result) => {
                self.refresh_in_flight = false;
                match result {
                    Ok(snapshot) => {
                        self.content_error = None;
                        self.disk = snapshot.disk;
                        self.server_info = snapshot.version;
                        self.recorder = snapshot.recorder;
                        self.channels = sort_channels(snapshot.channels);
                        self.video_page = snapshot.videos;
                        self.jobs = snapshot.jobs;
                        self.job_worker_active = snapshot.job_worker.is_processing;
                        self.clamp_selection();
                        self.prefetch_thumbnails(tx);
                    }
                    Err(error) => {
                        if error.contains("401") {
                            if let Some(session) = &self.session {
                                let _ = clear_saved_session(&session.base_url);
                            }
                            self.return_to_login(
                                "Saved session expired. Sign in again.",
                                Color::Yellow,
                            );
                        } else {
                            self.content_error = Some(error.clone());
                            self.set_status(error, Color::Red);
                        }
                    }
                }
            }
            AppMessage::SocketEvent(event) => {
                self.events.insert(0, event.clone());
                self.events.truncate(120);
                self.selected_event = 0;
                self.socket_message = event.summary.clone();
                self.set_status(event.summary, self.theme().accent);
                self.refresh_pending = true;
            }
            AppMessage::SocketStatus { status, message } => {
                self.socket_status = status;
                self.socket_message = message.unwrap_or_default();
            }
            AppMessage::ThumbnailLoaded {
                preview,
                row,
                target,
            } => {
                self.thumbnail_cache
                    .insert(target.key.clone(), ThumbnailEntry::Ready { preview, row });
            }
            AppMessage::ThumbnailFailed { error, target } => {
                self.set_status(
                    format!(
                        "Thumbnail error for {}: {}",
                        truncate(&target.label, 24),
                        truncate(&error, 72)
                    ),
                    Color::Yellow,
                );
                if target.key.starts_with("channel:") {
                    let theme = self.theme();
                    self.cache_placeholder_thumbnail(
                        target.key,
                        target.label,
                        theme.warning,
                        theme.surface_alt_bg,
                    );
                } else {
                    self.thumbnail_cache
                        .insert(target.key.clone(), ThumbnailEntry::Failed { error });
                }
            }
            AppMessage::RandomVideosLoaded(result) => {
                self.view_request_in_flight = false;
                match result {
                    Ok(videos) => {
                        self.content_error = None;
                        self.random_videos = videos;
                        self.selected_video =
                            clamp_index(self.selected_video, self.random_videos.len());
                        self.prefetch_thumbnails(tx);
                    }
                    Err(error) => {
                        self.content_error = Some(error.clone());
                        self.set_status(error, Color::Red);
                    }
                }
            }
            AppMessage::BookmarkVideosLoaded(result) => {
                self.view_request_in_flight = false;
                match result {
                    Ok(videos) => {
                        self.content_error = None;
                        self.bookmark_videos = videos;
                        self.selected_video =
                            clamp_index(self.selected_video, self.bookmark_videos.len());
                        self.prefetch_thumbnails(tx);
                    }
                    Err(error) => {
                        self.content_error = Some(error.clone());
                        self.set_status(error, Color::Red);
                    }
                }
            }
            AppMessage::SimilarityLoaded(result) => {
                self.view_request_in_flight = false;
                match result {
                    Ok(groups) => {
                        self.content_error = None;
                        self.similarity_groups = Some(groups);
                        self.selected_similarity = clamp_index(
                            self.selected_similarity,
                            self.similarity_groups
                                .as_ref()
                                .map(|group| group.groups.len())
                                .unwrap_or(0),
                        );
                        self.prefetch_thumbnails(tx);
                    }
                    Err(error) => {
                        self.content_error = Some(error.clone());
                        self.set_status(error, Color::Red);
                    }
                }
            }
            AppMessage::AdminLoaded(result) => {
                self.view_request_in_flight = false;
                match result {
                    Ok(status) => {
                        self.content_error = None;
                        self.admin_status = status;
                    }
                    Err(error) => {
                        self.content_error = Some(error.clone());
                        self.set_status(error, Color::Red);
                    }
                }
            }
            AppMessage::ChannelLoaded { channel_id, result } => {
                if self.channel_popup.apply_result(channel_id, result) {
                    self.prefetch_thumbnails(tx);
                } else if let Some(error) = self.channel_popup.error() {
                    self.set_status(error.to_string(), Color::Red);
                }
            }
            AppMessage::EnhanceDescriptionsLoaded(result) => match result {
                Ok(descriptions) => {
                    if let Some(form) = self.enhance_form.as_mut() {
                        form.apply_descriptions(descriptions);
                    }
                }
                Err(error) => {
                    if let Some(form) = self.enhance_form.as_mut() {
                        form.set_error(error.clone());
                    }
                    self.set_status(error, Color::Red);
                }
            },
            AppMessage::EnhancementEstimated(result) => match result {
                Ok(estimated_size) => {
                    if let Some(form) = self.enhance_form.as_mut() {
                        form.apply_estimate(estimated_size);
                    }
                    self.set_status(
                        format!("Estimated file size: {}", format_bytes(estimated_size)),
                        self.theme().accent,
                    );
                }
                Err(error) => {
                    if let Some(form) = self.enhance_form.as_mut() {
                        form.set_error(error.clone());
                    }
                    self.set_status(error, Color::Red);
                }
            },
            AppMessage::ProcessesLoaded(result) => {
                self.view_request_in_flight = false;
                match result {
                    Ok(processes) => {
                        self.content_error = None;
                        self.processes = processes;
                        self.selected_process =
                            clamp_index(self.selected_process, self.processes.len());
                    }
                    Err(error) => {
                        self.content_error = Some(error.clone());
                        self.set_status(error, Color::Red);
                    }
                }
            }
            AppMessage::SystemInfoLoaded(result) => {
                self.view_request_in_flight = false;
                match result {
                    Ok(system_info) => {
                        self.content_error = None;
                        self.disk = system_info.disk_info.clone();
                        self.monitor_history.push(MetricSample {
                            cpu_load_percent: average_cpu_percent(&system_info),
                            rx_megabytes: system_info.net_info.receive_bytes / 1024 / 1024,
                            timestamp: Local::now().format("%H:%M:%S").to_string(),
                            tx_megabytes: system_info.net_info.transmit_bytes / 1024 / 1024,
                        });
                        if self.monitor_history.len() > 60 {
                            self.monitor_history.remove(0);
                        }
                        self.system_info = Some(system_info);
                    }
                    Err(error) => {
                        self.content_error = Some(error.clone());
                        self.set_status(error, Color::Red);
                    }
                }
            }
            AppMessage::RecorderToggled(result) => match result {
                Ok(is_recording) => {
                    self.recorder.is_recording = is_recording;
                    self.set_status(
                        if is_recording {
                            "Recorder started."
                        } else {
                            "Recorder stopped."
                        },
                        Color::Green,
                    );
                    self.refresh_pending = true;
                }
                Err(error) => self.set_status(error, Color::Red),
            },
            AppMessage::LoggedOut(result) => match result {
                Ok(()) => self.return_to_login(
                    "Logged out. Enter another server or account to continue.",
                    Color::Green,
                ),
                Err(error) => self.set_status(error, Color::Red),
            },
            AppMessage::VideoPlayer(event) => match event {
                VideoPlayerEvent::Frame {
                    generation,
                    frame,
                    position_seconds,
                } => {
                    if let Some(popup) = self.video_popup.as_mut() {
                        if popup.generation == generation {
                            popup.frame = Some(frame);
                            popup.position_seconds =
                                position_seconds.clamp(0.0, popup.duration_seconds.max(0.0));
                            popup.loading = false;
                            popup.error = None;
                        }
                    }
                }
                VideoPlayerEvent::Ended {
                    generation,
                    position_seconds,
                } => {
                    let mut finished_label = None;
                    if let Some(popup) = self.video_popup.as_mut() {
                        if popup.generation == generation {
                            popup.loading = false;
                            popup.paused = true;
                            popup.position_seconds =
                                position_seconds.clamp(0.0, popup.duration_seconds.max(0.0));
                            finished_label = Some(popup.label.clone());
                        }
                    }
                    if let Some(label) = finished_label {
                        self.video_playback_worker = None;
                        self.set_status(format!("Playback finished: {label}"), self.theme().accent);
                    }
                }
                VideoPlayerEvent::Error { generation, error } => {
                    let mut should_report = false;
                    if let Some(popup) = self.video_popup.as_mut() {
                        if popup.generation == generation {
                            popup.loading = false;
                            popup.error = Some(error.clone());
                            popup.paused = true;
                            should_report = true;
                        }
                    }
                    if should_report {
                        self.video_playback_worker = None;
                        self.set_status(format!("Playback failed: {error}"), Color::Red);
                    }
                }
            },
        }
    }
}

fn build_file_url(root: &str, relative: &str) -> Option<String> {
    if root.trim().is_empty() || relative.trim().is_empty() {
        return None;
    }

    let mut base = root.to_string();
    if !base.ends_with('/') {
        base.push('/');
    }

    Url::parse(&base)
        .ok()?
        .join(relative.trim_start_matches('/'))
        .ok()
        .map(|url| url.to_string())
}

fn video_thumbnail_relative_path(video: &Recording) -> Option<String> {
    if let Some(preview) = &video.video_preview {
        let path = preview.preview_path.trim_end_matches('/');
        if path.is_empty() {
            return None;
        }
        if path.ends_with(".jpg") || path.ends_with(".jpeg") || path.ends_with(".png") {
            return Some(path.to_string());
        }
        return Some(format!("{path}/0.jpg"));
    }

    if video.channel_name.trim().is_empty() {
        None
    } else {
        Some(format!("{}/.previews/live.jpg", video.channel_name))
    }
}

fn summarize_event(name: &str, data: &Value) -> String {
    if let Some(number) = data.as_u64() {
        return format!("{name} #{number}");
    }
    if let Some(text) = data.as_str() {
        return format!("{name} {text}");
    }
    if let Some(filename) = data.get("filename").and_then(Value::as_str) {
        return format!("{name} {filename}");
    }
    if let Some(job) = data.get("job") {
        let filename = job.get("filename").and_then(Value::as_str).unwrap_or("job");
        let task = job.get("task").and_then(Value::as_str).unwrap_or("");
        return format!("{name} {task} {filename}").trim().to_string();
    }
    format!("{name} {data}").chars().take(120).collect()
}

async fn websocket_loop(runtime: RuntimeConfig, token: String, tx: UnboundedSender<AppMessage>) {
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

fn init_terminal() -> Result<DefaultTerminal> {
    let terminal = ratatui::init();
    crossterm::execute!(stdout(), EnableMouseCapture)
        .context("failed to enable terminal mouse capture")?;
    Ok(terminal)
}

fn restore_terminal(terminal: &mut DefaultTerminal) -> Result<()> {
    crossterm::execute!(stdout(), DisableMouseCapture)
        .context("failed to disable terminal mouse capture")?;
    ratatui::restore();
    terminal
        .show_cursor()
        .context("failed to show terminal cursor")
}

fn draw(frame: &mut Frame, app: &mut App) {
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

fn video_popup_layout(area: Rect) -> Option<(Rect, [Rect; 4])> {
    let popup_height = area.height.saturating_sub(4).clamp(16, 28);
    let popup = centered_rect(82, popup_height, area);
    if popup.width < 4 || popup.height < 4 {
        return None;
    }

    let inner = Rect {
        x: popup.x.saturating_add(1),
        y: popup.y.saturating_add(1),
        width: popup.width.saturating_sub(2),
        height: popup.height.saturating_sub(2),
    };
    if inner.width < 24 || inner.height < 8 {
        return None;
    }

    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Min(8),
            Constraint::Length(1),
            Constraint::Length(2),
            Constraint::Length(1),
        ])
        .split(inner);

    Some((popup, [sections[0], sections[1], sections[2], sections[3]]))
}

fn terminal_area() -> Rect {
    let (width, height) = crossterm::terminal::size().unwrap_or((120, 40));
    Rect::new(0, 0, width, height)
}

fn rect_contains(area: Rect, column: u16, row: u16) -> bool {
    column >= area.x
        && column < area.x.saturating_add(area.width)
        && row >= area.y
        && row < area.y.saturating_add(area.height)
}

fn should_clear_saved_session_on_auth_error(error: &str) -> bool {
    let normalized = error.to_ascii_lowercase();
    normalized.contains("401")
        || normalized.contains("403")
        || normalized.contains("unauthorized")
        || normalized.contains("forbidden")
        || normalized.contains("invalid token")
        || normalized.contains("token expired")
        || normalized.contains("jwt")
}

fn persist_session_on_exit(app: &App) -> anyhow::Result<Option<String>> {
    let Some(session) = app.session.as_ref() else {
        return Ok(None);
    };

    save_authenticated_session(
        &session.base_url,
        &session.username,
        &session.token,
        (!session.runtime.api_version.trim().is_empty()).then(|| session.runtime.api_version.clone()),
        (!session.runtime.file_url.trim().is_empty()).then(|| session.runtime.file_url.clone()),
    )
}

fn login_layout(area: Rect) -> (Rect, [Rect; 6]) {
    let compact = area.width < 72 || area.height < 18;
    let login_area = centered_rect(if compact { 96 } else { 70 }, 20, area);
    let inner = Rect {
        x: login_area.x.saturating_add(1),
        y: login_area.y.saturating_add(1),
        width: login_area.width.saturating_sub(2),
        height: login_area.height.saturating_sub(2),
    };
    let margin: u16 = if inner.width >= 48 && inner.height >= 14 {
        1
    } else {
        0
    };
    let content_height = inner.height.saturating_sub(margin.saturating_mul(2));
    let spacer_height = if content_height >= 12 { 1 } else { 0 };
    let info_height = content_height.saturating_sub(9 + spacer_height + 1).min(2);
    let rows = Layout::default()
        .direction(Direction::Vertical)
        .margin(margin)
        .constraints([
            Constraint::Length(spacer_height),
            Constraint::Length(3),
            Constraint::Length(3),
            Constraint::Length(3),
            Constraint::Length(info_height),
            Constraint::Length(1),
        ])
        .split(inner);

    (
        login_area,
        [rows[0], rows[1], rows[2], rows[3], rows[4], rows[5]],
    )
}

fn centered_action_rects<const N: usize>(area: Rect, labels: &[&str; N]) -> [Rect; N] {
    let slots = (N.saturating_sub(1)) as u16;
    let labels_width = labels
        .iter()
        .fold(0u16, |width, label| width.saturating_add(label.chars().count() as u16));
    let gap = if slots == 0 {
        0
    } else {
        area.width
            .saturating_sub(labels_width)
            .checked_div(slots)
            .unwrap_or(0)
            .min(2)
    };
    let total_width = labels_width.saturating_add(gap.saturating_mul(slots));
    let mut x = area
        .x
        .saturating_add(area.width.saturating_sub(total_width) / 2);
    std::array::from_fn(|index| {
        let width = labels[index].chars().count() as u16;
        let rect = Rect {
            x,
            y: area.y,
            width,
            height: area.height,
        };
        x = x.saturating_add(width).saturating_add(gap);
        rect
    })
}

fn login_action_labels(width: u16, mouse_enabled: bool) -> [&'static str; 4] {
    if width >= 50 {
        [
            "[Enter] Submit",
            if mouse_enabled {
                "[F6] Mouse On"
            } else {
                "[F6] Mouse Off"
            },
            "[F2] Mode",
            "[F10] Quit",
        ]
    } else if width >= 38 {
        ["[Enter]", "[F6] Mouse", "[F2] Mode", "[F10] Quit"]
    } else {
        ["Enter", "F6", "F2", "F10"]
    }
}

fn workspace_layout(area: Rect, app: &App) -> ([Rect; 5], [Rect; 4]) {
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

    (
        [
            vertical[0],
            vertical[1],
            vertical[2],
            vertical[3],
            vertical[4],
        ],
        [header[0], header[1], header[3], header[5]],
    )
}

fn primary_tab_regions(area: Rect) -> Vec<(Rect, View)> {
    let mut x = area.x;
    let mut regions = Vec::with_capacity(View::primary().len());
    for (index, view) in View::primary().iter().copied().enumerate() {
        let label = format!(" {}:{} ", index + 1, view.label());
        let width = label.chars().count() as u16;
        let rect = Rect {
            x,
            y: area.y,
            width: width.min(area.x.saturating_add(area.width).saturating_sub(x)),
            height: area.height,
        };
        regions.push((rect, view));
        x = x.saturating_add(width);
    }
    regions
}

fn main_panel_left_area(area: Rect, view: View) -> Rect {
    if view_has_thumbnail_preview(view) {
        Layout::default()
            .direction(Direction::Horizontal)
            .constraints([Constraint::Min(24), Constraint::Length(PREVIEW_PANEL_WIDTH)])
            .split(area)[0]
    } else {
        area
    }
}

fn streams_content_layout(area: Rect) -> Option<(Rect, Rect)> {
    let theme = ThemeName::Norton.palette();
    let inner = panel_block("Streams", "", theme).inner(area);
    if inner.width < 18 || inner.height < MEDIA_ROW_HEIGHT + 2 {
        return None;
    }
    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Length(2), Constraint::Min(MEDIA_ROW_HEIGHT)])
        .split(inner);
    Some((sections[0], sections[1]))
}

fn latest_content_layout(area: Rect) -> Option<(Rect, Rect)> {
    let theme = ThemeName::Norton.palette();
    let inner = panel_block("Latest", "", theme).inner(area);
    if inner.width < 18 || inner.height < MEDIA_ROW_HEIGHT + 2 {
        return None;
    }
    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Length(2), Constraint::Min(MEDIA_ROW_HEIGHT)])
        .split(inner);
    Some((sections[0], sections[1]))
}

fn recordings_content_layout(area: Rect) -> Option<Rect> {
    let theme = ThemeName::Norton.palette();
    let inner = panel_block("", "", theme).inner(area);
    (inner.width >= 18 && inner.height >= MEDIA_ROW_HEIGHT).then_some(inner)
}

fn stream_tab_regions(area: Rect, counts: StreamCounts) -> Vec<(Rect, StreamTab)> {
    let labels = [
        format!(" Recording [{}] ", counts.recording),
        format!(" Offline [{}] ", counts.offline),
        format!(" Disabled [{}] ", counts.disabled),
    ];
    let mut x = area.x;
    let mut regions = Vec::with_capacity(3);
    for (index, label) in labels.iter().enumerate() {
        let width = label.chars().count() as u16;
        let rect = Rect {
            x,
            y: area.y,
            width: width.min(area.x.saturating_add(area.width).saturating_sub(x)),
            height: area.height,
        };
        regions.push((
            rect,
            match index {
                0 => StreamTab::Live,
                1 => StreamTab::Offline,
                _ => StreamTab::Disabled,
            },
        ));
        x = x.saturating_add(width + 1);
    }
    regions
}

fn main_content_hit_index(app: &App, area: Rect, column: u16, row: u16) -> Option<usize> {
    let left = main_panel_left_area(area, app.view);
    match app.view {
        View::Streams => {
            let (_, rows_area) = streams_content_layout(left)?;
            row_hit_index(
                rows_area,
                column,
                row,
                MEDIA_ROW_HEIGHT,
                app.visible_stream_channels().len(),
                app.selected_channel,
            )
        }
        View::Channels => {
            let rows_area = recordings_content_layout(left)?;
            row_hit_index(
                rows_area,
                column,
                row,
                MEDIA_ROW_HEIGHT,
                app.channels.len(),
                app.selected_channel,
            )
        }
        View::Latest => {
            let (_, rows_area) = latest_content_layout(left)?;
            row_hit_index(
                rows_area,
                column,
                row,
                MEDIA_ROW_HEIGHT,
                app.video_page.videos.len(),
                app.selected_video,
            )
        }
        View::Random => {
            let (_, rows_area) = latest_content_layout(left)?;
            row_hit_index(
                rows_area,
                column,
                row,
                MEDIA_ROW_HEIGHT,
                app.random_videos.len(),
                app.selected_video,
            )
        }
        View::Favourites => {
            let rows_area = recordings_content_layout(left)?;
            row_hit_index(
                rows_area,
                column,
                row,
                MEDIA_ROW_HEIGHT,
                app.bookmark_videos.len(),
                app.selected_video,
            )
        }
        View::Similarity => {
            let rows_area = recordings_content_layout(left)?;
            row_hit_index(
                rows_area,
                column,
                row,
                2,
                app.similarity_groups
                    .as_ref()
                    .map(|groups| groups.groups.len())
                    .unwrap_or(0),
                app.selected_similarity,
            )
        }
        View::Processes => {
            let theme = ThemeName::Norton.palette();
            let inner = panel_block("Processes", "", theme).inner(left);
            let rows_area = Rect {
                x: inner.x,
                y: inner.y.saturating_add(1),
                width: inner.width,
                height: inner.height.saturating_sub(1),
            };
            row_hit_index(
                rows_area,
                column,
                row,
                1,
                app.processes.len(),
                app.selected_process,
            )
        }
        View::Jobs => {
            let theme = ThemeName::Norton.palette();
            let inner = panel_block("Jobs", "", theme).inner(left);
            let rows_area = Rect {
                x: inner.x,
                y: inner.y.saturating_add(1),
                width: inner.width,
                height: inner.height.saturating_sub(1),
            };
            row_hit_index(
                rows_area,
                column,
                row,
                1,
                app.jobs.jobs.len(),
                app.selected_job,
            )
        }
        View::Logs => {
            let theme = ThemeName::Norton.palette();
            let inner = panel_block("Logs", "", theme).inner(left);
            row_hit_index(
                inner,
                column,
                row,
                2,
                app.events.len().max(1),
                app.selected_event,
            )
        }
        View::Channel | View::Admin | View::Info | View::Monitoring => None,
    }
}

fn channel_popup_recordings_layout(area: Rect) -> Option<(Rect, Rect)> {
    let popup = centered_rect(88, area.height.saturating_sub(2), area);
    let theme = ThemeName::Norton.palette();
    let inner = panel_block("", "", theme).inner(popup);
    if inner.width < 20 || inner.height < 8 {
        return None;
    }
    let sections = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Min(28), Constraint::Length(PREVIEW_PANEL_WIDTH)])
        .split(inner);
    let rows_area = recordings_content_layout(sections[0])?;
    Some((popup, rows_area))
}

fn theme_picker_layout(area: Rect) -> Option<(Rect, Rect)> {
    let popup = centered_rect(46, 11, area);
    let theme = ThemeName::Norton.palette();
    let inner = panel_block("", "", theme).inner(popup);
    if inner.width < 20 || inner.height < 5 {
        return None;
    }
    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(4), Constraint::Length(1)])
        .split(inner);
    Some((popup, sections[0]))
}

fn player_picker_layout(area: Rect) -> Option<(Rect, Rect)> {
    let popup = centered_rect(58, 13, area);
    let theme = ThemeName::Norton.palette();
    let inner = panel_block("", "", theme).inner(popup);
    if inner.width < 24 || inner.height < 6 {
        return None;
    }
    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(5), Constraint::Length(1)])
        .split(inner);
    Some((popup, sections[0]))
}

fn help_popup_layout(area: Rect) -> Option<(Rect, Rect, Rect)> {
    let popup_height = area.height.saturating_sub(4).clamp(14, 24);
    let popup = centered_rect(78, popup_height, area);
    let theme = ThemeName::Norton.palette();
    let inner = panel_block("", "", theme).inner(popup);
    if inner.width < 28 || inner.height < 8 {
        return None;
    }
    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(6), Constraint::Length(1)])
        .split(inner);
    Some((popup, sections[0], sections[1]))
}

fn help_popup_max_scroll(context: HelpContext, body_height: u16) -> u16 {
    let line_count = help_sections(context)
        .iter()
        .map(|section| 1usize + section.lines.len() + 1)
        .sum::<usize>();
    line_count.saturating_sub(body_height as usize) as u16
}

fn item_menu_layout(area: Rect) -> Option<(Rect, Rect)> {
    let popup = centered_rect(58, 16, area);
    let theme = ThemeName::Norton.palette();
    let inner = panel_block("", "", theme).inner(popup);
    if inner.width < 20 || inner.height < 6 {
        return None;
    }
    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(4), Constraint::Length(1)])
        .split(inner);
    Some((popup, sections[0]))
}

fn channel_editor_layout(area: Rect) -> Option<(Rect, Rect)> {
    let popup = centered_rect(72, 16, area);
    let theme = ThemeName::Norton.palette();
    let inner = panel_block("", "", theme).inner(popup);
    if inner.width < 24 || inner.height < 8 {
        return None;
    }
    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(1),
            Constraint::Min(7),
            Constraint::Length(2),
        ])
        .split(inner);
    Some((popup, sections[1]))
}

fn enhance_form_layout(area: Rect) -> Option<(Rect, Rect)> {
    let popup = centered_rect(74, 17, area);
    let theme = ThemeName::Norton.palette();
    let inner = panel_block("", "", theme).inner(popup);
    if inner.width < 24 || inner.height < 8 {
        return None;
    }
    let sections = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Min(7),
            Constraint::Length(3),
            Constraint::Length(2),
        ])
        .split(inner);
    Some((popup, sections[0]))
}

fn confirm_layout(area: Rect) -> Option<(Rect, Rect, Rect)> {
    let popup = centered_rect(50, 7, area);
    if popup.width < 8 || popup.height < 5 {
        return None;
    }
    let inner = Rect {
        x: popup.x.saturating_add(1),
        y: popup.y.saturating_add(1),
        width: popup.width.saturating_sub(2),
        height: popup.height.saturating_sub(2),
    };
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
    Some((popup, buttons[0], buttons[1]))
}

fn row_hit_index(
    area: Rect,
    column: u16,
    row: u16,
    row_height: u16,
    count: usize,
    selected: usize,
) -> Option<usize> {
    if count == 0 || row_height == 0 || !rect_contains(area, column, row) {
        return None;
    }
    let capacity = max(1, area.height / row_height) as usize;
    let (start, end) = visible_window(selected, count, capacity);
    let offset = row.saturating_sub(area.y) / row_height;
    let index = start + offset as usize;
    (index < end).then_some(index)
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

fn display_channel_name(channel: &ChannelInfo) -> String {
    if channel.display_name.trim().is_empty() {
        channel.channel_name.clone()
    } else {
        channel.display_name.clone()
    }
}

fn channel_placeholder_accent(channel: &ChannelInfo, theme: ThemePalette) -> Color {
    if channel.is_recording {
        theme.danger
    } else if channel.is_paused {
        theme.warning
    } else if channel.is_online {
        theme.success
    } else {
        theme.accent
    }
}

fn view_has_thumbnail_preview(view: View) -> bool {
    matches!(
        view,
        View::Channel
            | View::Streams
            | View::Channels
            | View::Latest
            | View::Random
            | View::Favourites
            | View::Similarity
    )
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

fn average_cpu_percent(info: &UtilSysInfo) -> u64 {
    if info.cpu_info.load_cpu.is_empty() {
        return 0;
    }
    let total = info
        .cpu_info
        .load_cpu
        .iter()
        .map(|load| load.load.max(0.0))
        .sum::<f64>();
    ((total / info.cpu_info.load_cpu.len() as f64) * 100.0).round() as u64
}

fn latest_cpu_summary(app: &App) -> String {
    app.monitor_history
        .last()
        .map(|sample| format!("{}% @ {}", sample.cpu_load_percent, sample.timestamp))
        .unwrap_or_else(|| "no samples".to_string())
}

fn latest_rx_summary(app: &App) -> String {
    app.monitor_history
        .last()
        .map(|sample| format!("{} MB", sample.rx_megabytes))
        .unwrap_or_else(|| "no samples".to_string())
}

fn latest_tx_summary(app: &App) -> String {
    app.monitor_history
        .last()
        .map(|sample| format!("{} MB", sample.tx_megabytes))
        .unwrap_or_else(|| "no samples".to_string())
}

fn format_tags(tags: &Value) -> String {
    match tags {
        Value::Array(values) => values
            .iter()
            .filter_map(Value::as_str)
            .collect::<Vec<_>>()
            .join(", "),
        Value::String(text) => text.clone(),
        _ => String::new(),
    }
}

fn yes_no(value: bool) -> &'static str {
    if value { "yes" } else { "no" }
}

fn format_seconds(value: f64) -> String {
    if value <= 0.0 {
        return "0s".to_string();
    }
    let total = value.round() as u64;
    let hours = total / 3600;
    let minutes = (total % 3600) / 60;
    let seconds = total % 60;
    if hours > 0 {
        format!("{hours}h {minutes:02}m")
    } else if minutes > 0 {
        format!("{minutes}m {seconds:02}s")
    } else {
        format!("{seconds}s")
    }
}

fn format_duration(value: f64) -> String {
    let total = value.max(0.0).round() as u64;
    let hours = total / 3600;
    let minutes = (total % 3600) / 60;
    let seconds = total % 60;
    if hours > 0 {
        format!("{hours:02}:{minutes:02}:{seconds:02}")
    } else {
        format!("{minutes:02}:{seconds:02}")
    }
}

fn format_bytes(value: u64) -> String {
    if value == 0 {
        return "0 B".to_string();
    }
    let units = ["B", "KB", "MB", "GB", "TB"];
    let mut size = value as f64;
    let mut index = 0usize;
    while size >= 1024.0 && index < units.len() - 1 {
        size /= 1024.0;
        index += 1;
    }
    if size >= 100.0 || index == 0 {
        format!("{size:.0} {}", units[index])
    } else if size >= 10.0 {
        format!("{size:.1} {}", units[index])
    } else {
        format!("{size:.2} {}", units[index])
    }
}

fn truncate(value: &str, width: usize) -> String {
    if value.chars().count() <= width {
        return value.to_string();
    }
    if width <= 1 {
        return "…".to_string();
    }
    let mut output = value.chars().take(width - 1).collect::<String>();
    output.push('…');
    output
}

fn sanitize_filename(value: &str) -> String {
    let sanitized = value
        .chars()
        .map(|character| match character {
            '/' | '\\' | '\0' => '_',
            _ => character,
        })
        .collect::<String>();
    let trimmed = sanitized.trim();
    if trimmed.is_empty() {
        "download.bin".to_string()
    } else {
        trimmed.to_string()
    }
}

fn handle_process_args() -> Result<bool> {
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

async fn spawn_initial_auth(app: &App, tx: &UnboundedSender<AppMessage>) {
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

#[tokio::main]
async fn main() -> Result<()> {
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

#[cfg(test)]
mod tests {
    use super::should_clear_saved_session_on_auth_error;

    #[test]
    fn clears_saved_session_for_authentication_failures() {
        assert!(should_clear_saved_session_on_auth_error(
            "401 Unauthorized: token expired"
        ));
        assert!(should_clear_saved_session_on_auth_error(
            "Forbidden: invalid token"
        ));
        assert!(should_clear_saved_session_on_auth_error(
            "jwt malformed"
        ));
    }

    #[test]
    fn keeps_saved_session_for_transient_startup_failures() {
        assert!(!should_clear_saved_session_on_auth_error(
            "failed to decode response JSON"
        ));
        assert!(!should_clear_saved_session_on_auth_error(
            "connection refused"
        ));
        assert!(!should_clear_saved_session_on_auth_error(
            "client API version incompatible with server API version 0.1.0"
        ));
    }
}

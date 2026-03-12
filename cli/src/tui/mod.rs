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
    ActionTarget, ChannelEditorEvent, ChannelEditorField, ChannelEditorState, ChannelEditorSubmit,
    ChannelPopup, EnhanceField, EnhanceFormEvent, EnhanceFormState, HelpContext, HelpPopup,
    ItemAction, ItemActionMenu, PlayerModePicker, VIDEO_PLAYER_FPS, VIDEO_PLAYER_SEEK_STEP_SECONDS,
    VideoPlaybackWorker, VideoPlayerEvent, VideoPlayerRequest, VideoPopupState,
    desired_frame_cells, help_sections, spawn_video_worker,
};
use crate::player_mode::{PlayerCapabilities, PlayerMode};
use crate::selection::{clamp_index, visible_window};
use crate::ui::{
    FooterAction, PopupId, RenderedThumbnail, TextInput, TextInputAction, ThemeBackground,
    ThemeName, ThemePalette, ThemePicker, ThumbnailEntry, ThumbnailTarget, UiRegion, UiRegions,
    centered_rect, draw_footer_bar, draw_panel_notice, draw_rendered_thumbnail,
    draw_theme_background, draw_vertical_scrollbar, load_thumbnail_image, panel_block,
    render_panel_popup, render_placeholder_thumbnail, render_popup_shell, render_thumbnail,
    row_style, split_scrollbar_area,
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

mod app_core;
mod input;
mod layout;
mod messages;
mod render;
mod runtime;
mod support;

use self::layout::*;
use self::render::draw;
pub(crate) use self::runtime::run;
use self::runtime::websocket_loop;
use self::support::*;

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
pub(crate) enum LoginField {
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
pub(crate) enum LoginMouseAction {
    Submit,
    ToggleMode,
    Mouse,
    Quit,
}

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub(crate) enum WorkspaceHeaderAction {
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
        assert!(should_clear_saved_session_on_auth_error("jwt malformed"));
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

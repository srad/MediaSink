use crate::app::{SimilarityTab, StreamTab, View};
use ratatui::{prelude::Color, text::Line};

#[derive(Debug, Clone)]
pub struct ViewNotice {
    pub lines: Vec<Line<'static>>,
    pub tone: Color,
}

impl ViewNotice {
    fn new(tone: Color, lines: Vec<Line<'static>>) -> Self {
        Self { lines, tone }
    }
}

pub fn is_loading(view: View, refresh_in_flight: bool, view_request_in_flight: bool) -> bool {
    match view {
        View::Streams | View::Channels | View::Latest | View::Jobs | View::Logs => {
            refresh_in_flight
        }
        View::Channel
        | View::Random
        | View::Favourites
        | View::Similarity
        | View::Admin
        | View::Info
        | View::Processes
        | View::Monitoring => view_request_in_flight,
    }
}

pub fn collection_notice(
    view: View,
    item_count: usize,
    is_loading: bool,
    error: Option<&str>,
    stream_tab: StreamTab,
) -> Option<ViewNotice> {
    if item_count > 0 {
        return None;
    }

    if is_loading {
        return Some(loading_notice(view));
    }

    if let Some(error) = non_empty(error) {
        return Some(error_notice(error));
    }

    empty_collection_notice(view, stream_tab)
}

pub fn similarity_notice(
    tab: SimilarityTab,
    group_count: usize,
    is_loading: bool,
    error: Option<&str>,
) -> Option<ViewNotice> {
    if tab == SimilarityTab::Search || group_count > 0 {
        return None;
    }

    if is_loading {
        return Some(loading_notice(View::Similarity));
    }

    if let Some(error) = non_empty(error) {
        return Some(error_notice(error));
    }

    Some(ViewNotice::new(
        Color::White,
        vec![
            Line::from("No similarity groups found."),
            Line::from("Press g to load or retry."),
        ],
    ))
}

pub fn processes_notice(
    process_count: usize,
    is_loading: bool,
    error: Option<&str>,
) -> Option<ViewNotice> {
    if process_count > 0 {
        return None;
    }

    if is_loading {
        return Some(loading_notice(View::Processes));
    }

    if let Some(error) = non_empty(error) {
        return Some(error_notice(error));
    }

    Some(ViewNotice::new(
        Color::White,
        vec![Line::from("No active streaming processes.")],
    ))
}

fn loading_notice(view: View) -> ViewNotice {
    let label = match view {
        View::Streams => "Loading streams…",
        View::Channels => "Loading channels…",
        View::Channel => "Loading channel recordings…",
        View::Latest => "Loading latest recordings…",
        View::Random => "Loading random recordings…",
        View::Favourites => "Loading bookmarked recordings…",
        View::Similarity => "Loading similarity groups…",
        View::Processes => "Loading processes…",
        View::Admin => "Loading admin status…",
        View::Info => "Loading runtime info…",
        View::Monitoring => "Loading monitoring data…",
        View::Jobs => "Loading jobs…",
        View::Logs => "Loading websocket events…",
    };

    ViewNotice::new(
        Color::Yellow,
        vec![
            Line::from(label),
            Line::from("Fetching data from the server…"),
        ],
    )
}

fn error_notice(error: &str) -> ViewNotice {
    ViewNotice::new(
        Color::Red,
        vec![
            Line::from("Server request failed."),
            Line::from(error.to_string()),
            Line::from("Press g to retry."),
        ],
    )
}

fn empty_collection_notice(view: View, stream_tab: StreamTab) -> Option<ViewNotice> {
    match view {
        View::Streams => Some(ViewNotice::new(
            Color::White,
            vec![
                Line::from(format!(
                    "No {} streams in this tab.",
                    stream_tab.label().to_ascii_lowercase()
                )),
                Line::from("Press [ or ] to switch between Recording, Offline, and Disabled."),
            ],
        )),
        View::Channels => Some(ViewNotice::new(
            Color::White,
            vec![
                Line::from("No channels available."),
                Line::from("Press g to reload from the server."),
            ],
        )),
        View::Channel => Some(ViewNotice::new(
            Color::White,
            vec![Line::from("No recordings found in this channel.")],
        )),
        View::Latest => Some(ViewNotice::new(
            Color::White,
            vec![
                Line::from("No recordings available."),
                Line::from("Press g to reload from the server."),
            ],
        )),
        View::Random => Some(ViewNotice::new(
            Color::White,
            vec![
                Line::from("No random recordings loaded."),
                Line::from("Press g to fetch another set."),
            ],
        )),
        View::Favourites => Some(ViewNotice::new(
            Color::White,
            vec![Line::from("No bookmarked recordings found.")],
        )),
        View::Similarity
        | View::Admin
        | View::Info
        | View::Processes
        | View::Monitoring
        | View::Jobs
        | View::Logs => None,
    }
}

fn non_empty(error: Option<&str>) -> Option<&str> {
    error.and_then(|value| {
        if value.trim().is_empty() {
            None
        } else {
            Some(value)
        }
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn primary_views_use_workspace_refresh_loading_state() {
        assert!(is_loading(View::Channels, true, false));
        assert!(!is_loading(View::Channels, false, true));
    }

    #[test]
    fn secondary_views_use_view_request_loading_state() {
        assert!(is_loading(View::Random, false, true));
        assert!(!is_loading(View::Random, true, false));
    }

    #[test]
    fn error_notice_wins_over_empty_state() {
        let notice = collection_notice(
            View::Latest,
            0,
            false,
            Some("412 Precondition Failed"),
            StreamTab::Live,
        )
        .expect("notice");

        assert_eq!(notice.tone, Color::Red);
        assert_eq!(
            notice.lines[0].spans[0].content.as_ref(),
            "Server request failed."
        );
    }

    #[test]
    fn empty_stream_notice_mentions_current_tab() {
        let notice =
            collection_notice(View::Streams, 0, false, None, StreamTab::Offline).expect("notice");

        assert_eq!(
            notice.lines[0].spans[0].content.as_ref(),
            "No offline streams in this tab."
        );
    }
}

use super::*;

pub(super) fn video_popup_layout(area: Rect) -> Option<(Rect, [Rect; 4])> {
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

pub(super) fn terminal_area() -> Rect {
    let (width, height) = crossterm::terminal::size().unwrap_or((120, 40));
    Rect::new(0, 0, width, height)
}

pub(super) fn rect_contains(area: Rect, column: u16, row: u16) -> bool {
    column >= area.x
        && column < area.x.saturating_add(area.width)
        && row >= area.y
        && row < area.y.saturating_add(area.height)
}

pub(super) fn login_layout(area: Rect) -> (Rect, [Rect; 6]) {
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

pub(super) fn centered_action_rects<const N: usize>(area: Rect, labels: &[&str; N]) -> [Rect; N] {
    let slots = (N.saturating_sub(1)) as u16;
    let labels_width = labels.iter().fold(0u16, |width, label| {
        width.saturating_add(label.chars().count() as u16)
    });
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

pub(super) fn login_action_labels(width: u16, mouse_enabled: bool) -> [&'static str; 4] {
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

pub(super) fn workspace_layout(area: Rect, app: &App) -> ([Rect; 5], [Rect; 4]) {
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

pub(super) fn primary_tab_regions(area: Rect) -> Vec<(Rect, View)> {
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

pub(super) fn main_panel_left_area(area: Rect, view: View) -> Rect {
    if view_has_thumbnail_preview(view) {
        Layout::default()
            .direction(Direction::Horizontal)
            .constraints([Constraint::Min(24), Constraint::Length(PREVIEW_PANEL_WIDTH)])
            .split(area)[0]
    } else {
        area
    }
}

pub(super) fn streams_content_layout(area: Rect) -> Option<(Rect, Rect)> {
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

pub(super) fn latest_content_layout(area: Rect) -> Option<(Rect, Rect)> {
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

pub(super) fn recordings_content_layout(area: Rect) -> Option<Rect> {
    let theme = ThemeName::Norton.palette();
    let inner = panel_block("", "", theme).inner(area);
    (inner.width >= 18 && inner.height >= MEDIA_ROW_HEIGHT).then_some(inner)
}

pub(super) fn stream_tab_regions(area: Rect, counts: StreamCounts) -> Vec<(Rect, StreamTab)> {
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

pub(super) fn main_content_hit_index(
    app: &App,
    area: Rect,
    column: u16,
    row: u16,
) -> Option<usize> {
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

pub(super) fn channel_popup_recordings_layout(area: Rect) -> Option<(Rect, Rect)> {
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

pub(super) fn theme_picker_layout(area: Rect) -> Option<(Rect, Rect)> {
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

pub(super) fn player_picker_layout(area: Rect) -> Option<(Rect, Rect)> {
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

pub(super) fn help_popup_layout(area: Rect) -> Option<(Rect, Rect, Rect)> {
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

pub(super) fn help_popup_max_scroll(context: HelpContext, body_height: u16) -> u16 {
    let line_count = help_sections(context)
        .iter()
        .map(|section| 1usize + section.lines.len() + 1)
        .sum::<usize>();
    line_count.saturating_sub(body_height as usize) as u16
}

pub(super) fn item_menu_layout(area: Rect) -> Option<(Rect, Rect)> {
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

pub(super) fn channel_editor_layout(area: Rect) -> Option<(Rect, Rect)> {
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

pub(super) fn enhance_form_layout(area: Rect) -> Option<(Rect, Rect)> {
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

pub(super) fn confirm_layout(area: Rect) -> Option<(Rect, Rect, Rect)> {
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

pub(super) fn row_hit_index(
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

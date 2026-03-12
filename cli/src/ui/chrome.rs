use crate::app::ViewNotice;
use super::ThemePalette;
use ratatui::{
    Frame,
    buffer::Buffer,
    layout::{Alignment, Rect},
    prelude::{Modifier, Style},
    symbols::border,
    text::{Line, Span},
    widgets::{Block, Borders, Paragraph, Wrap},
};
use std::cmp::{max, min};

#[derive(Debug, Clone, Copy)]
pub struct FooterAction<'a> {
    pub key: &'a str,
    pub label: &'a str,
}

pub fn row_style(selected: bool, theme: ThemePalette) -> Style {
    theme.row_style(selected)
}

pub fn centered_rect(width_percent: u16, height: u16, area: Rect) -> Rect {
    let width = min(
        area.width.saturating_sub(4),
        max(20, area.width * width_percent / 100),
    );
    let popup_height = min(area.height.saturating_sub(2), height);
    Rect {
        x: area.x + area.width.saturating_sub(width) / 2,
        y: area.y + area.height.saturating_sub(popup_height) / 2,
        width,
        height: popup_height,
    }
}

pub fn panel_block(
    title: impl Into<String>,
    subtitle: impl Into<String>,
    theme: ThemePalette,
) -> Block<'static> {
    let title = title.into();
    let subtitle = subtitle.into();
    Block::default()
        .title(Line::from(vec![Span::styled(
            format!(" {} ", title),
            theme.title_style(),
        )]))
        .title(
            Line::from(Span::styled(subtitle, theme.subtitle_style())).alignment(Alignment::Right),
        )
        .borders(Borders::ALL)
        .border_set(border::ROUNDED)
        .border_style(theme.panel_border_style())
        .style(theme.surface_style())
}

pub fn popup_close_rect(popup: Rect) -> Option<Rect> {
    (popup.width >= 6 && popup.height >= 1).then_some(Rect {
        x: popup.x + popup.width.saturating_sub(4),
        y: popup.y,
        width: 3,
        height: 1,
    })
}

pub fn draw_popup_close_button(frame: &mut Frame, popup: Rect, theme: ThemePalette) {
    let Some(close_rect) = popup_close_rect(popup) else {
        return;
    };

    frame.render_widget(
        Paragraph::new("[x]")
            .alignment(Alignment::Center)
            .style(theme.chip_style(theme.danger)),
        close_rect,
    );
}

pub fn draw_panel_notice(frame: &mut Frame, area: Rect, notice: &ViewNotice, theme: ThemePalette) {
    frame.render_widget(
        Paragraph::new(notice.lines.clone())
            .alignment(Alignment::Center)
            .style(theme.notice_style(notice.tone))
            .wrap(Wrap { trim: true }),
        area,
    );
}

pub fn split_scrollbar_area(area: Rect, show_scrollbar: bool) -> (Rect, Option<Rect>) {
    if !show_scrollbar || area.width < 3 || area.height == 0 {
        return (area, None);
    }

    let scrollbar = Rect {
        x: area.x + area.width.saturating_sub(1),
        y: area.y,
        width: 1,
        height: area.height,
    };
    let content = Rect {
        x: area.x,
        y: area.y,
        width: area.width.saturating_sub(1),
        height: area.height,
    };
    (content, Some(scrollbar))
}

pub fn draw_vertical_scrollbar(
    frame: &mut Frame,
    area: Rect,
    total_items: usize,
    first_visible: usize,
    visible_items: usize,
    theme: ThemePalette,
) {
    if area.width == 0 || area.height == 0 || total_items <= visible_items || visible_items == 0 {
        return;
    }

    let track_height = if area.height >= 3 {
        area.height.saturating_sub(2) as usize
    } else {
        area.height as usize
    };
    let thumb_height = max(1, track_height.saturating_mul(visible_items) / total_items);
    let max_offset = total_items.saturating_sub(visible_items);
    let max_thumb_offset = track_height.saturating_sub(thumb_height);
    let thumb_offset = if max_offset == 0 {
        0
    } else {
        first_visible.saturating_mul(max_thumb_offset) / max_offset
    };

    draw_scrollbar_cells(frame.buffer_mut(), area, thumb_offset, thumb_height, theme);
}

fn draw_scrollbar_cells(
    buffer: &mut Buffer,
    area: Rect,
    thumb_offset: usize,
    thumb_height: usize,
    theme: ThemePalette,
) {
    let track_style = theme.chrome_style().fg(theme.border);
    let accent_style = theme
        .surface_alt_style()
        .fg(theme.border_focus)
        .add_modifier(Modifier::BOLD);
    let thumb_style = accent_style;

    if area.height >= 3 {
        if let Some(cell) = buffer.cell_mut((area.x, area.y)) {
            cell.set_symbol("▲");
            cell.set_style(accent_style);
        }
        if let Some(cell) = buffer.cell_mut((area.x, area.y + area.height.saturating_sub(1))) {
            cell.set_symbol("▼");
            cell.set_style(accent_style);
        }

        let track_y = area.y.saturating_add(1);
        let track_height = area.height.saturating_sub(2);
        for offset in 0..track_height {
            if let Some(cell) = buffer.cell_mut((area.x, track_y + offset)) {
                cell.set_symbol("│");
                cell.set_style(track_style);
            }
        }

        let clamped_thumb_height = thumb_height.min(track_height as usize);
        let max_thumb_offset = track_height.saturating_sub(clamped_thumb_height as u16) as usize;
        let clamped_thumb_offset = thumb_offset.min(max_thumb_offset);
        let thumb_end = clamped_thumb_offset
            .saturating_add(clamped_thumb_height)
            .min(track_height as usize);
        for offset in clamped_thumb_offset..thumb_end {
            if let Some(cell) = buffer.cell_mut((area.x, track_y + offset as u16)) {
                cell.set_symbol("█");
                cell.set_style(thumb_style);
            }
        }
        return;
    }

    for offset in 0..area.height {
        if let Some(cell) = buffer.cell_mut((area.x, area.y + offset)) {
            cell.set_symbol("│");
            cell.set_style(track_style);
        }
    }

    let thumb_end = thumb_offset
        .saturating_add(thumb_height)
        .min(area.height as usize);
    for offset in thumb_offset..thumb_end {
        if let Some(cell) = buffer.cell_mut((area.x, area.y + offset as u16)) {
            cell.set_symbol("█");
            cell.set_style(thumb_style);
        }
    }
}

pub fn draw_footer_bar(
    frame: &mut Frame,
    area: Rect,
    actions: &[FooterAction<'_>],
    function_actions: &[FooterAction<'_>],
    right_line: Option<Line<'static>>,
    theme: ThemePalette,
) {
    if area.width == 0 {
        return;
    }

    frame.render_widget(
        Paragraph::new(Line::from(""))
            .style(theme.chrome_style())
            .alignment(Alignment::Left),
        area,
    );

    let status_width = right_line
        .as_ref()
        .map(|line| {
            min(
                area.width.saturating_sub(8),
                line_width(line).saturating_add(1) as u16,
            )
        })
        .unwrap_or(0);
    let max_function_width = area.width.saturating_sub(status_width).saturating_sub(8);
    let function_line = (!function_actions.is_empty() && max_function_width > 0)
        .then(|| grouped_action_line(function_actions, max_function_width as usize, theme));
    let function_width = function_line
        .as_ref()
        .map(|line| line_width(line).saturating_add(1) as u16)
        .unwrap_or(0)
        .min(max_function_width);

    let right_edge = area.x.saturating_add(area.width);
    let status_rect = Rect {
        x: right_edge.saturating_sub(status_width),
        y: area.y,
        width: status_width,
        height: area.height,
    };
    let function_rect = Rect {
        x: status_rect.x.saturating_sub(function_width),
        y: area.y,
        width: function_width,
        height: area.height,
    };
    let actions_rect = Rect {
        x: area.x,
        y: area.y,
        width: function_rect.x.saturating_sub(area.x),
        height: area.height,
    };

    if actions_rect.width > 0 {
        frame.render_widget(
            Paragraph::new(grouped_action_line(
                actions,
                actions_rect.width as usize,
                theme,
            ))
            .style(theme.chrome_style()),
            actions_rect,
        );
    }

    if let Some(function_line) = function_line {
        if function_rect.width > 0 {
            frame.render_widget(
                Paragraph::new(function_line)
                    .alignment(Alignment::Right)
                    .style(theme.chrome_style()),
                function_rect,
            );
        }
    }

    if let Some(right_line) = right_line {
        if status_rect.width > 0 {
            frame.render_widget(
                Paragraph::new(right_line)
                    .alignment(Alignment::Right)
                    .style(theme.chrome_style()),
                status_rect,
            );
        }
    }
}

fn line_width(line: &Line<'_>) -> usize {
    line.spans
        .iter()
        .map(|span| span.content.chars().count())
        .sum()
}

pub fn grouped_action_line(
    actions: &[FooterAction<'_>],
    width: usize,
    theme: ThemePalette,
) -> Line<'static> {
    let mut spans = Vec::new();
    let mut used = 0usize;

    for (index, action) in actions.iter().enumerate() {
        let key = format!(" {} ", action.key);
        let separator = if index + 1 < actions.len() { 2 } else { 0 };
        let group_width = key.chars().count() + 1 + action.label.chars().count() + separator;
        if !spans.is_empty() && used + group_width > width {
            break;
        }

        spans.push(Span::styled(key, theme.footer_key_style()));
        spans.push(Span::styled(" ", theme.footer_separator_style()));
        spans.push(Span::styled(
            action.label.to_string(),
            theme.footer_label_style(),
        ));
        used += group_width;

        if index + 1 < actions.len() && used < width {
            spans.push(Span::styled("  ", theme.footer_separator_style()));
            used += 2;
        }
    }

    Line::from(spans)
}

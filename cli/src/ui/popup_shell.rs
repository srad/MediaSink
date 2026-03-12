use super::{
    PopupId, ThemePalette, UiRegion, UiRegions, draw_popup_close_button, panel_block,
    popup_close_rect,
};
use ratatui::{
    Frame,
    layout::Rect,
    widgets::{Block, Clear},
};

#[derive(Debug, Clone, Copy)]
pub struct PopupShell {
    pub inner: Rect,
}

pub fn render_panel_popup(
    frame: &mut Frame,
    popup: Rect,
    title: impl Into<String>,
    subtitle: impl Into<String>,
    theme: ThemePalette,
    regions: &mut UiRegions,
    popup_id: PopupId,
) -> PopupShell {
    let block = panel_block(title, subtitle, theme);
    render_popup_shell(frame, popup, block, theme, regions, popup_id)
}

pub fn render_popup_shell(
    frame: &mut Frame,
    popup: Rect,
    block: Block<'static>,
    theme: ThemePalette,
    regions: &mut UiRegions,
    popup_id: PopupId,
) -> PopupShell {
    frame.render_widget(Clear, popup);
    frame.render_widget(block.clone(), popup);
    draw_popup_close_button(frame, popup, theme);
    if let Some(close_rect) = popup_close_rect(popup) {
        regions.register(close_rect, UiRegion::PopupClose(popup_id));
    }

    PopupShell {
        inner: block.inner(popup),
    }
}

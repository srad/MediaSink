pub mod background;
pub mod chrome;
pub mod popup_shell;
pub mod text_input;
pub mod theme;
pub mod theme_picker;
pub mod thumbnail;
pub mod ui_regions;

pub use background::draw_theme_background;
pub use chrome::{
    FooterAction, centered_rect, draw_footer_bar, draw_panel_notice, draw_popup_close_button,
    draw_vertical_scrollbar, panel_block, popup_close_rect, row_style, split_scrollbar_area,
};
pub use popup_shell::{render_panel_popup, render_popup_shell};
pub use text_input::{TextInput, TextInputAction};
pub use theme::{ThemeBackground, ThemeName, ThemePalette};
pub use theme_picker::ThemePicker;
pub use thumbnail::{
    RenderedThumbnail, ThumbnailEntry, ThumbnailTarget, draw_rendered_thumbnail,
    load_thumbnail_image, render_placeholder_thumbnail, render_thumbnail, render_video_frame,
    rendered_video_frame_pixel_dimensions,
};
pub use ui_regions::{PopupId, UiRegion, UiRegions};

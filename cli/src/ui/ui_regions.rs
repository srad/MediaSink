use crate::{
    app::{StreamTab, View},
    tui::{LoginField, LoginMouseAction, WorkspaceHeaderAction},
};
use ratatui::layout::Rect;

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum PopupId {
    ChannelPopup,
    ItemMenu,
    ChannelEditor,
    EnhanceForm,
    VideoPlayer,
    ThemePicker,
    PlayerPicker,
    Help,
    Confirm,
}

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum UiRegion {
    LoginField(LoginField),
    LoginAction(LoginMouseAction),
    WorkspaceHeader(WorkspaceHeaderAction),
    PrimaryTab(View),
    StreamTab(StreamTab),
    PopupClose(PopupId),
    VideoSeekBar,
}

#[derive(Debug, Default, Clone)]
pub struct UiRegions {
    entries: Vec<(Rect, UiRegion)>,
}

impl UiRegions {
    pub fn clear(&mut self) {
        self.entries.clear();
    }

    pub fn register(&mut self, rect: Rect, region: UiRegion) {
        if rect.width == 0 || rect.height == 0 {
            return;
        }
        self.entries.push((rect, region));
    }

    pub fn hit(&self, column: u16, row: u16) -> Option<UiRegion> {
        self.entries
            .iter()
            .rev()
            .find_map(|(rect, region)| contains(*rect, column, row).then_some(*region))
    }
}

fn contains(rect: Rect, column: u16, row: u16) -> bool {
    column >= rect.x
        && column < rect.x.saturating_add(rect.width)
        && row >= rect.y
        && row < rect.y.saturating_add(rect.height)
}

#[cfg(test)]
mod tests {
    use super::{PopupId, UiRegion, UiRegions};
    use crate::{app::View, tui::LoginField};
    use ratatui::layout::Rect;

    #[test]
    fn hit_uses_last_registered_region() {
        let mut regions = UiRegions::default();
        regions.register(
            Rect::new(1, 1, 10, 2),
            UiRegion::LoginField(LoginField::Server),
        );
        regions.register(Rect::new(1, 1, 4, 1), UiRegion::PrimaryTab(View::Streams));

        assert_eq!(regions.hit(2, 1), Some(UiRegion::PrimaryTab(View::Streams)));
        assert_eq!(
            regions.hit(9, 2),
            Some(UiRegion::LoginField(LoginField::Server))
        );
    }

    #[test]
    fn hit_ignores_empty_rectangles() {
        let mut regions = UiRegions::default();
        regions.register(Rect::new(0, 0, 0, 1), UiRegion::PopupClose(PopupId::Help));

        assert_eq!(regions.hit(0, 0), None);
    }
}

use crate::api::{ChannelInfo, Recording};

#[derive(Debug, Clone)]
pub enum ActionTarget {
    Channel(ChannelInfo),
    Video(Recording),
}

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum ItemAction {
    OpenChannelRecordings,
    EditChannel,
    ToggleChannelPause,
    ToggleChannelFavourite,
    DeleteChannel,
    DownloadVideo,
    ToggleVideoBookmark,
    AnalyzeVideo,
    GenerateVideoPreview,
    EnhanceVideo,
    ConvertVideo720,
    ConvertVideo1080,
    DeleteVideo,
}

#[derive(Debug, Clone)]
pub struct ActionItem {
    pub action: ItemAction,
    pub hint: String,
    pub label: String,
}

#[derive(Debug, Clone, Default)]
pub struct ItemActionMenu {
    items: Vec<ActionItem>,
    selected: usize,
    target: Option<ActionTarget>,
}

impl ItemActionMenu {
    pub fn is_open(&self) -> bool {
        self.target.is_some()
    }

    pub fn close(&mut self) {
        *self = Self::default();
    }

    pub fn items(&self) -> &[ActionItem] {
        &self.items
    }

    pub fn selected(&self) -> usize {
        self.selected
    }

    pub fn set_selected(&mut self, index: usize) {
        self.selected = index.min(self.items.len().saturating_sub(1));
    }

    pub fn target(&self) -> Option<&ActionTarget> {
        self.target.as_ref()
    }

    pub fn selected_action(&self) -> Option<ItemAction> {
        self.items.get(self.selected).map(|item| item.action)
    }

    pub fn move_selection(&mut self, delta: isize) {
        if self.items.is_empty() {
            self.selected = 0;
            return;
        }

        self.selected = if delta < 0 {
            if self.selected == 0 {
                self.items.len() - 1
            } else {
                self.selected - 1
            }
        } else if self.selected + 1 >= self.items.len() {
            0
        } else {
            self.selected + 1
        };
    }

    pub fn open_channel(&mut self, channel: ChannelInfo) {
        self.target = Some(ActionTarget::Channel(channel.clone()));
        self.selected = 0;
        self.items = vec![
            ActionItem {
                action: ItemAction::OpenChannelRecordings,
                hint: "Open channel recordings popup".to_string(),
                label: "Open recordings".to_string(),
            },
            ActionItem {
                action: ItemAction::EditChannel,
                hint: "Edit stream settings and tags".to_string(),
                label: "Edit stream".to_string(),
            },
            ActionItem {
                action: ItemAction::ToggleChannelPause,
                hint: "Toggle channel recording state".to_string(),
                label: if channel.is_paused {
                    "Resume recording".to_string()
                } else {
                    "Pause recording".to_string()
                },
            },
            ActionItem {
                action: ItemAction::ToggleChannelFavourite,
                hint: "Toggle favourite flag".to_string(),
                label: if channel.fav {
                    "Remove favourite".to_string()
                } else {
                    "Add favourite".to_string()
                },
            },
            ActionItem {
                action: ItemAction::DeleteChannel,
                hint: "Delete the channel and all its recordings".to_string(),
                label: "Delete channel".to_string(),
            },
        ];
    }

    pub fn open_video(&mut self, video: Recording) {
        self.target = Some(ActionTarget::Video(video.clone()));
        self.selected = 0;
        let mut items = vec![
            ActionItem {
                action: ItemAction::DownloadVideo,
                hint: "Download to the current working directory".to_string(),
                label: "Download recording".to_string(),
            },
            ActionItem {
                action: ItemAction::ToggleVideoBookmark,
                hint: "Toggle bookmark state".to_string(),
                label: if video.bookmark {
                    "Remove bookmark".to_string()
                } else {
                    "Add bookmark".to_string()
                },
            },
            ActionItem {
                action: ItemAction::AnalyzeVideo,
                hint: "Queue visual analysis".to_string(),
                label: "Run analysis".to_string(),
            },
            ActionItem {
                action: ItemAction::GenerateVideoPreview,
                hint: "Regenerate preview frames".to_string(),
                label: "Generate preview".to_string(),
            },
            ActionItem {
                action: ItemAction::EnhanceVideo,
                hint: "Open enhancement options".to_string(),
                label: "Enhance recording".to_string(),
            },
        ];

        if video.height != 720 {
            items.push(ActionItem {
                action: ItemAction::ConvertVideo720,
                hint: "Queue 720p conversion".to_string(),
                label: "Convert to 720p".to_string(),
            });
        }
        if video.height != 1080 {
            items.push(ActionItem {
                action: ItemAction::ConvertVideo1080,
                hint: "Queue 1080p conversion".to_string(),
                label: "Convert to 1080p".to_string(),
            });
        }
        items.push(ActionItem {
            action: ItemAction::DeleteVideo,
            hint: "Delete this recording".to_string(),
            label: "Delete recording".to_string(),
        });

        self.items = items;
    }
}

#[cfg(test)]
mod tests {
    use super::{ItemAction, ItemActionMenu};
    use crate::api::{ChannelInfo, Recording};

    #[test]
    fn channel_actions_reflect_pause_and_favourite_state() {
        let mut menu = ItemActionMenu::default();
        menu.open_channel(ChannelInfo {
            fav: true,
            is_paused: true,
            ..ChannelInfo::default()
        });

        let labels = menu
            .items()
            .iter()
            .map(|item| item.label.as_str())
            .collect::<Vec<_>>();

        assert!(labels.contains(&"Resume recording"));
        assert!(labels.contains(&"Remove favourite"));
    }

    #[test]
    fn video_actions_hide_matching_convert_resolution() {
        let mut menu = ItemActionMenu::default();
        menu.open_video(Recording {
            bookmark: true,
            height: 1080,
            ..Recording::default()
        });

        assert_eq!(menu.selected_action(), Some(ItemAction::DownloadVideo));
        let actions = menu
            .items()
            .iter()
            .map(|item| item.action)
            .collect::<Vec<_>>();
        assert!(actions.contains(&ItemAction::ToggleVideoBookmark));
        assert!(actions.contains(&ItemAction::ConvertVideo720));
        assert!(!actions.contains(&ItemAction::ConvertVideo1080));
    }
}

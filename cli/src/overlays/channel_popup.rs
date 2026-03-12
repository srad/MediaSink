use crate::api::{ChannelInfo, Recording};

#[derive(Debug, Clone, Default)]
pub struct ChannelPopup {
    channel: Option<ChannelInfo>,
    error: Option<String>,
    loading: bool,
    requested_channel_id: Option<u64>,
    selected_recording: usize,
}

impl ChannelPopup {
    pub fn is_open(&self) -> bool {
        self.requested_channel_id.is_some()
    }

    pub fn open(&mut self, channel: ChannelInfo) {
        self.requested_channel_id = Some(channel.channel_id);
        self.channel = Some(channel);
        self.error = None;
        self.loading = true;
        self.selected_recording = 0;
    }

    pub fn close(&mut self) {
        *self = Self::default();
    }

    pub fn requested_channel_id(&self) -> Option<u64> {
        self.requested_channel_id
    }

    pub fn start_loading(&mut self) {
        if self.requested_channel_id.is_some() {
            self.loading = true;
            self.error = None;
        }
    }

    pub fn loading(&self) -> bool {
        self.loading
    }

    pub fn error(&self) -> Option<&str> {
        self.error.as_deref()
    }

    pub fn channel(&self) -> Option<&ChannelInfo> {
        self.channel.as_ref()
    }

    pub fn selected_recording(&self) -> usize {
        self.selected_recording
    }

    pub fn set_selected_recording(&mut self, index: usize) {
        self.selected_recording = index;
        self.clamp_selection();
    }

    pub fn recordings(&self) -> Vec<Recording> {
        let Some(channel) = self.channel.as_ref() else {
            return Vec::new();
        };

        let mut recordings = channel.recordings.clone();
        recordings.sort_by(|left, right| right.created_at.cmp(&left.created_at));
        recordings
    }

    pub fn apply_result(&mut self, channel_id: u64, result: Result<ChannelInfo, String>) -> bool {
        if self.requested_channel_id != Some(channel_id) {
            return false;
        }

        self.loading = false;
        match result {
            Ok(channel) => {
                self.error = None;
                self.channel = Some(channel);
                self.clamp_selection();
                true
            }
            Err(error) => {
                self.error = Some(error);
                self.clamp_selection();
                false
            }
        }
    }

    pub fn move_selection(&mut self, delta: isize) {
        let count = self.recordings().len();
        if count == 0 {
            self.selected_recording = 0;
            return;
        }

        self.selected_recording = if delta < 0 {
            if self.selected_recording == 0 {
                count - 1
            } else {
                self.selected_recording - 1
            }
        } else if self.selected_recording + 1 >= count {
            0
        } else {
            self.selected_recording + 1
        };
    }

    pub fn clamp_selection(&mut self) {
        let len = self.recordings().len();
        self.selected_recording = if len == 0 {
            0
        } else {
            self.selected_recording.min(len - 1)
        };
    }
}

#[cfg(test)]
mod tests {
    use super::ChannelPopup;
    use crate::api::{ChannelInfo, Recording};

    #[test]
    fn closes_and_resets_state() {
        let mut popup = ChannelPopup::default();
        popup.open(ChannelInfo {
            channel_id: 1,
            ..ChannelInfo::default()
        });

        popup.close();

        assert!(!popup.is_open());
        assert!(popup.channel().is_none());
        assert!(popup.error().is_none());
        assert!(!popup.loading());
    }

    #[test]
    fn ignores_stale_channel_results() {
        let mut popup = ChannelPopup::default();
        popup.open(ChannelInfo {
            channel_id: 7,
            ..ChannelInfo::default()
        });

        let applied = popup.apply_result(9, Ok(ChannelInfo::default()));

        assert!(!applied);
        assert_eq!(popup.requested_channel_id(), Some(7));
    }

    #[test]
    fn sorts_recordings_newest_first() {
        let mut popup = ChannelPopup::default();
        popup.open(ChannelInfo {
            channel_id: 1,
            recordings: vec![
                Recording {
                    recording_id: 1,
                    created_at: "2024-01-01T00:00:00Z".to_string(),
                    ..Recording::default()
                },
                Recording {
                    recording_id: 2,
                    created_at: "2024-01-02T00:00:00Z".to_string(),
                    ..Recording::default()
                },
            ],
            ..ChannelInfo::default()
        });

        let recordings = popup.recordings();

        assert_eq!(recordings[0].recording_id, 2);
        assert_eq!(recordings[1].recording_id, 1);
    }
}

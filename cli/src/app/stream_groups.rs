use crate::{api::ChannelInfo, app::StreamTab};
use std::cmp::Ordering;

#[derive(Debug, Clone, Copy, Default, Eq, PartialEq)]
pub struct StreamCounts {
    pub recording: usize,
    pub offline: usize,
    pub disabled: usize,
}

impl StreamCounts {
    pub const fn count_for(self, tab: StreamTab) -> usize {
        match tab {
            StreamTab::Live => self.recording,
            StreamTab::Offline => self.offline,
            StreamTab::Disabled => self.disabled,
        }
    }
}

pub fn sort_channels(mut channels: Vec<ChannelInfo>) -> Vec<ChannelInfo> {
    channels.sort_by(compare_channels);
    channels
}

pub fn stream_counts(channels: &[ChannelInfo]) -> StreamCounts {
    let mut counts = StreamCounts::default();
    for channel in channels {
        if matches_stream_tab(channel, StreamTab::Live) {
            counts.recording += 1;
        }
        if matches_stream_tab(channel, StreamTab::Offline) {
            counts.offline += 1;
        }
        if matches_stream_tab(channel, StreamTab::Disabled) {
            counts.disabled += 1;
        }
    }
    counts
}

pub fn channels_for_tab(channels: &[ChannelInfo], tab: StreamTab) -> Vec<ChannelInfo> {
    channels
        .iter()
        .filter(|channel| matches_stream_tab(channel, tab))
        .cloned()
        .collect()
}

pub fn matches_stream_tab(channel: &ChannelInfo, tab: StreamTab) -> bool {
    match tab {
        StreamTab::Live => channel.is_recording && !channel.is_terminating,
        StreamTab::Offline => !channel.is_recording && !channel.is_paused,
        StreamTab::Disabled => channel.is_paused,
    }
}

fn compare_channels(left: &ChannelInfo, right: &ChannelInfo) -> Ordering {
    left.channel_name
        .to_lowercase()
        .cmp(&right.channel_name.to_lowercase())
        .then_with(|| left.channel_id.cmp(&right.channel_id))
}

#[cfg(test)]
mod tests {
    use super::{StreamCounts, channels_for_tab, sort_channels, stream_counts};
    use crate::{api::ChannelInfo, app::StreamTab};

    fn channel(
        channel_id: u64,
        channel_name: &str,
        is_recording: bool,
        is_paused: bool,
        is_terminating: bool,
    ) -> ChannelInfo {
        ChannelInfo {
            channel_id,
            channel_name: channel_name.to_string(),
            display_name: channel_name.to_string(),
            is_recording,
            is_paused,
            is_terminating,
            ..ChannelInfo::default()
        }
    }

    #[test]
    fn sort_channels_matches_frontend_order() {
        let sorted = sort_channels(vec![
            channel(2, "bravo", false, false, false),
            channel(1, "Alpha", false, false, false),
        ]);

        assert_eq!(sorted[0].channel_name, "Alpha");
        assert_eq!(sorted[1].channel_name, "bravo");
    }

    #[test]
    fn stream_filters_match_frontend_buckets() {
        let channels = sort_channels(vec![
            channel(3, "paused", false, true, false),
            channel(2, "terminating", true, false, true),
            channel(1, "recording", true, false, false),
            channel(4, "offline", false, false, false),
        ]);

        assert_eq!(
            channels_for_tab(&channels, StreamTab::Live)
                .into_iter()
                .map(|channel| channel.channel_name)
                .collect::<Vec<_>>(),
            vec!["recording"]
        );
        assert_eq!(
            channels_for_tab(&channels, StreamTab::Offline)
                .into_iter()
                .map(|channel| channel.channel_name)
                .collect::<Vec<_>>(),
            vec!["offline"]
        );
        assert_eq!(
            channels_for_tab(&channels, StreamTab::Disabled)
                .into_iter()
                .map(|channel| channel.channel_name)
                .collect::<Vec<_>>(),
            vec!["paused"]
        );
    }

    #[test]
    fn stream_counts_follow_bucket_rules() {
        let counts = stream_counts(&sort_channels(vec![
            channel(1, "recording", true, false, false),
            channel(2, "terminating", true, false, true),
            channel(3, "offline", false, false, false),
            channel(4, "paused", false, true, false),
        ]));

        assert_eq!(
            counts,
            StreamCounts {
                recording: 1,
                offline: 1,
                disabled: 1,
            }
        );
        assert_eq!(counts.count_for(StreamTab::Live), 1);
        assert_eq!(counts.count_for(StreamTab::Offline), 1);
        assert_eq!(counts.count_for(StreamTab::Disabled), 1);
    }
}

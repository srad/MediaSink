use super::*;

pub(super) fn build_file_url(root: &str, relative: &str) -> Option<String> {
    if root.trim().is_empty() || relative.trim().is_empty() {
        return None;
    }

    let mut base = root.to_string();
    if !base.ends_with('/') {
        base.push('/');
    }

    Url::parse(&base)
        .ok()?
        .join(relative.trim_start_matches('/'))
        .ok()
        .map(|url| url.to_string())
}

pub(super) fn video_thumbnail_relative_path(video: &Recording) -> Option<String> {
    if let Some(preview) = &video.video_preview {
        let path = preview.preview_path.trim_end_matches('/');
        if path.is_empty() {
            return None;
        }
        if path.ends_with(".jpg") || path.ends_with(".jpeg") || path.ends_with(".png") {
            return Some(path.to_string());
        }
        return Some(format!("{path}/0.jpg"));
    }

    if video.channel_name.trim().is_empty() {
        None
    } else {
        Some(format!("{}/.previews/live.jpg", video.channel_name))
    }
}

pub(super) fn summarize_event(name: &str, data: &Value) -> String {
    if let Some(number) = data.as_u64() {
        return format!("{name} #{number}");
    }
    if let Some(text) = data.as_str() {
        return format!("{name} {text}");
    }
    if let Some(filename) = data.get("filename").and_then(Value::as_str) {
        return format!("{name} {filename}");
    }
    if let Some(job) = data.get("job") {
        let filename = job.get("filename").and_then(Value::as_str).unwrap_or("job");
        let task = job.get("task").and_then(Value::as_str).unwrap_or("");
        return format!("{name} {task} {filename}").trim().to_string();
    }
    format!("{name} {data}").chars().take(120).collect()
}

pub(super) fn should_clear_saved_session_on_auth_error(error: &str) -> bool {
    let normalized = error.to_ascii_lowercase();
    normalized.contains("401")
        || normalized.contains("403")
        || normalized.contains("unauthorized")
        || normalized.contains("forbidden")
        || normalized.contains("invalid token")
        || normalized.contains("token expired")
        || normalized.contains("jwt")
}

pub(super) fn persist_session_on_exit(app: &App) -> anyhow::Result<Option<String>> {
    let Some(session) = app.session.as_ref() else {
        return Ok(None);
    };

    save_authenticated_session(
        &session.base_url,
        &session.username,
        &session.token,
        (!session.runtime.api_version.trim().is_empty())
            .then(|| session.runtime.api_version.clone()),
        (!session.runtime.file_url.trim().is_empty()).then(|| session.runtime.file_url.clone()),
    )
}

pub(super) fn display_channel_name(channel: &ChannelInfo) -> String {
    if channel.display_name.trim().is_empty() {
        channel.channel_name.clone()
    } else {
        channel.display_name.clone()
    }
}

pub(super) fn channel_placeholder_accent(channel: &ChannelInfo, theme: ThemePalette) -> Color {
    if channel.is_recording {
        theme.danger
    } else if channel.is_paused {
        theme.warning
    } else if channel.is_online {
        theme.success
    } else {
        theme.accent
    }
}

pub(super) fn view_has_thumbnail_preview(view: View) -> bool {
    matches!(
        view,
        View::Channel
            | View::Streams
            | View::Channels
            | View::Latest
            | View::Random
            | View::Favourites
            | View::Similarity
    )
}

pub(super) fn average_cpu_percent(info: &UtilSysInfo) -> u64 {
    if info.cpu_info.load_cpu.is_empty() {
        return 0;
    }
    let total = info
        .cpu_info
        .load_cpu
        .iter()
        .map(|load| load.load.max(0.0))
        .sum::<f64>();
    ((total / info.cpu_info.load_cpu.len() as f64) * 100.0).round() as u64
}

pub(super) fn latest_cpu_summary(app: &App) -> String {
    app.monitor_history
        .last()
        .map(|sample| format!("{}% @ {}", sample.cpu_load_percent, sample.timestamp))
        .unwrap_or_else(|| "no samples".to_string())
}

pub(super) fn latest_rx_summary(app: &App) -> String {
    app.monitor_history
        .last()
        .map(|sample| format!("{} MB", sample.rx_megabytes))
        .unwrap_or_else(|| "no samples".to_string())
}

pub(super) fn latest_tx_summary(app: &App) -> String {
    app.monitor_history
        .last()
        .map(|sample| format!("{} MB", sample.tx_megabytes))
        .unwrap_or_else(|| "no samples".to_string())
}

pub(super) fn format_tags(tags: &Value) -> String {
    match tags {
        Value::Array(values) => values
            .iter()
            .filter_map(Value::as_str)
            .collect::<Vec<_>>()
            .join(", "),
        Value::String(text) => text.clone(),
        _ => String::new(),
    }
}

pub(super) fn yes_no(value: bool) -> &'static str {
    if value { "yes" } else { "no" }
}

pub(super) fn format_seconds(value: f64) -> String {
    if value <= 0.0 {
        return "0s".to_string();
    }
    let total = value.round() as u64;
    let hours = total / 3600;
    let minutes = (total % 3600) / 60;
    let seconds = total % 60;
    if hours > 0 {
        format!("{hours}h {minutes:02}m")
    } else if minutes > 0 {
        format!("{minutes}m {seconds:02}s")
    } else {
        format!("{seconds}s")
    }
}

pub(super) fn format_duration(value: f64) -> String {
    let total = value.max(0.0).round() as u64;
    let hours = total / 3600;
    let minutes = (total % 3600) / 60;
    let seconds = total % 60;
    if hours > 0 {
        format!("{hours:02}:{minutes:02}:{seconds:02}")
    } else {
        format!("{minutes:02}:{seconds:02}")
    }
}

pub(super) fn format_bytes(value: u64) -> String {
    if value == 0 {
        return "0 B".to_string();
    }
    let units = ["B", "KB", "MB", "GB", "TB"];
    let mut size = value as f64;
    let mut index = 0usize;
    while size >= 1024.0 && index < units.len() - 1 {
        size /= 1024.0;
        index += 1;
    }
    if size >= 100.0 || index == 0 {
        format!("{size:.0} {}", units[index])
    } else if size >= 10.0 {
        format!("{size:.1} {}", units[index])
    } else {
        format!("{size:.2} {}", units[index])
    }
}

pub(super) fn truncate(value: &str, width: usize) -> String {
    if value.chars().count() <= width {
        return value.to_string();
    }
    if width <= 1 {
        return "…".to_string();
    }
    let mut output = value.chars().take(width - 1).collect::<String>();
    output.push('…');
    output
}

pub(super) fn sanitize_filename(value: &str) -> String {
    let sanitized = value
        .chars()
        .map(|character| match character {
            '/' | '\\' | '\0' => '_',
            _ => character,
        })
        .collect::<String>();
    let trimmed = sanitized.trim();
    if trimmed.is_empty() {
        "download.bin".to_string()
    } else {
        trimmed.to_string()
    }
}

use std::process::Command;

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum PlayerMode {
    Auto,
    SmoothColor,
    SharpColor,
    AsciiGray,
    AsciiMono,
}

impl Default for PlayerMode {
    fn default() -> Self {
        Self::Auto
    }
}

#[derive(Debug, Clone, Default)]
pub struct PlayerCapabilities {
    ffmpeg_available: bool,
    supports_unsharp: bool,
    supports_eq: bool,
}

impl PlayerCapabilities {
    pub fn detect() -> Self {
        Self {
            ffmpeg_available: ffmpeg_exists(),
            supports_unsharp: ffmpeg_supports_filter("unsharp"),
            supports_eq: ffmpeg_supports_filter("eq"),
        }
    }

    pub fn has_ffmpeg(&self) -> bool {
        self.ffmpeg_available
    }

    pub fn available_modes(&self) -> Vec<PlayerMode> {
        let mut modes = Vec::new();
        for mode in PlayerMode::all() {
            if mode.is_supported(self) {
                modes.push(*mode);
            }
        }

        if modes.is_empty() {
            vec![PlayerMode::Auto]
        } else {
            modes
        }
    }

    pub fn supports(&self, mode: PlayerMode) -> bool {
        mode.is_supported(self)
    }
}

impl PlayerMode {
    pub const fn all() -> &'static [PlayerMode] {
        &[
            Self::Auto,
            Self::SmoothColor,
            Self::SharpColor,
            Self::AsciiGray,
            Self::AsciiMono,
        ]
    }

    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Auto => "auto",
            Self::SmoothColor => "smooth-color",
            Self::SharpColor => "sharp-color",
            Self::AsciiGray => "ascii-gray",
            Self::AsciiMono => "ascii-mono",
        }
    }

    pub const fn label(self) -> &'static str {
        match self {
            Self::Auto => "Auto",
            Self::SmoothColor => "Smooth Color",
            Self::SharpColor => "Sharp Color",
            Self::AsciiGray => "ASCII Gray",
            Self::AsciiMono => "ASCII Mono",
        }
    }

    pub const fn hint(self) -> &'static str {
        match self {
            Self::Auto => "Use the default popup renderer settings for this machine.",
            Self::SmoothColor => "Balanced full-color popup playback.",
            Self::SharpColor => "Color playback with stronger sharpening.",
            Self::AsciiGray => "Grayscale popup rendering for lower distraction.",
            Self::AsciiMono => "High-contrast monochrome popup rendering.",
        }
    }

    pub fn from_config(value: Option<&str>) -> Self {
        let Some(value) = value else {
            return Self::default();
        };

        match value.trim().to_ascii_lowercase().as_str() {
            "auto" => Self::Auto,
            "smooth-color" | "smooth" => Self::SmoothColor,
            "sharp-color" | "sharp" => Self::SharpColor,
            "ascii-gray" | "gray" => Self::AsciiGray,
            "ascii-mono" | "mono" => Self::AsciiMono,
            _ => Self::default(),
        }
    }

    pub fn resolved(self, capabilities: &PlayerCapabilities) -> Self {
        if capabilities.supports(self) {
            self
        } else {
            Self::Auto
        }
    }

    fn is_supported(self, capabilities: &PlayerCapabilities) -> bool {
        match self {
            Self::Auto => capabilities.has_ffmpeg(),
            Self::SmoothColor => capabilities.has_ffmpeg(),
            Self::SharpColor => capabilities.has_ffmpeg() && capabilities.supports_unsharp,
            Self::AsciiGray => capabilities.has_ffmpeg(),
            Self::AsciiMono => capabilities.has_ffmpeg() && capabilities.supports_eq,
        }
    }
}

fn ffmpeg_exists() -> bool {
    Command::new("ffmpeg")
        .args(["-hide_banner", "-version"])
        .output()
        .map(|output| output.status.success())
        .unwrap_or(false)
}

fn ffmpeg_supports_filter(filter_name: &str) -> bool {
    let Ok(output) = Command::new("ffmpeg")
        .args(["-hide_banner", "-h", &format!("filter={filter_name}")])
        .output()
    else {
        return false;
    };

    let mut text = String::from_utf8_lossy(&output.stdout).to_string();
    text.push_str(&String::from_utf8_lossy(&output.stderr));
    text.contains(&format!("Filter {filter_name}"))
}

#[cfg(test)]
mod tests {
    use super::{PlayerCapabilities, PlayerMode};

    #[test]
    fn parses_known_player_mode_names() {
        assert_eq!(
            PlayerMode::from_config(Some("smooth-color")),
            PlayerMode::SmoothColor
        );
        assert_eq!(PlayerMode::from_config(Some("mono")), PlayerMode::AsciiMono);
        assert_eq!(PlayerMode::from_config(Some("unknown")), PlayerMode::Auto);
    }

    #[test]
    fn filters_modes_by_detected_capabilities() {
        let capabilities = PlayerCapabilities {
            ffmpeg_available: true,
            supports_unsharp: true,
            supports_eq: false,
        };

        let available = capabilities.available_modes();

        assert!(available.contains(&PlayerMode::Auto));
        assert!(available.contains(&PlayerMode::SmoothColor));
        assert!(available.contains(&PlayerMode::SharpColor));
        assert!(available.contains(&PlayerMode::AsciiGray));
        assert!(!available.contains(&PlayerMode::AsciiMono));
    }
}

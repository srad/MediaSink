use ratatui::prelude::{Color, Modifier, Style};

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum ThemeBackground {
    None,
    MatrixRain,
}

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum ThemeName {
    Norton,
    Doom,
    Duke,
    Midnight,
    Quake,
    Fallout,
    Matrix,
    Amber,
}

impl Default for ThemeName {
    fn default() -> Self {
        Self::Norton
    }
}

impl ThemeName {
    pub const fn all() -> &'static [ThemeName] {
        &[
            Self::Norton,
            Self::Doom,
            Self::Duke,
            Self::Midnight,
            Self::Quake,
            Self::Fallout,
            Self::Matrix,
            Self::Amber,
        ]
    }

    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Norton => "norton",
            Self::Doom => "doom",
            Self::Duke => "duke",
            Self::Midnight => "midnight",
            Self::Quake => "quake",
            Self::Fallout => "fallout",
            Self::Matrix => "matrix",
            Self::Amber => "amber",
        }
    }

    pub const fn label(self) -> &'static str {
        match self {
            Self::Norton => "Norton Commander",
            Self::Doom => "Doom",
            Self::Duke => "Duke Nukem",
            Self::Midnight => "Midnight",
            Self::Quake => "Quake",
            Self::Fallout => "Fallout",
            Self::Matrix => "Matrix",
            Self::Amber => "Amber Terminal",
        }
    }

    pub fn from_config(value: Option<&str>) -> Self {
        let Some(value) = value else {
            return Self::default();
        };
        match value.trim().to_ascii_lowercase().as_str() {
            "norton" | "norton-commander" => Self::Norton,
            "doom" => Self::Doom,
            "duke" | "duke-nukem" => Self::Duke,
            "midnight" => Self::Midnight,
            "quake" => Self::Quake,
            "fallout" | "pip-boy" | "pipboy" => Self::Fallout,
            "matrix" => Self::Matrix,
            "amber" | "amber-terminal" => Self::Amber,
            _ => Self::default(),
        }
    }

    pub const fn background(self) -> ThemeBackground {
        match self {
            Self::Matrix => ThemeBackground::MatrixRain,
            _ => ThemeBackground::None,
        }
    }

    pub const fn palette(self) -> ThemePalette {
        match self {
            Self::Norton => ThemePalette {
                app_bg: Color::Rgb(0, 0, 152),
                surface_bg: Color::Rgb(0, 0, 152),
                surface_alt_bg: Color::Rgb(0, 72, 152),
                chrome_bg: Color::Rgb(0, 0, 152),
                border: Color::Rgb(72, 216, 216),
                border_focus: Color::Rgb(224, 224, 104),
                text: Color::Rgb(232, 232, 232),
                muted: Color::Rgb(152, 152, 152),
                accent: Color::Rgb(224, 224, 104),
                accent_soft: Color::Rgb(0, 144, 144),
                warning: Color::Rgb(224, 224, 104),
                success: Color::Rgb(96, 216, 96),
                danger: Color::Rgb(224, 104, 104),
            },
            Self::Doom => ThemePalette {
                app_bg: Color::Rgb(22, 10, 10),
                surface_bg: Color::Rgb(38, 17, 17),
                surface_alt_bg: Color::Rgb(63, 22, 18),
                chrome_bg: Color::Rgb(28, 12, 12),
                border: Color::Rgb(128, 52, 38),
                border_focus: Color::Rgb(198, 93, 64),
                text: Color::Rgb(241, 224, 204),
                muted: Color::Rgb(189, 149, 126),
                accent: Color::Rgb(219, 108, 67),
                accent_soft: Color::Rgb(123, 50, 37),
                warning: Color::Rgb(238, 169, 85),
                success: Color::Rgb(112, 150, 80),
                danger: Color::Rgb(214, 72, 58),
            },
            Self::Duke => ThemePalette {
                app_bg: Color::Rgb(17, 15, 18),
                surface_bg: Color::Rgb(29, 26, 31),
                surface_alt_bg: Color::Rgb(59, 48, 24),
                chrome_bg: Color::Rgb(21, 19, 24),
                border: Color::Rgb(136, 112, 45),
                border_focus: Color::Rgb(224, 188, 66),
                text: Color::Rgb(239, 233, 214),
                muted: Color::Rgb(188, 176, 140),
                accent: Color::Rgb(235, 195, 74),
                accent_soft: Color::Rgb(126, 98, 35),
                warning: Color::Rgb(242, 144, 60),
                success: Color::Rgb(109, 167, 108),
                danger: Color::Rgb(194, 78, 78),
            },
            Self::Midnight => ThemePalette {
                app_bg: Color::Rgb(5, 10, 22),
                surface_bg: Color::Rgb(10, 18, 36),
                surface_alt_bg: Color::Rgb(18, 34, 68),
                chrome_bg: Color::Rgb(8, 13, 28),
                border: Color::Rgb(71, 95, 153),
                border_focus: Color::Rgb(122, 172, 255),
                text: Color::Rgb(225, 233, 249),
                muted: Color::Rgb(142, 163, 203),
                accent: Color::Rgb(116, 180, 255),
                accent_soft: Color::Rgb(52, 92, 162),
                warning: Color::Rgb(215, 173, 95),
                success: Color::Rgb(67, 160, 135),
                danger: Color::Rgb(196, 91, 112),
            },
            Self::Quake => ThemePalette {
                app_bg: Color::Rgb(14, 12, 12),
                surface_bg: Color::Rgb(24, 20, 18),
                surface_alt_bg: Color::Rgb(43, 32, 26),
                chrome_bg: Color::Rgb(17, 14, 14),
                border: Color::Rgb(105, 88, 72),
                border_focus: Color::Rgb(164, 133, 102),
                text: Color::Rgb(229, 220, 205),
                muted: Color::Rgb(166, 149, 130),
                accent: Color::Rgb(179, 126, 88),
                accent_soft: Color::Rgb(95, 63, 47),
                warning: Color::Rgb(214, 160, 86),
                success: Color::Rgb(112, 146, 96),
                danger: Color::Rgb(196, 84, 68),
            },
            Self::Fallout => ThemePalette {
                app_bg: Color::Rgb(7, 12, 7),
                surface_bg: Color::Rgb(11, 22, 11),
                surface_alt_bg: Color::Rgb(19, 42, 19),
                chrome_bg: Color::Rgb(8, 17, 8),
                border: Color::Rgb(61, 126, 61),
                border_focus: Color::Rgb(109, 214, 109),
                text: Color::Rgb(169, 233, 149),
                muted: Color::Rgb(108, 167, 96),
                accent: Color::Rgb(136, 245, 121),
                accent_soft: Color::Rgb(49, 94, 49),
                warning: Color::Rgb(214, 187, 84),
                success: Color::Rgb(129, 231, 130),
                danger: Color::Rgb(204, 111, 92),
            },
            Self::Matrix => ThemePalette {
                app_bg: Color::Rgb(1, 6, 1),
                surface_bg: Color::Rgb(4, 14, 4),
                surface_alt_bg: Color::Rgb(8, 24, 8),
                chrome_bg: Color::Rgb(2, 10, 2),
                border: Color::Rgb(38, 110, 46),
                border_focus: Color::Rgb(88, 236, 110),
                text: Color::Rgb(172, 246, 180),
                muted: Color::Rgb(90, 170, 105),
                accent: Color::Rgb(102, 255, 126),
                accent_soft: Color::Rgb(30, 82, 39),
                warning: Color::Rgb(199, 187, 90),
                success: Color::Rgb(117, 255, 146),
                danger: Color::Rgb(208, 92, 92),
            },
            Self::Amber => ThemePalette {
                app_bg: Color::Rgb(16, 10, 3),
                surface_bg: Color::Rgb(27, 17, 5),
                surface_alt_bg: Color::Rgb(44, 28, 8),
                chrome_bg: Color::Rgb(20, 13, 4),
                border: Color::Rgb(150, 98, 28),
                border_focus: Color::Rgb(255, 176, 68),
                text: Color::Rgb(255, 218, 151),
                muted: Color::Rgb(203, 157, 92),
                accent: Color::Rgb(255, 186, 73),
                accent_soft: Color::Rgb(118, 73, 21),
                warning: Color::Rgb(255, 205, 94),
                success: Color::Rgb(141, 190, 110),
                danger: Color::Rgb(214, 109, 66),
            },
        }
    }
}

#[derive(Debug, Clone, Copy)]
pub struct ThemePalette {
    pub app_bg: Color,
    pub surface_bg: Color,
    pub surface_alt_bg: Color,
    pub chrome_bg: Color,
    pub border: Color,
    pub border_focus: Color,
    pub text: Color,
    pub muted: Color,
    pub accent: Color,
    pub accent_soft: Color,
    pub warning: Color,
    pub success: Color,
    pub danger: Color,
}

impl ThemePalette {
    pub fn app_style(self) -> Style {
        Style::default().fg(self.text).bg(self.app_bg)
    }

    pub fn surface_style(self) -> Style {
        Style::default().fg(self.text).bg(self.surface_bg)
    }

    pub fn surface_alt_style(self) -> Style {
        Style::default().fg(self.text).bg(self.surface_alt_bg)
    }

    pub fn chrome_style(self) -> Style {
        Style::default().fg(self.muted).bg(self.chrome_bg)
    }

    pub fn footer_key_style(self) -> Style {
        Style::default()
            .fg(self.app_bg)
            .bg(self.border_focus)
            .add_modifier(Modifier::BOLD)
    }

    pub fn footer_label_style(self) -> Style {
        Style::default()
            .fg(self.text)
            .bg(self.chrome_bg)
            .add_modifier(Modifier::BOLD)
    }

    pub fn footer_separator_style(self) -> Style {
        Style::default().fg(self.muted).bg(self.chrome_bg)
    }

    pub fn title_style(self) -> Style {
        Style::default()
            .fg(self.accent)
            .add_modifier(Modifier::BOLD)
    }

    pub fn subtitle_style(self) -> Style {
        Style::default().fg(self.muted)
    }

    pub fn panel_border_style(self) -> Style {
        Style::default().fg(self.border)
    }

    pub fn row_style(self, selected: bool) -> Style {
        if selected {
            Style::default()
                .fg(self.text)
                .bg(self.surface_alt_bg)
                .add_modifier(Modifier::BOLD)
        } else {
            self.surface_style()
        }
    }

    pub fn tab_style(self) -> Style {
        self.chrome_style()
    }

    pub fn tab_highlight_style(self) -> Style {
        Style::default()
            .fg(self.text)
            .bg(self.surface_alt_bg)
            .add_modifier(Modifier::BOLD)
    }

    pub fn input_border_style(self, selected: bool) -> Style {
        if selected {
            Style::default()
                .fg(self.border_focus)
                .bg(self.surface_alt_bg)
        } else {
            Style::default().fg(self.border).bg(self.surface_bg)
        }
    }

    pub fn input_text_style(self, selected: bool) -> Style {
        if selected {
            self.surface_alt_style()
        } else {
            self.surface_style()
        }
    }

    pub fn notice_style(self, tone: Color) -> Style {
        Style::default().fg(tone).bg(self.surface_bg)
    }

    pub fn table_header_style(self) -> Style {
        Style::default()
            .fg(self.text)
            .bg(self.surface_alt_bg)
            .add_modifier(Modifier::BOLD)
    }

    pub fn selection_style(self) -> Style {
        Style::default()
            .fg(self.text)
            .bg(self.surface_alt_bg)
            .add_modifier(Modifier::BOLD)
    }

    pub fn chip_style(self, background: Color) -> Style {
        Style::default()
            .fg(self.text)
            .bg(background)
            .add_modifier(Modifier::BOLD)
    }

    pub fn event_time_style(self) -> Style {
        Style::default().fg(self.warning)
    }

    pub fn event_name_style(self) -> Style {
        Style::default().fg(self.accent)
    }
}

#[cfg(test)]
mod tests {
    use super::ThemeName;

    #[test]
    fn defaults_to_norton_theme() {
        assert_eq!(ThemeName::default(), ThemeName::Norton);
        assert_eq!(ThemeName::from_config(None), ThemeName::Norton);
    }

    #[test]
    fn parses_known_theme_names() {
        assert_eq!(ThemeName::from_config(Some("doom")), ThemeName::Doom);
        assert_eq!(ThemeName::from_config(Some("duke-nukem")), ThemeName::Duke);
        assert_eq!(ThemeName::from_config(Some("quake")), ThemeName::Quake);
        assert_eq!(ThemeName::from_config(Some("pip-boy")), ThemeName::Fallout);
        assert_eq!(ThemeName::from_config(Some("matrix")), ThemeName::Matrix);
        assert_eq!(
            ThemeName::from_config(Some("amber-terminal")),
            ThemeName::Amber
        );
        assert_eq!(
            ThemeName::from_config(Some("norton-commander")),
            ThemeName::Norton
        );
    }
}

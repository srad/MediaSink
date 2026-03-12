#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum HelpContext {
    Login,
    Workspace,
    VideoPlayer,
}

impl Default for HelpContext {
    fn default() -> Self {
        Self::Workspace
    }
}

#[derive(Debug, Clone, Copy)]
pub struct HelpSection {
    pub title: &'static str,
    pub lines: &'static [&'static str],
}

const LOGIN_SECTIONS: &[HelpSection] = &[
    HelpSection {
        title: "Login",
        lines: &[
            "Enter server URL, login, and password, then press Enter to sign in.",
            "F2 switches between Login and Register mode.",
            "Saved credentials are reused automatically when available.",
        ],
    },
    HelpSection {
        title: "Editing",
        lines: &[
            "Tab and Shift+Tab move between fields.",
            "Ctrl+A selects all. Ctrl+C copies. Ctrl+X cuts. Ctrl+V pastes.",
            "Esc cancels auto-login or closes the current popup.",
        ],
    },
    HelpSection {
        title: "Mouse",
        lines: &[
            "Click fields and buttons directly when mouse mode is enabled.",
            "F6 toggles mouse support on or off.",
        ],
    },
    HelpSection {
        title: "Global Keys",
        lines: &[
            "F1 opens this help. F3 opens themes. F5 opens player mode presets.",
            "F10 quits the TUI.",
        ],
    },
];

const WORKSPACE_SECTIONS: &[HelpSection] = &[
    HelpSection {
        title: "Navigation",
        lines: &[
            "Left and Right switch primary tabs. Up and Down move through rows.",
            "Enter on a channel opens its recordings popup.",
            "Enter on a recording opens the popup video player.",
        ],
    },
    HelpSection {
        title: "Actions",
        lines: &[
            "N opens Add Stream. L logs out. R starts or stops recording.",
            "F4 opens the item action menu for the selected stream, channel, or recording.",
            "Most destructive actions show a confirmation dialog first.",
        ],
    },
    HelpSection {
        title: "Video Player",
        lines: &[
            "Space toggles play and pause.",
            "Left and Right seek 5 seconds. PgUp and PgDn seek 30 seconds.",
            "Home and End jump to start or end. Esc closes the player.",
        ],
    },
    HelpSection {
        title: "Mouse",
        lines: &[
            "Click tabs, rows, header buttons, popup entries, and form fields.",
            "Mouse wheel scrolls lists and this help popup.",
            "Click the video seek bar to jump to a position.",
        ],
    },
    HelpSection {
        title: "Forms",
        lines: &[
            "Tab moves between fields. Left and Right adjust selector-style values.",
            "Ctrl+A, Ctrl+C, Ctrl+X, and Ctrl+V work in text fields.",
        ],
    },
    HelpSection {
        title: "Function Keys",
        lines: &["F1 Help, F3 Theme, F4 Menu, F5 Player Mode, F6 Mouse."],
    },
];

const VIDEO_PLAYER_SECTIONS: &[HelpSection] = &[
    HelpSection {
        title: "Playback",
        lines: &[
            "Space toggles play and pause.",
            "Left and Right seek 5 seconds. PgUp and PgDn seek 30 seconds.",
            "Home and End jump to the start or end of the recording.",
        ],
    },
    HelpSection {
        title: "Mouse",
        lines: &[
            "Click the seek bar to jump to a time index.",
            "Esc closes the popup player and returns to the list.",
        ],
    },
    HelpSection {
        title: "Player Modes",
        lines: &["F5 opens the player render-mode picker for this machine."],
    },
];

#[derive(Debug, Clone, Default)]
pub struct HelpPopup {
    context: HelpContext,
    open: bool,
    scroll: u16,
}

impl HelpPopup {
    pub fn open(&mut self, context: HelpContext) {
        self.context = context;
        self.open = true;
        self.scroll = 0;
    }

    pub fn close(&mut self) {
        self.open = false;
    }

    pub fn is_open(&self) -> bool {
        self.open
    }

    pub fn context(&self) -> HelpContext {
        self.context
    }

    pub fn scroll(&self) -> u16 {
        self.scroll
    }

    pub fn scroll_by(&mut self, delta: i16, max_scroll: u16) {
        if delta < 0 {
            self.scroll = self.scroll.saturating_sub(delta.unsigned_abs());
        } else {
            self.scroll = self.scroll.saturating_add(delta as u16).min(max_scroll);
        }
    }

    pub fn page_by(&mut self, delta: i16, page_height: u16, max_scroll: u16) {
        let page = page_height.max(1) as i16;
        self.scroll_by(delta.saturating_mul(page), max_scroll);
    }

    pub fn scroll_to_top(&mut self) {
        self.scroll = 0;
    }

    pub fn scroll_to_bottom(&mut self, max_scroll: u16) {
        self.scroll = max_scroll;
    }
}

pub fn help_sections(context: HelpContext) -> &'static [HelpSection] {
    match context {
        HelpContext::Login => LOGIN_SECTIONS,
        HelpContext::Workspace => WORKSPACE_SECTIONS,
        HelpContext::VideoPlayer => VIDEO_PLAYER_SECTIONS,
    }
}

#[cfg(test)]
mod tests {
    use super::{HelpContext, HelpPopup, help_sections};

    #[test]
    fn opens_with_context_and_resets_scroll() {
        let mut popup = HelpPopup::default();
        popup.open(HelpContext::VideoPlayer);
        popup.scroll_by(9, 20);

        popup.open(HelpContext::Login);

        assert!(popup.is_open());
        assert_eq!(popup.context(), HelpContext::Login);
        assert_eq!(popup.scroll(), 0);
    }

    #[test]
    fn scrolling_clamps_to_bounds() {
        let mut popup = HelpPopup::default();
        popup.open(HelpContext::Workspace);

        popup.scroll_by(5, 7);
        assert_eq!(popup.scroll(), 5);

        popup.scroll_by(9, 7);
        assert_eq!(popup.scroll(), 7);

        popup.scroll_by(-3, 7);
        assert_eq!(popup.scroll(), 4);

        popup.scroll_to_bottom(6);
        assert_eq!(popup.scroll(), 6);

        popup.scroll_to_top();
        assert_eq!(popup.scroll(), 0);
    }

    #[test]
    fn each_context_has_help_content() {
        for context in [
            HelpContext::Login,
            HelpContext::Workspace,
            HelpContext::VideoPlayer,
        ] {
            assert!(!help_sections(context).is_empty());
        }
    }
}

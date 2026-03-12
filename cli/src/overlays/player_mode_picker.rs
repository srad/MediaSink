use crate::player_mode::PlayerMode;

#[derive(Debug, Clone, Default)]
pub struct PlayerModePicker {
    open: bool,
    selected: usize,
}

impl PlayerModePicker {
    pub fn open(&mut self, current: PlayerMode, modes: &[PlayerMode]) {
        self.open = true;
        self.selected = modes.iter().position(|mode| *mode == current).unwrap_or(0);
    }

    pub fn close(&mut self) {
        self.open = false;
    }

    pub fn is_open(&self) -> bool {
        self.open
    }

    pub fn move_selection(&mut self, delta: isize, count: usize) {
        if count == 0 {
            self.selected = 0;
            return;
        }

        if delta < 0 {
            self.selected = if self.selected == 0 {
                count - 1
            } else {
                self.selected - 1
            };
        } else if self.selected + 1 >= count {
            self.selected = 0;
        } else {
            self.selected += 1;
        }
    }

    pub fn selected_mode(&self, modes: &[PlayerMode]) -> PlayerMode {
        modes.get(self.selected).copied().unwrap_or_default()
    }

    pub fn set_selected_index(&mut self, index: usize, count: usize) {
        self.selected = index.min(count.saturating_sub(1));
    }
}

#[cfg(test)]
mod tests {
    use super::PlayerModePicker;
    use crate::player_mode::PlayerMode;

    #[test]
    fn opens_on_current_mode() {
        let mut picker = PlayerModePicker::default();
        let modes = [
            PlayerMode::Auto,
            PlayerMode::SharpColor,
            PlayerMode::AsciiMono,
        ];

        picker.open(PlayerMode::SharpColor, &modes);

        assert!(picker.is_open());
        assert_eq!(picker.selected_mode(&modes), PlayerMode::SharpColor);
    }

    #[test]
    fn wraps_selection() {
        let mut picker = PlayerModePicker::default();
        let modes = [PlayerMode::Auto, PlayerMode::SharpColor];

        picker.open(PlayerMode::Auto, &modes);
        picker.move_selection(-1, modes.len());
        assert_eq!(picker.selected_mode(&modes), PlayerMode::SharpColor);
        picker.move_selection(1, modes.len());
        assert_eq!(picker.selected_mode(&modes), PlayerMode::Auto);
    }
}

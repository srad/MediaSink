use crate::ui::ThemeName;

#[derive(Debug, Clone, Default)]
pub struct ThemePicker {
    open: bool,
    original: ThemeName,
    selected: usize,
}

impl ThemePicker {
    pub fn open(&mut self, current: ThemeName) {
        self.open = true;
        self.original = current;
        self.selected = ThemeName::all()
            .iter()
            .position(|theme| *theme == current)
            .unwrap_or(0);
    }

    pub fn close(&mut self) {
        self.open = false;
    }

    pub fn is_open(&self) -> bool {
        self.open
    }

    pub fn original_theme(&self) -> ThemeName {
        self.original
    }

    pub fn move_selection(&mut self, delta: isize) {
        let count = ThemeName::all().len();
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

    pub fn selected_theme(&self) -> ThemeName {
        ThemeName::all()
            .get(self.selected)
            .copied()
            .unwrap_or_default()
    }

    pub fn set_selected_index(&mut self, index: usize) {
        self.selected = index.min(ThemeName::all().len().saturating_sub(1));
    }
}

#[cfg(test)]
mod tests {
    use super::ThemePicker;
    use crate::ui::ThemeName;

    #[test]
    fn opens_on_current_theme() {
        let mut picker = ThemePicker::default();
        picker.open(ThemeName::Duke);

        assert!(picker.is_open());
        assert_eq!(picker.original_theme(), ThemeName::Duke);
        assert_eq!(picker.selected_theme(), ThemeName::Duke);
    }

    #[test]
    fn wraps_selection() {
        let mut picker = ThemePicker::default();
        picker.open(ThemeName::Norton);
        picker.move_selection(-1);
        assert_eq!(
            picker.selected_theme(),
            *ThemeName::all().last().unwrap_or(&ThemeName::Norton)
        );
        picker.move_selection(1);
        assert_eq!(picker.selected_theme(), ThemeName::Norton);
    }
}

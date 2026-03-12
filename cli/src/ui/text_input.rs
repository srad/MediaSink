use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};
use ratatui::{prelude::Style, text::Span};

#[derive(Debug, Clone, Eq, PartialEq)]
pub struct TextInput {
    anchor: Option<usize>,
    cursor: usize,
    value: String,
}

#[derive(Debug, Clone, Eq, PartialEq)]
pub enum TextInputAction {
    Copied(String),
    Handled,
    Ignored,
}

impl TextInput {
    pub fn new(value: impl Into<String>) -> Self {
        let value = value.into();
        let cursor = value.chars().count();
        Self {
            anchor: None,
            cursor,
            value,
        }
    }

    pub fn clear(&mut self) {
        self.value.clear();
        self.cursor = 0;
        self.anchor = None;
    }

    pub fn paste(&mut self, text: &str) -> bool {
        self.replace_selection(text);
        true
    }

    pub fn display_text(&self, masked: bool) -> String {
        if masked {
            "*".repeat(self.value.chars().count())
        } else {
            self.value.clone()
        }
    }

    pub fn selection_range(&self) -> Option<(usize, usize)> {
        let anchor = self.anchor?;
        let start = anchor.min(self.cursor);
        let end = anchor.max(self.cursor);
        Some((start, end))
    }

    pub fn set_text(&mut self, value: impl Into<String>) {
        self.value = value.into();
        self.cursor = self.value.chars().count();
        self.anchor = None;
    }

    pub fn text(&self) -> &str {
        &self.value
    }

    pub fn value_spans(
        &self,
        masked: bool,
        normal_style: Style,
        selection_style: Style,
        cursor_style: Style,
    ) -> Vec<Span<'static>> {
        let display = self.display_text(masked);
        let len = display.chars().count();
        if let Some((start, end)) = self.selection_range().filter(|(start, end)| start < end) {
            let before = slice_chars(&display, 0, start);
            let selected = slice_chars(&display, start, end);
            let after = slice_chars(&display, end, len);
            let mut spans = Vec::new();
            if !before.is_empty() {
                spans.push(Span::styled(before, normal_style));
            }
            if !selected.is_empty() {
                spans.push(Span::styled(selected, selection_style));
            }
            if !after.is_empty() {
                spans.push(Span::styled(after, normal_style));
            }
            if spans.is_empty() {
                spans.push(Span::styled(" ", cursor_style));
            }
            return spans;
        }

        let before = slice_chars(&display, 0, self.cursor.min(len));
        let cursor_char = if self.cursor < len {
            slice_chars(&display, self.cursor, self.cursor + 1)
        } else {
            " ".to_string()
        };
        let after = if self.cursor < len {
            slice_chars(&display, self.cursor + 1, len)
        } else {
            String::new()
        };
        let mut spans = Vec::new();
        if !before.is_empty() {
            spans.push(Span::styled(before, normal_style));
        }
        spans.push(Span::styled(cursor_char, cursor_style));
        if !after.is_empty() {
            spans.push(Span::styled(after, normal_style));
        }
        spans
    }

    pub fn handle_key(&mut self, key: KeyEvent, clipboard: &str) -> TextInputAction {
        let modifiers = key.modifiers;
        let control_shortcut =
            modifiers.contains(KeyModifiers::CONTROL) && !modifiers.contains(KeyModifiers::ALT);
        match key.code {
            KeyCode::Left => {
                self.move_cursor_left(modifiers.contains(KeyModifiers::SHIFT));
                TextInputAction::Handled
            }
            KeyCode::Right => {
                self.move_cursor_right(modifiers.contains(KeyModifiers::SHIFT));
                TextInputAction::Handled
            }
            KeyCode::Home => {
                self.move_cursor_to(0, modifiers.contains(KeyModifiers::SHIFT));
                TextInputAction::Handled
            }
            KeyCode::End => {
                self.move_cursor_to(
                    self.value.chars().count(),
                    modifiers.contains(KeyModifiers::SHIFT),
                );
                TextInputAction::Handled
            }
            KeyCode::Backspace => {
                self.backspace();
                TextInputAction::Handled
            }
            KeyCode::Delete => {
                self.delete();
                TextInputAction::Handled
            }
            KeyCode::Insert if modifiers.contains(KeyModifiers::SHIFT) => {
                self.replace_selection(clipboard);
                TextInputAction::Handled
            }
            KeyCode::Char('a') if control_shortcut => {
                self.anchor = Some(0);
                self.cursor = self.value.chars().count();
                TextInputAction::Handled
            }
            KeyCode::Char('c') if control_shortcut => {
                if let Some(selection) = self.selected_text() {
                    TextInputAction::Copied(selection)
                } else {
                    TextInputAction::Ignored
                }
            }
            KeyCode::Char('x') if control_shortcut => {
                if let Some(selection) = self.delete_selection() {
                    TextInputAction::Copied(selection)
                } else {
                    TextInputAction::Ignored
                }
            }
            KeyCode::Char('v') if control_shortcut => {
                self.replace_selection(clipboard);
                TextInputAction::Handled
            }
            KeyCode::Char(character) => {
                if control_shortcut {
                    return TextInputAction::Ignored;
                }
                self.replace_selection(&character.to_string());
                TextInputAction::Handled
            }
            _ => TextInputAction::Ignored,
        }
    }

    fn backspace(&mut self) {
        if self.delete_selection().is_some() {
            return;
        }
        if self.cursor == 0 {
            return;
        }
        let start = self.cursor - 1;
        self.delete_char_range(start, self.cursor);
        self.cursor = start;
    }

    fn delete(&mut self) {
        if self.delete_selection().is_some() {
            return;
        }
        if self.cursor >= self.value.chars().count() {
            return;
        }
        self.delete_char_range(self.cursor, self.cursor + 1);
    }

    fn delete_char_range(&mut self, start: usize, end: usize) {
        let start_byte = char_to_byte_index(&self.value, start);
        let end_byte = char_to_byte_index(&self.value, end);
        self.value.replace_range(start_byte..end_byte, "");
        self.anchor = None;
    }

    fn delete_selection(&mut self) -> Option<String> {
        let (start, end) = self.selection_range()?;
        if start == end {
            self.anchor = None;
            return None;
        }
        let selected = slice_chars(&self.value, start, end);
        self.delete_char_range(start, end);
        self.cursor = start;
        Some(selected)
    }

    fn move_cursor_left(&mut self, extend_selection: bool) {
        let next = self.cursor.saturating_sub(1);
        self.move_cursor_to(next, extend_selection);
    }

    fn move_cursor_right(&mut self, extend_selection: bool) {
        let next = (self.cursor + 1).min(self.value.chars().count());
        self.move_cursor_to(next, extend_selection);
    }

    fn move_cursor_to(&mut self, next: usize, extend_selection: bool) {
        if extend_selection {
            if self.anchor.is_none() {
                self.anchor = Some(self.cursor);
            }
        } else {
            self.anchor = None;
        }
        self.cursor = next.min(self.value.chars().count());
    }

    fn replace_selection(&mut self, replacement: &str) {
        let (start, end) = self.selection_range().unwrap_or((self.cursor, self.cursor));
        let start_byte = char_to_byte_index(&self.value, start);
        let end_byte = char_to_byte_index(&self.value, end);
        self.value.replace_range(start_byte..end_byte, replacement);
        self.cursor = start + replacement.chars().count();
        self.anchor = None;
    }

    fn selected_text(&self) -> Option<String> {
        let (start, end) = self.selection_range()?;
        if start == end {
            return None;
        }
        Some(slice_chars(&self.value, start, end))
    }
}

fn char_to_byte_index(value: &str, char_index: usize) -> usize {
    if char_index == 0 {
        return 0;
    }
    value
        .char_indices()
        .nth(char_index)
        .map(|(index, _)| index)
        .unwrap_or(value.len())
}

fn slice_chars(value: &str, start: usize, end: usize) -> String {
    value
        .chars()
        .skip(start)
        .take(end.saturating_sub(start))
        .collect()
}

#[cfg(test)]
mod tests {
    use super::{TextInput, TextInputAction};
    use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};

    #[test]
    fn ctrl_a_and_copy_returns_selected_text() {
        let mut input = TextInput::new("hello");
        assert_eq!(
            input.handle_key(KeyEvent::new(KeyCode::Char('a'), KeyModifiers::CONTROL), ""),
            TextInputAction::Handled
        );
        assert_eq!(
            input.handle_key(KeyEvent::new(KeyCode::Char('c'), KeyModifiers::CONTROL), ""),
            TextInputAction::Copied("hello".to_string())
        );
    }

    #[test]
    fn ctrl_v_replaces_selection() {
        let mut input = TextInput::new("hello");
        input.handle_key(KeyEvent::new(KeyCode::Home, KeyModifiers::NONE), "");
        input.handle_key(KeyEvent::new(KeyCode::Right, KeyModifiers::SHIFT), "");
        input.handle_key(KeyEvent::new(KeyCode::Right, KeyModifiers::SHIFT), "");
        input.handle_key(
            KeyEvent::new(KeyCode::Char('v'), KeyModifiers::CONTROL),
            "XY",
        );

        assert_eq!(input.text(), "XYllo");
    }

    #[test]
    fn altgr_printable_character_is_inserted() {
        let mut input = TextInput::new("");

        assert_eq!(
            input.handle_key(
                KeyEvent::new(KeyCode::Char('@'), KeyModifiers::CONTROL | KeyModifiers::ALT),
                "",
            ),
            TextInputAction::Handled
        );

        assert_eq!(input.text(), "@");
    }
}

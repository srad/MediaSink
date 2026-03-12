use crate::{
    api::{ChannelInfo, ChannelRequest},
    ui::{TextInput, TextInputAction},
};
use crossterm::event::{KeyCode, KeyEvent};
use regex::Regex;
use serde_json::Value;
use url::Url;

const CHANNEL_NAME_PATTERN: &str = r"^[_a-z0-9]+$";
const DISPLAY_NAME_PATTERN: &str = r"^[^\s\\]+(\s[^\s\\]+)*$";
const TAG_PATTERN: &str = r"^[0-9a-z]+(-*[0-9a-z]+)*$";

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum ChannelEditorField {
    Url,
    DisplayName,
    ChannelName,
    MinDuration,
    SkipStart,
    Paused,
    Tags,
    Save,
    Cancel,
}

#[derive(Debug, Clone)]
pub struct ChannelEditorSubmit {
    pub channel_id: Option<u64>,
    pub request: ChannelRequest,
}

#[derive(Debug, Clone)]
pub enum ChannelEditorEvent {
    Close,
    Copied(String),
    None,
    Submit(ChannelEditorSubmit),
}

#[derive(Debug, Clone)]
pub struct ChannelEditorState {
    channel_id: Option<u64>,
    channel_name: TextInput,
    channel_name_editable: bool,
    deleted: bool,
    display_name: TextInput,
    error: Option<String>,
    fav: bool,
    is_paused: bool,
    min_duration: TextInput,
    selected: ChannelEditorField,
    skip_start: TextInput,
    tags: TextInput,
    url: TextInput,
}

impl ChannelEditorState {
    pub fn from_channel(channel: &ChannelInfo) -> Self {
        Self {
            channel_id: Some(channel.channel_id),
            channel_name: TextInput::new(channel.channel_name.clone()),
            channel_name_editable: false,
            deleted: channel.deleted,
            display_name: TextInput::new(channel.display_name.clone()),
            error: None,
            fav: channel.fav,
            is_paused: channel.is_paused,
            min_duration: TextInput::new(
                (channel.min_duration.round().max(0.0) as u64).to_string(),
            ),
            selected: ChannelEditorField::Url,
            skip_start: TextInput::new((channel.skip_start.round().max(0.0) as u64).to_string()),
            tags: TextInput::new(tags_to_csv(&channel.tags)),
            url: TextInput::new(channel.url.clone()),
        }
    }

    pub fn new_stream() -> Self {
        Self {
            channel_id: None,
            channel_name: TextInput::new(String::new()),
            channel_name_editable: true,
            deleted: false,
            display_name: TextInput::new(String::new()),
            error: None,
            fav: false,
            is_paused: false,
            min_duration: TextInput::new("20"),
            selected: ChannelEditorField::Url,
            skip_start: TextInput::new("0"),
            tags: TextInput::new(String::new()),
            url: TextInput::new(String::new()),
        }
    }

    pub fn channel_id(&self) -> Option<u64> {
        self.channel_id
    }

    pub fn channel_name(&self) -> &str {
        self.channel_name.text()
    }

    pub fn channel_name_editable(&self) -> bool {
        self.channel_name_editable
    }

    pub fn error(&self) -> Option<&str> {
        self.error.as_deref()
    }

    pub fn field_value(&self, field: ChannelEditorField) -> String {
        match field {
            ChannelEditorField::Url => self.url.display_text(false),
            ChannelEditorField::DisplayName => self.display_name.display_text(false),
            ChannelEditorField::ChannelName => self.channel_name.display_text(false),
            ChannelEditorField::MinDuration => self.min_duration.display_text(false),
            ChannelEditorField::SkipStart => self.skip_start.display_text(false),
            ChannelEditorField::Paused => {
                if self.is_paused {
                    "yes".to_string()
                } else {
                    "no".to_string()
                }
            }
            ChannelEditorField::Tags => self.tags.display_text(false),
            ChannelEditorField::Save => {
                if self.channel_id.is_some() {
                    "Save changes".to_string()
                } else {
                    "Create stream".to_string()
                }
            }
            ChannelEditorField::Cancel => "Cancel".to_string(),
        }
    }

    pub fn input(&self, field: ChannelEditorField) -> Option<&TextInput> {
        match field {
            ChannelEditorField::Url => Some(&self.url),
            ChannelEditorField::DisplayName => Some(&self.display_name),
            ChannelEditorField::ChannelName if self.channel_name_editable => {
                Some(&self.channel_name)
            }
            ChannelEditorField::MinDuration => Some(&self.min_duration),
            ChannelEditorField::SkipStart => Some(&self.skip_start),
            ChannelEditorField::Tags => Some(&self.tags),
            ChannelEditorField::ChannelName
            | ChannelEditorField::Paused
            | ChannelEditorField::Save
            | ChannelEditorField::Cancel => None,
        }
    }

    pub fn selected(&self) -> ChannelEditorField {
        self.selected
    }

    pub fn set_selected(&mut self, field: ChannelEditorField) {
        self.selected = field;
        self.error = None;
    }

    pub fn fields() -> &'static [ChannelEditorField] {
        &[
            ChannelEditorField::Url,
            ChannelEditorField::DisplayName,
            ChannelEditorField::ChannelName,
            ChannelEditorField::MinDuration,
            ChannelEditorField::SkipStart,
            ChannelEditorField::Paused,
            ChannelEditorField::Tags,
            ChannelEditorField::Save,
            ChannelEditorField::Cancel,
        ]
    }

    pub fn handle_key(&mut self, key: KeyEvent, clipboard: &str) -> ChannelEditorEvent {
        self.error = None;
        match key.code {
            KeyCode::Esc => ChannelEditorEvent::Close,
            KeyCode::Tab | KeyCode::Down => {
                self.move_selection(1);
                ChannelEditorEvent::None
            }
            KeyCode::BackTab | KeyCode::Up => {
                self.move_selection(-1);
                ChannelEditorEvent::None
            }
            KeyCode::Left | KeyCode::Right | KeyCode::Char(' ')
                if self.selected == ChannelEditorField::Paused =>
            {
                self.is_paused = !self.is_paused;
                ChannelEditorEvent::None
            }
            KeyCode::Enter => match self.selected {
                ChannelEditorField::Paused => {
                    self.is_paused = !self.is_paused;
                    ChannelEditorEvent::None
                }
                ChannelEditorField::Save => match self.build_request() {
                    Ok(request) => ChannelEditorEvent::Submit(ChannelEditorSubmit {
                        channel_id: self.channel_id,
                        request,
                    }),
                    Err(error) => {
                        self.error = Some(error);
                        ChannelEditorEvent::None
                    }
                },
                ChannelEditorField::Cancel => ChannelEditorEvent::Close,
                _ => {
                    self.move_selection(1);
                    ChannelEditorEvent::None
                }
            },
            _ => self.apply_text_input_action(key, clipboard),
        }
    }

    pub fn paste(&mut self, text: &str) {
        self.error = None;
        if let Some(input) = self.selected_input_mut() {
            input.paste(text);
        }
    }

    fn build_request(&self) -> Result<ChannelRequest, String> {
        let url = self.url.text().trim();
        let display_name = self.display_name.text().trim();
        let channel_name = self.channel_name.text().trim();
        let min_duration = self
            .min_duration
            .text()
            .trim()
            .parse::<u64>()
            .map_err(|_| "Minimum duration must be a whole number.".to_string())?;
        let skip_start = self
            .skip_start
            .text()
            .trim()
            .parse::<u64>()
            .map_err(|_| "Skip start must be a whole number.".to_string())?;
        let tags = parse_tags(self.tags.text())?;

        let display_name_re =
            Regex::new(DISPLAY_NAME_PATTERN).map_err(|error| format!("regex error: {error}"))?;
        let channel_name_re =
            Regex::new(CHANNEL_NAME_PATTERN).map_err(|error| format!("regex error: {error}"))?;
        let parsed_url = Url::parse(url)
            .map_err(|_| "URL must be a valid http:// or https:// address.".to_string())?;
        if parsed_url.scheme() != "http" && parsed_url.scheme() != "https" {
            return Err("URL must use http or https.".to_string());
        }
        if display_name.is_empty() || !display_name_re.is_match(display_name) {
            return Err(
                "Display name cannot have leading, trailing, or repeated spaces.".to_string(),
            );
        }
        if channel_name.is_empty() || !channel_name_re.is_match(channel_name) {
            return Err(
                "Channel name must use only letters, numbers, and underscores.".to_string(),
            );
        }

        Ok(ChannelRequest {
            channel_name: channel_name.to_string(),
            deleted: self.deleted,
            display_name: display_name.to_string(),
            fav: self.fav,
            is_paused: self.is_paused,
            min_duration,
            skip_start,
            tags: Some(tags),
            url: url.to_string(),
        })
    }

    fn apply_text_input_action(&mut self, key: KeyEvent, clipboard: &str) -> ChannelEditorEvent {
        let Some(input) = self.selected_input_mut() else {
            return ChannelEditorEvent::None;
        };
        match input.handle_key(key, clipboard) {
            TextInputAction::Copied(text) => ChannelEditorEvent::Copied(text),
            TextInputAction::Handled | TextInputAction::Ignored => ChannelEditorEvent::None,
        }
    }

    fn selected_input_mut(&mut self) -> Option<&mut TextInput> {
        match self.selected {
            ChannelEditorField::Url => Some(&mut self.url),
            ChannelEditorField::DisplayName => Some(&mut self.display_name),
            ChannelEditorField::ChannelName if self.channel_name_editable => {
                Some(&mut self.channel_name)
            }
            ChannelEditorField::MinDuration => Some(&mut self.min_duration),
            ChannelEditorField::SkipStart => Some(&mut self.skip_start),
            ChannelEditorField::Tags => Some(&mut self.tags),
            ChannelEditorField::ChannelName
            | ChannelEditorField::Paused
            | ChannelEditorField::Save
            | ChannelEditorField::Cancel => None,
        }
    }

    fn move_selection(&mut self, delta: isize) {
        let fields = Self::fields();
        let current = fields
            .iter()
            .position(|field| *field == self.selected)
            .unwrap_or(0);
        let next = if delta < 0 {
            if current == 0 {
                fields.len() - 1
            } else {
                current - 1
            }
        } else if current + 1 >= fields.len() {
            0
        } else {
            current + 1
        };
        self.selected = fields[next];
    }
}

fn parse_tags(raw: &str) -> Result<Vec<String>, String> {
    let tag_re = Regex::new(TAG_PATTERN).map_err(|error| format!("regex error: {error}"))?;
    let mut tags = Vec::new();
    for token in raw.split(',') {
        let tag = token.trim();
        if tag.is_empty() {
            continue;
        }
        if !tag_re.is_match(tag) {
            return Err(format!("Illegal tag: {tag}"));
        }
        tags.push(tag.to_string());
    }
    Ok(tags)
}

fn tags_to_csv(value: &Value) -> String {
    match value {
        Value::Array(items) => items
            .iter()
            .filter_map(Value::as_str)
            .collect::<Vec<_>>()
            .join(","),
        Value::String(text) => text.clone(),
        _ => String::new(),
    }
}

#[cfg(test)]
mod tests {
    use super::{ChannelEditorEvent, ChannelEditorField, ChannelEditorState};
    use crate::api::ChannelInfo;
    use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};
    use serde_json::json;

    #[test]
    fn preloads_channel_data() {
        let editor = ChannelEditorState::from_channel(&ChannelInfo {
            channel_id: 3,
            channel_name: "demo".to_string(),
            display_name: "Demo".to_string(),
            min_duration: 20.0,
            skip_start: 15.0,
            tags: json!(["one", "two"]),
            url: "https://example.com".to_string(),
            ..ChannelInfo::default()
        });

        assert_eq!(editor.channel_id(), Some(3));
        assert_eq!(editor.channel_name(), "demo");
        assert_eq!(editor.selected(), ChannelEditorField::Url);
        assert_eq!(editor.field_value(ChannelEditorField::Tags), "one,two");
        assert!(!editor.channel_name_editable());
    }

    #[test]
    fn new_stream_starts_editable() {
        let editor = ChannelEditorState::new_stream();

        assert_eq!(editor.channel_id(), None);
        assert!(editor.channel_name_editable());
        assert_eq!(editor.field_value(ChannelEditorField::MinDuration), "20");
    }

    #[test]
    fn validates_illegal_tag_values() {
        let mut editor = ChannelEditorState::from_channel(&ChannelInfo {
            channel_name: "demo".to_string(),
            display_name: "Demo".to_string(),
            url: "https://example.com".to_string(),
            ..ChannelInfo::default()
        });
        editor.handle_key(KeyEvent::from(KeyCode::Tab), "");
        editor.handle_key(KeyEvent::from(KeyCode::Tab), "");
        editor.handle_key(KeyEvent::from(KeyCode::Tab), "");
        editor.handle_key(KeyEvent::from(KeyCode::Tab), "");
        editor.handle_key(KeyEvent::from(KeyCode::Tab), "");
        editor.handle_key(KeyEvent::from(KeyCode::Tab), "");
        editor.handle_key(KeyEvent::from(KeyCode::Char('b')), "");
        editor.handle_key(KeyEvent::from(KeyCode::Char('a')), "");
        editor.handle_key(KeyEvent::from(KeyCode::Char('d')), "");
        editor.handle_key(KeyEvent::from(KeyCode::Char('-')), "");
        editor.handle_key(KeyEvent::from(KeyCode::Tab), "");

        editor.handle_key(KeyEvent::from(KeyCode::Enter), "");

        assert!(editor.error().is_some());
    }

    #[test]
    fn supports_copy_and_paste_in_text_fields() {
        let mut editor = ChannelEditorState::new_stream();
        editor.paste("https://example.com/live");

        let copied =
            editor.handle_key(KeyEvent::new(KeyCode::Char('a'), KeyModifiers::CONTROL), "");
        assert!(matches!(copied, ChannelEditorEvent::None));

        let copied =
            editor.handle_key(KeyEvent::new(KeyCode::Char('c'), KeyModifiers::CONTROL), "");
        assert!(matches!(
            copied,
            ChannelEditorEvent::Copied(text) if text == "https://example.com/live"
        ));
    }
}

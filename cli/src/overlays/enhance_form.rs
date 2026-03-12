use crate::{
    api::{EnhanceRequest, EnhancementDescriptions, EstimateEnhanceRequest, Recording},
    ui::{TextInput, TextInputAction},
};
use crossterm::event::{KeyCode, KeyEvent};

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum EnhanceField {
    Resolution,
    Preset,
    Crf,
    Denoise,
    Sharpen,
    Normalize,
    Estimate,
    Save,
    Cancel,
}

#[derive(Debug, Clone)]
pub enum EnhanceFormEvent {
    Close,
    Copied(String),
    Estimate(EstimateEnhanceRequest),
    None,
    Submit(EnhanceRequest),
}

#[derive(Debug, Clone)]
pub struct EnhanceFormState {
    apply_normalize: bool,
    crf: TextInput,
    denoise: TextInput,
    descriptions: Option<EnhancementDescriptions>,
    error: Option<String>,
    estimated_size: Option<u64>,
    filename: String,
    loading: bool,
    recording_id: u64,
    selected: EnhanceField,
    selected_preset: usize,
    selected_resolution: usize,
    sharpen: TextInput,
}

impl EnhanceFormState {
    pub fn new(recording: &Recording) -> Self {
        Self {
            apply_normalize: false,
            crf: TextInput::new(String::new()),
            denoise: TextInput::new(String::new()),
            descriptions: None,
            error: None,
            estimated_size: None,
            filename: recording.filename.clone(),
            loading: true,
            recording_id: recording.recording_id,
            selected: EnhanceField::Resolution,
            selected_preset: 0,
            selected_resolution: 0,
            sharpen: TextInput::new(String::new()),
        }
    }

    pub fn apply_descriptions(&mut self, descriptions: EnhancementDescriptions) {
        self.loading = false;
        self.error = None;
        self.selected_resolution = recommended_resolution_index(&descriptions);
        self.selected_preset = recommended_preset_index(&descriptions);
        self.crf.set_text(
            descriptions
                .crf_values
                .iter()
                .find(|value| value.label.to_ascii_lowercase().contains("recommended"))
                .map(|value| value.value.to_string())
                .unwrap_or_default(),
        );
        self.denoise
            .set_text(if descriptions.filters.denoise_strength.recommended > 0.0 {
                format!("{:.1}", descriptions.filters.denoise_strength.recommended)
            } else {
                "1.0".to_string()
            });
        self.sharpen.set_text(format!(
            "{:.1}",
            descriptions.filters.sharpen_strength.recommended
        ));
        self.apply_normalize = descriptions.filters.apply_normalize.recommended;
        self.descriptions = Some(descriptions);
    }

    pub fn apply_estimate(&mut self, estimated_size: u64) {
        self.estimated_size = Some(estimated_size);
        self.error = None;
    }

    pub fn set_error(&mut self, error: impl Into<String>) {
        self.error = Some(error.into());
    }

    pub fn error(&self) -> Option<&str> {
        self.error.as_deref()
    }

    pub fn estimated_size(&self) -> Option<u64> {
        self.estimated_size
    }

    pub fn filename(&self) -> &str {
        &self.filename
    }

    pub fn recording_id(&self) -> u64 {
        self.recording_id
    }

    pub fn fields() -> &'static [EnhanceField] {
        &[
            EnhanceField::Resolution,
            EnhanceField::Preset,
            EnhanceField::Crf,
            EnhanceField::Denoise,
            EnhanceField::Sharpen,
            EnhanceField::Normalize,
            EnhanceField::Estimate,
            EnhanceField::Save,
            EnhanceField::Cancel,
        ]
    }

    pub fn is_loading(&self) -> bool {
        self.loading
    }

    pub fn selected(&self) -> EnhanceField {
        self.selected
    }

    pub fn set_selected(&mut self, field: EnhanceField) {
        self.selected = field;
        self.error = None;
    }

    pub fn field_value(&self, field: EnhanceField) -> String {
        match field {
            EnhanceField::Resolution => self
                .selected_resolution_label()
                .unwrap_or_else(|| "No options".to_string()),
            EnhanceField::Preset => self
                .selected_preset_label()
                .unwrap_or_else(|| "No options".to_string()),
            EnhanceField::Crf => {
                if self.crf.text().is_empty() {
                    "auto".to_string()
                } else {
                    self.crf.display_text(false)
                }
            }
            EnhanceField::Denoise => self.denoise.display_text(false),
            EnhanceField::Sharpen => self.sharpen.display_text(false),
            EnhanceField::Normalize => {
                if self.apply_normalize {
                    "enabled".to_string()
                } else {
                    "disabled".to_string()
                }
            }
            EnhanceField::Estimate => "Estimate size".to_string(),
            EnhanceField::Save => "Queue enhancement".to_string(),
            EnhanceField::Cancel => "Cancel".to_string(),
        }
    }

    pub fn input(&self, field: EnhanceField) -> Option<&TextInput> {
        match field {
            EnhanceField::Crf => Some(&self.crf),
            EnhanceField::Denoise => Some(&self.denoise),
            EnhanceField::Sharpen => Some(&self.sharpen),
            EnhanceField::Resolution
            | EnhanceField::Preset
            | EnhanceField::Normalize
            | EnhanceField::Estimate
            | EnhanceField::Save
            | EnhanceField::Cancel => None,
        }
    }

    pub fn handle_key(&mut self, key: KeyEvent, clipboard: &str) -> EnhanceFormEvent {
        if self.loading {
            return if matches!(key.code, KeyCode::Esc) {
                EnhanceFormEvent::Close
            } else {
                EnhanceFormEvent::None
            };
        }

        match key.code {
            KeyCode::Esc => EnhanceFormEvent::Close,
            KeyCode::Tab | KeyCode::Down => {
                self.move_selection(1);
                EnhanceFormEvent::None
            }
            KeyCode::BackTab | KeyCode::Up => {
                self.move_selection(-1);
                EnhanceFormEvent::None
            }
            KeyCode::Char('g') => match self.build_estimate_request() {
                Ok(request) => EnhanceFormEvent::Estimate(request),
                Err(error) => {
                    self.error = Some(error);
                    EnhanceFormEvent::None
                }
            },
            KeyCode::Enter => match self.selected {
                EnhanceField::Estimate => match self.build_estimate_request() {
                    Ok(request) => EnhanceFormEvent::Estimate(request),
                    Err(error) => {
                        self.error = Some(error);
                        EnhanceFormEvent::None
                    }
                },
                EnhanceField::Save => match self.build_request() {
                    Ok(request) => EnhanceFormEvent::Submit(request),
                    Err(error) => {
                        self.error = Some(error);
                        EnhanceFormEvent::None
                    }
                },
                EnhanceField::Cancel => EnhanceFormEvent::Close,
                EnhanceField::Normalize => {
                    self.apply_normalize = !self.apply_normalize;
                    EnhanceFormEvent::None
                }
                _ => {
                    self.move_selection(1);
                    EnhanceFormEvent::None
                }
            },
            KeyCode::Left if self.is_selector_field() => {
                self.step_choice(-1);
                EnhanceFormEvent::None
            }
            KeyCode::Right | KeyCode::Char(' ') if self.is_selector_field() => {
                self.step_choice(1);
                EnhanceFormEvent::None
            }
            _ => self.apply_text_input_action(key, clipboard),
        }
    }

    pub fn paste(&mut self, text: &str) {
        self.error = None;
        if let Some(input) = self.selected_input_mut() {
            input.paste(text);
        }
    }

    fn build_estimate_request(&self) -> Result<EstimateEnhanceRequest, String> {
        Ok(EstimateEnhanceRequest {
            apply_normalize: self.apply_normalize,
            crf: parse_optional_integer(self.crf.text())?,
            denoise_strength: parse_float(self.denoise.text(), "Denoise strength")?,
            encoding_preset: self
                .selected_preset_label()
                .ok_or_else(|| "Select an encoding preset.".to_string())?,
            sharpen_strength: parse_float(self.sharpen.text(), "Sharpen strength")?,
            target_resolution: self
                .selected_resolution_label()
                .ok_or_else(|| "Select a target resolution.".to_string())?,
        })
    }

    fn build_request(&self) -> Result<EnhanceRequest, String> {
        let estimate = self.build_estimate_request()?;
        Ok(EnhanceRequest {
            apply_normalize: estimate.apply_normalize,
            crf: estimate.crf,
            denoise_strength: estimate.denoise_strength,
            encoding_preset: estimate.encoding_preset,
            recording_id: self.recording_id,
            sharpen_strength: estimate.sharpen_strength,
            target_resolution: estimate.target_resolution,
        })
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

    fn selected_preset_label(&self) -> Option<String> {
        self.descriptions
            .as_ref()?
            .presets
            .get(self.selected_preset)
            .map(|preset| preset.preset.clone())
    }

    fn selected_resolution_label(&self) -> Option<String> {
        self.descriptions
            .as_ref()?
            .resolutions
            .get(self.selected_resolution)
            .map(|resolution| resolution.resolution.clone())
    }

    fn step_choice(&mut self, delta: isize) {
        self.error = None;
        match self.selected {
            EnhanceField::Resolution => {
                if let Some(descriptions) = &self.descriptions {
                    self.selected_resolution = cycle_index(
                        self.selected_resolution,
                        descriptions.resolutions.len(),
                        delta,
                    );
                }
            }
            EnhanceField::Preset => {
                if let Some(descriptions) = &self.descriptions {
                    self.selected_preset =
                        cycle_index(self.selected_preset, descriptions.presets.len(), delta);
                }
            }
            EnhanceField::Normalize => {
                self.apply_normalize = !self.apply_normalize;
            }
            EnhanceField::Crf
            | EnhanceField::Denoise
            | EnhanceField::Sharpen
            | EnhanceField::Estimate
            | EnhanceField::Save
            | EnhanceField::Cancel => {}
        }
    }

    fn apply_text_input_action(&mut self, key: KeyEvent, clipboard: &str) -> EnhanceFormEvent {
        let Some(input) = self.selected_input_mut() else {
            return EnhanceFormEvent::None;
        };
        match input.handle_key(key, clipboard) {
            TextInputAction::Copied(text) => EnhanceFormEvent::Copied(text),
            TextInputAction::Handled | TextInputAction::Ignored => EnhanceFormEvent::None,
        }
    }

    fn is_selector_field(&self) -> bool {
        matches!(
            self.selected,
            EnhanceField::Resolution | EnhanceField::Preset | EnhanceField::Normalize
        )
    }

    fn selected_input_mut(&mut self) -> Option<&mut TextInput> {
        match self.selected {
            EnhanceField::Crf => Some(&mut self.crf),
            EnhanceField::Denoise => Some(&mut self.denoise),
            EnhanceField::Sharpen => Some(&mut self.sharpen),
            EnhanceField::Resolution
            | EnhanceField::Preset
            | EnhanceField::Normalize
            | EnhanceField::Estimate
            | EnhanceField::Save
            | EnhanceField::Cancel => None,
        }
    }
}

fn cycle_index(current: usize, len: usize, delta: isize) -> usize {
    if len == 0 {
        return 0;
    }
    if delta < 0 {
        if current == 0 { len - 1 } else { current - 1 }
    } else if current + 1 >= len {
        0
    } else {
        current + 1
    }
}

fn parse_float(value: &str, label: &str) -> Result<f64, String> {
    value
        .trim()
        .parse::<f64>()
        .map_err(|_| format!("{label} must be a number."))
}

fn parse_optional_integer(value: &str) -> Result<Option<u64>, String> {
    if value.trim().is_empty() {
        return Ok(None);
    }
    value
        .trim()
        .parse::<u64>()
        .map(Some)
        .map_err(|_| "CRF must be a whole number.".to_string())
}

fn recommended_preset_index(descriptions: &EnhancementDescriptions) -> usize {
    descriptions
        .presets
        .iter()
        .position(|preset| {
            preset.preset == "medium" || preset.label.to_ascii_lowercase().contains("recommended")
        })
        .unwrap_or(0)
}

fn recommended_resolution_index(descriptions: &EnhancementDescriptions) -> usize {
    descriptions
        .resolutions
        .iter()
        .position(|resolution| resolution.resolution == "1080p")
        .unwrap_or(0)
}

#[cfg(test)]
mod tests {
    use super::{EnhanceField, EnhanceFormEvent, EnhanceFormState};
    use crate::api::{
        EnhancementCrfDescription, EnhancementDescriptions, EnhancementFilters,
        EnhancementPresetDescription, EnhancementRangeSetting, EnhancementResolutionDescription,
        EnhancementToggleSetting, Recording,
    };
    use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};

    #[test]
    fn loads_recommended_defaults() {
        let mut state = EnhanceFormState::new(&Recording {
            recording_id: 9,
            filename: "demo.mp4".to_string(),
            ..Recording::default()
        });
        state.apply_descriptions(EnhancementDescriptions {
            crf_values: vec![EnhancementCrfDescription {
                label: "Recommended".to_string(),
                value: 21,
                ..EnhancementCrfDescription::default()
            }],
            filters: EnhancementFilters {
                apply_normalize: EnhancementToggleSetting {
                    recommended: true,
                    ..EnhancementToggleSetting::default()
                },
                denoise_strength: EnhancementRangeSetting {
                    recommended: 4.0,
                    ..EnhancementRangeSetting::default()
                },
                sharpen_strength: EnhancementRangeSetting {
                    recommended: 1.0,
                    ..EnhancementRangeSetting::default()
                },
            },
            presets: vec![EnhancementPresetDescription {
                preset: "medium".to_string(),
                ..EnhancementPresetDescription::default()
            }],
            resolutions: vec![EnhancementResolutionDescription {
                resolution: "1080p".to_string(),
                ..EnhancementResolutionDescription::default()
            }],
        });

        assert!(!state.is_loading());
        assert_eq!(state.selected(), EnhanceField::Resolution);
        assert_eq!(state.field_value(EnhanceField::Crf), "21");
        assert_eq!(state.field_value(EnhanceField::Normalize), "enabled");
    }

    #[test]
    fn supports_copy_and_paste_in_numeric_fields() {
        let mut state = EnhanceFormState::new(&Recording {
            recording_id: 9,
            filename: "demo.mp4".to_string(),
            ..Recording::default()
        });
        state.apply_descriptions(EnhancementDescriptions {
            crf_values: vec![EnhancementCrfDescription {
                label: "Recommended".to_string(),
                value: 21,
                ..EnhancementCrfDescription::default()
            }],
            filters: EnhancementFilters {
                apply_normalize: EnhancementToggleSetting {
                    recommended: true,
                    ..EnhancementToggleSetting::default()
                },
                denoise_strength: EnhancementRangeSetting {
                    recommended: 4.0,
                    ..EnhancementRangeSetting::default()
                },
                sharpen_strength: EnhancementRangeSetting {
                    recommended: 1.0,
                    ..EnhancementRangeSetting::default()
                },
            },
            presets: vec![EnhancementPresetDescription {
                preset: "medium".to_string(),
                ..EnhancementPresetDescription::default()
            }],
            resolutions: vec![EnhancementResolutionDescription {
                resolution: "1080p".to_string(),
                ..EnhancementResolutionDescription::default()
            }],
        });

        state.handle_key(KeyEvent::from(KeyCode::Tab), "");
        state.handle_key(KeyEvent::from(KeyCode::Tab), "");
        let _ = state.handle_key(KeyEvent::new(KeyCode::Char('a'), KeyModifiers::CONTROL), "");
        state.paste("22");

        let _ = state.handle_key(KeyEvent::new(KeyCode::Char('a'), KeyModifiers::CONTROL), "");
        let copied = state.handle_key(KeyEvent::new(KeyCode::Char('c'), KeyModifiers::CONTROL), "");

        assert_eq!(state.field_value(EnhanceField::Crf), "22");
        assert!(matches!(
            copied,
            EnhanceFormEvent::Copied(text) if text == "22"
        ));
    }
}

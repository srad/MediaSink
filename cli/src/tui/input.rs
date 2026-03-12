use super::*;

impl App {
    pub(super) fn handle_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        if key.kind != crossterm::event::KeyEventKind::Press {
            return;
        }

        if self.help_popup.is_open() {
            self.handle_help_popup_key(key);
            return;
        }

        if key.code == KeyCode::F(1) {
            self.open_help_popup();
            return;
        }

        if key.code == KeyCode::F(10) {
            self.running = false;
            return;
        }

        if self.confirm.is_some() {
            self.handle_confirm_key(key, tx);
            return;
        }

        if self.theme_picker.is_open() {
            self.handle_theme_picker_key(key, tx);
            return;
        }

        if self.player_picker.is_open() {
            self.handle_player_picker_key(key, tx);
            return;
        }

        if self.video_popup.is_some() {
            self.handle_video_popup_key(key, tx);
            return;
        }

        if self.enhance_form.is_some() {
            self.handle_enhance_form_key(key, tx);
            return;
        }

        if self.channel_editor.is_some() {
            self.handle_channel_editor_key(key, tx);
            return;
        }

        if self.item_menu.is_open() {
            self.handle_item_menu_key(key, tx);
            return;
        }

        match self.screen {
            Screen::Login => self.handle_login_key(key, tx),
            Screen::Workspace => self.handle_workspace_key(key, tx),
        }
    }

    pub(super) fn handle_help_popup_key(&mut self, key: KeyEvent) {
        let area = terminal_area();
        let Some((_, body_area, _)) = help_popup_layout(area) else {
            return;
        };
        let max_scroll = help_popup_max_scroll(self.help_popup.context(), body_area.height);

        match key.code {
            KeyCode::Esc | KeyCode::F(1) => self.help_popup.close(),
            KeyCode::Up => self.help_popup.scroll_by(-1, max_scroll),
            KeyCode::Down => self.help_popup.scroll_by(1, max_scroll),
            KeyCode::PageUp => self.help_popup.page_by(-1, body_area.height, max_scroll),
            KeyCode::PageDown => self.help_popup.page_by(1, body_area.height, max_scroll),
            KeyCode::Home => self.help_popup.scroll_to_top(),
            KeyCode::End => self.help_popup.scroll_to_bottom(max_scroll),
            _ => {}
        }
    }

    pub(super) fn handle_confirm_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        match key.code {
            KeyCode::Esc | KeyCode::Char('n') => self.confirm = None,
            KeyCode::Enter | KeyCode::Char('y') => {
                let action = self.confirm.take();
                if let Some(action) = action {
                    self.execute_confirm_action(action, tx);
                }
            }
            _ => {}
        }
    }

    pub(super) fn handle_theme_picker_key(
        &mut self,
        key: KeyEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        match key.code {
            KeyCode::Esc => self.restore_theme_preview(tx),
            KeyCode::Up => {
                self.theme_picker.move_selection(-1);
                self.preview_theme(self.theme_picker.selected_theme(), tx);
            }
            KeyCode::Down => {
                self.theme_picker.move_selection(1);
                self.preview_theme(self.theme_picker.selected_theme(), tx);
            }
            KeyCode::Enter => {
                let theme_name = self.theme_picker.selected_theme();
                self.theme_picker.close();
                self.commit_theme(theme_name, tx);
            }
            _ => {}
        }
    }

    pub(super) fn handle_player_picker_key(
        &mut self,
        key: KeyEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let modes = self.available_player_modes();
        match key.code {
            KeyCode::Esc => self.player_picker.close(),
            KeyCode::Up => self.player_picker.move_selection(-1, modes.len()),
            KeyCode::Down => self.player_picker.move_selection(1, modes.len()),
            KeyCode::Enter => {
                let restart_single_frame = self.video_popup.as_ref().map(|popup| popup.paused);
                let player_mode = self.player_picker.selected_mode(&modes);
                self.player_picker.close();
                self.set_player_mode(player_mode);
                if let Some(single_frame) = restart_single_frame {
                    self.restart_video_popup_playback(tx, single_frame);
                }
            }
            _ => {}
        }
    }

    pub(super) fn handle_video_popup_key(
        &mut self,
        key: KeyEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        match key.code {
            KeyCode::Esc => {
                self.close_video_popup();
                self.set_status("Closed popup player.", self.theme().accent);
            }
            KeyCode::Char(' ') => self.toggle_video_popup_pause(tx),
            KeyCode::Left => self.seek_video_popup(-VIDEO_PLAYER_SEEK_STEP_SECONDS, tx),
            KeyCode::Right => self.seek_video_popup(VIDEO_PLAYER_SEEK_STEP_SECONDS, tx),
            KeyCode::PageUp => self.seek_video_popup(-30.0, tx),
            KeyCode::PageDown => self.seek_video_popup(30.0, tx),
            KeyCode::Home => self.seek_video_popup_absolute(0.0, tx),
            KeyCode::End => {
                if let Some(popup) = self.video_popup.as_ref() {
                    self.seek_video_popup_absolute(popup.duration_seconds.max(0.0), tx);
                }
            }
            KeyCode::F(5) | KeyCode::Char('v') => self.open_player_picker(),
            KeyCode::F(6) => self.toggle_mouse_enabled(),
            _ => {}
        }
    }

    pub(super) fn handle_mouse(&mut self, mouse: MouseEvent, tx: &UnboundedSender<AppMessage>) {
        if !self.mouse_enabled {
            return;
        }

        if self.help_popup.is_open() {
            self.handle_help_popup_mouse(mouse);
            return;
        }

        if self.confirm.is_some() {
            self.handle_confirm_mouse(mouse, tx);
            return;
        }

        if self.theme_picker.is_open() {
            self.handle_theme_picker_mouse(mouse, tx);
            return;
        }

        if self.player_picker.is_open() {
            self.handle_player_picker_mouse(mouse, tx);
            return;
        }

        if self.video_popup.is_some() && self.handle_video_popup_mouse(mouse, tx) {
            return;
        }

        if self.enhance_form.is_some() {
            self.handle_enhance_form_mouse(mouse, tx);
            return;
        }

        if self.channel_editor.is_some() {
            self.handle_channel_editor_mouse(mouse, tx);
            return;
        }

        if self.item_menu.is_open() {
            self.handle_item_menu_mouse(mouse, tx);
            return;
        }

        match self.screen {
            Screen::Login => self.handle_login_mouse(mouse, tx),
            Screen::Workspace => self.handle_workspace_mouse(mouse, tx),
        }
    }

    pub(super) fn handle_help_popup_mouse(&mut self, mouse: MouseEvent) {
        let area = terminal_area();
        let Some((popup, body_area, _)) = help_popup_layout(area) else {
            return;
        };
        let max_scroll = help_popup_max_scroll(self.help_popup.context(), body_area.height);

        match mouse.kind {
            MouseEventKind::Down(MouseButton::Left) => {
                if matches!(
                    self.ui_regions.hit(mouse.column, mouse.row),
                    Some(UiRegion::PopupClose(PopupId::Help))
                ) {
                    self.help_popup.close();
                } else if !rect_contains(popup, mouse.column, mouse.row) {
                    self.help_popup.close();
                }
            }
            MouseEventKind::ScrollDown => self.help_popup.scroll_by(2, max_scroll),
            MouseEventKind::ScrollUp => self.help_popup.scroll_by(-2, max_scroll),
            _ => {}
        }
    }

    pub(super) fn handle_video_popup_mouse(
        &mut self,
        mouse: MouseEvent,
        tx: &UnboundedSender<AppMessage>,
    ) -> bool {
        let Some(popup) = self.video_popup.as_ref() else {
            return false;
        };
        let terminal = crossterm::terminal::size().unwrap_or((120, 40));
        let area = Rect::new(0, 0, terminal.0, terminal.1);
        let Some((popup_rect, sections)) = video_popup_layout(area) else {
            return true;
        };
        let progress_area = sections[1];

        match mouse.kind {
            MouseEventKind::Down(MouseButton::Left) => {
                let hit = self.ui_regions.hit(mouse.column, mouse.row);
                if matches!(hit, Some(UiRegion::PopupClose(PopupId::VideoPlayer))) {
                    self.close_video_popup();
                    self.set_status("Closed popup player.", self.theme().accent);
                    return true;
                }

                if !rect_contains(popup_rect, mouse.column, mouse.row) {
                    return true;
                }

                if !matches!(hit, Some(UiRegion::VideoSeekBar)) {
                    return true;
                }

                let duration_seconds = popup.duration_seconds.max(0.0);
                let relative = mouse
                    .column
                    .saturating_sub(progress_area.x)
                    .min(progress_area.width.saturating_sub(1));
                let ratio = if progress_area.width <= 1 {
                    0.0
                } else {
                    relative as f64 / progress_area.width.saturating_sub(1) as f64
                };
                self.seek_video_popup_absolute(duration_seconds * ratio, tx);
                true
            }
            _ => true,
        }
    }

    pub(super) fn handle_confirm_mouse(
        &mut self,
        mouse: MouseEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let Some(action) = self.confirm.clone() else {
            return;
        };

        let area = terminal_area();
        let Some((popup, yes_button, no_button)) = confirm_layout(area) else {
            return;
        };

        if matches!(mouse.kind, MouseEventKind::Down(MouseButton::Left)) {
            if matches!(
                self.ui_regions.hit(mouse.column, mouse.row),
                Some(UiRegion::PopupClose(PopupId::Confirm))
            ) {
                self.confirm = None;
            } else if rect_contains(yes_button, mouse.column, mouse.row) {
                self.confirm = None;
                self.execute_confirm_action(action, tx);
            } else if rect_contains(no_button, mouse.column, mouse.row)
                || !rect_contains(popup, mouse.column, mouse.row)
            {
                self.confirm = None;
            }
        }
    }

    pub(super) fn handle_theme_picker_mouse(
        &mut self,
        mouse: MouseEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let area = terminal_area();
        let Some((popup, list_area)) = theme_picker_layout(area) else {
            return;
        };

        if matches!(mouse.kind, MouseEventKind::Down(MouseButton::Left)) {
            if matches!(
                self.ui_regions.hit(mouse.column, mouse.row),
                Some(UiRegion::PopupClose(PopupId::ThemePicker))
            ) {
                self.restore_theme_preview(tx);
                return;
            }
            if !rect_contains(popup, mouse.column, mouse.row) {
                self.restore_theme_preview(tx);
                return;
            }

            if let Some(index) = row_hit_index(
                list_area,
                mouse.column,
                mouse.row,
                1,
                ThemeName::all().len(),
                0,
            ) {
                self.theme_picker.set_selected_index(index);
                let theme_name = self.theme_picker.selected_theme();
                self.theme_picker.close();
                self.commit_theme(theme_name, tx);
            }
        } else if mouse.kind == MouseEventKind::ScrollDown {
            self.theme_picker.move_selection(1);
            self.preview_theme(self.theme_picker.selected_theme(), tx);
        } else if mouse.kind == MouseEventKind::ScrollUp {
            self.theme_picker.move_selection(-1);
            self.preview_theme(self.theme_picker.selected_theme(), tx);
        }
    }

    pub(super) fn handle_player_picker_mouse(
        &mut self,
        mouse: MouseEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let modes = self.available_player_modes();
        let area = terminal_area();
        let Some((popup, list_area)) = player_picker_layout(area) else {
            return;
        };

        if matches!(mouse.kind, MouseEventKind::Down(MouseButton::Left)) {
            if matches!(
                self.ui_regions.hit(mouse.column, mouse.row),
                Some(UiRegion::PopupClose(PopupId::PlayerPicker))
            ) {
                self.player_picker.close();
                return;
            }
            if !rect_contains(popup, mouse.column, mouse.row) {
                self.player_picker.close();
                return;
            }

            if let Some(index) =
                row_hit_index(list_area, mouse.column, mouse.row, 2, modes.len(), 0)
            {
                self.player_picker.set_selected_index(index, modes.len());
                let restart_single_frame = self.video_popup.as_ref().map(|popup| popup.paused);
                let player_mode = self.player_picker.selected_mode(&modes);
                self.player_picker.close();
                self.set_player_mode(player_mode);
                if let Some(single_frame) = restart_single_frame {
                    self.restart_video_popup_playback(tx, single_frame);
                }
            }
        } else if mouse.kind == MouseEventKind::ScrollDown {
            self.player_picker.move_selection(1, modes.len());
        } else if mouse.kind == MouseEventKind::ScrollUp {
            self.player_picker.move_selection(-1, modes.len());
        }
    }

    pub(super) fn handle_item_menu_mouse(
        &mut self,
        mouse: MouseEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let area = terminal_area();
        let Some((popup, list_area)) = item_menu_layout(area) else {
            return;
        };

        if matches!(mouse.kind, MouseEventKind::Down(MouseButton::Left)) {
            if matches!(
                self.ui_regions.hit(mouse.column, mouse.row),
                Some(UiRegion::PopupClose(PopupId::ItemMenu))
            ) {
                self.item_menu.close();
                return;
            }
            if !rect_contains(popup, mouse.column, mouse.row) {
                self.item_menu.close();
                return;
            }

            if let Some(index) = row_hit_index(
                list_area,
                mouse.column,
                mouse.row,
                2,
                self.item_menu.items().len(),
                0,
            ) {
                self.item_menu.set_selected(index);
                let selected = self.item_menu.selected_action();
                let target = self.item_menu.target().cloned();
                if let (Some(action), Some(target)) = (selected, target) {
                    self.dispatch_item_action(action, target, tx);
                }
            }
        } else if mouse.kind == MouseEventKind::ScrollDown {
            self.item_menu.move_selection(1);
        } else if mouse.kind == MouseEventKind::ScrollUp {
            self.item_menu.move_selection(-1);
        }
    }

    pub(super) fn handle_channel_editor_mouse(
        &mut self,
        mouse: MouseEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let area = terminal_area();
        let Some((popup, rows_area)) = channel_editor_layout(area) else {
            return;
        };

        if !matches!(mouse.kind, MouseEventKind::Down(MouseButton::Left)) {
            return;
        }
        if matches!(
            self.ui_regions.hit(mouse.column, mouse.row),
            Some(UiRegion::PopupClose(PopupId::ChannelEditor))
        ) {
            self.channel_editor = None;
            return;
        }
        if !rect_contains(popup, mouse.column, mouse.row) {
            self.channel_editor = None;
            return;
        }

        let Some(index) = row_hit_index(
            rows_area,
            mouse.column,
            mouse.row,
            1,
            ChannelEditorState::fields().len(),
            0,
        ) else {
            return;
        };

        let clipboard = self.clipboard.clone();
        let event = {
            let Some(editor) = self.channel_editor.as_mut() else {
                return;
            };
            let field = ChannelEditorState::fields()[index];
            editor.set_selected(field);
            match field {
                ChannelEditorField::Paused
                | ChannelEditorField::Save
                | ChannelEditorField::Cancel => {
                    editor.handle_key(KeyEvent::from(KeyCode::Enter), &clipboard)
                }
                _ => ChannelEditorEvent::None,
            }
        };
        self.apply_channel_editor_event(event, tx);
    }

    pub(super) fn handle_enhance_form_mouse(
        &mut self,
        mouse: MouseEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let area = terminal_area();
        let Some((popup, rows_area)) = enhance_form_layout(area) else {
            return;
        };

        if !matches!(mouse.kind, MouseEventKind::Down(MouseButton::Left)) {
            return;
        }
        if matches!(
            self.ui_regions.hit(mouse.column, mouse.row),
            Some(UiRegion::PopupClose(PopupId::EnhanceForm))
        ) {
            self.enhance_form = None;
            return;
        }
        if !rect_contains(popup, mouse.column, mouse.row) {
            self.enhance_form = None;
            return;
        }

        let Some(index) = row_hit_index(
            rows_area,
            mouse.column,
            mouse.row,
            1,
            EnhanceFormState::fields().len(),
            0,
        ) else {
            return;
        };

        let clipboard = self.clipboard.clone();
        let event = {
            let Some(form) = self.enhance_form.as_mut() else {
                return;
            };
            let field = EnhanceFormState::fields()[index];
            form.set_selected(field);
            match field {
                EnhanceField::Resolution | EnhanceField::Preset | EnhanceField::Normalize => {
                    form.handle_key(KeyEvent::from(KeyCode::Right), &clipboard)
                }
                EnhanceField::Estimate | EnhanceField::Save | EnhanceField::Cancel => {
                    form.handle_key(KeyEvent::from(KeyCode::Enter), &clipboard)
                }
                _ => EnhanceFormEvent::None,
            }
        };
        self.apply_enhance_form_event(event, tx);
    }

    pub(super) fn handle_login_mouse(
        &mut self,
        mouse: MouseEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let area = terminal_area();
        let (login_area, _) = login_layout(area);

        match mouse.kind {
            MouseEventKind::ScrollDown => self.login_field = self.login_field.next(),
            MouseEventKind::ScrollUp => self.login_field = self.login_field.previous(),
            MouseEventKind::Down(MouseButton::Left) => {
                if !rect_contains(login_area, mouse.column, mouse.row) {
                    return;
                }
                match self.ui_regions.hit(mouse.column, mouse.row) {
                    Some(UiRegion::LoginField(field)) => self.login_field = field,
                    Some(UiRegion::LoginAction(action)) => match action {
                        LoginMouseAction::Submit => self.submit_login(tx),
                        LoginMouseAction::ToggleMode => {
                            self.login_mode = self.login_mode.toggle();
                            self.set_status(
                                format!("{} mode selected.", self.login_mode.label()),
                                self.theme().accent,
                            );
                        }
                        LoginMouseAction::Mouse => self.toggle_mouse_enabled(),
                        LoginMouseAction::Quit => self.running = false,
                    },
                    _ => {}
                }
            }
            _ => {}
        }
    }

    pub(super) fn handle_workspace_mouse(
        &mut self,
        mouse: MouseEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        if self.channel_popup.is_open() {
            self.handle_channel_popup_mouse(mouse, tx);
            return;
        }

        let area = terminal_area();
        let (vertical, _) = workspace_layout(area, self);

        match mouse.kind {
            MouseEventKind::ScrollDown => self.move_selection(1),
            MouseEventKind::ScrollUp => self.move_selection(-1),
            MouseEventKind::Down(MouseButton::Left) => {
                match self.ui_regions.hit(mouse.column, mouse.row) {
                    Some(UiRegion::WorkspaceHeader(action)) => {
                        match action {
                            WorkspaceHeaderAction::AddStream => self.open_create_stream_editor(),
                            WorkspaceHeaderAction::Logout => {
                                self.confirm = Some(ConfirmAction::Logout)
                            }
                            WorkspaceHeaderAction::Recorder => {
                                self.confirm = Some(ConfirmAction::ToggleRecorder)
                            }
                        }
                        return;
                    }
                    Some(UiRegion::PrimaryTab(view)) => {
                        self.set_view(view, tx);
                        return;
                    }
                    Some(UiRegion::StreamTab(tab)) => {
                        self.stream_tab = tab;
                        self.selected_channel = clamp_index(
                            self.selected_channel,
                            self.visible_stream_channels().len(),
                        );
                        return;
                    }
                    _ => {}
                }

                if !rect_contains(vertical[3], mouse.column, mouse.row) {
                    return;
                }

                self.handle_workspace_content_click(vertical[3], mouse.column, mouse.row, tx);
            }
            _ => {}
        }
    }

    pub(super) fn handle_channel_popup_mouse(
        &mut self,
        mouse: MouseEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let area = terminal_area();
        let vertical = workspace_layout(area, self).0;
        let Some((popup, rows_area)) = channel_popup_recordings_layout(vertical[3]) else {
            return;
        };

        match mouse.kind {
            MouseEventKind::ScrollDown => self.channel_popup.move_selection(1),
            MouseEventKind::ScrollUp => self.channel_popup.move_selection(-1),
            MouseEventKind::Down(MouseButton::Left) => {
                if matches!(
                    self.ui_regions.hit(mouse.column, mouse.row),
                    Some(UiRegion::PopupClose(PopupId::ChannelPopup))
                ) {
                    self.channel_popup.close();
                    return;
                }
                if !rect_contains(popup, mouse.column, mouse.row) {
                    self.channel_popup.close();
                    return;
                }

                let recordings = self.channel_recordings();
                if let Some(index) = row_hit_index(
                    rows_area,
                    mouse.column,
                    mouse.row,
                    MEDIA_ROW_HEIGHT,
                    recordings.len(),
                    self.channel_popup.selected_recording(),
                ) {
                    let was_selected = index == self.channel_popup.selected_recording();
                    self.channel_popup.set_selected_recording(index);
                    if was_selected {
                        self.play_selected_video(tx);
                    }
                    self.prefetch_thumbnails(tx);
                }
            }
            _ => {}
        }
    }

    pub(super) fn handle_workspace_content_click(
        &mut self,
        area: Rect,
        column: u16,
        row: u16,
        tx: &UnboundedSender<AppMessage>,
    ) {
        if let Some(index) = main_content_hit_index(self, area, column, row) {
            let was_selected = index == self.current_selection();
            self.set_selection(index);
            match self.view {
                View::Streams | View::Channels => {
                    if was_selected {
                        self.open_selected_channel(tx);
                    }
                }
                View::Latest | View::Random | View::Favourites => {
                    if was_selected {
                        self.play_selected_video(tx);
                    }
                }
                View::Similarity | View::Processes | View::Jobs | View::Logs | View::Channel => {}
                View::Admin | View::Info | View::Monitoring => {}
            }
            self.prefetch_thumbnails(tx);
        }
    }

    pub(super) fn handle_item_menu_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        match key.code {
            KeyCode::Esc => self.item_menu.close(),
            KeyCode::Up => self.item_menu.move_selection(-1),
            KeyCode::Down => self.item_menu.move_selection(1),
            KeyCode::Enter => {
                let selected = self.item_menu.selected_action();
                let target = self.item_menu.target().cloned();
                if let (Some(action), Some(target)) = (selected, target) {
                    self.dispatch_item_action(action, target, tx);
                }
            }
            _ => {}
        }
    }

    pub(super) fn handle_channel_editor_key(
        &mut self,
        key: KeyEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let clipboard = self.clipboard.clone();
        let event = {
            let Some(editor) = self.channel_editor.as_mut() else {
                return;
            };
            editor.handle_key(key, &clipboard)
        };
        self.apply_channel_editor_event(event, tx);
    }

    pub(super) fn handle_enhance_form_key(
        &mut self,
        key: KeyEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        let clipboard = self.clipboard.clone();
        let event = {
            let Some(form) = self.enhance_form.as_mut() else {
                return;
            };
            form.handle_key(key, &clipboard)
        };
        self.apply_enhance_form_event(event, tx);
    }

    pub(super) fn apply_channel_editor_event(
        &mut self,
        event: ChannelEditorEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        match event {
            ChannelEditorEvent::Close => self.channel_editor = None,
            ChannelEditorEvent::Copied(text) => {
                self.apply_text_input_action(TextInputAction::Copied(text))
            }
            ChannelEditorEvent::None => {}
            ChannelEditorEvent::Submit(ChannelEditorSubmit {
                channel_id,
                request,
            }) => {
                let Some(session) = self.session.clone() else {
                    return;
                };
                let channel_name = request.channel_name.clone();
                let sender = tx.clone();
                self.set_status(
                    if channel_id.is_some() {
                        format!("Saving {channel_name}…")
                    } else {
                        format!("Creating {channel_name}…")
                    },
                    self.theme().accent,
                );
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        if let Some(channel_id) = channel_id {
                            client.update_channel(channel_id, &request).await?;
                        } else {
                            client.create_channel(&request).await?;
                        }
                        Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                            message: if channel_id.is_some() {
                                format!("Saved {channel_name}")
                            } else {
                                format!("Created {channel_name}")
                            },
                            refresh_channel_popup: false,
                            refresh_snapshot: true,
                            refresh_view_data: false,
                            tone: Color::Green,
                        })
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::ActionFinished(result));
                });
                self.channel_editor = None;
            }
        }
    }

    pub(super) fn apply_enhance_form_event(
        &mut self,
        event: EnhanceFormEvent,
        tx: &UnboundedSender<AppMessage>,
    ) {
        match event {
            EnhanceFormEvent::Close => self.enhance_form = None,
            EnhanceFormEvent::Copied(text) => {
                self.apply_text_input_action(TextInputAction::Copied(text))
            }
            EnhanceFormEvent::Estimate(body) => self.request_enhancement_estimate(body, tx),
            EnhanceFormEvent::None => {}
            EnhanceFormEvent::Submit(body) => {
                let Some(session) = self.session.clone() else {
                    return;
                };
                let recording_id = body.recording_id;
                let sender = tx.clone();
                self.set_status(
                    format!("Queueing enhancement for #{recording_id}…"),
                    self.theme().accent,
                );
                tokio::spawn(async move {
                    let result = async move {
                        let client = session.client()?;
                        client.enhance_video(recording_id, &body).await?;
                        Ok::<ActionFeedback, anyhow::Error>(ActionFeedback {
                            message: format!("Queued enhancement for #{recording_id}"),
                            refresh_channel_popup: true,
                            refresh_snapshot: true,
                            refresh_view_data: true,
                            tone: Color::Green,
                        })
                    }
                    .await
                    .map_err(|error| error.to_string());
                    let _ = sender.send(AppMessage::ActionFinished(result));
                });
                self.enhance_form = None;
            }
        }
    }

    pub(super) fn handle_login_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        if self.auto_login_pending {
            if matches!(key.code, KeyCode::Esc) {
                self.auto_login_pending = false;
                self.session = None;
                self.set_status("Auto-login canceled.", self.theme().warning);
            }
            return;
        }

        match key.code {
            KeyCode::Char('q') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                self.running = false;
            }
            KeyCode::Char('c') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                let clipboard = self.clipboard.clone();
                let selection_action = self.selected_login_input_mut().handle_key(key, &clipboard);
                self.apply_text_input_action(selection_action);
            }
            KeyCode::F(2) => {
                self.login_mode = self.login_mode.toggle();
                self.set_status(
                    format!("{} mode selected.", self.login_mode.label()),
                    self.theme().accent,
                );
            }
            KeyCode::F(3) => self.open_theme_picker(),
            KeyCode::F(5) => self.open_player_picker(),
            KeyCode::F(6) => self.toggle_mouse_enabled(),
            KeyCode::Tab | KeyCode::Down => self.login_field = self.login_field.next(),
            KeyCode::BackTab | KeyCode::Up => self.login_field = self.login_field.previous(),
            KeyCode::Enter => self.submit_login(tx),
            _ => {
                let clipboard = self.clipboard.clone();
                let action = self.selected_login_input_mut().handle_key(key, &clipboard);
                self.apply_text_input_action(action);
            }
        }
    }

    pub(super) fn submit_login(&mut self, tx: &UnboundedSender<AppMessage>) {
        if self.login_server.text().trim().is_empty() {
            self.login_error = Some("Server URL is required.".to_string());
            return;
        }
        if self.login_username.text().trim().is_empty() {
            self.login_error = Some("Login is required.".to_string());
            return;
        }
        if self.login_password.text().is_empty() {
            self.login_error = Some("Password is required.".to_string());
            return;
        }

        self.login_error = None;
        self.auto_login_pending = true;
        self.set_status(
            format!("{} in progress…", self.login_mode.label()),
            self.theme().accent,
        );

        let sender = tx.clone();
        let base_url = self.login_server.text().trim().to_string();
        let username = self.login_username.text().trim().to_string();
        let password = self.login_password.text().to_string();
        let signup = self.login_mode == AuthMode::Signup;

        tokio::spawn(async move {
            let result = async move {
                let normalized = crate::config::normalize_server_url(&base_url)?;
                let AuthSuccess { runtime, token } =
                    ApiClient::authenticate(&normalized, None, &username, &password, signup)
                        .await?;
                let warning = save_authenticated_session(
                    &normalized,
                    &username,
                    &token,
                    Some(runtime.api_version.clone()),
                    Some(runtime.file_url.clone()),
                )?;

                Ok::<_, anyhow::Error>((normalized, runtime, token, username, warning))
            }
            .await;

            match result {
                Ok((base_url, runtime, token, username, warning)) => {
                    let _ = sender.send(AppMessage::AuthSucceeded {
                        base_url,
                        runtime,
                        token,
                        username,
                        warning,
                    });
                }
                Err(error) => {
                    let _ = sender.send(AppMessage::AuthFailed(error.to_string()));
                }
            }
        });
    }

    pub(super) fn handle_workspace_key(&mut self, key: KeyEvent, tx: &UnboundedSender<AppMessage>) {
        if self.channel_popup.is_open() {
            match key.code {
                KeyCode::Char('c') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                    self.running = false
                }
                KeyCode::Char('q') => self.running = false,
                KeyCode::Esc | KeyCode::Backspace => self.channel_popup.close(),
                KeyCode::Enter => self.play_selected_video(tx),
                KeyCode::F(3) => self.open_theme_picker(),
                KeyCode::F(5) | KeyCode::Char('v') => self.open_player_picker(),
                KeyCode::F(6) => self.toggle_mouse_enabled(),
                KeyCode::F(4) => self.open_item_actions(),
                KeyCode::Char('n') => self.open_create_stream_editor(),
                KeyCode::Up => self.channel_popup.move_selection(-1),
                KeyCode::Down => self.channel_popup.move_selection(1),
                KeyCode::Char('g') => self.request_channel_popup(tx),
                _ => {}
            }

            self.prefetch_thumbnails(tx);
            return;
        }

        match key.code {
            KeyCode::Char('c') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                self.running = false
            }
            KeyCode::Char('q') => self.running = false,
            KeyCode::Left => {
                let items = View::primary();
                let index = items
                    .iter()
                    .position(|item| *item == self.primary_view)
                    .unwrap_or(0);
                let next = items[(index + items.len() - 1) % items.len()];
                self.set_view(next, tx);
            }
            KeyCode::Right => {
                let items = View::primary();
                let index = items
                    .iter()
                    .position(|item| *item == self.primary_view)
                    .unwrap_or(0);
                let next = items[(index + 1) % items.len()];
                self.set_view(next, tx);
            }
            KeyCode::Up => self.move_selection(-1),
            KeyCode::Down => self.move_selection(1),
            KeyCode::PageDown => {
                if matches!(self.view, View::Latest) {
                    self.video_page.skip = self
                        .video_page
                        .skip
                        .saturating_add(self.video_page.take.max(self.latest_filter.limit()));
                    self.request_refresh(tx);
                }
            }
            KeyCode::PageUp => {
                if matches!(self.view, View::Latest) {
                    self.video_page.skip = self
                        .video_page
                        .skip
                        .saturating_sub(self.video_page.take.max(self.latest_filter.limit()));
                    self.request_refresh(tx);
                }
            }
            KeyCode::Char('g') => {
                self.request_refresh(tx);
                self.request_view_data(tx);
            }
            KeyCode::Char('a') => self.set_view(View::Admin, tx),
            KeyCode::Char('i') => self.set_view(View::Info, tx),
            KeyCode::Char('j') => {
                self.set_view(View::Jobs, tx);
            }
            KeyCode::Char('l') => self.confirm = Some(ConfirmAction::Logout),
            KeyCode::Char('k') => self.set_view(View::Logs, tx),
            KeyCode::Char('m') => self.set_view(View::Monitoring, tx),
            KeyCode::Char('n') => self.open_create_stream_editor(),
            KeyCode::Char('p') => self.set_view(View::Processes, tx),
            KeyCode::Char('r') => self.confirm = Some(ConfirmAction::ToggleRecorder),
            KeyCode::Char('t') => self.open_theme_picker(),
            KeyCode::F(3) => self.open_theme_picker(),
            KeyCode::Char('v') | KeyCode::F(5) => self.open_player_picker(),
            KeyCode::F(6) => self.toggle_mouse_enabled(),
            KeyCode::F(4) => self.open_item_actions(),
            KeyCode::Char('1') => self.set_view(View::Streams, tx),
            KeyCode::Char('2') => self.set_view(View::Channels, tx),
            KeyCode::Char('3') => self.set_view(View::Latest, tx),
            KeyCode::Char('4') => self.set_view(View::Random, tx),
            KeyCode::Char('5') => self.set_view(View::Favourites, tx),
            KeyCode::Char('6') => self.set_view(View::Similarity, tx),
            KeyCode::Enter if matches!(self.view, View::Streams | View::Channels) => {
                self.open_selected_channel(tx);
            }
            KeyCode::Enter
                if matches!(self.view, View::Latest | View::Random | View::Favourites) =>
            {
                self.play_selected_video(tx);
            }
            KeyCode::Char('[') => match self.view {
                View::Streams => {
                    self.stream_tab = self.stream_tab.previous();
                    self.selected_channel =
                        clamp_index(self.selected_channel, self.visible_stream_channels().len());
                }
                View::Latest => {
                    self.latest_filter.previous_column();
                    self.video_page.skip = 0;
                    self.video_page.take = self.latest_filter.limit();
                    self.request_refresh(tx);
                }
                View::Similarity => {
                    self.similarity_tab = self.similarity_tab.previous();
                    self.selected_similarity = clamp_index(
                        self.selected_similarity,
                        self.similarity_groups
                            .as_ref()
                            .map(|groups| groups.groups.len())
                            .unwrap_or(0),
                    );
                }
                _ => {}
            },
            KeyCode::Char(']') => match self.view {
                View::Streams => {
                    self.stream_tab = self.stream_tab.next();
                    self.selected_channel =
                        clamp_index(self.selected_channel, self.visible_stream_channels().len());
                }
                View::Latest => {
                    self.latest_filter.next_column();
                    self.video_page.skip = 0;
                    self.video_page.take = self.latest_filter.limit();
                    self.request_refresh(tx);
                }
                View::Similarity => {
                    self.similarity_tab = self.similarity_tab.next();
                    self.selected_similarity = clamp_index(
                        self.selected_similarity,
                        self.similarity_groups
                            .as_ref()
                            .map(|groups| groups.groups.len())
                            .unwrap_or(0),
                    );
                }
                _ => {}
            },
            KeyCode::Char('o') if matches!(self.view, View::Latest) => {
                self.latest_filter.toggle_order();
                self.video_page.skip = 0;
                self.video_page.take = self.latest_filter.limit();
                self.request_refresh(tx);
            }
            KeyCode::Char('-') => match self.view {
                View::Latest => {
                    self.latest_filter.previous_limit();
                    self.video_page.skip = 0;
                    self.video_page.take = self.latest_filter.limit();
                    self.request_refresh(tx);
                }
                View::Random => {
                    self.random_filter.previous_limit();
                    self.request_view_data(tx);
                }
                _ => {}
            },
            KeyCode::Char('+') | KeyCode::Char('=') => match self.view {
                View::Latest => {
                    self.latest_filter.next_limit();
                    self.video_page.skip = 0;
                    self.video_page.take = self.latest_filter.limit();
                    self.request_refresh(tx);
                }
                View::Random => {
                    self.random_filter.next_limit();
                    self.request_view_data(tx);
                }
                _ => {}
            },
            KeyCode::Char('0') => match self.view {
                View::Latest => {
                    self.latest_filter.reset();
                    self.video_page.skip = 0;
                    self.video_page.take = self.latest_filter.limit();
                    self.request_refresh(tx);
                }
                View::Random => {
                    self.random_filter.reset();
                    self.request_view_data(tx);
                }
                _ => {}
            },
            _ => {}
        }

        self.prefetch_thumbnails(tx);
    }

    pub(super) fn toggle_recorder(&mut self, tx: &UnboundedSender<AppMessage>) {
        let Some(session) = self.session.clone() else {
            return;
        };

        self.set_status(
            if self.recorder.is_recording {
                "Stopping recorder…"
            } else {
                "Starting recorder…"
            },
            self.theme().accent,
        );
        let currently_recording = self.recorder.is_recording;
        let sender = tx.clone();
        tokio::spawn(async move {
            let result = async move {
                let client = session.client()?;
                if currently_recording {
                    client.stop_recorder().await?;
                    Ok::<bool, anyhow::Error>(false)
                } else {
                    client.start_recorder().await?;
                    Ok::<bool, anyhow::Error>(true)
                }
            }
            .await
            .map_err(|error| error.to_string());

            let _ = sender.send(AppMessage::RecorderToggled(result));
        });
    }

    pub(super) fn logout(&mut self, tx: &UnboundedSender<AppMessage>) {
        let Some(session) = self.session.clone() else {
            self.return_to_login(
                "Logged out. Enter another server or account to continue.",
                Color::Green,
            );
            return;
        };
        let sender = tx.clone();
        self.set_status("Logging out…", self.theme().accent);
        tokio::spawn(async move {
            let result = async move {
                let client = session.client()?;
                let _ = client.logout().await;
                clear_saved_session(&session.base_url)?;
                Ok::<(), anyhow::Error>(())
            }
            .await
            .map_err(|error| error.to_string());
            let _ = sender.send(AppMessage::LoggedOut(result));
        });
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::{CliProfile, LoadedSession};

    #[test]
    fn login_plain_v_updates_text_input_instead_of_opening_player_picker() {
        let loaded = LoadedSession {
            base_url: String::new(),
            profile: CliProfile::default(),
            token: None,
        };
        let mut app = App::from_loaded_session(loaded);
        let (tx, _rx) = unbounded_channel();

        app.handle_login_key(KeyEvent::new(KeyCode::Char('v'), KeyModifiers::NONE), &tx);

        assert_eq!(app.login_server.text(), "v");
        assert!(!app.player_picker.is_open());
    }
}

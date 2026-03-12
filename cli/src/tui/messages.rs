use super::*;

impl App {
    pub(super) fn handle_message(&mut self, message: AppMessage, tx: &UnboundedSender<AppMessage>) {
        match message {
            AppMessage::ActionFinished(result) => match result {
                Ok(feedback) => {
                    self.set_status(feedback.message.clone(), feedback.tone);
                    self.refresh_after_action(tx, &feedback);
                }
                Err(error) => self.set_status(error, Color::Red),
            },
            AppMessage::AuthFailed(error) => {
                self.auto_login_pending = false;
                self.login_error = Some(error.clone());
                self.session = None;
                self.set_status(error, Color::Red);
            }
            AppMessage::AuthSucceeded {
                base_url,
                runtime,
                token,
                username,
                warning,
            } => {
                self.switch_to_workspace(base_url, runtime, token, username, warning, tx);
                self.request_refresh(tx);
                self.request_view_data(tx);
            }
            AppMessage::WorkspaceRefreshed(result) => {
                self.refresh_in_flight = false;
                match result {
                    Ok(snapshot) => {
                        self.content_error = None;
                        self.disk = snapshot.disk;
                        self.server_info = snapshot.version;
                        self.recorder = snapshot.recorder;
                        self.channels = sort_channels(snapshot.channels);
                        self.video_page = snapshot.videos;
                        self.jobs = snapshot.jobs;
                        self.job_worker_active = snapshot.job_worker.is_processing;
                        self.clamp_selection();
                        self.prefetch_thumbnails(tx);
                    }
                    Err(error) => {
                        if error.contains("401") {
                            if let Some(session) = &self.session {
                                let _ = clear_saved_session(&session.base_url);
                            }
                            self.return_to_login(
                                "Saved session expired. Sign in again.",
                                Color::Yellow,
                            );
                        } else {
                            self.content_error = Some(error.clone());
                            self.set_status(error, Color::Red);
                        }
                    }
                }
            }
            AppMessage::SocketEvent(event) => {
                self.events.insert(0, event.clone());
                self.events.truncate(120);
                self.selected_event = 0;
                self.socket_message = event.summary.clone();
                self.set_status(event.summary, self.theme().accent);
                self.refresh_pending = true;
            }
            AppMessage::SocketStatus { status, message } => {
                self.socket_status = status;
                self.socket_message = message.unwrap_or_default();
            }
            AppMessage::ThumbnailLoaded {
                preview,
                row,
                target,
            } => {
                self.thumbnail_cache
                    .insert(target.key.clone(), ThumbnailEntry::Ready { preview, row });
            }
            AppMessage::ThumbnailFailed { error, target } => {
                self.set_status(
                    format!(
                        "Thumbnail error for {}: {}",
                        truncate(&target.label, 24),
                        truncate(&error, 72)
                    ),
                    Color::Yellow,
                );
                if target.key.starts_with("channel:") {
                    let theme = self.theme();
                    self.cache_placeholder_thumbnail(
                        target.key,
                        target.label,
                        theme.warning,
                        theme.surface_alt_bg,
                    );
                } else {
                    self.thumbnail_cache
                        .insert(target.key.clone(), ThumbnailEntry::Failed { error });
                }
            }
            AppMessage::RandomVideosLoaded(result) => {
                self.view_request_in_flight = false;
                match result {
                    Ok(videos) => {
                        self.content_error = None;
                        self.random_videos = videos;
                        self.selected_video =
                            clamp_index(self.selected_video, self.random_videos.len());
                        self.prefetch_thumbnails(tx);
                    }
                    Err(error) => {
                        self.content_error = Some(error.clone());
                        self.set_status(error, Color::Red);
                    }
                }
            }
            AppMessage::BookmarkVideosLoaded(result) => {
                self.view_request_in_flight = false;
                match result {
                    Ok(videos) => {
                        self.content_error = None;
                        self.bookmark_videos = videos;
                        self.selected_video =
                            clamp_index(self.selected_video, self.bookmark_videos.len());
                        self.prefetch_thumbnails(tx);
                    }
                    Err(error) => {
                        self.content_error = Some(error.clone());
                        self.set_status(error, Color::Red);
                    }
                }
            }
            AppMessage::SimilarityLoaded(result) => {
                self.view_request_in_flight = false;
                match result {
                    Ok(groups) => {
                        self.content_error = None;
                        self.similarity_groups = Some(groups);
                        self.selected_similarity = clamp_index(
                            self.selected_similarity,
                            self.similarity_groups
                                .as_ref()
                                .map(|group| group.groups.len())
                                .unwrap_or(0),
                        );
                        self.prefetch_thumbnails(tx);
                    }
                    Err(error) => {
                        self.content_error = Some(error.clone());
                        self.set_status(error, Color::Red);
                    }
                }
            }
            AppMessage::AdminLoaded(result) => {
                self.view_request_in_flight = false;
                match result {
                    Ok(status) => {
                        self.content_error = None;
                        self.admin_status = status;
                    }
                    Err(error) => {
                        self.content_error = Some(error.clone());
                        self.set_status(error, Color::Red);
                    }
                }
            }
            AppMessage::ChannelLoaded { channel_id, result } => {
                if self.channel_popup.apply_result(channel_id, result) {
                    self.prefetch_thumbnails(tx);
                } else if let Some(error) = self.channel_popup.error() {
                    self.set_status(error.to_string(), Color::Red);
                }
            }
            AppMessage::EnhanceDescriptionsLoaded(result) => match result {
                Ok(descriptions) => {
                    if let Some(form) = self.enhance_form.as_mut() {
                        form.apply_descriptions(descriptions);
                    }
                }
                Err(error) => {
                    if let Some(form) = self.enhance_form.as_mut() {
                        form.set_error(error.clone());
                    }
                    self.set_status(error, Color::Red);
                }
            },
            AppMessage::EnhancementEstimated(result) => match result {
                Ok(estimated_size) => {
                    if let Some(form) = self.enhance_form.as_mut() {
                        form.apply_estimate(estimated_size);
                    }
                    self.set_status(
                        format!("Estimated file size: {}", format_bytes(estimated_size)),
                        self.theme().accent,
                    );
                }
                Err(error) => {
                    if let Some(form) = self.enhance_form.as_mut() {
                        form.set_error(error.clone());
                    }
                    self.set_status(error, Color::Red);
                }
            },
            AppMessage::ProcessesLoaded(result) => {
                self.view_request_in_flight = false;
                match result {
                    Ok(processes) => {
                        self.content_error = None;
                        self.processes = processes;
                        self.selected_process =
                            clamp_index(self.selected_process, self.processes.len());
                    }
                    Err(error) => {
                        self.content_error = Some(error.clone());
                        self.set_status(error, Color::Red);
                    }
                }
            }
            AppMessage::SystemInfoLoaded(result) => {
                self.view_request_in_flight = false;
                match result {
                    Ok(system_info) => {
                        self.content_error = None;
                        self.disk = system_info.disk_info.clone();
                        self.monitor_history.push(MetricSample {
                            cpu_load_percent: average_cpu_percent(&system_info),
                            rx_megabytes: system_info.net_info.receive_bytes / 1024 / 1024,
                            timestamp: Local::now().format("%H:%M:%S").to_string(),
                            tx_megabytes: system_info.net_info.transmit_bytes / 1024 / 1024,
                        });
                        if self.monitor_history.len() > 60 {
                            self.monitor_history.remove(0);
                        }
                        self.system_info = Some(system_info);
                    }
                    Err(error) => {
                        self.content_error = Some(error.clone());
                        self.set_status(error, Color::Red);
                    }
                }
            }
            AppMessage::RecorderToggled(result) => match result {
                Ok(is_recording) => {
                    self.recorder.is_recording = is_recording;
                    self.set_status(
                        if is_recording {
                            "Recorder started."
                        } else {
                            "Recorder stopped."
                        },
                        Color::Green,
                    );
                    self.refresh_pending = true;
                }
                Err(error) => self.set_status(error, Color::Red),
            },
            AppMessage::LoggedOut(result) => match result {
                Ok(()) => self.return_to_login(
                    "Logged out. Enter another server or account to continue.",
                    Color::Green,
                ),
                Err(error) => self.set_status(error, Color::Red),
            },
            AppMessage::VideoPlayer(event) => match event {
                VideoPlayerEvent::Frame {
                    generation,
                    frame,
                    position_seconds,
                } => {
                    if let Some(popup) = self.video_popup.as_mut() {
                        if popup.generation == generation {
                            popup.frame = Some(frame);
                            popup.position_seconds =
                                position_seconds.clamp(0.0, popup.duration_seconds.max(0.0));
                            popup.loading = false;
                            popup.error = None;
                        }
                    }
                }
                VideoPlayerEvent::Ended {
                    generation,
                    position_seconds,
                } => {
                    let mut finished_label = None;
                    if let Some(popup) = self.video_popup.as_mut() {
                        if popup.generation == generation {
                            popup.loading = false;
                            popup.paused = true;
                            popup.position_seconds =
                                position_seconds.clamp(0.0, popup.duration_seconds.max(0.0));
                            finished_label = Some(popup.label.clone());
                        }
                    }
                    if let Some(label) = finished_label {
                        self.video_playback_worker = None;
                        self.set_status(format!("Playback finished: {label}"), self.theme().accent);
                    }
                }
                VideoPlayerEvent::Error { generation, error } => {
                    let mut should_report = false;
                    if let Some(popup) = self.video_popup.as_mut() {
                        if popup.generation == generation {
                            popup.loading = false;
                            popup.error = Some(error.clone());
                            popup.paused = true;
                            should_report = true;
                        }
                    }
                    if should_report {
                        self.video_playback_worker = None;
                        self.set_status(format!("Playback failed: {error}"), Color::Red);
                    }
                }
            },
        }
    }
}

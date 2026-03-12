use crate::{
    player_mode::PlayerMode,
    ui::{RenderedThumbnail, render_video_frame, rendered_video_frame_pixel_dimensions},
};
use anyhow::{Context, Result};
use image::{DynamicImage, RgbImage};
use std::{
    io::{self, BufReader, Read},
    process::{Child, Command, Stdio},
    sync::{
        Arc, Mutex,
        atomic::{AtomicBool, Ordering},
    },
    thread,
    time::{Duration, Instant},
};

pub const VIDEO_PLAYER_FPS: u16 = 6;
pub const VIDEO_PLAYER_SEEK_STEP_SECONDS: f64 = 5.0;
const PLAYBACK_CATCHUP_FRAMES: f64 = 1.25;

#[derive(Debug, Clone)]
pub struct VideoPopupState {
    pub duration_seconds: f64,
    pub error: Option<String>,
    pub frame: Option<RenderedThumbnail>,
    pub generation: u64,
    pub label: String,
    pub loading: bool,
    pub paused: bool,
    pub position_seconds: f64,
    pub url: String,
}

impl VideoPopupState {
    pub fn new(label: String, url: String, duration_seconds: f64, generation: u64) -> Self {
        Self {
            duration_seconds: duration_seconds.max(0.0),
            error: None,
            frame: None,
            generation,
            label,
            loading: true,
            paused: false,
            position_seconds: 0.0,
            url,
        }
    }

    pub fn clamped_position(&self) -> f64 {
        self.position_seconds
            .clamp(0.0, self.duration_seconds.max(0.0))
    }
}

#[derive(Debug, Clone)]
pub enum VideoPlayerEvent {
    Ended {
        generation: u64,
        position_seconds: f64,
    },
    Error {
        generation: u64,
        error: String,
    },
    Frame {
        generation: u64,
        frame: RenderedThumbnail,
        position_seconds: f64,
    },
}

#[derive(Debug, Clone)]
pub struct VideoPlayerRequest {
    pub fps: u16,
    pub generation: u64,
    pub mode: PlayerMode,
    pub single_frame: bool,
    pub start_seconds: f64,
    pub target_height: u16,
    pub target_width: u16,
    pub total_duration_seconds: f64,
    pub url: String,
}

pub struct VideoPlaybackWorker {
    child: Arc<Mutex<Option<Child>>>,
    stop_requested: Arc<AtomicBool>,
}

impl VideoPlaybackWorker {
    pub fn stop(&mut self) {
        self.stop_requested.store(true, Ordering::SeqCst);
        if let Ok(mut guard) = self.child.lock() {
            if let Some(child) = guard.as_mut() {
                let _ = child.kill();
                let _ = child.wait();
            }
            *guard = None;
        }
    }
}

impl Drop for VideoPlaybackWorker {
    fn drop(&mut self) {
        self.stop();
    }
}

pub fn desired_frame_cells(terminal_cols: u16, terminal_rows: u16) -> (u16, u16) {
    let popup_width = terminal_cols.saturating_sub(8).max(40);
    let popup_height = terminal_rows.saturating_sub(6).max(16);
    let frame_width = popup_width.saturating_sub(6).max(24);
    let frame_height = popup_height.saturating_sub(8).max(8);
    (frame_width, frame_height)
}

pub fn spawn_video_worker<F>(request: VideoPlayerRequest, emit: F) -> Result<VideoPlaybackWorker>
where
    F: Fn(VideoPlayerEvent) + Send + 'static,
{
    let mut command = Command::new("ffmpeg");
    command
        .args(ffmpeg_args(&request))
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::piped());

    let mut child = command
        .spawn()
        .context("failed to spawn ffmpeg for popup playback")?;
    let stdout = child
        .stdout
        .take()
        .context("failed to capture ffmpeg stdout for popup playback")?;
    let stderr = child
        .stderr
        .take()
        .context("failed to capture ffmpeg stderr for popup playback")?;

    let child = Arc::new(Mutex::new(Some(child)));
    let stop_requested = Arc::new(AtomicBool::new(false));

    let child_thread = Arc::clone(&child);
    let stop_thread = Arc::clone(&stop_requested);
    thread::spawn(move || {
        let stderr_text = Arc::new(Mutex::new(String::new()));
        let stderr_store = Arc::clone(&stderr_text);
        let stderr_thread = thread::spawn(move || {
            let mut message = String::new();
            let _ = BufReader::new(stderr).read_to_string(&mut message);
            if let Ok(mut guard) = stderr_store.lock() {
                *guard = message;
            }
        });

        let mut reader = BufReader::new(stdout);
        let mut frame_index = 0u64;
        let mut delivered_frame = false;
        let frame_duration = 1.0 / request.fps.max(1) as f64;
        let playback_start = Instant::now();
        let (pixel_width, pixel_height) =
            rendered_video_frame_pixel_dimensions(request.target_width, request.target_height);

        loop {
            match read_rgb24_frame(&mut reader, pixel_width, pixel_height) {
                Ok(Some(image)) => {
                    if stop_thread.load(Ordering::SeqCst) {
                        break;
                    }

                    let frame_offset_seconds = frame_index as f64 * frame_duration;
                    if !request.single_frame {
                        let elapsed_seconds = playback_start.elapsed().as_secs_f64();
                        if elapsed_seconds
                            > frame_offset_seconds + frame_duration * PLAYBACK_CATCHUP_FRAMES
                        {
                            frame_index += 1;
                            continue;
                        }
                    }

                    let rendered = render_video_frame(
                        &DynamicImage::ImageRgb8(image),
                        request.target_width,
                        request.target_height,
                    );
                    let position_seconds = if request.single_frame {
                        request.start_seconds
                    } else {
                        request.start_seconds + frame_offset_seconds
                    }
                    .clamp(0.0, request.total_duration_seconds.max(0.0));

                    if !request.single_frame {
                        let elapsed_seconds = playback_start.elapsed().as_secs_f64();
                        if frame_offset_seconds > elapsed_seconds {
                            thread::sleep(Duration::from_secs_f64(
                                frame_offset_seconds - elapsed_seconds,
                            ));
                        }
                    }

                    delivered_frame = true;
                    emit(VideoPlayerEvent::Frame {
                        generation: request.generation,
                        frame: rendered,
                        position_seconds,
                    });
                    frame_index += 1;

                    if request.single_frame {
                        break;
                    }
                }
                Ok(None) => break,
                Err(error) => {
                    if !stop_thread.load(Ordering::SeqCst) {
                        emit(VideoPlayerEvent::Error {
                            generation: request.generation,
                            error: format!("Video frame decode failed: {error}"),
                        });
                    }
                    break;
                }
            }
        }

        let status = if let Ok(mut guard) = child_thread.lock() {
            if let Some(mut child) = guard.take() {
                child.wait().ok()
            } else {
                None
            }
        } else {
            None
        };

        let _ = stderr_thread.join();

        if stop_thread.load(Ordering::SeqCst) {
            return;
        }

        if let Some(status) = status {
            if !status.success() {
                let stderr = stderr_text
                    .lock()
                    .ok()
                    .map(|guard| guard.trim().to_string())
                    .unwrap_or_default();
                let error = if stderr.is_empty() {
                    format!("ffmpeg exited with {status}")
                } else {
                    stderr
                        .lines()
                        .rev()
                        .find(|line| !line.trim().is_empty())
                        .unwrap_or(stderr.as_str())
                        .to_string()
                };
                emit(VideoPlayerEvent::Error {
                    generation: request.generation,
                    error,
                });
            } else if delivered_frame && !request.single_frame {
                let final_position = if request.total_duration_seconds > 0.0 {
                    request.total_duration_seconds
                } else {
                    (request.start_seconds + frame_index.saturating_sub(1) as f64 * frame_duration)
                        .max(request.start_seconds)
                };
                emit(VideoPlayerEvent::Ended {
                    generation: request.generation,
                    position_seconds: final_position,
                });
            }
        }
    });

    Ok(VideoPlaybackWorker {
        child,
        stop_requested,
    })
}

fn ffmpeg_args(request: &VideoPlayerRequest) -> Vec<String> {
    let (pixel_width, pixel_height) =
        rendered_video_frame_pixel_dimensions(request.target_width, request.target_height);
    let mut args = vec![
        "-nostdin".to_string(),
        "-hide_banner".to_string(),
        "-loglevel".to_string(),
        "error".to_string(),
    ];

    args.push("-ss".to_string());
    args.push(format!("{:.3}", request.start_seconds.max(0.0)));
    args.push("-i".to_string());
    args.push(request.url.clone());
    args.push("-an".to_string());
    args.push("-sn".to_string());
    args.push("-dn".to_string());
    args.push("-vf".to_string());
    args.push(video_filter(
        request.mode,
        request.fps.max(1),
        pixel_width,
        pixel_height,
    ));
    if request.single_frame {
        args.push("-frames:v".to_string());
        args.push("1".to_string());
    }
    args.push("-f".to_string());
    args.push("rawvideo".to_string());
    args.push("-pix_fmt".to_string());
    args.push("rgb24".to_string());
    args.push("-".to_string());
    args
}

fn video_filter(mode: PlayerMode, fps: u16, pixel_width: u32, pixel_height: u32) -> String {
    let mut filters = vec![
        format!("fps={fps}"),
        format!("scale={pixel_width}:{pixel_height}:flags=fast_bilinear"),
    ];

    match mode {
        PlayerMode::Auto | PlayerMode::SmoothColor => {}
        PlayerMode::SharpColor => filters.push("unsharp=5:5:0.8:5:5:0.0".to_string()),
        PlayerMode::AsciiGray => filters.push("format=gray".to_string()),
        PlayerMode::AsciiMono => {
            filters.push("format=gray,eq=contrast=2.0:brightness=0.02".to_string())
        }
    }

    filters.join(",")
}

fn read_rgb24_frame<R: Read>(
    reader: &mut R,
    width: u32,
    height: u32,
) -> io::Result<Option<RgbImage>> {
    let byte_len = width
        .checked_mul(height)
        .and_then(|pixels| pixels.checked_mul(3))
        .ok_or_else(|| io::Error::new(io::ErrorKind::InvalidData, "rgb24 frame too large"))?
        as usize;
    let mut bytes = vec![0u8; byte_len];
    let mut offset = 0usize;
    while offset < byte_len {
        match reader.read(&mut bytes[offset..])? {
            0 if offset == 0 => return Ok(None),
            0 => {
                return Err(io::Error::new(
                    io::ErrorKind::UnexpectedEof,
                    "unexpected end of stream inside rgb24 frame",
                ));
            }
            read => offset += read,
        }
    }
    let image = RgbImage::from_raw(width, height, bytes).ok_or_else(|| {
        io::Error::new(
            io::ErrorKind::InvalidData,
            "failed to construct rgb24 frame image",
        )
    })?;
    Ok(Some(image))
}

#[cfg(test)]
mod tests {
    use super::{
        VIDEO_PLAYER_SEEK_STEP_SECONDS, VideoPlayerRequest, desired_frame_cells, ffmpeg_args,
        read_rgb24_frame, video_filter,
    };
    use crate::player_mode::PlayerMode;

    #[test]
    fn parses_single_rgb24_frame() {
        let data = b"\xff\x00\x00\x00\xff\x00";
        let mut reader = &data[..];
        let frame = read_rgb24_frame(&mut reader, 2, 1).unwrap().unwrap();

        assert_eq!(frame.width(), 2);
        assert_eq!(frame.height(), 1);
    }

    #[test]
    fn desired_size_stays_positive() {
        let (width, height) = desired_frame_cells(40, 20);
        assert!(width >= 24);
        assert!(height >= 8);
        assert_eq!(VIDEO_PLAYER_SEEK_STEP_SECONDS, 5.0);
    }

    #[test]
    fn video_filter_reflects_mode() {
        let sharp = video_filter(PlayerMode::SharpColor, 6, 80, 40);
        let mono = video_filter(PlayerMode::AsciiMono, 6, 80, 40);

        assert!(sharp.contains("unsharp="));
        assert!(mono.contains("format=gray"));
    }

    #[test]
    fn popup_ffmpeg_args_do_not_use_realtime_input_throttling() {
        let args = ffmpeg_args(&VideoPlayerRequest {
            fps: 6,
            generation: 1,
            mode: PlayerMode::SmoothColor,
            single_frame: false,
            start_seconds: 0.0,
            target_height: 12,
            target_width: 32,
            total_duration_seconds: 60.0,
            url: "http://localhost:3000/videos/test.mp4".to_string(),
        });

        assert!(!args.iter().any(|arg| arg == "-re"));
        assert!(args.iter().any(|arg| arg == "rawvideo"));
        assert!(args.iter().any(|arg| arg == "rgb24"));
    }
}

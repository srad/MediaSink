use crate::serde_ext::null_default;
use anyhow::{Context, Result, anyhow, bail};
use regex::Regex;
use reqwest::{Client, Method, RequestBuilder};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use std::{fs, path::Path};
use url::Url;

#[allow(dead_code)]
#[derive(Debug, Clone)]
pub struct RuntimeConfig {
    pub api_url: String,
    pub api_version: String,
    pub base_url: String,
    pub build: Option<String>,
    pub file_url: String,
    pub socket_url: String,
    pub version: Option<String>,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct RecorderStatusResponse {
    #[serde(rename = "isRecording", default)]
    pub is_recording: bool,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct JobWorkerStatus {
    #[serde(rename = "isProcessing", default)]
    pub is_processing: bool,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct VideoPreview {
    #[serde(rename = "previewPath", default)]
    pub preview_path: String,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct ChannelInfo {
    #[serde(rename = "channelId", default)]
    pub channel_id: u64,
    #[serde(rename = "channelName", default)]
    pub channel_name: String,
    #[serde(default)]
    pub deleted: bool,
    #[serde(rename = "displayName", default)]
    pub display_name: String,
    #[serde(default)]
    pub url: String,
    #[serde(default)]
    pub fav: bool,
    #[serde(rename = "isPaused", default)]
    pub is_paused: bool,
    #[serde(rename = "isOnline", default)]
    pub is_online: bool,
    #[serde(rename = "isRecording", default)]
    pub is_recording: bool,
    #[serde(rename = "isTerminating", default)]
    pub is_terminating: bool,
    #[serde(rename = "recordingsCount", default)]
    pub recordings_count: u64,
    #[serde(rename = "recordingsSize", default)]
    pub recordings_size: u64,
    #[serde(rename = "minRecording", default)]
    pub min_recording: f64,
    #[serde(default)]
    pub preview: String,
    #[serde(default, deserialize_with = "null_default")]
    pub recordings: Vec<Recording>,
    #[serde(rename = "skipStart", default)]
    pub skip_start: f64,
    #[serde(default)]
    pub tags: Value,
    #[serde(rename = "minDuration", default)]
    pub min_duration: f64,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct Recording {
    #[serde(rename = "recordingId", default)]
    pub recording_id: u64,
    #[serde(rename = "channelName", default)]
    pub channel_name: String,
    #[serde(rename = "channelId", default)]
    pub channel_id: u64,
    #[serde(default)]
    pub filename: String,
    #[serde(rename = "createdAt", default)]
    pub created_at: String,
    #[serde(default)]
    pub duration: f64,
    #[serde(default)]
    pub size: u64,
    #[serde(rename = "bitRate", default)]
    pub bit_rate: u64,
    #[serde(rename = "videoType", default)]
    pub video_type: String,
    #[serde(default)]
    pub width: u64,
    #[serde(default)]
    pub height: u64,
    #[serde(rename = "pathRelative", default)]
    pub path_relative: String,
    #[serde(default)]
    pub bookmark: bool,
    #[serde(rename = "videoPreview", default)]
    pub video_preview: Option<VideoPreview>,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct Job {
    #[serde(rename = "jobId", default)]
    pub job_id: u64,
    #[serde(rename = "recordingId", default)]
    pub recording_id: u64,
    #[serde(rename = "channelId", default)]
    pub channel_id: u64,
    #[serde(rename = "channelName", default)]
    pub channel_name: String,
    #[serde(default)]
    pub filename: String,
    #[serde(default)]
    pub task: String,
    #[serde(default)]
    pub status: String,
    #[serde(default)]
    pub progress: Option<String>,
    #[serde(default)]
    pub pid: Option<i64>,
    #[serde(default)]
    pub info: Option<String>,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct JobsResponse {
    #[serde(default, deserialize_with = "null_default")]
    pub jobs: Vec<Job>,
    #[serde(rename = "totalCount", default)]
    pub total_count: i64,
    #[serde(default)]
    pub skip: usize,
    #[serde(default)]
    pub take: usize,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct VideosResponse {
    #[serde(default, deserialize_with = "null_default")]
    pub videos: Vec<Recording>,
    #[serde(rename = "totalCount", default)]
    pub total_count: i64,
    #[serde(default)]
    pub skip: usize,
    #[serde(default)]
    pub take: usize,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct ServerInfo {
    #[serde(default)]
    pub commit: String,
    #[serde(default)]
    pub version: String,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct ImportInfoResponse {
    #[serde(rename = "isImporting", default)]
    pub is_importing: bool,
    #[serde(default)]
    pub progress: u64,
    #[serde(default)]
    pub size: u64,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct PreviewRegenerationProgress {
    #[serde(rename = "current", default)]
    pub current: u64,
    #[serde(rename = "currentVideo", default)]
    pub current_video: String,
    #[serde(rename = "isRunning", default)]
    pub is_running: bool,
    #[serde(default)]
    pub total: u64,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct ProcessInfo {
    #[serde(default)]
    pub args: String,
    #[serde(default)]
    pub path: String,
    #[serde(default)]
    pub pid: u64,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct UtilCPULoad {
    #[serde(default)]
    pub load: f64,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct UtilCPUInfo {
    #[serde(rename = "loadCpu", default)]
    pub load_cpu: Vec<UtilCPULoad>,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct UtilDiskInfo {
    #[serde(default)]
    pub pcent: f64,
    #[serde(rename = "sizeFormattedGb", default)]
    pub size_formatted_gb: f64,
    #[serde(rename = "usedFormattedGb", default)]
    pub used_formatted_gb: f64,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct UtilNetInfo {
    #[serde(default)]
    pub dev: String,
    #[serde(rename = "receiveBytes", default)]
    pub receive_bytes: u64,
    #[serde(rename = "transmitBytes", default)]
    pub transmit_bytes: u64,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct UtilSysInfo {
    #[serde(rename = "cpuInfo", default)]
    pub cpu_info: UtilCPUInfo,
    #[serde(rename = "diskInfo", default)]
    pub disk_info: UtilDiskInfo,
    #[serde(rename = "netInfo", default)]
    pub net_info: UtilNetInfo,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct SimilarVideoGroup {
    #[serde(rename = "groupId", default)]
    pub group_id: u64,
    #[serde(rename = "maxSimilarity", default)]
    pub max_similarity: f64,
    #[serde(default, deserialize_with = "null_default")]
    pub videos: Vec<Recording>,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct SimilarityGroupsResponse {
    #[serde(rename = "analyzedCount", default)]
    pub analyzed_count: u64,
    #[serde(rename = "groupCount", default)]
    pub group_count: u64,
    #[serde(default, deserialize_with = "null_default")]
    pub groups: Vec<SimilarVideoGroup>,
    #[serde(rename = "similarityThreshold", default)]
    pub similarity_threshold: f64,
}

#[derive(Debug, Clone)]
pub struct WorkspaceSnapshot {
    pub channels: Vec<ChannelInfo>,
    pub disk: UtilDiskInfo,
    pub job_worker: JobWorkerStatus,
    pub jobs: JobsResponse,
    pub recorder: RecorderStatusResponse,
    pub version: ServerInfo,
    pub videos: VideosResponse,
}

#[derive(Debug, Clone)]
pub struct AuthSuccess {
    pub runtime: RuntimeConfig,
    pub token: String,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ChannelRequest {
    #[serde(rename = "channelName")]
    pub channel_name: String,
    #[serde(rename = "displayName")]
    pub display_name: String,
    #[serde(rename = "skipStart")]
    pub skip_start: u64,
    #[serde(rename = "minDuration")]
    pub min_duration: u64,
    #[serde(rename = "url")]
    pub url: String,
    #[serde(rename = "isPaused")]
    pub is_paused: bool,
    #[serde(default)]
    pub tags: Option<Vec<String>>,
    #[serde(default)]
    pub fav: bool,
    #[serde(default)]
    pub deleted: bool,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct EstimateEnhanceRequest {
    #[serde(rename = "targetResolution")]
    pub target_resolution: String,
    #[serde(rename = "denoiseStrength")]
    pub denoise_strength: f64,
    #[serde(rename = "sharpenStrength")]
    pub sharpen_strength: f64,
    #[serde(rename = "applyNormalize")]
    pub apply_normalize: bool,
    #[serde(rename = "encodingPreset")]
    pub encoding_preset: String,
    #[serde(rename = "crf", default)]
    pub crf: Option<u64>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct EnhanceRequest {
    #[serde(rename = "recordingId")]
    pub recording_id: u64,
    #[serde(rename = "targetResolution")]
    pub target_resolution: String,
    #[serde(rename = "denoiseStrength")]
    pub denoise_strength: f64,
    #[serde(rename = "sharpenStrength")]
    pub sharpen_strength: f64,
    #[serde(rename = "applyNormalize")]
    pub apply_normalize: bool,
    #[serde(rename = "encodingPreset")]
    pub encoding_preset: String,
    #[serde(rename = "crf", default)]
    pub crf: Option<u64>,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct EnhancementPresetDescription {
    #[serde(default)]
    pub description: String,
    #[serde(rename = "encodeSpeed", default)]
    pub encode_speed: String,
    #[serde(default)]
    pub label: String,
    #[serde(default)]
    pub preset: String,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct EnhancementCrfDescription {
    #[serde(rename = "approxRatio", default)]
    pub approx_ratio: f64,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub label: String,
    #[serde(default)]
    pub quality: String,
    #[serde(default)]
    pub value: u64,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct EnhancementResolutionDescription {
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub dimensions: String,
    #[serde(default)]
    pub resolution: String,
    #[serde(rename = "useCase", default)]
    pub use_case: String,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct EnhancementRangeSetting {
    #[serde(default)]
    pub description: String,
    #[serde(rename = "maxValue", default)]
    pub max_value: f64,
    #[serde(rename = "minValue", default)]
    pub min_value: f64,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub recommended: f64,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Default, Deserialize)]
pub struct EnhancementToggleSetting {
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub recommended: bool,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct EnhancementFilters {
    #[serde(rename = "applyNormalize", default)]
    pub apply_normalize: EnhancementToggleSetting,
    #[serde(rename = "denoiseStrength", default)]
    pub denoise_strength: EnhancementRangeSetting,
    #[serde(rename = "sharpenStrength", default)]
    pub sharpen_strength: EnhancementRangeSetting,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct EnhancementDescriptions {
    #[serde(rename = "crfValues", default, deserialize_with = "null_default")]
    pub crf_values: Vec<EnhancementCrfDescription>,
    #[serde(default)]
    pub filters: EnhancementFilters,
    #[serde(default, deserialize_with = "null_default")]
    pub presets: Vec<EnhancementPresetDescription>,
    #[serde(default, deserialize_with = "null_default")]
    pub resolutions: Vec<EnhancementResolutionDescription>,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct EstimateEnhancementResponse {
    #[serde(rename = "estimatedFileSize", default)]
    pub estimated_file_size: u64,
}

#[derive(Debug, Clone)]
pub struct ApiClient {
    http: Client,
    pub runtime: RuntimeConfig,
    token: Option<String>,
}

#[derive(Debug, Clone)]
pub struct AdminStatus {
    pub import: ImportInfoResponse,
    pub previews: PreviewRegenerationProgress,
    pub video_updating: bool,
}

fn parse_assignments(source: &str) -> Result<std::collections::HashMap<String, String>> {
    let pattern = Regex::new(r#"window\.(APP_[A-Z_]+)\s*=\s*(?:"([^"]*)"|'([^']*)');"#)
        .context("failed to compile env.js parser")?;
    let mut values = std::collections::HashMap::new();
    for captures in pattern.captures_iter(source) {
        let value = captures
            .get(2)
            .or_else(|| captures.get(3))
            .map(|capture| capture.as_str())
            .unwrap_or_default();
        values.insert(captures[1].to_string(), value.to_string());
    }
    Ok(values)
}

async fn safe_fetch_text(client: &Client, url: &str) -> Option<String> {
    let response = client.get(url).send().await.ok()?;
    if !response.status().is_success() {
        return None;
    }
    response.text().await.ok()
}

fn non_empty_string(value: Option<String>) -> Option<String> {
    value.and_then(|value| {
        let trimmed = value.trim();
        if trimmed.is_empty() {
            None
        } else {
            Some(trimmed.to_string())
        }
    })
}

fn non_empty_str(value: Option<&str>) -> Option<String> {
    value.and_then(|value| {
        let trimmed = value.trim();
        if trimmed.is_empty() {
            None
        } else {
            Some(trimmed.to_string())
        }
    })
}

fn expected_api_version(profile_api_version: Option<&str>) -> String {
    non_empty_string(std::env::var("MEDIASINK_API_VERSION").ok())
        .or_else(|| non_empty_str(profile_api_version))
        .unwrap_or_else(|| "0.1.0".to_string())
}

pub async fn resolve_runtime_config(
    base_url: &str,
    profile_api_version: Option<&str>,
) -> Result<RuntimeConfig> {
    let client = Client::builder()
        .redirect(reqwest::redirect::Policy::limited(5))
        .build()
        .context("failed to construct HTTP client")?;

    let env_script = safe_fetch_text(&client, &format!("{base_url}/env.js")).await;
    let build_script = safe_fetch_text(&client, &format!("{base_url}/build.js")).await;

    let env_values = env_script
        .as_deref()
        .map(parse_assignments)
        .transpose()?
        .unwrap_or_default();
    let build_values = build_script
        .as_deref()
        .map(parse_assignments)
        .transpose()?
        .unwrap_or_default();

    let api_url = Url::parse(base_url)?
        .join(
            env_values
                .get("APP_APIURL")
                .map(String::as_str)
                .unwrap_or("/api/v1"),
        )?
        .to_string();
    let file_url = Url::parse(base_url)?
        .join(
            env_values
                .get("APP_FILEURL")
                .map(String::as_str)
                .unwrap_or("/videos"),
        )?
        .to_string();

    let socket_url = env_values
        .get("APP_SOCKETURL")
        .cloned()
        .unwrap_or_else(|| format!("{base_url}/api/v1/ws"))
        .replacen("http", "ws", 1);

    let client_api_version = expected_api_version(profile_api_version);
    let server_api_version = non_empty_string(build_values.get("APP_API_VERSION").cloned())
        .ok_or_else(|| anyhow!("server did not expose APP_API_VERSION in /build.js"))?;

    if server_api_version != client_api_version {
        bail!(
            "client API version incompatible with server API version {}",
            server_api_version
        );
    }

    Ok(RuntimeConfig {
        api_url,
        api_version: server_api_version,
        base_url: base_url.to_string(),
        build: build_values.get("APP_BUILD").cloned(),
        file_url,
        socket_url,
        version: build_values.get("APP_VERSION").cloned(),
    })
}

impl ApiClient {
    pub fn new(runtime: RuntimeConfig, token: Option<String>) -> Result<Self> {
        let http = Client::builder()
            .redirect(reqwest::redirect::Policy::limited(5))
            .build()
            .context("failed to construct HTTP client")?;
        Ok(Self {
            http,
            runtime,
            token,
        })
    }

    fn request(&self, method: Method, path: &str) -> Result<RequestBuilder> {
        let mut api_root = self.runtime.api_url.clone();
        if !api_root.ends_with('/') {
            api_root.push('/');
        }
        let url = Url::parse(&api_root)?.join(path)?;
        let mut request = self
            .http
            .request(method, url)
            .header("X-API-Version", &self.runtime.api_version);

        if let Some(token) = &self.token {
            request = request.bearer_auth(token);
        }

        Ok(request)
    }

    async fn expect_json<T>(&self, request: RequestBuilder) -> Result<T>
    where
        T: for<'de> Deserialize<'de>,
    {
        let response = request.send().await.context("request failed")?;
        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            bail!("{status}: {body}");
        }
        response
            .json::<T>()
            .await
            .context("failed to decode response JSON")
    }

    async fn expect_ok(&self, request: RequestBuilder) -> Result<()> {
        let response = request.send().await.context("request failed")?;
        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            bail!("{status}: {body}");
        }
        Ok(())
    }

    pub async fn authenticate(
        base_url: &str,
        profile_api_version: Option<&str>,
        username: &str,
        password: &str,
        signup: bool,
    ) -> Result<AuthSuccess> {
        let runtime = resolve_runtime_config(base_url, profile_api_version).await?;
        let client = Self::new(runtime.clone(), None)?;
        let body = json!({
            "username": username,
            "password": password,
        });

        if signup {
            client
                .expect_ok(client.request(Method::POST, "auth/signup")?.json(&body))
                .await
                .context("signup failed")?;
        }

        let response: Value = client
            .expect_json(client.request(Method::POST, "auth/login")?.json(&body))
            .await
            .context("login failed")?;
        let token = response
            .get("token")
            .and_then(Value::as_str)
            .ok_or_else(|| anyhow!("The server did not return an auth token."))?
            .to_string();

        Ok(AuthSuccess { runtime, token })
    }

    pub async fn verify(&self) -> Result<Value> {
        self.expect_json(self.request(Method::GET, "admin/version")?)
            .await
    }

    pub async fn logout(&self) -> Result<()> {
        self.expect_ok(self.request(Method::POST, "auth/logout")?)
            .await
    }

    pub async fn refresh_snapshot(
        &self,
        video_skip: usize,
        video_take: usize,
        video_sort_column: &str,
        video_sort_order: &str,
        jobs_open_only: bool,
    ) -> Result<WorkspaceSnapshot> {
        let version_request = self.request(Method::GET, "admin/version")?;
        let disk_request = self.request(Method::GET, "info/disk")?;
        let recorder_request = self.request(Method::GET, "recorder")?;
        let channels_request = self.request(Method::GET, "channels")?;
        let videos_request = self.request(Method::POST, "videos/filter")?.json(&json!({
            "skip": video_skip,
            "take": video_take,
            "sortColumn": video_sort_column,
            "sortOrder": video_sort_order,
        }));
        let jobs_request = self.request(Method::POST, "jobs/list")?.json(&json!({
            "skip": 0,
            "take": 20,
            "states": if jobs_open_only { json!(["open"]) } else { json!(["open", "completed", "error", "canceled"]) },
            "sortOrder": "DESC",
        }));
        let worker_request = self.request(Method::GET, "jobs/worker")?;

        let (version, disk, recorder, channels, videos, jobs, job_worker) = tokio::try_join!(
            self.expect_json::<ServerInfo>(version_request),
            self.expect_json::<UtilDiskInfo>(disk_request),
            self.expect_json::<RecorderStatusResponse>(recorder_request),
            self.expect_json::<Vec<ChannelInfo>>(channels_request),
            self.expect_json::<VideosResponse>(videos_request),
            self.expect_json::<JobsResponse>(jobs_request),
            self.expect_json::<JobWorkerStatus>(worker_request),
        )?;

        Ok(WorkspaceSnapshot {
            channels,
            disk,
            job_worker,
            jobs,
            recorder,
            version,
            videos,
        })
    }

    pub async fn start_recorder(&self) -> Result<()> {
        self.expect_ok(self.request(Method::POST, "recorder/resume")?)
            .await
    }

    pub async fn stop_recorder(&self) -> Result<()> {
        self.expect_ok(self.request(Method::POST, "recorder/pause")?)
            .await
    }

    pub async fn random_videos(&self, limit: usize) -> Result<Vec<Recording>> {
        self.expect_json(self.request(Method::GET, &format!("videos/random/{limit}"))?)
            .await
    }

    pub async fn bookmarks(&self) -> Result<Vec<Recording>> {
        self.expect_json(self.request(Method::GET, "videos/bookmarks")?)
            .await
    }

    pub async fn similarity_groups(
        &self,
        similarity: f64,
        pair_limit: usize,
        include_singletons: bool,
    ) -> Result<SimilarityGroupsResponse> {
        self.expect_json(self.request(Method::POST, "analysis/group")?.json(&json!({
            "similarity": similarity,
            "pairLimit": pair_limit,
            "includeSingletons": include_singletons,
        })))
        .await
    }

    pub async fn channel(&self, id: u64) -> Result<ChannelInfo> {
        self.expect_json(self.request(Method::GET, &format!("channels/{id}"))?)
            .await
    }

    pub async fn create_channel(&self, body: &ChannelRequest) -> Result<()> {
        self.expect_ok(self.request(Method::POST, "channels")?.json(body))
            .await
    }

    pub async fn update_channel(&self, id: u64, body: &ChannelRequest) -> Result<()> {
        self.expect_ok(
            self.request(Method::PATCH, &format!("channels/{id}"))?
                .json(body),
        )
        .await
    }

    pub async fn delete_channel(&self, id: u64) -> Result<()> {
        self.expect_ok(self.request(Method::DELETE, &format!("channels/{id}"))?)
            .await
    }

    pub async fn pause_channel(&self, id: u64) -> Result<()> {
        self.expect_ok(self.request(Method::POST, &format!("channels/{id}/pause"))?)
            .await
    }

    pub async fn resume_channel(&self, id: u64) -> Result<()> {
        self.expect_ok(self.request(Method::POST, &format!("channels/{id}/resume"))?)
            .await
    }

    pub async fn fav_channel(&self, id: u64) -> Result<()> {
        self.expect_ok(self.request(Method::PATCH, &format!("channels/{id}/fav"))?)
            .await
    }

    pub async fn unfav_channel(&self, id: u64) -> Result<()> {
        self.expect_ok(self.request(Method::PATCH, &format!("channels/{id}/unfav"))?)
            .await
    }

    pub async fn delete_video(&self, id: u64) -> Result<()> {
        self.expect_ok(self.request(Method::DELETE, &format!("videos/{id}"))?)
            .await
    }

    pub async fn fav_video(&self, id: u64) -> Result<()> {
        self.expect_ok(self.request(Method::PATCH, &format!("videos/{id}/fav"))?)
            .await
    }

    pub async fn unfav_video(&self, id: u64) -> Result<()> {
        self.expect_ok(self.request(Method::PATCH, &format!("videos/{id}/unfav"))?)
            .await
    }

    pub async fn analyze_video(&self, id: u64) -> Result<()> {
        self.expect_ok(self.request(Method::POST, &format!("analysis/{id}"))?)
            .await
    }

    pub async fn generate_video_preview(&self, id: u64) -> Result<()> {
        self.expect_ok(self.request(Method::POST, &format!("videos/{id}/preview"))?)
            .await
    }

    pub async fn convert_video(&self, id: u64, media_type: &str) -> Result<()> {
        self.expect_ok(self.request(Method::POST, &format!("videos/{id}/{media_type}/convert"))?)
            .await
    }

    pub async fn enhancement_descriptions(&self) -> Result<EnhancementDescriptions> {
        self.expect_json(self.request(Method::GET, "videos/enhance/descriptions")?)
            .await
    }

    pub async fn estimate_video_enhancement(
        &self,
        id: u64,
        body: &EstimateEnhanceRequest,
    ) -> Result<EstimateEnhancementResponse> {
        self.expect_json(
            self.request(Method::POST, &format!("videos/{id}/estimate-enhancement"))?
                .json(body),
        )
        .await
    }

    pub async fn enhance_video(&self, id: u64, body: &EnhanceRequest) -> Result<()> {
        self.expect_ok(
            self.request(Method::POST, &format!("videos/{id}/enhance"))?
                .json(body),
        )
        .await
    }

    pub async fn download_video(&self, id: u64, destination: &Path) -> Result<()> {
        let request = self.request(Method::GET, &format!("videos/{id}/download"))?;
        let response = request.send().await.context("request failed")?;
        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            bail!("{status}: {body}");
        }

        if let Some(parent) = destination.parent() {
            fs::create_dir_all(parent).context("failed to create download directory")?;
        }
        let bytes = response
            .bytes()
            .await
            .context("failed to read download response")?;
        fs::write(destination, bytes).context("failed to write download file")?;
        Ok(())
    }

    pub async fn admin_status(&self) -> Result<AdminStatus> {
        let version_request = self.request(Method::GET, "admin/version")?;
        let import_request = self.request(Method::GET, "admin/import")?;
        let preview_request = self.request(Method::GET, "previews/regenerate")?;
        let video_updating_request = self.request(Method::GET, "videos/isupdating")?;

        let (_version, import, previews, video_updating) = tokio::try_join!(
            self.expect_json::<ServerInfo>(version_request),
            self.expect_json::<ImportInfoResponse>(import_request),
            self.expect_json::<PreviewRegenerationProgress>(preview_request),
            self.expect_json::<bool>(video_updating_request),
        )?;

        Ok(AdminStatus {
            import,
            previews,
            video_updating,
        })
    }

    pub async fn system_info(&self, seconds: u64) -> Result<UtilSysInfo> {
        self.expect_json(self.request(Method::GET, &format!("info/{seconds}"))?)
            .await
    }

    pub async fn processes(&self) -> Result<Vec<ProcessInfo>> {
        self.expect_json(self.request(Method::GET, "processes")?)
            .await
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SocketEnvelope {
    pub data: Value,
    pub name: String,
}

#[cfg(test)]
mod tests {
    use super::*;
    use anyhow::Result;
    use serde_json::json;
    use std::{
        collections::HashMap,
        sync::{Arc, Mutex},
    };
    use tokio::{
        io::{AsyncReadExt, AsyncWriteExt},
        net::TcpListener,
        task::JoinHandle,
    };

    #[derive(Debug, Clone)]
    struct ObservedRequest {
        body: String,
        headers: HashMap<String, String>,
        method: String,
        path: String,
    }

    async fn spawn_test_server(
        responses: Vec<(&'static str, &'static str, String)>,
    ) -> Result<(String, Arc<Mutex<Vec<ObservedRequest>>>, JoinHandle<()>)> {
        let listener = TcpListener::bind("127.0.0.1:0").await?;
        let address = listener.local_addr()?;
        let requests = Arc::new(Mutex::new(Vec::new()));
        let recorded = requests.clone();

        let handle = tokio::spawn(async move {
            for (status, content_type, body) in responses {
                let (mut stream, _) = listener.accept().await.expect("accept failed");
                let request = read_request(&mut stream)
                    .await
                    .expect("read request failed");
                recorded
                    .lock()
                    .expect("request lock poisoned")
                    .push(parse_request(&request));

                let response = format!(
                    "HTTP/1.1 {status}\r\nContent-Type: {content_type}\r\nContent-Length: {}\r\nConnection: close\r\n\r\n{body}",
                    body.len()
                );
                stream
                    .write_all(response.as_bytes())
                    .await
                    .expect("write response failed");
            }
        });

        Ok((format!("http://{address}"), requests, handle))
    }

    async fn read_request(stream: &mut tokio::net::TcpStream) -> Result<String> {
        let mut buffer = Vec::new();
        let mut chunk = [0u8; 4096];
        let mut header_end = None;
        let mut content_length = 0usize;

        loop {
            let read = stream.read(&mut chunk).await?;
            if read == 0 {
                break;
            }
            buffer.extend_from_slice(&chunk[..read]);

            if header_end.is_none() {
                if let Some(position) = buffer.windows(4).position(|window| window == b"\r\n\r\n") {
                    let end = position + 4;
                    header_end = Some(end);
                    content_length = parse_content_length(&buffer[..end]);
                }
            }

            if let Some(end) = header_end {
                if buffer.len() >= end + content_length {
                    break;
                }
            }
        }

        Ok(String::from_utf8(buffer).context("request was not valid UTF-8")?)
    }

    fn parse_content_length(headers: &[u8]) -> usize {
        let text = String::from_utf8_lossy(headers);
        text.lines()
            .find_map(|line| {
                let (name, value) = line.split_once(':')?;
                if name.eq_ignore_ascii_case("content-length") {
                    value.trim().parse::<usize>().ok()
                } else {
                    None
                }
            })
            .unwrap_or(0)
    }

    fn parse_request(raw: &str) -> ObservedRequest {
        let (head, body) = raw.split_once("\r\n\r\n").unwrap_or((raw, ""));
        let mut lines = head.lines();
        let request_line = lines.next().unwrap_or_default();
        let mut parts = request_line.split_whitespace();
        let method = parts.next().unwrap_or_default().to_string();
        let path = parts.next().unwrap_or_default().to_string();
        let headers = lines
            .filter_map(|line| {
                let (name, value) = line.split_once(':')?;
                Some((name.trim().to_ascii_lowercase(), value.trim().to_string()))
            })
            .collect::<HashMap<_, _>>();

        ObservedRequest {
            body: body.to_string(),
            headers,
            method,
            path,
        }
    }

    #[tokio::test]
    async fn authenticate_posts_login_and_returns_token() -> Result<()> {
        let responses = vec![
            ("200 OK", "application/javascript", "window.APP_APIURL='/api/v1'; window.APP_FILEURL='/videos'; window.APP_SOCKETURL='ws://127.0.0.1/ws';".to_string()),
            ("200 OK", "application/javascript", "window.APP_API_VERSION='0.1.0';".to_string()),
            ("200 OK", "application/json", json!({ "token": "session-token" }).to_string()),
        ];
        let (base_url, requests, handle) = spawn_test_server(responses).await?;

        let success = ApiClient::authenticate(&base_url, None, "alice", "secret", false).await?;

        handle.await?;
        let requests = requests.lock().expect("request lock poisoned").clone();
        assert_eq!(success.token, "session-token");
        assert_eq!(success.runtime.api_version, "0.1.0");
        assert_eq!(requests.len(), 3);
        assert_eq!(requests[0].method, "GET");
        assert_eq!(requests[0].path, "/env.js");
        assert_eq!(requests[1].path, "/build.js");
        assert_eq!(requests[2].method, "POST");
        assert_eq!(requests[2].path, "/api/v1/auth/login");
        assert_eq!(
            serde_json::from_str::<Value>(&requests[2].body)?,
            json!({
                "username": "alice",
                "password": "secret",
            })
        );
        assert_eq!(
            requests[2].headers.get("x-api-version").map(String::as_str),
            Some("0.1.0")
        );

        Ok(())
    }

    #[tokio::test]
    async fn authenticate_signup_runs_signup_before_login() -> Result<()> {
        let responses = vec![
            (
                "200 OK",
                "application/javascript",
                "window.APP_APIURL='/api/v1';".to_string(),
            ),
            (
                "200 OK",
                "application/javascript",
                "window.APP_API_VERSION='fallback-v1';".to_string(),
            ),
            ("200 OK", "application/json", "{}".to_string()),
            (
                "200 OK",
                "application/json",
                json!({ "token": "new-token" }).to_string(),
            ),
        ];
        let (base_url, requests, handle) = spawn_test_server(responses).await?;

        let success =
            ApiClient::authenticate(&base_url, Some("fallback-v1"), "bob", "s3cr3t", true).await?;

        handle.await?;
        let requests = requests.lock().expect("request lock poisoned").clone();
        assert_eq!(success.token, "new-token");
        assert_eq!(success.runtime.api_version, "fallback-v1");
        assert_eq!(requests[2].path, "/api/v1/auth/signup");
        assert_eq!(requests[3].path, "/api/v1/auth/login");

        Ok(())
    }

    #[tokio::test]
    async fn authenticate_errors_when_server_returns_no_token() -> Result<()> {
        let responses = vec![
            (
                "200 OK",
                "application/javascript",
                "window.APP_APIURL='/api/v1';".to_string(),
            ),
            (
                "200 OK",
                "application/javascript",
                "window.APP_API_VERSION='0.1.0';".to_string(),
            ),
            ("200 OK", "application/json", "{}".to_string()),
        ];
        let (base_url, _requests, handle) = spawn_test_server(responses).await?;

        let error = ApiClient::authenticate(&base_url, None, "alice", "secret", false)
            .await
            .expect_err("authenticate should fail without a token");

        handle.await?;
        assert!(error.to_string().contains("did not return an auth token"));

        Ok(())
    }

    #[tokio::test]
    async fn authenticate_rejects_empty_server_api_version() -> Result<()>
    {
        let responses = vec![
            (
                "200 OK",
                "application/javascript",
                "window.APP_APIURL='/api/v1';".to_string(),
            ),
            (
                "200 OK",
                "application/javascript",
                "window.APP_API_VERSION='';".to_string(),
            ),
            (
                "200 OK",
                "application/json",
                json!({ "token": "fallback-token" }).to_string(),
            ),
        ];
        let (base_url, _requests, handle) = spawn_test_server(responses).await?;

        let error = ApiClient::authenticate(&base_url, Some("fallback-v1"), "alice", "secret", false)
            .await
            .expect_err("authenticate should fail when APP_API_VERSION is empty");

        handle.await?;
        assert!(error
            .to_string()
            .contains("server did not expose APP_API_VERSION"));

        Ok(())
    }

    #[tokio::test]
    async fn authenticate_rejects_mismatched_server_api_version() -> Result<()> {
        let responses = vec![
            (
                "200 OK",
                "application/javascript",
                "window.APP_APIURL='/api/v1';".to_string(),
            ),
            (
                "200 OK",
                "application/javascript",
                "window.APP_API_VERSION='server-v2';".to_string(),
            ),
        ];
        let (base_url, _requests, handle) = spawn_test_server(responses).await?;

        let error = ApiClient::authenticate(&base_url, Some("client-v1"), "alice", "secret", false)
            .await
            .expect_err("authenticate should fail when API versions differ");

        handle.await?;
        assert!(error
            .to_string()
            .contains("client API version incompatible with server API version server-v2"));

        Ok(())
    }

    #[test]
    fn channel_info_accepts_null_recordings_array() {
        let payload = json!({
            "channelId": 7,
            "channelName": "demo",
            "displayName": "Demo",
            "url": "https://example.com",
            "fav": false,
            "isPaused": false,
            "isOnline": false,
            "isRecording": false,
            "recordingsCount": 0,
            "recordingsSize": 0,
            "minRecording": 0,
            "preview": "demo/.previews/live.jpg",
            "recordings": null,
            "skipStart": 0,
            "tags": null,
            "minDuration": 0
        });

        let parsed: ChannelInfo =
            serde_json::from_value(payload).expect("channel info should decode");

        assert!(parsed.recordings.is_empty());
    }
}

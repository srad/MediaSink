use anyhow::{Context, Result};
use chrono::Utc;
use keyring::Entry;
use serde::{Deserialize, Serialize};
use std::{collections::BTreeMap, fs, path::PathBuf};
use url::Url;

const DEFAULT_SERVER: &str = "http://localhost:3000";
const SERVICE_NAME: &str = "mediasink-cli";

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct CliProfile {
    #[serde(rename = "apiVersion", default)]
    pub api_version: Option<String>,
    #[serde(rename = "baseUrl")]
    pub base_url: String,
    #[serde(rename = "fileUrl", default)]
    pub file_url: Option<String>,
    #[serde(default)]
    pub mouse: Option<bool>,
    #[serde(default)]
    pub player: Option<String>,
    #[serde(default)]
    pub theme: Option<String>,
    #[serde(default)]
    pub token: Option<String>,
    #[serde(rename = "tokenStorage", default)]
    pub token_storage: Option<String>,
    #[serde(rename = "updatedAt", default)]
    pub updated_at: Option<String>,
    #[serde(default)]
    pub username: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CliConfig {
    #[serde(rename = "currentProfile", default)]
    pub current_profile: Option<String>,
    #[serde(rename = "defaultPlayer", default)]
    pub default_player: Option<String>,
    #[serde(default)]
    pub profiles: BTreeMap<String, CliProfile>,
    pub version: u8,
}

impl Default for CliConfig {
    fn default() -> Self {
        Self {
            current_profile: None,
            default_player: None,
            profiles: BTreeMap::new(),
            version: 1,
        }
    }
}

#[derive(Debug, Clone)]
pub struct LoadedSession {
    pub base_url: String,
    pub profile: CliProfile,
    pub token: Option<String>,
}

fn token_account(base_url: &str) -> String {
    format!("session:{base_url}")
}

fn keyring_entry(base_url: &str) -> Result<Entry> {
    Entry::new(SERVICE_NAME, &token_account(base_url)).context("failed to create keyring entry")
}

pub fn get_config_path() -> PathBuf {
    let config_base = std::env::var_os("XDG_CONFIG_HOME")
        .map(PathBuf::from)
        .or_else(|| dirs::home_dir().map(|dir| dir.join(".config")))
        .unwrap_or_else(|| PathBuf::from("."));

    config_base.join("mediasink").join("config.json")
}

pub fn normalize_server_url(value: &str) -> Result<String> {
    let candidate = if value.contains("://") {
        value.to_string()
    } else {
        format!("http://{value}")
    };

    let mut url = Url::parse(&candidate).context("invalid server URL")?;
    url.set_fragment(None);
    url.set_query(None);

    let mut path = url.path().trim_end_matches('/').to_string();
    if path.ends_with("/api/v1") {
        path.truncate(path.len() - "/api/v1".len());
    }
    if path.is_empty() {
        path.push('/');
    }
    url.set_path(&path);

    Ok(url.to_string().trim_end_matches('/').to_string())
}

pub fn load_config() -> Result<CliConfig> {
    let config_path = get_config_path();
    let content = match fs::read_to_string(&config_path) {
        Ok(content) => content,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
            return Ok(CliConfig::default());
        }
        Err(error) => return Err(error).context("failed to read CLI config"),
    };

    let mut parsed: CliConfig =
        serde_json::from_str(&content).context("failed to parse CLI config")?;
    if parsed.version == 0 {
        parsed.version = 1;
    }
    Ok(parsed)
}

fn sanitized_config(mut config: CliConfig) -> CliConfig {
    for profile in config.profiles.values_mut() {
        if !matches!(
            profile.token_storage.as_deref(),
            Some("config") | Some("keychain+config")
        ) {
            profile.token = None;
        }
    }
    config
}

pub fn save_config(config: &CliConfig) -> Result<()> {
    let config_path = get_config_path();
    if let Some(parent) = config_path.parent() {
        fs::create_dir_all(parent).context("failed to create config directory")?;
    }

    let content = serde_json::to_string_pretty(&sanitized_config(config.clone()))
        .context("failed to serialize CLI config")?;
    fs::write(&config_path, format!("{content}\n")).context("failed to write CLI config")?;
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        let _ = fs::set_permissions(&config_path, fs::Permissions::from_mode(0o600));
    }
    Ok(())
}

pub fn resolve_base_url(config: &CliConfig, explicit: Option<&str>) -> Result<String> {
    if let Some(value) = explicit {
        return normalize_server_url(value);
    }

    if let Ok(value) = std::env::var("MEDIASINK_URL") {
        return normalize_server_url(&value);
    }

    if let Some(current) = &config.current_profile {
        return Ok(current.clone());
    }

    Ok(DEFAULT_SERVER.to_string())
}

pub fn get_profile(config: &CliConfig, base_url: &str) -> CliProfile {
    config
        .profiles
        .get(base_url)
        .cloned()
        .unwrap_or_else(|| CliProfile {
            base_url: base_url.to_string(),
            ..CliProfile::default()
        })
}

fn upsert_profile(config: &mut CliConfig, profile: CliProfile) {
    config.current_profile = Some(profile.base_url.clone());
    config.profiles.insert(profile.base_url.clone(), profile);
}

pub fn load_saved_session(explicit_base_url: Option<&str>) -> Result<LoadedSession> {
    let config = load_config()?;
    let base_url = resolve_base_url(&config, explicit_base_url)?;
    let profile = get_profile(&config, &base_url);
    let token = load_stored_token(&base_url).or_else(|| profile.token.clone());

    Ok(LoadedSession {
        base_url,
        profile,
        token,
    })
}

pub fn load_stored_token(base_url: &str) -> Option<String> {
    let entry = keyring_entry(base_url).ok()?;
    entry.get_password().ok()
}

fn save_stored_token(base_url: &str, token: &str) -> Result<()> {
    let entry = keyring_entry(base_url)?;
    entry
        .set_password(token)
        .map_err(|error| anyhow::anyhow!("System keychain write failed: {error}"))
}

fn delete_stored_token(base_url: &str) {
    if let Ok(entry) = keyring_entry(base_url) {
        let _ = entry.delete_credential();
    }
}

pub fn save_authenticated_session(
    base_url: &str,
    username: &str,
    token: &str,
    api_version: Option<String>,
    file_url: Option<String>,
) -> Result<Option<String>> {
    let mut config = load_config()?;
    let mut warning = None;
    let mut token_storage = Some("keychain+config".to_string());
    let token_for_config = Some(token.to_string());

    if let Err(error) = save_stored_token(base_url, token) {
        token_storage = Some("config".to_string());
        warning = Some(format!(
            "{error}. Falling back to config-file token storage."
        ));
    }

    let existing = get_profile(&config, base_url);
    let profile = CliProfile {
        api_version,
        base_url: base_url.to_string(),
        file_url,
        player: existing.player,
        theme: existing.theme,
        mouse: existing.mouse,
        token: token_for_config,
        token_storage,
        updated_at: Some(Utc::now().to_rfc3339()),
        username: Some(username.to_string()),
    };

    upsert_profile(&mut config, profile);
    save_config(&config)?;

    Ok(warning)
}

pub fn clear_saved_session(base_url: &str) -> Result<()> {
    let mut config = load_config()?;
    let mut profile = get_profile(&config, base_url);
    profile.token = None;
    profile.token_storage = None;
    profile.updated_at = Some(Utc::now().to_rfc3339());
    upsert_profile(&mut config, profile);
    save_config(&config)?;
    delete_stored_token(base_url);
    Ok(())
}

pub fn save_profile_theme(base_url: &str, theme: &str) -> Result<()> {
    let mut config = load_config()?;
    let mut profile = get_profile(&config, base_url);
    profile.theme = Some(theme.to_string());
    profile.updated_at = Some(Utc::now().to_rfc3339());
    upsert_profile(&mut config, profile);
    save_config(&config)
}

pub fn save_profile_player(base_url: &str, player: &str) -> Result<()> {
    let mut config = load_config()?;
    let mut profile = get_profile(&config, base_url);
    profile.player = Some(player.to_string());
    profile.updated_at = Some(Utc::now().to_rfc3339());
    upsert_profile(&mut config, profile);
    save_config(&config)
}

pub fn save_profile_mouse(base_url: &str, mouse: bool) -> Result<()> {
    let mut config = load_config()?;
    let mut profile = get_profile(&config, base_url);
    profile.mouse = Some(mouse);
    profile.updated_at = Some(Utc::now().to_rfc3339());
    upsert_profile(&mut config, profile);
    save_config(&config)
}

#[cfg(test)]
mod tests {
    use super::{load_saved_session, save_config, CliConfig, CliProfile};
    use std::{
        collections::BTreeMap,
        fs,
        path::PathBuf,
        sync::{Mutex, OnceLock},
        time::{SystemTime, UNIX_EPOCH},
    };

    fn env_lock() -> &'static Mutex<()> {
        static LOCK: OnceLock<Mutex<()>> = OnceLock::new();
        LOCK.get_or_init(|| Mutex::new(()))
    }

    fn temp_config_home() -> PathBuf {
        let unique = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .expect("clock before epoch")
            .as_nanos();
        std::env::temp_dir().join(format!("mediasink-cli-config-test-{unique}"))
    }

    #[test]
    fn load_saved_session_uses_config_token_fallback() {
        let _guard = env_lock().lock().expect("env lock poisoned");
        let config_home = temp_config_home();
        let config_dir = config_home.join("mediasink");
        fs::create_dir_all(&config_dir).expect("create config dir");

        let base_url = "http://persisted.example.invalid:3000";
        let mut profiles = BTreeMap::new();
        profiles.insert(
            base_url.to_string(),
            CliProfile {
                api_version: Some("0.1.0".to_string()),
                base_url: base_url.to_string(),
                file_url: Some(format!("{base_url}/videos")),
                mouse: Some(true),
                player: Some("auto".to_string()),
                theme: Some("norton".to_string()),
                token: Some("persisted-token".to_string()),
                token_storage: Some("keychain+config".to_string()),
                updated_at: None,
                username: Some("saved@example.com".to_string()),
            },
        );
        let config = CliConfig {
            current_profile: Some(base_url.to_string()),
            default_player: None,
            profiles,
            version: 1,
        };

        let previous = std::env::var_os("XDG_CONFIG_HOME");
        unsafe {
            std::env::set_var("XDG_CONFIG_HOME", &config_home);
        }
        save_config(&config).expect("save config");
        let loaded = load_saved_session(None).expect("load saved session");
        if let Some(previous) = previous {
            unsafe {
                std::env::set_var("XDG_CONFIG_HOME", previous);
            }
        } else {
            unsafe {
                std::env::remove_var("XDG_CONFIG_HOME");
            }
        }

        assert_eq!(loaded.base_url, base_url);
        assert_eq!(loaded.profile.username.as_deref(), Some("saved@example.com"));
        assert_eq!(loaded.token.as_deref(), Some("persisted-token"));

        let _ = fs::remove_dir_all(config_home);
    }
}

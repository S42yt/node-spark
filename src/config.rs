use directories::ProjectDirs;
use std::path::PathBuf;
use std::fs;
use anyhow::{Result, Context};
use serde::{Serialize, Deserialize};

#[derive(Debug, Serialize, Deserialize)]
pub struct Config {
    pub active_version: Option<String>,
}

pub struct NodeSparkDirs {
    pub config_dir: PathBuf,
    pub versions_dir: PathBuf,
    pub bin_dir: PathBuf,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            active_version: None,
        }
    }
}

pub fn get_dirs() -> Result<NodeSparkDirs> {
    let project_dirs = ProjectDirs::from("com", "node-spark", "node-spark")
        .context("Failed to determine project directories")?;
    
    let config_dir = project_dirs.config_dir().to_path_buf();
    let data_dir = project_dirs.data_dir().to_path_buf();
    
    let versions_dir = data_dir.join("versions");
    let bin_dir = data_dir.join("bin");
    
    
    fs::create_dir_all(&config_dir)?;
    fs::create_dir_all(&versions_dir)?;
    fs::create_dir_all(&bin_dir)?;
    
    Ok(NodeSparkDirs {
        config_dir,
        versions_dir,
        bin_dir,
    })
}

pub fn load_config() -> Result<Config> {
    let dirs = get_dirs()?;
    let config_path = dirs.config_dir.join("config.json");
    
    if config_path.exists() {
        let content = fs::read_to_string(&config_path)?;
        let config = serde_json::from_str(&content)?;
        Ok(config)
    } else {
        let config = Config::default();
        save_config(&config)?;
        Ok(config)
    }
}

pub fn save_config(config: &Config) -> Result<()> {
    let dirs = get_dirs()?;
    let config_path = dirs.config_dir.join("config.json");
    
    let content = serde_json::to_string_pretty(config)?;
    fs::write(&config_path, content)?;
    
    Ok(())
}

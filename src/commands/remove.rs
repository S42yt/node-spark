use anyhow::{Result, anyhow};
use colored::Colorize;
use std::fs;
use crate::config;
use crate::utils;

pub fn execute(version: &str) -> Result<()> {
    let dirs = config::get_dirs()?;
    let config = config::load_config()?;
    
    let actual_version = utils::parse_version(version)?;
    
    let version_dir = dirs.versions_dir.join(&actual_version);
    if !version_dir.exists() {
        return Err(anyhow!("Node.js {} is not installed", actual_version));
    }
    
    if let Some(ref active) = config.active_version {
        if *active == actual_version {
            return Err(anyhow!(
                "Cannot remove the active Node.js version. Switch to another version first."
            ));
        }
    }
    
    fs::remove_dir_all(&version_dir)?;
    
    println!("Successfully removed Node.js {}", actual_version.green());
    
    Ok(())
}

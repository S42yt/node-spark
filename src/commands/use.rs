use anyhow::{Result, anyhow};
use colored::Colorize;
use crate::config;
use crate::commands::install::create_node_symlinks;
use crate::utils;

pub fn execute(version: &str) -> Result<()> {
    let dirs = config::get_dirs()?;
    
    let actual_version = utils::parse_version(version)?;
    
    let version_dir = dirs.versions_dir.join(&actual_version);
    if !version_dir.exists() {
        return Err(anyhow!("Node.js {} is not installed. Use 'node-spark install {}' first.",
                            actual_version, actual_version));
    }
    
    let mut config = config::load_config()?;
    config.active_version = Some(actual_version.clone());
    config::save_config(&config)?;
    
    create_node_symlinks(&actual_version)?;
    
    println!("Now using Node.js {}", actual_version.green());
    
    Ok(())
}

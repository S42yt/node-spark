use anyhow::{Result, anyhow};
use colored::Colorize;
use std::fs;
use crate::config;
use crate::utils::{self, download, extract};

pub fn execute(version: &str) -> Result<()> {
    let dirs = config::get_dirs()?;
    
    let actual_version = if version == "latest" || version == "lts" {
        println!("Fetching {} Node.js version...", version);
        let available_versions = download::get_available_versions()?;
        
        if available_versions.is_empty() {
            return Err(anyhow!("No available Node.js versions found"));
        }
        
        if version == "latest" {
            available_versions.first().unwrap().clone()
        } else {
            available_versions.first().unwrap().clone()
        }
    } else {
        utils::parse_version(version)?
    };
    
    println!("Installing Node.js {}", actual_version.green());
    
    let version_dir = dirs.versions_dir.join(&actual_version);
    if version_dir.exists() {
        println!("Node.js {} is already installed", actual_version);
        return Ok(());
    }
    
    let temp_dir = dirs.config_dir.join("temp");
    fs::create_dir_all(&temp_dir)?;
    
    let download_url = utils::get_download_url(&actual_version);
    let extension = if cfg!(target_os = "windows") { "zip" } else { "tar.gz" };
    let download_path = temp_dir.join(format!("node-v{}.{}", actual_version, extension));
    
    download::download_file(&download_url, &download_path)?;
    
    println!("Extracting Node.js {}...", actual_version);
    fs::create_dir_all(&version_dir)?;
    extract::extract_archive(&download_path, &version_dir)?;
    
    fs::remove_file(download_path)?;
    
    println!("Successfully installed Node.js {}", actual_version.green());
    
    let mut config = config::load_config()?;
    if config.active_version.is_none() {
        println!("Setting Node.js {} as the default version", actual_version);
        config.active_version = Some(actual_version.clone());
        config::save_config(&config)?;
        
        create_node_symlinks(&actual_version)?;
    }
    
    Ok(())
}

pub fn create_node_symlinks(version: &str) -> Result<()> {
    let dirs = config::get_dirs()?;
    let version_bin_dir = dirs.versions_dir.join(version).join("bin");
    
    let node_path = version_bin_dir.join("node");
    let npm_path = version_bin_dir.join("npm");
    let npx_path = version_bin_dir.join("npx");
    
    let node_link = dirs.bin_dir.join("node");
    let npm_link = dirs.bin_dir.join("npm");
    let npx_link = dirs.bin_dir.join("npx");
    
    #[cfg(unix)]
    {
        use std::os::unix::fs as unix_fs;
        if node_link.exists() {
            fs::remove_file(&node_link)?;
        }
        if npm_link.exists() {
            fs::remove_file(&npm_link)?;
        }
        if npx_link.exists() {
            fs::remove_file(&npx_link)?;
        }
        
        unix_fs::symlink(&node_path, &node_link)?;
        unix_fs::symlink(&npm_path, &npm_link)?;
        unix_fs::symlink(&npx_path, &npx_link)?;
    }
    
    #[cfg(windows)]
    {
        use std::os::windows::fs as windows_fs;
        if node_link.exists() {
            fs::remove_file(&node_link)?;
        }
        if npm_link.exists() {
            fs::remove_file(&npm_link)?;
        }
        if npx_link.exists() {
            fs::remove_file(&npx_link)?;
        }
        
        windows_fs::symlink_file(&node_path, &node_link)?;
        windows_fs::symlink_file(&npm_path, &npm_link)?;
        windows_fs::symlink_file(&npx_path, &npx_link)?;
    }
    
    Ok(())
}

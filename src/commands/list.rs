use anyhow::Result;
use colored::Colorize;
use std::fs;
use crate::config;
use crate::utils::download;

pub fn execute(remote: bool) -> Result<()> {
    if remote {
        list_remote_versions()?;
    } else {
        list_local_versions()?;
    }
    
    Ok(())
}

fn list_local_versions() -> Result<()> {
    let dirs = config::get_dirs()?;
    let config = config::load_config()?;
    
    println!("Installed Node.js versions:");
    
    let entries = match fs::read_dir(&dirs.versions_dir) {
        Ok(entries) => entries,
        Err(_) => {
            println!("  No versions installed");
            return Ok(());
        }
    };
    
    let mut versions = Vec::new();
    for entry in entries {
        let entry = entry?;
        if entry.file_type()?.is_dir() {
            if let Some(name) = entry.file_name().to_str() {
                versions.push(name.to_string());
            }
        }
    }
    
    if versions.is_empty() {
        println!("  No versions installed");
        return Ok(());
    }
    
    versions.sort_by(|a, b| {
        match (semver::Version::parse(a), semver::Version::parse(b)) {
            (Ok(a_ver), Ok(b_ver)) => a_ver.cmp(&b_ver).reverse(),
            (Ok(_), Err(_)) => std::cmp::Ordering::Less,
            (Err(_), Ok(_)) => std::cmp::Ordering::Greater,
            (Err(_), Err(_)) => a.cmp(b).reverse()
        }
    });
    
    for version in versions {
        if let Some(ref active) = config.active_version {
            if version == *active {
                println!("* {} (current)", version.green());
            } else {
                println!("  {}", version);
            }
        } else {
            println!("  {}", version);
        }
    }
    
    Ok(())
}

fn list_remote_versions() -> Result<()> {
    println!("Fetching available Node.js versions...");
    
    let available_versions = download::get_available_versions()?;
    
    if available_versions.is_empty() {
        println!("No available versions found");
        return Ok(());
    }
    
    println!("\nAvailable Node.js versions:");
    
    let config = config::load_config()?;
    let dirs = config::get_dirs()?;
    
    for (i, version) in available_versions.iter().enumerate().take(30) {
        let installed = dirs.versions_dir.join(version).exists();
        let is_current = config.active_version.as_ref().map_or(false, |v| v == version);
        
        if installed {
            if is_current {
                println!("* {} (installed, current)", version.green());
            } else {
                println!("* {} (installed)", version.yellow());
            }
        } else {
            println!("  {}", version);
        }
        
        if i == 29 {
            println!("  ... and more");
            break;
        }
    }
    
    Ok(())
}

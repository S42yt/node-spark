pub mod download;
pub mod extract;

use anyhow::{Result, anyhow};
use semver::Version;

pub fn parse_version(version: &str) -> Result<String> {
    if let Ok(_) = Version::parse(version) {
        return Ok(version.to_string());
    }
    
    if version.starts_with('v') {
        if let Ok(_) = Version::parse(&version[1..]) {
            return Ok(version[1..].to_string());
        }
    }

    Err(anyhow!("Invalid version format: {}", version))
}

pub fn get_download_url(version: &str) -> String {
    let arch = if cfg!(target_arch = "x86_64") {
        "x64"
    } else if cfg!(target_arch = "x86") {
        "x86"
    } else if cfg!(target_arch = "aarch64") {
        "arm64"
    } else {
        "x64" 
    };

    let os = if cfg!(target_os = "windows") {
        "win"
    } else if cfg!(target_os = "macos") {
        "darwin"
    } else {
        "linux"
    };

    let ext = if cfg!(target_os = "windows") {
        "zip"
    } else {
        "tar.gz"
    };

    format!(
        "https://nodejs.org/dist/v{}/node-v{}-{}-{}.{}",
        version, version, os, arch, ext
    )
}

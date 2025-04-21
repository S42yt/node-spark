use anyhow::{Result, Context};
use indicatif::{ProgressBar, ProgressStyle};
use reqwest::blocking::Client;
use std::fs::File;
use std::io::Write;
use std::path::Path;

pub fn download_file(url: &str, dest_path: &Path) -> Result<()> {
    println!("Downloading from {}", url);
    
    let client = Client::new();
    let resp = client.get(url)
        .send()
        .context("Failed to send request")?;
    
    let total_size = resp.content_length().unwrap_or(0);
    
    let pb = ProgressBar::new(total_size);
    pb.set_style(ProgressStyle::default_bar()
        .template("{spinner:.green} [{elapsed_precise}] [{bar:40.cyan/blue}] {bytes}/{total_bytes} ({eta})")
        .unwrap()
        .progress_chars("#>-"));
    
    let mut file = File::create(dest_path)?;
    let content = resp.bytes()?;
    file.write_all(&content)?;
    pb.finish_with_message("Download complete");
    
    Ok(())
}

pub fn get_available_versions() -> Result<Vec<String>> {
    let client = Client::new();
    let resp = client.get("https://nodejs.org/dist/index.json")
        .send()
        .context("Failed to fetch available Node.js versions")?;
    
    let versions: Vec<serde_json::Value> = resp.json()?;
    
    let mut result = Vec::new();
    for version in versions {
        if let Some(version_str) = version["version"].as_str() {
            let cleaned_version = version_str.trim_start_matches('v').to_string();
            result.push(cleaned_version);
        }
    }
    
    Ok(result)
}

//pub fn is_lts_version(version_data: &serde_json::Value) -> bool {
//    version_data.get("lts").map_or(false, |v| !v.is_null())
//}

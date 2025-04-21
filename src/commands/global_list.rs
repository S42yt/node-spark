use anyhow::Result;
use colored::Colorize;
use std::process::Command;
use crate::options::verbose;

pub fn execute() -> Result<()> {
    verbose::log("Executing global-list command");
    println!("Listing globally installed npm packages...");
    
    let npm_cmd = if cfg!(target_os = "windows") {
        "npm.cmd"
    } else {
        "npm"
    };

    let output = Command::new(npm_cmd)
        .args(["list", "--global", "--depth=0"])
        .output()?;
    
    if !output.status.success() {
        verbose::log(&format!("npm list command failed with status: {}", output.status));
    }

    let output_str = String::from_utf8_lossy(&output.stdout);
    
    for line in output_str.lines() {
        if line.contains("npm ERR!") || line.trim().is_empty() {
            continue;
        }
        
        if line.contains("@") && !line.starts_with("+") && !line.starts_with("`") {
            let parts: Vec<&str> = line.splitn(2, '@').collect();
            if parts.len() == 2 {
                let name = parts[0].trim();
                let version = parts[1].trim();
                println!("{} {}", name.green(), format!("@{}", version).yellow());
            } else {
                println!("{}", line);
            }
        } else {
            println!("{}", line);
        }
    }
    
    Ok(())
}
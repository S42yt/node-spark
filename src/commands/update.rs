use anyhow::{Result, anyhow};
use colored::Colorize;
use std::process::Command;
use crate::options::verbose;

pub fn execute() -> Result<()> {
    verbose::log("Executing update command");
    println!("Checking for updates to node-spark...");
    
    let cargo_cmd = if cfg!(target_os = "windows") {
        "cargo.exe"
    } else {
        "cargo"
    };

    match Command::new(cargo_cmd).arg("--version").output() {
        Ok(_) => {
            verbose::log("Cargo is available, proceeding with update");
        },
        Err(_) => {
            return Err(anyhow!("Cargo not found. Make sure it's installed and in your PATH"));
        }
    }

    println!("Updating node-spark to the latest version...");
    
    let output = Command::new(cargo_cmd)
        .args(["install", "--force", "node-spark"])
        .output()?;
    
    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        verbose::log(&format!("Update command failed: {}", stderr));
        return Err(anyhow!("Failed to update node-spark: {}", stderr));
    }

    println!("{}", "node-spark updated successfully!".green());
    
    if let Err(e) = crate::create_alias() {
        verbose::log(&format!("Failed to create alias: {}", e));
        println!("Note: Failed to create 'nsk' alias, but node-spark was updated successfully.");
    }
    
    Ok(())
}
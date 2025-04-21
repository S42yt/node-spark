mod commands;
mod config;
mod options;
mod utils;

use clap::{Parser, CommandFactory};
use colored::Colorize;

fn main() -> anyhow::Result<()> {
    let cli = options::Cli::parse();

    options::verbose::set_verbose(cli.verbose);

    if cli.verbose && cli.version {
        println!("Verbose mode: {}", "enabled".green());
        options::version::show();
        return Ok(());
    }

    if cli.version {
        options::version::show();
        return Ok(());
    }

    check_and_create_alias()?;

    match cli.command {
        Some(options::Commands::Install { version }) => {
            commands::install::execute(&version)?;
        }
        Some(options::Commands::Use { version }) => {
            commands::r#use::execute(&version)?;
        }
        Some(options::Commands::List { remote }) => {
            commands::list::execute(remote)?;
        }
        Some(options::Commands::Remove { version }) => {
            commands::remove::execute(&version)?;
        }
        Some(options::Commands::GlobalList) => {
            commands::global_list::execute()?;
        }
        Some(options::Commands::Update) => {
            commands::update::execute()?;
        }
        None => {
            let mut cmd = options::Cli::command();
            cmd.print_help()?;
            println!();
        }
    }

    Ok(())
}

pub fn create_alias() -> anyhow::Result<()> {
    options::verbose::log("Creating 'nsk' alias for node-spark");
    
    #[cfg(target_os = "windows")]
    {
        use std::fs::File;
        use std::io::Write;
        use std::env;
        
        let executable = env::current_exe()?;
        
        let nsk_path = executable.parent().unwrap().join("nsk.bat");
        
        let mut file = File::create(&nsk_path)?;
        writeln!(file, "@echo off")?;
        writeln!(file, "\"{}\" %*", executable.display())?;
        
        println!("Created alias: {} -> {}", "nsk".green(), "node-spark".bright_green());
    }
    
    #[cfg(not(target_os = "windows"))]
    {
        use std::process::Command;
        use std::env;
        
        let executable = env::current_exe()?;
        
        let nsk_path = executable.parent().unwrap().join("nsk");
        
        if nsk_path.exists() {
            std::fs::remove_file(&nsk_path)?;
        }
        
        Command::new("ln")
            .args(["-s", &executable.to_string_lossy(), &nsk_path.to_string_lossy()])
            .output()?;
            
        println!("Created alias: {} -> {}", "nsk".green(), "node-spark".bright_green());
    }
    
    Ok(())
}

fn check_and_create_alias() -> anyhow::Result<()> {
    let executable = std::env::current_exe()?;
    let nsk_path = executable.parent().unwrap().join(if cfg!(target_os = "windows") {
        "nsk.bat"
    } else {
        "nsk"
    });

    if !nsk_path.exists() {
        create_alias()?;
    }

    Ok(())
}

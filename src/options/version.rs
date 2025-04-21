use colored::Colorize;

pub fn show() {
    let version = env!("CARGO_PKG_VERSION");
    let name = env!("CARGO_PKG_NAME");
    
    println!("{} v{}", name.bright_green(), version.bright_white());
    println!("Author: {}", env!("CARGO_PKG_AUTHORS").bright_blue());
}
pub mod verbose;
pub mod version;

use clap::{Parser, Subcommand, ArgAction};

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
#[command(disable_version_flag = true)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Option<Commands>,

    #[arg(short = 'V', long, action = ArgAction::SetTrue)]
    pub version: bool,

    #[arg(short, long, action = ArgAction::SetTrue)]
    pub verbose: bool,
}

#[derive(Subcommand, Debug)]
pub enum Commands {
    Install {
        version: String,
    },

    #[command(name = "use")]
    Use {
        version: String,
    },

    Remove {
        version: String,
    },

    List {
        #[arg(short, long)]
        remote: bool,
    },

    #[command(name = "global-list")]
    GlobalList,

    Update,
}
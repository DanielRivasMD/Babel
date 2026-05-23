mod completion;
mod compose;
mod construct;
mod display;
mod embed;
mod identity;
mod interpret;

use crate::cli::{Cli, Commands};
use anyhow::Result;

pub fn run(cli: Cli) -> Result<()> {
    let global = GlobalOpts {
        verbose: cli.verbose,
        program: cli.program,
        root: cli.root.unwrap_or_else(default_root_dir),
    };
    match cli.command {
        Commands::Completion { shell } => {
            completion::run(shell);
            Ok(())
        }
        Commands::Identity => {
            identity::run();
            Ok(())
        }
        Commands::Construct => construct::run(global),
        Commands::Compose { template } => compose::run(global, template),
        Commands::Display { file, render, sort } => display::run(global, file, render, sort),
        Commands::Embed { target } => embed::run(global, target),
        Commands::Interpret { target } => interpret::run(global, target),
    }
}

pub struct GlobalOpts {
    pub verbose: bool,
    pub program: Option<String>,
    pub root: std::path::PathBuf,
}

pub fn default_root_dir() -> std::path::PathBuf {
    let home = dirs::home_dir().expect("cannot determine home directory");
    home.join(".saiyajin/edn")
}

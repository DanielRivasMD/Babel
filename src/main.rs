////////////////////////////////////////////////////////////////////////////////////////////////////

use anyhow::Result as anyResult;
use clap::Parser;

////////////////////////////////////////////////////////////////////////////////////////////////////


////////////////////////////////////////////////////////////////////////////////////////////////////

mod cli;
mod cmd;
mod edn;
mod lookup;
mod util;

////////////////////////////////////////////////////////////////////////////////////////////////////

fn main() -> anyResult<()> {
    let cli = cli::Cli::parse();
    let global = cli::GlobalOpts {
        program: cli.program,
        root: cli.root.unwrap_or_else(util::default_root_dir),
    };
    match cli.command {
        cli::Command::Construct => cmd::construct::run()?,
        cli::Command::Compose { template } => cmd::compose::run(global, template)?,
        cli::Command::Display { file, render, sort } => cmd::display::run(global, file, render, sort)?,
        cli::Command::Embed { target } => cmd::embed::run(global, target)?,
        cli::Command::Interpret { target } => cmd::interpret::run(global, target)?,
        cli::Command::Identity => cmd::identity::run()?,
        cli::Command::Completion { shell } => cmd::completion::run(shell)?,
    }
    Ok(())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

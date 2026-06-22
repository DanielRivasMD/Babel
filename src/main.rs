////////////////////////////////////////////////////////////////////////////////////////////////////

use anyhow::Result as anyResult;
use clap::Parser;

////////////////////////////////////////////////////////////////////////////////////////////////////

use cli::Command;
use cli::GlobalOpts;
use cmd::{completion, compose, construct, display, embed, identity, interpret};
use util::default_root_dir;

////////////////////////////////////////////////////////////////////////////////////////////////////

mod cli;
mod cmd;
mod edn;
mod lookup;
mod util;

////////////////////////////////////////////////////////////////////////////////////////////////////

fn main() -> anyResult<()> {
    let cli = cli::Cli::parse();
    let global = GlobalOpts {
        program: cli.program,
        root: cli.root.unwrap_or_else(default_root_dir),
    };
    match cli.command {
        Command::Construct => construct::run(global)?,
        Command::Compose { template } => compose::run(global, template)?,
        Command::Display { file, render, sort } => display::run(global, file, render, sort)?,
        Command::Embed { target } => embed::run(global, target)?,
        Command::Interpret { target } => interpret::run(global, target)?,
        Command::Identity => identity::run()?,
        Command::Completion { shell } => completion::run(shell)?,
    }
    Ok(())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

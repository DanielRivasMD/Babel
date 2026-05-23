mod cli;
mod cmds;
mod forger;
mod lookup;
mod parser;
mod util;

use clap::Parser;

fn main() -> anyhow::Result<()> {
    let cli = cli::Cli::parse();
    cmds::run(cli)
}

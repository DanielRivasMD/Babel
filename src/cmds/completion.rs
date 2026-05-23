use clap::CommandFactory;
use clap_complete::{generate, Shell};
use std::io;

pub fn run(shell: Shell) {
    let mut cmd = crate::cli::Cli::command();
    generate(shell, &mut cmd, "babel", &mut io::stdout());
}

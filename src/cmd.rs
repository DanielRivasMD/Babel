////////////////////////////////////////////////////////////////////////////////////////////////////

use anyhow::Result as anyResult;

////////////////////////////////////////////////////////////////////////////////////////////////////

pub mod compose;
pub mod construct;
pub mod display;
pub mod embed;
pub mod interpret;

////////////////////////////////////////////////////////////////////////////////////////////////////

pub mod completion {

    use clap::{Command, CommandFactory};
    use clap_complete::{generate, shells::*};
    use std::io;

    use crate::cli;

    pub fn run(shell: cli::Shell) -> super::anyResult<()> {
        let visible: Vec<_> = cli::Cli::command()
            .get_subcommands()
            .filter(|s| !s.is_hide_set())
            .cloned()
            .collect();

        let mut cmd = Command::new(env!("CARGO_BIN_NAME")).subcommands(visible);

        // Manually add global flags from the full CLI definition
        let full = cli::Cli::command();
        for arg in full.get_arguments() {
            let name = arg.get_id().as_str();
            if name == "program" || name == "root" || name == "verbose" {
                cmd = cmd.arg(arg.clone());
            }
        }

        let name = cmd.get_name().to_string();

        match shell {
            cli::Shell::Bash => generate(Bash, &mut cmd, name, &mut io::stdout()),
            cli::Shell::Zsh => generate(Zsh, &mut cmd, name, &mut io::stdout()),
            cli::Shell::Fish => generate(Fish, &mut cmd, name, &mut io::stdout()),
            cli::Shell::Powershell => generate(PowerShell, &mut cmd, name, &mut io::stdout()),
        }
        Ok(())
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub mod identity {
    const IDENTITY: &str = r#"Babel"#;

    pub fn run() -> super::anyResult<()> {
        println!("{}", IDENTITY);
        Ok(())
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

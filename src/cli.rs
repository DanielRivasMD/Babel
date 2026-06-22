////////////////////////////////////////////////////////////////////////////////////////////////////

use clap::{Parser, Subcommand, ValueEnum, ValueHint};
use std::path::PathBuf;

////////////////////////////////////////////////////////////////////////////////////////////////////

pub struct GlobalOpts {
    pub program: Option<String>,
    pub root: std::path::PathBuf,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

const HELP: &str = r"";

////////////////////////////////////////////////////////////////////////////////////////////////////

/// Interpret hotkeys into markdown keyboard visuals
#[derive(Parser)]
#[command(
    name = env!("CARGO_PKG_NAME"),
    version = env!("CARGO_PKG_VERSION"),
    author = env!("CARGO_PKG_AUTHORS"),
    about = env!("CARGO_PKG_DESCRIPTION"),
    before_help = concat!(env!("CARGO_PKG_AUTHORS"), "\n", env!("CARGO_PKG_NAME"), " v", env!("CARGO_PKG_VERSION")),
    long_about = HELP,
)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Command,

    /// Regex or substring to filter Program names (e.g. helix)
    #[arg(long, global = true, value_hint = ValueHint::Other)]
    pub program: Option<String>,

    /// Config root (recurses .edn files)
    #[arg(long, global = true, value_hint = ValueHint::DirPath)]
    pub root: Option<PathBuf>,

    /// Enable verbose diagnostics
    #[arg(short, long, global = true)]
    pub verbose: bool,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

#[derive(Subcommand)]
pub enum Command {
    /// Display current bindings
    Display {
        /// Path to your EDN file
        #[arg(short, long)]
        file: Option<PathBuf>,
        /// Which rows to render: EMPTY (only empty program+action), FULL (all), DEFAULT (non-empty program+action)
        #[arg(short, long, default_value = "DEFAULT")]
        render: String,
        /// Sort output by one of: program, action, trigger, binding
        #[arg(short, long, default_value = "trigger")]
        sort: String,
    },

    /// Generate program‑specific configs from EDN annotations
    Interpret {
        /// Write output to this file instead of stdout
        #[arg(short, long)]
        target: Option<PathBuf>,
    },

    /// Install application
    Construct,

    /// Create templates
    Compose {
        /// Output file for the generated template (default: stdout)
        #[arg(short, long)]
        template: Option<PathBuf>,
    },

    /// Insert program‑specific configs from EDN annotations
    Embed {
        /// Config file to supplement
        #[arg(short, long)]
        target: PathBuf,
    },

    /// Print identity
    #[command(hide = true)]
    #[command(aliases = &["id"])]
    Identity,

    /// Generate shell completions
    #[command(hide = true)]
    Completion {
        /// Shell for which to generate completions
        #[arg(value_enum)]
        shell: Shell,
    },
}

////////////////////////////////////////////////////////////////////////////////////////////////////

#[derive(Clone, Copy, ValueEnum)]
pub enum Shell {
    Bash,
    Zsh,
    Fish,
    Powershell,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

use clap::{Parser, Subcommand, ValueHint};
use clap_complete::Shell;
use std::path::PathBuf;

/// Interpret hotkeys into markdown keyboard visuals
#[derive(Parser)]
#[command(
    name = "babel",
    version,
    author = "Daniel Rivas <danielrivasmd@gmail.com>"
)]
#[command(about, long_about = None)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Commands,

    /// Enable verbose diagnostics
    #[arg(short, long, global = true)]
    pub verbose: bool,

    /// Regex or substring to filter Program names (e.g. helix)
    #[arg(long, global = true, value_hint = ValueHint::Other)]
    pub program: Option<String>,

    /// Config root (recurses .edn files)
    #[arg(long, global = true, value_hint = ValueHint::DirPath)]
    pub root: Option<PathBuf>,
}

#[derive(Subcommand)]
pub enum Commands {
    /// Display current bindings
    Display {
        /// Path to your EDN file
        #[arg(short, long)]
        file: Option<PathBuf>,
        /// Which rows to render: EMPTY (only empty program+action), FULL (all), DEFAULT (non-empty program+action)
        #[arg(short = 'm', long, default_value = "DEFAULT")]
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
        #[arg(long)]
        target: Option<String>,
    },
    /// Print identity
    #[command(aliases = &["id"])]
    Identity,
    /// Generate shell completions
    Completion {
        /// The shell to generate completions for
        #[arg(value_enum)]
        shell: Shell,
    },
}

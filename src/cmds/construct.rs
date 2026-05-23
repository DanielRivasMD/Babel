use crate::cmds::GlobalOpts;
use crate::lookup;
use anyhow::{Context, Result};
use std::fs;

pub fn run(global: GlobalOpts) -> Result<()> {
    let config_dirs = lookup::config_dirs(&global)?;
    for (label, path) in [
        ("babel root", &config_dirs.babel),
        ("config", &config_dirs.config),
    ] {
        fs::create_dir_all(path)
            .with_context(|| format!("creating {label} directory at {}", path.display()))?;
    }
    Ok(())
}

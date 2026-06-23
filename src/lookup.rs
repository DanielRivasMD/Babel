////////////////////////////////////////////////////////////////////////////////////////////////////

use anyhow::{Context, Result as anyResult};
use std::collections::HashMap;
use std::path::{Path, PathBuf};

////////////////////////////////////////////////////////////////////////////////////////////////////

pub type LookupHash = HashMap<String, HashMap<String, String>>;

////////////////////////////////////////////////////////////////////////////////////////////////////

pub struct Lookups {
    pub display_binding: LookupHash,
    pub display_trigger: LookupHash,
    pub interpret: LookupHash,
    pub embed: LookupHash,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

impl Lookups {
    pub fn load() -> anyResult<Self> {
        let config = config_dirs()?;
        Ok(Lookups {
            display_binding: load_toml(&config.config.join("display_binding.toml"))?,
            display_trigger: load_toml(&config.config.join("display_trigger.toml"))?,
            interpret: load_toml(&config.config.join("interpret.toml"))?,
            embed: load_toml(&config.config.join("embed.toml"))?,
        })
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub struct ConfigDirs {
    pub babel: PathBuf,
    pub config: PathBuf,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn config_dirs() -> anyResult<ConfigDirs> {
    let home = dirs::home_dir().context("no home dir")?;
    let babel = home.join(".babel");
    let config = babel.join("config");
    Ok(ConfigDirs {
        babel,
        config,
    })
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn load_toml(path: &Path) -> anyResult<LookupHash> {
    let contents = std::fs::read_to_string(path)?;
    let cfg: LookupHash = toml::from_str(&contents)?;
    Ok(cfg)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

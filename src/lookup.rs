////////////////////////////////////////////////////////////////////////////////////////////////////

use anyhow::{Context, Result};
use std::collections::HashMap;
use std::path::{Path, PathBuf};

////////////////////////////////////////////////////////////////////////////////////////////////////

use crate::cli::GlobalOpts;

////////////////////////////////////////////////////////////////////////////////////////////////////

pub struct Lookups {
    pub display_binding: HashMap<String, HashMap<String, String>>,
    pub display_trigger: HashMap<String, HashMap<String, String>>,
    pub interpret: HashMap<String, HashMap<String, String>>,
    pub embed: HashMap<String, HashMap<String, String>>,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

impl Lookups {
    pub fn load(global: &GlobalOpts) -> Result<Self> {
        let config = config_dirs(global)?;
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
    pub home: PathBuf,
    pub babel: PathBuf,
    pub config: PathBuf,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn config_dirs(_global: &GlobalOpts) -> Result<ConfigDirs> {
    let home = dirs::home_dir().context("no home dir")?;
    let babel = home.join(".babel");
    let config = babel.join("config");
    Ok(ConfigDirs {
        home,
        babel,
        config,
    })
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn load_toml(path: &Path) -> Result<HashMap<String, HashMap<String, String>>> {
    let contents = std::fs::read_to_string(path)?;
    let cfg: HashMap<String, HashMap<String, String>> = toml::from_str(&contents)?;
    Ok(cfg)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

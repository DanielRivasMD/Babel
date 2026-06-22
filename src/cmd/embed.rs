////////////////////////////////////////////////////////////////////////////////////////////////////

use anyhow::{Context, Result as anyResult, bail};

////////////////////////////////////////////////////////////////////////////////////////////////////

use crate::cli::GlobalOpts;
use crate::forge;
use crate::lookup::Lookups;
use crate::edn;

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn run(global: GlobalOpts, target: Option<String>) -> anyResult<()> {
    let lookups = Lookups::load(&global)?;
    let target = target
        .or_else(|| global.program.clone())
        .context("no target specified")?;
    if global.program.is_none() {
        bail!("`--program` is required");
    }
    let paths = edn::resolve_edn_files(None, &global.root);
    let all_entries = edn::parse_edn_files(&paths)?;
    forge::embed_config(all_entries, &target, &lookups, &global)?;
    Ok(())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

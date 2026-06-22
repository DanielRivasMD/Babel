////////////////////////////////////////////////////////////////////////////////////////////////////

use anyhow::{Result as anyResult, bail};
use std::path::PathBuf;

////////////////////////////////////////////////////////////////////////////////////////////////////

use crate::cli::GlobalOpts;
use crate::edn;
use crate::lookup::Lookups;
use crate::util;

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn run(global: GlobalOpts, target: PathBuf) -> anyResult<()> {
    if global.program.is_none() {
        bail!("`--program` is required");
    }
    let lookups = Lookups::load(&global)?;
    let program = global.program.as_deref().unwrap();
    let paths = edn::resolve_edn_files(None, &global.root);
    let all_entries = edn::parse_edn_files(&paths)?;
    util::embed_config(all_entries, program, &target, &lookups)?;
    Ok(())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

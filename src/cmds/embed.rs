use crate::cmds::GlobalOpts;
use crate::forger;
use crate::lookup::Lookups;
use crate::parser;
use anyhow::{bail, Context, Result};

pub fn run(global: GlobalOpts, target: Option<String>) -> Result<()> {
    let lookups = Lookups::load(&global)?;
    let target = target
        .or_else(|| global.program.clone())
        .context("no target specified")?;
    if global.program.is_none() {
        bail!("`--program` is required");
    }
    let paths = parser::resolve_edn_files(None, &global.root);
    let all_entries = parser::parse_edn_files(&paths)?;
    forger::embed_config(all_entries, &target, &lookups, &global)
}

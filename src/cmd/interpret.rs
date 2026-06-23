////////////////////////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////////////////////////

use anyhow::{Result as anyResult, bail};
use std::fs::File;
use std::io::{self, Write};
use std::path::PathBuf;

////////////////////////////////////////////////////////////////////////////////////////////////////

use crate::cli;
use crate::lookup;
use crate::edn;

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn run(global: cli::GlobalOpts, target: Option<PathBuf>) -> anyResult<()> {
    if global.program.is_none() {
        bail!("`--program` is required");
    }
    let lookups = lookup::Lookups::load()?;
    let paths = edn::resolve_edn_files(None, &global.root);
    let all_entries = edn::parse_edn_files(&paths)?;

    let mut writer: Box<dyn Write> = if let Some(path) = target {
        Box::new(File::create(path)?)
    } else {
        Box::new(io::stdout())
    };

    let families = std::collections::HashMap::from([
        (
            "helix",
            vec![
                "helix-common",
                "helix-insert",
                "helix-normal",
                "helix-select",
            ],
        ),
        ("micro", vec!["micro"]),
    ]);
    let program = global.program.as_deref().unwrap();
    if let Some(bases) = families.get(program) {
        for b in bases {
            crate::util::emit_config(&mut *writer, &all_entries, b, &lookups)?;
            writeln!(writer)?;
        }
    } else {
        crate::util::emit_config(&mut *writer, &all_entries, program, &lookups)?;
    }
    Ok(())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

use crate::cmds::GlobalOpts;
use crate::lookup::Lookups;
use crate::parser;
use anyhow::{bail, Result};
use std::fs::File;
use std::io::{self, Write};
use std::path::PathBuf;

pub fn run(global: GlobalOpts, target: Option<PathBuf>) -> Result<()> {
    if global.program.is_none() {
        bail!("`--program` is required");
    }
    let lookups = Lookups::load(&global)?;
    let paths = parser::resolve_edn_files(None, &global.root);
    let all_entries = parser::parse_edn_files(&paths)?;

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

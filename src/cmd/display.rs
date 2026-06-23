////////////////////////////////////////////////////////////////////////////////////////////////////

use anyhow::Result as anyResult;
use std::path::PathBuf;

////////////////////////////////////////////////////////////////////////////////////////////////////

use crate::cli;
use crate::lookup;
use crate::edn;
use crate::util;

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn run(
    global: cli::GlobalOpts,
    file: Option<PathBuf>,
    render: String,
    sort: String,
) -> anyResult<()> {
    let lookups = lookup::Lookups::load()?;
    let paths = edn::resolve_edn_files(file, &global.root);
    let all_entries = edn::parse_edn_files(&paths)?;
    let filtered = edn::filter_by_program(all_entries, global.program.as_deref().unwrap_or(""));

    let final_entries = match render.to_uppercase().as_str() {
        "FULL" => filtered,
        "EMPTY" => filtered
            .into_iter()
            .filter(|e| util::is_empty_entry(e))
            .collect(),
        &_ => filtered
            .into_iter()
            .filter(|e| !util::is_empty_entry(e))
            .collect(),
    };

    let mut rows = Vec::new();
    for entry in &final_entries {
        for action in &entry.actions {
            let trigger = util::format_trigger_display(
                &entry.trigger,
                &lookups.display_trigger,
                &action.program,
            );
            let binding =
                util::format_binding_display(entry, &lookups.display_binding, &action.program);
            rows.push(util::TableRow {
                program: action.program.clone(),
                action: action.action.clone(),
                trigger,
                binding,
                empty: util::is_empty_entry(entry),
            });
        }
    }

    rows.sort_by(|a, b| match sort.to_lowercase().as_str() {
        "program" => a.program.cmp(&b.program),
        "action" => a.action.cmp(&b.action),
        "binding" => a.binding.cmp(&b.binding),
        _ => a.trigger.cmp(&b.trigger),
    });

    util::print_table(&rows);
    Ok(())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

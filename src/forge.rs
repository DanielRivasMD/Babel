////////////////////////////////////////////////////////////////////////////////////////////////////

use anyhow::{Context, Result as anyResult, bail};
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;

////////////////////////////////////////////////////////////////////////////////////////////////////

use crate::cli::GlobalOpts;
use crate::lookup::Lookups;
use crate::edn;
use crate::util;

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn embed_config(
    entries: Vec<edn::BindingEntry>,
    target: &str,
    lookups: &Lookups,
    global: &GlobalOpts,
) -> anyResult<()> {
    match target {
        "kanata" => embed_kanata(entries, lookups, global),
        "serpl" => embed_bindings(entries, target, lookups, global, serpl_format),
        "lazygit" => embed_bindings(entries, target, lookups, global, lazygit_format),
        z if z.starts_with("zellij") => embed_zellij(entries, lookups, global),
        _ => bail!("unsupported --program: {}", target),
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn kanata_transform_map() -> HashMap<String, String> {
    HashMap::from([
        ("up_arrow".to_string(), "up".to_string()),
        ("down_arrow".to_string(), "dn".to_string()),
        ("left_arrow".to_string(), "lf".to_string()),
        ("right_arrow".to_string(), "rg".to_string()),
        ("hyphen".to_string(), "hy".to_string()),
        ("equal_sign".to_string(), "eq".to_string()),
        ("delete_or_backspace".to_string(), "db".to_string()),
        ("open_bracket".to_string(), "ob".to_string()),
        ("close_bracket".to_string(), "cb".to_string()),
        ("semicolon".to_string(), "sc".to_string()),
        ("quote".to_string(), "qu".to_string()),
        ("backslash".to_string(), "bl".to_string()),
        ("comma".to_string(), "cm".to_string()),
        ("period".to_string(), "pe".to_string()),
        ("slash".to_string(), "sl".to_string()),
        ("return_or_enter".to_string(), "ret".to_string()),
        ("spacebar".to_string(), "spc".to_string()),
        ("right_shift".to_string(), "kR".to_string()),
        ("right_option".to_string(), "kE".to_string()),
        ("right_command".to_string(), "kQ".to_string()),
        ("right_control".to_string(), "kW".to_string()),
        ("left_command".to_string(), "kC".to_string()),
        ("left_option".to_string(), "kO".to_string()),
        ("left_control".to_string(), "kT".to_string()),
        ("left_shift".to_string(), "kS".to_string()),
    ])
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn embed_kanata(entries: Vec<edn::BindingEntry>, lookups: &Lookups, global: &GlobalOpts) -> anyResult<()> {
    let allowed: Vec<&str> = vec![
        "helix", "serpl", "lazygit", "zellij", "term", "micro", "kanata",
    ];
    let mut replaces = Vec::new();
    let transforms = kanata_transform_map();
    for entry in &entries {
        let has_allowed = entry
            .actions
            .iter()
            .any(|a| allowed.contains(&util::normalize_program(&a.program).as_str()));
        if !has_allowed {
            continue;
        }
        let trigger_key =
            util::format_trigger_embed(&entry.trigger, &lookups.embed, "kanata", &transforms);
        if trigger_key.is_empty() {
            continue;
        }
        let bind_key = util::format_key_seq(&entry.binding, &lookups.embed, "kanata", "");
        if bind_key.is_empty() {
            continue;
        }
        let prefix = format!("  {}", trigger_key);
        let padding = 10usize.saturating_sub(prefix.len()).max(1);
        let old = prefix.clone();
        let new = format!("{}{}{}:line", prefix, " ".repeat(padding), bind_key);
        replaces.push((old, new));
    }
    if replaces.is_empty() {
        eprintln!("Warning: No kanata bindings found for allowed programs");
    }
    let target_file = global.root.join("..").join("kanata").join("babel.kdb");
    forge_file(&target_file, replaces)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn embed_bindings(
    entries: Vec<edn::BindingEntry>,
    target: &str,
    lookups: &Lookups,
    global: &GlobalOpts,
    fmt: fn(&str, &str) -> (String, String),
) -> anyResult<()> {
    let filtered = edn::filter_by_program(entries, target);
    let mut raw: HashMap<String, String> = HashMap::new();
    for entry in &filtered {
        for action in &entry.actions {
            let key = util::format_key_seq(&entry.binding, &lookups.embed, &action.program, "-");
            raw.insert(key, action.command.clone());
        }
    }
    let formatted = util::format_binds(raw, target);
    let mut replaces = Vec::new();
    for (key, val) in &formatted {
        let (old, new) = fmt(key, val);
        replaces.push((old, new));
    }
    let target_file = global.root.join("..").join(target).join("babel.conf");
    forge_file(&target_file, replaces)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn serpl_format(key: &str, val: &str) -> (String, String) {
    (val.to_string(), format!("\"<{}>\" = \"{}\":line", key, val))
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn lazygit_format(key: &str, val: &str) -> (String, String) {
    (val.to_string(), format!("    {}: '<{}>':line", val, key))
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn embed_zellij(entries: Vec<edn::BindingEntry>, lookups: &Lookups, global: &GlobalOpts) -> anyResult<()> {
    let norm = util::normalize_program("zellij");
    let filtered = edn::filter_by_program(entries, &norm);
    let mut replaces = Vec::new();
    for entry in &filtered {
        for action in &entry.actions {
            let bind_key = util::format_key_seq(&entry.binding, &lookups.embed, &norm, " ");
            let cmd = action.command.trim_matches(&['[', ']'] as &[_]);
            let lhs = cmd.to_string();
            let rhs = format!("        bind \"{}\" {{ {} }}:line", bind_key, cmd);
            replaces.push((format!("\"{}\"", lhs), rhs));
        }
    }
    let target_file = global.root.join("..").join("zellij").join("config.kdl");
    forge_file(&target_file, replaces)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

/// Pure in‑process replacement using the `issac` library.
/// Reads `target_file`, applies all `replaces` (old / new strings with optional :line suffix),
/// then writes the forged result back to the same file.
fn forge_file(target_file: &PathBuf, replaces: Vec<(String, String)>) -> anyResult<()> {
    let raw = fs::read_to_string(target_file)
        .with_context(|| format!("failed to read {}", target_file.display()))?;

    let replacements: Vec<issac::Replacement> = replaces
        .iter()
        .map(|(old, new)| {
            let pair = format!("{old}={new}");
            pair.parse::<issac::Replacement>()
                .map_err(|e| anyhow::anyhow!("invalid replacement pair '{pair}': {e}"))
        })
        .collect::<anyResult<_>>()?;

    let forged = issac::forge(&[raw.as_str()], &replacements);

    fs::write(target_file, forged)
        .with_context(|| format!("failed to write {}", target_file.display()))
}

////////////////////////////////////////////////////////////////////////////////////////////////////

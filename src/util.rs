////////////////////////////////////////////////////////////////////////////////////////////////////

use anyhow::{Context, Result as anyResult, bail};
use colored::*;
use std::collections::HashMap;
use std::fs;
use std::io::Write;
use std::path::PathBuf;

////////////////////////////////////////////////////////////////////////////////////////////////////

use issac::{Replacement, forge};

////////////////////////////////////////////////////////////////////////////////////////////////////

use crate::cli::GlobalOpts;
use crate::edn;
use crate::lookup::Lookups;

////////////////////////////////////////////////////////////////////////////////////////////////////

pub struct TableRow {
    pub program: String,
    pub action: String,
    pub trigger: String,
    pub binding: String,
    pub empty: bool,
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn default_root_dir() -> std::path::PathBuf {
    let home = dirs::home_dir().expect("cannot determine home directory");
    home.join(".saiyajin/edn")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn is_empty_entry(e: &edn::BindingEntry) -> bool {
    e.actions.is_empty()
        || e.actions.iter().all(|a| {
            a.program.trim().is_empty()
                || a.program == "<nil>"
                || a.action.trim().is_empty()
                || a.action == "<nil>"
                || match &a.command {
                    edn::Command::Simple(s) => s.trim().is_empty() || s == "<nil>",
                    edn::Command::List(v) => {
                        v.is_empty() || v.iter().all(|s| s.trim().is_empty() || s == "<nil>")
                    }
                }
        })
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn format_trigger_display(
    k: &edn::KeySeq,
    lookups: &HashMap<String, HashMap<String, String>>,
    program: &str,
) -> String {
    format_key_seq(k, lookups, program, " ")
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn format_binding_display(
    b: &edn::BindingEntry,
    lookups: &HashMap<String, HashMap<String, String>>,
    program: &str,
) -> String {
    let program_norm = normalize_program(program);
    let empty = HashMap::new();
    let lookup_map = lookups
        .get(&program_norm)
        .or_else(|| lookups.get("default"))
        .unwrap_or(&empty);

    let key = if b.sequence.is_empty() {
        &b.binding.key
    } else {
        &b.sequence
    };
    let key = function_key_to_upper(key);
    let mod_parts: Vec<String> = b
        .binding
        .modifier
        .chars()
        .map(|c| lookup(&c.to_string(), lookup_map))
        .collect();
    let mod_str = mod_parts.join("-");
    if mod_str.is_empty() {
        lookup(&key, lookup_map)
    } else {
        format!("{}-{}", mod_str, lookup(&key, lookup_map))
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn format_key_seq(
    k: &edn::KeySeq,
    lookups: &HashMap<String, HashMap<String, String>>,
    program: &str,
    sep: &str,
) -> String {
    let program_norm = normalize_program(program);
    let empty = HashMap::new();
    let map = lookups
        .get(&program_norm)
        .or_else(|| lookups.get("default"))
        .unwrap_or(&empty);

    let mod_parts: Vec<String> = k
        .modifier
        .chars()
        .map(|c| lookup(&c.to_string(), map))
        .collect();
    let mod_str = mod_parts.join(sep);
    let key = function_key_to_upper(&k.key);
    let mapped = lookup(&key, map);
    if mod_str.is_empty() {
        mapped
    } else {
        format!("{}{}{}", mod_str, sep, mapped)
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn format_trigger_embed(
    k: &edn::KeySeq,
    lookups: &HashMap<String, HashMap<String, String>>,
    program: &str,
    transforms: &HashMap<String, String>,
) -> String {
    let norm = normalize_program(program);
    let empty = HashMap::new();
    let map = lookups
        .get(&norm)
        .or_else(|| lookups.get("default"))
        .unwrap_or(&empty);

    let mod_parts: String = k
        .modifier
        .chars()
        .map(|c| lookup(&c.to_string(), map))
        .collect();
    let key = transforms
        .get(&k.key)
        .cloned()
        .unwrap_or_else(|| k.key.clone());
    if k.mode.is_empty() {
        format!("{}{}", mod_parts, key)
    } else {
        format!("{}{}{}", k.mode, mod_parts, key)
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn lookup(key: &str, map: &HashMap<String, String>) -> String {
    map.get(key).cloned().unwrap_or_else(|| key.to_string())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn normalize_program(p: &str) -> String {
    match p {
        "kanata" => "kanata".to_string(),
        _ if p.contains("zellij") => "zellij".to_string(),
        _ if p.contains("micro") => "micro".to_string(),
        _ if p.contains("helix") => "helix".to_string(),
        _ => p.to_string(),
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn function_key_to_upper(key: &str) -> String {
    if key.len() > 1 && key.starts_with('f') && key[1..].parse::<u32>().is_ok() {
        format!("F{}", &key[1..])
    } else {
        key.to_string()
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn kanata_transform_map() -> HashMap<String, String> {
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

pub fn emit_config(
    w: &mut dyn Write,
    entries: &[edn::BindingEntry],
    target: &str,
    lookups: &Lookups,
) -> anyResult<()> {
    let filtered = crate::edn::filter_by_program(entries.to_vec(), target);
    let mut raw: HashMap<String, String> = HashMap::new();
    for entry in &filtered {
        for action in &entry.actions {
            let key = format_key_seq(&entry.binding, &lookups.interpret, &action.program, "-");
            raw.insert(key, action.command.as_target_string(target));
        }
    }

    let headers: HashMap<&str, &[&str]> = HashMap::from([
        ("helix-common", &[][..]),
        (
            "helix-insert",
            &["[keys.insert]", "A-ret = [\"completion\"]"],
        ),
        (
            "helix-normal",
            &[
                "[keys.normal]",
                "A-ret = [\"hover\"]",
                "\",\" = \"repeat_last_motion\"",
                "g = \"no_op\"",
                "G = \"no_op\"",
                "Z = \"no_op\"",
                "\"~\" = \"no_op\"",
                "\"=\" = \"no_op\"",
                "\"<\" = \"no_op\"",
                "\">\" = \"no_op\"",
                "q = \"no_op\"",
                "Q = \"no_op\"",
                "\"|\" = \"no_op\"",
                "\"A-|\" = \"no_op\"",
                "\"!\" = \"no_op\"",
                "\"A-!\" = \"no_op\"",
                "\"$\" = \"no_op\"",
                "S = \"no_op\"",
                "\"A-_\" = \"no_op\"",
                "\"&\" = \"no_op\"",
                "\"_\" = \"no_op\"",
                "\"A-;\" = \"no_op\"",
                "\"A-:\" = \"no_op\"",
                "\"A-,\" = \"no_op\"",
                "C = \"no_op\"",
                "\"(\" = \"no_op\"",
                "\")\" = \"no_op\"",
                "\"A-(\" = \"no_op\"",
                "\"A-)\" = \"no_op\"",
                "\"%\" = \"no_op\"",
                "x = \"no_op\"",
                "X = \"no_op\"",
                "J = \"no_op\"",
                "K = \"no_op\"",
                "\"C-c\" = \"no_op\"",
                "\"?\" = \"no_op\"",
                "m = \"no_op\"",
                "n = \"no_op\"",
                "N = \"no_op\"",
                "\"A-*\" = \"no_op\"",
            ],
        ),
        ("helix-select", &["[keys.select]", "A-ret = [\"hover\"]"]),
        (
            "micro",
            &[
                "\"MouseRight\": \"MouseMultiCursor\",",
                "\"AltEnter\": \"Autocomplete\",",
            ],
        ),
    ]);

    if target.starts_with("helix-") || target == "micro" {
        if target == "micro" {
            writeln!(w, "{{")?;
            if let Some(lines) = headers.get(target) {
                for line in *lines {
                    writeln!(w, "  {line}")?;
                }
            }
            for (key, val) in &raw {
                writeln!(w, "  {:?}: {:?},", key, val)?;
            }
            writeln!(w, "}}")?;
        } else {
            if let Some(lines) = headers.get(target) {
                for line in *lines {
                    writeln!(w, "{line}")?;
                }
            }
            for (key, val) in &raw {
                writeln!(w, "{} = {}", key, val)?;
            }
        }
    } else {
        anyhow::bail!("unsupported program {}", target);
    }
    Ok(())
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub(crate) fn format_binds(raw: HashMap<String, String>, program: &str) -> HashMap<String, String> {
    raw.into_iter()
        .map(|(k, v)| {
            let v = match program {
                p if p.starts_with("helix-") => toml_list(&v),
                "micro" | "lazygit" | "serpl" | "zellij" => {
                    v.trim_matches(&['[', ']'] as &[_]).to_string()
                }
                _ => v,
            };
            (k, v)
        })
        .collect()
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn toml_list(raw: &str) -> String {
    let inner = raw.trim().trim_start_matches('[').trim_end_matches(']');
    if inner.starts_with(":sh ") || inner.starts_with(":echo ") {
        format!("[\"{}\"]", inner)
    } else if inner.is_empty() {
        "[]".to_string()
    } else {
        let parts: Vec<_> = inner
            .split_whitespace()
            .map(|p| format!("\"{}\"", p))
            .collect();
        format!("[{}]", parts.join(","))
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn embed_config(
    entries: Vec<edn::BindingEntry>,
    program: &str,
    target_file: &PathBuf,
    lookups: &Lookups,
) -> anyResult<()> {
    match program {
        "kanata" => embed_kanata(entries, target_file, lookups),
        "serpl" => embed_bindings(entries, program, target_file, lookups, serpl_format),
        "lazygit" => embed_bindings(entries, program, target_file, lookups, lazygit_format),
        z if z.starts_with("zellij") => embed_zellij(entries, program, target_file, lookups),
        _ => bail!("unsupported --program: {}", program),
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn embed_kanata(
    entries: Vec<edn::BindingEntry>,
    target_file: &PathBuf,
    lookups: &Lookups,
) -> anyResult<()> {
    let allowed: Vec<&str> = vec![
        "helix", "serpl", "lazygit", "zellij", "term", "micro", "kanata",
    ];
    let mut replaces = Vec::new();
    let transforms = kanata_transform_map();
    for entry in &entries {
        let has_allowed = entry
            .actions
            .iter()
            .any(|a| allowed.contains(&normalize_program(&a.program).as_str()));
        if !has_allowed {
            continue;
        }
        let trigger_key =
            format_trigger_embed(&entry.trigger, &lookups.embed, "kanata", &transforms);
        if trigger_key.is_empty() {
            continue;
        }
        let bind_key = format_key_seq(&entry.binding, &lookups.embed, "kanata", "");
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
    forge_file(target_file, replaces)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn embed_bindings(
    entries: Vec<edn::BindingEntry>,
    program: &str,
    target_file: &PathBuf,
    lookups: &Lookups,
    fmt: fn(&str, &str) -> (String, String),
) -> anyResult<()> {
    let filtered = edn::filter_by_program(entries, program);
    let mut raw: HashMap<String, String> = HashMap::new();
    for entry in &filtered {
        for action in &entry.actions {
            let key = format_key_seq(&entry.binding, &lookups.embed, &action.program, "-");
            raw.insert(key, action.command.as_target_string(program));
        }
    }
    let mut replaces = Vec::new();
    for (key, val) in &raw {
        let (old, new) = fmt(key, val);
        replaces.push((old, new));
    }
    forge_file(target_file, replaces)
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

fn embed_zellij(
    entries: Vec<edn::BindingEntry>,
    program: &str,
    target_file: &PathBuf,
    lookups: &Lookups,
) -> anyResult<()> {
    let norm = normalize_program(program);
    let filtered = edn::filter_by_program(entries, &norm);
    let mut replaces = Vec::new();
    for entry in &filtered {
        for action in &entry.actions {
            let bind_key = format_key_seq(&entry.binding, &lookups.embed, &norm, " ");
            let cmd = action.command.as_target_string("zellij");
            let lhs = cmd.clone();
            let rhs = format!("        bind \"{}\" {{ {} }}:line", bind_key, cmd);
            replaces.push((format!("\"{}\"", lhs), rhs));
        }
    }
    forge_file(target_file, replaces)
}

////////////////////////////////////////////////////////////////////////////////////////////////////

/// Pure in‑process replacement using the `issac` library.
/// Reads `target_file`, applies all `replaces` (old / new strings with optional :line suffix),
/// then writes the forged result back to the same file.
fn forge_file(target_file: &PathBuf, replaces: Vec<(String, String)>) -> anyResult<()> {
    let raw = fs::read_to_string(target_file)
        .with_context(|| format!("failed to read {}", target_file.display()))?;

    let replacements: Vec<Replacement> = replaces
        .iter()
        .map(|(old, new)| {
            let pair = format!("{old}={new}");
            pair.parse::<Replacement>()
                .map_err(|e| anyhow::anyhow!("invalid replacement pair '{pair}': {e}"))
        })
        .collect::<anyResult<_>>()?;

    let forged = forge(&[raw.as_str()], &replacements);

    fs::write(target_file, forged)
        .with_context(|| format!("failed to write {}", target_file.display()))
}

////////////////////////////////////////////////////////////////////////////////////////////////////

pub fn print_table(rows: &[TableRow]) {
    if rows.is_empty() {
        println!("No bindings found.");
        return;
    }
    let border = "==================================================================================================";
    println!("{}", border);
    println!(
        "| Program         | Action                         | Trigger              | Binding              |"
    );
    println!(
        "|-----------------|--------------------------------|----------------------|----------------------|"
    );
    for row in rows {
        let color = program_color(&row.program);
        let line = format!(
            "| {} | {} | {} | {} |",
            render_cell(&row.program, 15, color),
            render_cell(&row.action, 30, None),
            render_cell(&row.trigger, 20, None),
            render_cell(&row.binding, 20, None)
        );
        let line = if row.empty {
            line.dimmed()
        } else {
            line.normal()
        };
        println!("{}", line);
    }
    println!("{}", border);
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn render_cell(val: &str, width: usize, color: Option<Color>) -> String {
    let raw = format!("{:<width$}", val, width = width);
    if let Some(c) = color {
        raw.color(c).to_string()
    } else {
        raw
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

fn program_color(prog: &str) -> Option<Color> {
    match prog {
        "micro" | "helix-common" | "helix-insert" | "helix-normal" | "helix-pop"
        | "helix-select" => Some(Color::Cyan),
        "lazygit" | "serpl" => Some(Color::Green),
        "terminal" => Some(Color::Blue),
        "zellij" => Some(Color::Yellow),
        _ => None,
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////

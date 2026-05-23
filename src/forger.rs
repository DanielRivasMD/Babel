use crate::cmds::GlobalOpts;
use crate::lookup::Lookups;
use crate::parser::{self, BindingEntry};
use crate::util::{self, normalize_program};
use anyhow::{bail, Context, Result};
use std::collections::HashMap;
use std::path::PathBuf;
use std::process::Command;

pub fn embed_config(
    entries: Vec<BindingEntry>,
    target: &str,
    lookups: &Lookups,
    global: &GlobalOpts,
) -> Result<()> {
    match target {
        "kanata" => embed_kanata(entries, lookups, global),
        "serpl" => embed_bindings(entries, target, lookups, global, serpl_format),
        "lazygit" => embed_bindings(entries, target, lookups, global, lazygit_format),
        z if z.starts_with("zellij") => embed_zellij(entries, lookups, global),
        _ => bail!("unsupported --program: {}", target),
    }
}

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

fn embed_kanata(entries: Vec<BindingEntry>, lookups: &Lookups, global: &GlobalOpts) -> Result<()> {
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
    run_mbombo(&target_file, &[target_file.clone()], replaces)
}

fn embed_bindings(
    entries: Vec<BindingEntry>,
    target: &str,
    lookups: &Lookups,
    global: &GlobalOpts,
    fmt: fn(&str, &str) -> (String, String),
) -> Result<()> {
    let filtered = parser::filter_by_program(entries, target);
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
    run_mbombo(&target_file, &[target_file.clone()], replaces)
}

fn serpl_format(key: &str, val: &str) -> (String, String) {
    (val.to_string(), format!("\"<{}>\" = \"{}\":line", key, val))
}

fn lazygit_format(key: &str, val: &str) -> (String, String) {
    (val.to_string(), format!("    {}: '<{}>':line", val, key))
}

fn embed_zellij(entries: Vec<BindingEntry>, lookups: &Lookups, global: &GlobalOpts) -> Result<()> {
    let norm = normalize_program("zellij");
    let filtered = parser::filter_by_program(entries, &norm);
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
    run_mbombo(&target_file, &[target_file.clone()], replaces)
}

fn run_mbombo(out: &PathBuf, files: &[PathBuf], replaces: Vec<(String, String)>) -> Result<()> {
    let mut cmd = Command::new("mbombo");
    cmd.arg("--out").arg(out);
    for f in files {
        cmd.arg("--files").arg(f);
    }
    for (old, new) in &replaces {
        cmd.arg("--replace").arg(format!("{}={}", old, new));
    }
    let status = cmd.status().context("failed to run mbombo")?;
    if !status.success() {
        bail!("mbombo command failed");
    }
    Ok(())
}

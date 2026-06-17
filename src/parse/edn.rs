use anyhow::Result;
use once_cell::sync::Lazy;
use regex::Regex;
use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};

use edn::parser::Parser;
use edn::Value;

// regex helpers using once_cell
static RE_FN: Lazy<Regex> = Lazy::new(|| Regex::new(r"^([OESRTWCQ]+)(f[0-9]+)$").unwrap());
static RE_CH: Lazy<Regex> = Lazy::new(|| Regex::new(r"^([OESRTWCQ]+)([a-z])$").unwrap());
static RE_NB: Lazy<Regex> = Lazy::new(|| Regex::new(r"^([OESRTWCQ]+)([0-9])$").unwrap());
static RE_OT: Lazy<Regex> = Lazy::new(|| Regex::new(r"^([OESRTWCQ]+)([a-z_]+)$").unwrap());
static RE_KW: Lazy<Regex> = Lazy::new(|| Regex::new(r"^([OESRTWCQ]*)#P(.+)$").unwrap());

#[derive(Debug, Clone)]
pub struct KeySeq {
    pub mode: String,
    pub modifier: String,
    pub key: String,
}

#[derive(Debug, Clone)]
pub struct ProgramAction {
    pub program: String,
    pub action: String,
    pub command: String,
}

#[derive(Debug, Clone)]
pub struct BindingEntry {
    pub trigger: KeySeq,
    pub binding: KeySeq,
    pub sequence: String,
    pub actions: Vec<ProgramAction>,
    pub annotations: HashMap<String, Vec<String>>,
}

pub fn resolve_edn_files(file: Option<PathBuf>, root: &Path) -> Vec<PathBuf> {
    if let Some(f) = file {
        return vec![f];
    }
    let mut out = Vec::new();
    for entry in walkdir::WalkDir::new(root) {
        let entry = entry.unwrap();
        if entry.file_type().is_file() && entry.path().extension().map_or(false, |e| e == "edn") {
            out.push(entry.into_path());
        }
    }
    out
}

pub fn parse_edn_files(paths: &[PathBuf]) -> Result<Vec<BindingEntry>> {
    let mut all = Vec::new();
    for path in paths {
        all.extend(parse_edn_file(path)?);
    }
    Ok(all)
}

fn parse_edn_file(path: &Path) -> Result<Vec<BindingEntry>> {
    let text = fs::read_to_string(path)?;
    let mode = extract_mode(&text);
    let entries = parse_binding_entries(&text, &mode);
    Ok(entries)
}

fn extract_mode(text: &str) -> String {
    if let Some(rules_idx) = text.find(":rules") {
        let after = &text[rules_idx..];
        if let Some(bracket) = after.find('[') {
            if let Some(colon) = after[bracket..].find(':') {
                let mode_str = &after[bracket..][colon..];
                if let Some(dash) = mode_str.find('-') {
                    return mode_str[1..dash].to_string();
                }
            }
        }
    }
    String::new()
}

fn parse_binding_entries(text: &str, mode: &str) -> Vec<BindingEntry> {
    let mut entries = Vec::new();
    let mut pos = 0;
    while let Some((meta, vec_str, new_pos)) = extract_entry(text, pos) {
        pos = new_pos;
        if let Ok(raw_meta) = decode_metadata(&meta) {
            if let Ok(vec) = decode_rule(&vec_str) {
                if let Some(entry) = parse_binding_entry(raw_meta, &vec, mode) {
                    entries.push(entry);
                }
            }
        }
    }
    entries
}

fn extract_entry(text: &str, start: usize) -> Option<(String, String, usize)> {
    let delta = text[start..].find('^')?;
    let i = start + delta;
    let mut j = i + 1;
    for ch in text[j..].chars() {
        if ch.is_whitespace() {
            j += 1;
        } else {
            break;
        }
    }
    if text[j..].starts_with('{') {
        let mut depth = 0;
        let mut k = j;
        while k < text.len() {
            match text.chars().nth(k).unwrap() {
                '{' => depth += 1,
                '}' => {
                    depth -= 1;
                    if depth == 0 {
                        k += 1;
                        break;
                    }
                }
                _ => {}
            }
            k += 1;
        }
        if depth != 0 {
            return None;
        }
        let meta_end = k;
        let meta = text[j..meta_end].to_string();

        let mut p = meta_end;
        for ch in text[p..].chars() {
            if ch.is_whitespace() {
                p += 1;
            } else {
                break;
            }
        }
        if text[p..].starts_with('[') {
            let mut depth = 0;
            let mut q = p;
            while q < text.len() {
                match text.chars().nth(q).unwrap() {
                    '[' => depth += 1,
                    ']' => {
                        depth -= 1;
                        if depth == 0 {
                            q += 1;
                            break;
                        }
                    }
                    _ => {}
                }
                q += 1;
            }
            if depth != 0 {
                return None;
            }
            let vec_end = q;
            let vec_str = text[p..vec_end].to_string();
            return Some((meta, vec_str, vec_end));
        }
    }
    // continue search
    extract_entry(text, i + 1)
}

// Convert a Value to a string in EDN-like representation (e.g., ":trigger" for keywords)
fn value_to_edn_string(v: &Value) -> String {
    match v {
        Value::Keyword(kw) => format!(":{}", kw),
        Value::String(s) => s.clone(),
        Value::Symbol(s) => s.clone(),
        // fallback: use Debug format for other types
        _ => format!("{:?}", v),
    }
}

// Convert a Value used as a map key to a plain string (without colon for keywords)
fn value_to_key_string(v: &Value) -> String {
    match v {
        Value::Keyword(kw) => kw.clone(),
        Value::String(s) => s.clone(),
        _ => format!("{:?}", v),
    }
}

fn decode_metadata(meta_str: &str) -> Result<HashMap<String, Value>> {
    let mut parser = Parser::new(meta_str);
    let opt_result = parser.read();
    let val = opt_result
        .ok_or_else(|| anyhow::anyhow!("no more EDN values in metadata"))
        .and_then(|res| res.map_err(|e| anyhow::anyhow!("EDN parse error: {:?}", e)))?;
    match val {
        Value::Map(map) => {
            let mut res = HashMap::new();
            for (k, v) in map {
                let key_str = value_to_key_string(&k);
                res.insert(key_str, v);
            }
            Ok(res)
        }
        _ => anyhow::bail!("metadata is not a map"),
    }
}

fn decode_rule(vec_str: &str) -> Result<Vec<Value>> {
    let mut parser = Parser::new(vec_str);
    let opt_result = parser.read();
    let val = opt_result
        .ok_or_else(|| anyhow::anyhow!("no more EDN values in rule"))
        .and_then(|res| res.map_err(|e| anyhow::anyhow!("EDN parse error: {:?}", e)))?;
    match val {
        Value::Vector(vec) => Ok(vec),
        _ => anyhow::bail!("rule is not a vector"),
    }
}

fn parse_binding_entry(
    raw_meta: HashMap<String, Value>,
    vec: &[Value],
    mode: &str,
) -> Option<BindingEntry> {
    if vec.len() < 2 {
        return None;
    }
    let trigger_raw = value_to_edn_string(&vec[0]);
    let (tm, tk) = split_edn_key(&trigger_raw);
    let trigger = KeySeq {
        mode: mode.to_string(),
        modifier: tm,
        key: tk,
    };
    let binding_raw = build_key_sequence(&vec[1]);
    let (bm, bk) = split_edn_key(&binding_raw);
    let mut binding = KeySeq {
        mode: String::new(),
        modifier: bm,
        key: bk,
    };

    let mut actions = Vec::new();
    let mut seq = String::new();
    if let Some(val) = raw_meta.get("doc/actions") {
        if let Value::Vector(acts) = val {
            for a in acts {
                if let Value::Map(map) = a {
                    let mut pa = ProgramAction {
                        program: String::new(),
                        action: String::new(),
                        command: String::new(),
                    };
                    for (k, v) in map {
                        let key_str = value_to_key_string(k);
                        let val_str = value_to_edn_string(v);
                        match key_str.as_str() {
                            "program" => pa.program = val_str,
                            "action" => pa.action = val_str,
                            "exec" => pa.command = val_str,
                            "sequence" => seq = val_str,
                            _ => {}
                        }
                    }
                    actions.push(pa);
                }
            }
        }
    }

    let annotations = parse_annotations(vec);
    if let Some(alone) = annotations.get("alone") {
        if let Some(val) = alone.first() {
            let (bm, bk) = split_edn_key(val);
            binding = KeySeq {
                mode: String::new(),
                modifier: bm,
                key: bk,
            };
        }
    }

    Some(BindingEntry {
        trigger,
        binding,
        sequence: seq,
        actions,
        annotations,
    })
}

fn build_key_sequence(x: &Value) -> String {
    match x {
        Value::Vector(v) => v
            .iter()
            .map(value_to_edn_string)
            .collect::<Vec<_>>()
            .join(" "),
        _ => value_to_edn_string(x),
    }
}

pub fn split_edn_key(s: &str) -> (String, String) {
    let s = s.trim().trim_start_matches(':').trim_start_matches('!');
    for re in &[&*RE_FN, &*RE_CH, &*RE_NB, &*RE_OT, &*RE_KW] {
        if let Some(caps) = re.captures(s) {
            return (caps[1].to_string(), caps[2].to_string());
        }
    }
    (String::new(), s.to_string())
}

fn parse_annotations(vec: &[Value]) -> HashMap<String, Vec<String>> {
    let mut anns = HashMap::new();
    if vec.len() < 4 {
        return anns;
    }
    if let Value::Map(map) = &vec[3] {
        for (k, v) in map {
            let key_str = value_to_key_string(k);
            match v {
                Value::Vector(vv) => {
                    for item in vv {
                        anns.entry(key_str.clone())
                            .or_default()
                            .push(value_to_edn_string(item));
                    }
                }
                _ => {
                    anns.entry(key_str.clone())
                        .or_default()
                        .push(value_to_edn_string(v));
                }
            }
        }
    }
    anns
}

pub fn filter_by_program(entries: Vec<BindingEntry>, filter: &str) -> Vec<BindingEntry> {
    if filter.is_empty() {
        return entries;
    }
    let re = Regex::new(filter).unwrap();
    entries
        .into_iter()
        .filter_map(|mut e| {
            let actions: Vec<_> = e
                .actions
                .into_iter()
                .filter(|a| re.is_match(&a.program))
                .collect();
            if actions.is_empty() {
                None
            } else {
                e.actions = actions;
                Some(e)
            }
        })
        .collect()
}

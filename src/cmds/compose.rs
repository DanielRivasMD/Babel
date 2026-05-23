use crate::cmds::GlobalOpts;
use anyhow::Result;
use std::fmt::Write as FmtWrite;
use std::fs;
use std::io::{self, Write};
use std::path::PathBuf;

pub fn run(global: GlobalOpts, template: Option<PathBuf>) -> Result<()> {
    if global.program.as_deref() != Some("kanata") {
        anyhow::bail!("unsupported program {:?} for compose", global.program);
    }
    let prefixes = &[
        "T", "TS", "O", "OS", "C", "CS", "J", "S", "R", "Q", "QR", "E", "ER", "W", "WS", "tab",
        "q", "w", "z", "zS",
    ];
    let suffixes: Vec<String> = {
        let mut v = Vec::new();
        for i in 1..=9 {
            v.push(i.to_string());
        }
        v.push("0".into());
        for c in 'a'..='z' {
            v.push(c.to_string());
        }
        v.extend(
            vec![
                "lf", "rg", "up", "dn", "hy", "eq", "db", "ob", "cb", "sc", "qu", "bl", "cm", "pe",
                "sl", "ret", "spc", "kR", "kE", "kQ", "kC", "kO", "kT", "kS", "kW",
            ]
            .into_iter()
            .map(String::from),
        );
        v
    };

    let mut out = String::new();
    out.push_str("(defalias\n");
    for (i, p) in prefixes.iter().enumerate() {
        for s in &suffixes {
            let key = format!("{p}{s}");
            let line = format!("  {key}");
            let padding = 10usize.saturating_sub(line.len()).max(1);
            writeln!(out, "{:<padding$}XX", line)?;
        }
        if i + 1 != prefixes.len() {
            out.push('\n');
        }
    }
    out.push_str(")\n");

    if let Some(path) = template {
        fs::write(&path, &out)?;
    } else {
        io::stdout().write_all(out.as_bytes())?;
    }
    Ok(())
}

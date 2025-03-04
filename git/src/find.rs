use std::{fs::read_dir, io, path::Path};

use crate::WorkspaceRepo;

pub fn find_repos(workspace_root: &Path, search_depth: u8) -> io::Result<Vec<WorkspaceRepo>> {
    let mut found: Vec<WorkspaceRepo> = Vec::new();
    for de in read_dir(workspace_root)? {
        let p = de?.path();
        if p.is_dir() {
            if p.join(".git").is_dir() {
                found.push(WorkspaceRepo::new(workspace_root, p));
            } else if search_depth > 1 {
                found.append(&mut find_repos(&p, search_depth - 1)?);
            }
        }
    }
    Ok(found)
}

mod find;
mod host;
mod pull;

#[cfg(test)]
mod host_test;

#[cfg(test)]
mod pull_test;

use pull::pull_ff;
use std::path::{Path, PathBuf};
use tokio::{sync::mpsc::UnboundedReceiver, task::JoinSet};

pub use find::*;
pub use host::*;
pub use pull::PullResult;

#[derive(Debug)]
pub struct WorkspaceRepo {
    /// unique id of rel path from workspare root to repo
    pub label: String,
    /// abs path to repo dir
    pub path: PathBuf,
}

impl WorkspaceRepo {
    fn new(workspace_root: &Path, p: PathBuf) -> Self {
        Self {
            label: p
                .to_string_lossy()
                .to_string()
                .strip_prefix(
                    format!("{}/", workspace_root.to_string_lossy().to_string().as_str(),).as_str(),
                )
                .unwrap()
                .to_string(),
            path: p,
        }
    }
}

pub struct SyncOptions {
    pub repos: Vec<WorkspaceRepo>,
}

pub struct Sync {
    _join_set: JoinSet<()>,
    pub repos: Vec<String>,
    pub rx: UnboundedReceiver<SyncResult>,
}

impl Sync {
    pub fn max_repo_label_len(&self) -> usize {
        let mut max = 0;
        for repo in &self.repos {
            let len = repo.len();
            if len > max {
                max = len;
            }
        }
        max
    }
}

#[derive(Debug)]
pub enum SyncKind {
    Pull(PullResult),
}

#[derive(Debug)]
pub struct SyncResult {
    pub repo: WorkspaceRepo,
    pub kind: SyncKind,
}

pub fn sync(opts: SyncOptions) -> Result<Sync, anyhow::Error> {
    let (tx, rx) = tokio::sync::mpsc::unbounded_channel::<SyncResult>();
    let mut join_set = JoinSet::new();
    let mut repos = Vec::with_capacity(opts.repos.len());
    for repo in opts.repos {
        repos.push(repo.label.clone());
        let tx = tx.clone();
        join_set.spawn(async move {
            let kind = match pull_ff(&repo.path) {
                Err(err) => SyncKind::Pull(PullResult::Error(err.to_string())),
                Ok(pull_result) => SyncKind::Pull(pull_result),
            };
            tx.send(SyncResult { repo, kind }).expect("tx send");
        });
    }
    Ok(Sync {
        _join_set: join_set,
        repos,
        rx,
    })
}

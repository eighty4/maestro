mod find;
mod host;
mod pull;
mod status;

#[cfg(test)]
mod host_test;

#[cfg(test)]
mod pull_test;

#[cfg(test)]
mod status_test;

#[cfg(test)]
mod testing;

use pull::pull_ff;
use status::status;
use std::{
    net::ToSocketAddrs,
    path::{Path, PathBuf},
};
use tokio::{
    sync::mpsc::{unbounded_channel, UnboundedReceiver, UnboundedSender},
    task::JoinSet,
};

pub use find::*;
pub use host::*;
pub use pull::PullResult;
pub use status::{LocalChanges, RepoState};

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
    pub offline: bool,
    pub repos: Vec<WorkspaceRepo>,
}

pub struct Sync {
    _join_set: JoinSet<()>,
    // network available if not offline mode
    pub network: bool,
    // user preference offline mode
    pub offline: bool,
    pub repos: Vec<String>,
    pub rx: UnboundedReceiver<SyncResult>,
}

impl Sync {
    // return true if not offline mode and network unavailable
    pub fn is_network_unavailable(&self) -> bool {
        !self.offline && !self.network
    }

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
    Skipped,
}

#[derive(Debug)]
pub struct SyncResult {
    pub kind: SyncKind,
    pub repo: WorkspaceRepo,
    pub state: RepoState,
}

pub fn sync(opts: SyncOptions) -> Result<Sync, anyhow::Error> {
    let (tx, rx) = unbounded_channel::<SyncResult>();
    let mut join_set = JoinSet::new();
    let mut repos = Vec::with_capacity(opts.repos.len());
    let network = !opts.offline && is_network_available();
    for repo in opts.repos {
        repos.push(repo.label.clone());
        join_set.spawn(sync_flow(repo, !network, tx.clone()));
    }
    Ok(Sync {
        _join_set: join_set,
        network,
        offline: opts.offline,
        repos,
        rx,
    })
}

fn is_network_available() -> bool {
    "github.com:443".to_socket_addrs().is_ok()
}

async fn sync_flow(repo: WorkspaceRepo, offline: bool, tx: UnboundedSender<SyncResult>) {
    let kind = if offline {
        SyncKind::Skipped
    } else {
        match pull_ff(&repo.path) {
            Err(err) => SyncKind::Pull(PullResult::Error(err.to_string())),
            Ok(pull_result) => SyncKind::Pull(pull_result),
        }
    };
    let state = status(&repo.path).expect("bad rust control flow and data structuring");
    tx.send(SyncResult { kind, repo, state }).expect("tx send");
}

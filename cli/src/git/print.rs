use crossterm::style::Stylize;
use maestro_git::{LocalChanges, PullResult, Sync, SyncKind};

use crate::git::indicator::SyncIndicator;

pub async fn print_sync_updates(mut syncing: Sync) {
    let label_max_len = syncing.max_repo_label_len();
    println!("Syncing {} repos...\n", syncing.repos.len());
    let mut _done: usize = 0;
    while let Some(result) = syncing.rx.recv().await {
        _done += 1;
        let text = match &result.kind {
            SyncKind::Pull(PullResult::DetachedHead) => "detached head".to_string(),
            SyncKind::Pull(PullResult::Error(err_msg)) => err_msg.clone(),
            SyncKind::Pull(PullResult::FastForward { commits, .. }) => {
                format!("pulled {commits} commits",)
            }
            SyncKind::Pull(PullResult::UnpullableMerge) => {
                "unable to ff merge from remote".to_string()
            }
            SyncKind::Pull(PullResult::UpToDate) => "already up to date".to_string(),
        };
        let local_changes = match result.state.changes {
            LocalChanges::Clean => String::new(),
            LocalChanges::Present {
                stashes,
                staged,
                unstaged,
                untracked,
            } => {
                let mut changes = Vec::new();
                if stashes > 0 {
                    changes.push(format!(
                        "{stashes} stash{}",
                        if stashes == 1 { "" } else { "es" }
                    ));
                }
                if staged > 0 || unstaged > 0 || untracked > 0 {
                    let total = staged + unstaged + untracked;
                    changes.push(format!(
                        "{total} change{}",
                        if total == 1 { "" } else { "s" }
                    ));
                }
                format!(", local has {}", changes.join(", "))
            }
        };
        let display_name = format!("{:w$}", result.repo.label, w = label_max_len).bold();
        let indicator = match SyncIndicator::from(&result) {
            SyncIndicator::Clean => '✔'.green(),
            SyncIndicator::LocalChanges => '✔'.yellow(),
            SyncIndicator::Error => '✗'.red(),
        };
        println!(
            "  {}  {} {}{}",
            display_name, indicator, text, local_changes,
        );
    }
    println!("\nAll repositories synced!");
}

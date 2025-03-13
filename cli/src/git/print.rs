use crossterm::style::Stylize;
use maestro_git::{LocalChanges, PullResult, Sync, SyncKind};

use crate::git::indicator::SyncIndicator;

pub async fn print_sync_updates(mut syncing: Sync) {
    let label_max_len = syncing.max_repo_label_len();
    println!("Syncing {} repos...\n", syncing.repos.len());
    let mut _done: usize = 0;
    while let Some(result) = syncing.rx.recv().await {
        _done += 1;
        let mut statuses = Vec::new();
        let sync_status = match &result.kind {
            SyncKind::Pull(PullResult::UpToDate) => None,
            SyncKind::Pull(PullResult::DetachedHead) => Some("detached head".to_string()),
            SyncKind::Pull(PullResult::Error(err_msg)) => Some(err_msg.clone().red().to_string()),
            SyncKind::Pull(PullResult::FastForward { commits, .. }) => {
                Some(format!("pulled {commits} commits"))
            }
            SyncKind::Pull(PullResult::UnpullableMerge) => Some("unable to ff merge".to_string()),
        };
        if let Some(sync_status) = sync_status {
            statuses.push(sync_status);
        }
        if let LocalChanges::Present {
            stashes,
            staged,
            unstaged,
            untracked,
        } = result.state.changes
        {
            if stashes > 0 {
                statuses.push(format!(
                    "{stashes} stash{}",
                    if stashes == 1 { "" } else { "es" }
                ));
            }
            if staged > 0 || unstaged > 0 || untracked > 0 {
                let total = staged + unstaged + untracked;
                statuses.push(format!(
                    "{total} local change{}",
                    if total == 1 { "" } else { "s" }
                ));
            }
        }
        let display_name = format!("{:w$}", result.repo.label, w = label_max_len).bold();
        let indicator = match SyncIndicator::from(&result) {
            SyncIndicator::Clean => '✔'.green(),
            SyncIndicator::LocalChanges => '✔'.yellow(),
            SyncIndicator::Error => '✗'.red(),
        };
        println!("  {display_name}  {indicator} {}", statuses.join(", "));
    }
    println!("\nAll repositories synced!");
}

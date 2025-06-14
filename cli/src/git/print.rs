use crossterm::style::Stylize;
use maestro_git::{LocalChanges, PullResult, Sync, SyncKind};

use crate::git::indicator::SyncIndicator;

pub async fn print_sync_updates(mut syncing: Sync) {
    if syncing.offline || !syncing.network {
        println!("Offline report of {} repos...\n", syncing.repos.len());
    } else {
        println!("Syncing {} repos...\n", syncing.repos.len());
    }
    let label_max_len = syncing.max_repo_label_len();
    while let Some(result) = syncing.rx.recv().await {
        let mut statuses = Vec::new();
        let sync_status = match &result.kind {
            SyncKind::Pull(PullResult::UpToDate) => None,
            SyncKind::Pull(PullResult::DetachedHead) => Some("detached head".to_string()),
            SyncKind::Pull(PullResult::Error(err_msg)) => Some(err_msg.clone().red().to_string()),
            SyncKind::Pull(PullResult::FastForward { commits, .. }) => {
                Some(format!("pulled {commits} commits"))
            }
            SyncKind::Pull(PullResult::UnpullableMerge) => Some("unable to ff merge".to_string()),
            SyncKind::Skipped => None,
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
            SyncIndicator::Error => '✗'.red(),
            SyncIndicator::LocalChanges => '✔'.yellow(),
            SyncIndicator::Noop => '✔'.dark_grey(),
        };
        println!("  {display_name}  {indicator} {}", statuses.join(", "));
    }
    if !syncing.offline && syncing.network {
        println!("\nAll repositories synced!");
    } else if syncing.is_network_unavailable() {
        println!("\n    {} network was unavailable for syncing\n", '✗'.red());
    }
}

use crossterm::style::Stylize;
use maestro_git::{PullResult, Sync, SyncKind};

pub async fn print_sync_updates(mut syncing: Sync) {
    let label_max_len = syncing.max_repo_label_len();
    println!("Syncing {} repos...\n", syncing.repos.len());
    let mut _done: usize = 0;
    while let Some(result) = syncing.rx.recv().await {
        _done += 1;
        let SyncKind::Pull(pull_result) = result.kind;
        let indicator = match pull_result {
            PullResult::FastForward(_) | PullResult::UpToDate => "✔".green(),
            _ => "✗".red(),
        };
        let text = match pull_result {
            PullResult::DetachedHead => "detached head".to_string(),
            PullResult::Error(err_msg) => err_msg,
            PullResult::FastForward(commits) => format!("pulled {commits} commits"),
            PullResult::UnpullableMerge => "unable to ff merge from remote".to_string(),
            PullResult::UpToDate => "already up to date".to_string(),
        };
        println!(
            "  {}  {} {}",
            format!("{:w$}", result.repo.label, w = label_max_len).bold(),
            indicator,
            text,
        );
    }
    println!("\nAll repositories synced!");
}

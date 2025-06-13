use std::process::Command;

use maestro_git::{find_repos, sync, SyncOptions};
use temp_dir::TempDir;

#[tokio::main]
async fn main() {
    let temp_dir = TempDir::new().unwrap();
    let repo = temp_dir.path().join("pear.ng");
    Command::new("git")
        .arg("clone")
        .arg("https://github.com/eighty4/pear.ng")
        .current_dir(temp_dir.path())
        .output()
        .unwrap();
    Command::new("git")
        .arg("reset")
        .arg("--hard")
        .arg("HEAD~2")
        .current_dir(&repo)
        .output()
        .unwrap();
    let repos = find_repos(temp_dir.path(), 1).unwrap();
    dbg!(&repos);
    let mut syncing = sync(SyncOptions {
        repos,
        offline: false,
    })
    .unwrap();
    while let Some(sync_result) = syncing.rx.recv().await {
        dbg!(sync_result);
    }
}

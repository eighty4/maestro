mod interactive;
mod print;

use clap::Parser;
use interactive::InteractiveSync;
use maestro_git::{SyncOptions, sync};
use print::print_sync_updates;

use crate::{MaestroCommandRun, MaestroCommandRunResult};

#[derive(Parser)]
pub struct GitCommand {
    #[clap(
        short,
        long,
        default_value = "false",
        long_help = "Sync a workspace of git repositories"
    )]
    interactive: bool,
}

impl MaestroCommandRun for GitCommand {
    async fn run(&self) -> MaestroCommandRunResult {
        let workspace_root = std::env::current_dir().expect("cwd");
        // find repos in workspace root
        let repos = maestro_git::find_repos(&workspace_root, 1).expect("find repos");
        if repos.is_empty() {
            println!("No repos found to sync");
            std::process::exit(0);
        }

        // start syncing and stream updates to ui
        let syncing = sync(SyncOptions { repos }).expect("sync start");
        if self.interactive {
            InteractiveSync::new(workspace_root.to_string_lossy().to_string(), syncing)
                .run()
                .await
                .expect("tui");
        } else {
            print_sync_updates(syncing).await;
        }
        Ok(())
    }
}

mod git;

use clap::{Parser, Subcommand};
use git::GitCommand;

type MaestroCommandRunResult = Result<(), anyhow::Error>;

trait MaestroCommandRun {
    async fn run(&self) -> MaestroCommandRunResult;
}

#[derive(Parser)]
#[command(author, version, about)]
struct MaestroCli {
    #[command(subcommand)]
    command: MaestroCommand,
}

#[derive(Subcommand)]
enum MaestroCommand {
    #[clap(about = "Operate a workspace of Git repos")]
    Git(GitCommand),
}

#[tokio::main]
async fn main() {
    let result = match MaestroCli::parse().command {
        MaestroCommand::Git(git) => git.run().await,
    };
    if let Err(err) = result {
        println!("\x1b[0;31;1merror:\x1b[0m {err}");
        std::process::exit(1);
    }
}

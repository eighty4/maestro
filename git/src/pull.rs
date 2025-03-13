use anyhow::anyhow;
use git2::{Direction, Oid, RemoteCallbacks};
use std::path::Path;

use crate::RemoteHost;

#[derive(Debug, Eq, PartialEq)]
pub enum PullResult {
    DetachedHead,
    Error(String),
    /// A ff merge performed with commit count
    FastForward {
        remote: RemoteHost,
        commits: u16,
        from: String,
        to: String,
    },
    /// Fetched changes cannot be ff merged
    UnpullableMerge,
    /// Remote had nothing to pull
    UpToDate,
}

pub fn pull_ff(p: &Path) -> Result<PullResult, anyhow::Error> {
    let repo = git2::Repository::open(p)?;

    if repo.head_detached()? {
        return Ok(PullResult::DetachedHead);
    }

    // creating remote connection for retrieving remote's default branch
    // and reused for fetching changes from remote
    let mut remote = repo.find_remote("origin")?;
    let mut remote_connection =
        remote.connect_auth(Direction::Fetch, Some(remote_auth_callbacks()), None)?;
    let remote_default_branch_name = remote_connection
        .default_branch()?
        .as_str()
        .map(String::from)
        .ok_or_else(|| anyhow!("remote default branch was not utf8"))?;

    // todo sync both default branch and HEAD ref branch, erroring for now
    let start_head_oid = get_ref_oid(&repo, "HEAD")?;
    if start_head_oid != get_ref_oid(&repo, &remote_default_branch_name)? {
        return Err(anyhow!("not on {remote_default_branch_name}"));
    }

    // fetch from remote
    let mut fetch_options = git2::FetchOptions::new();
    fetch_options.remote_callbacks(remote_auth_callbacks());
    remote_connection.remote().fetch(
        &[&remote_default_branch_name],
        Some(&mut fetch_options),
        None,
    )?;

    // analyze whether FETCH_HEAD is mergable
    let fetch_commit = repo.reference_to_annotated_commit(&repo.find_reference("FETCH_HEAD")?)?;
    let merge_analysis = repo.merge_analysis(&[&fetch_commit])?;

    // nothing to see here
    if merge_analysis.0.is_up_to_date() {
        return Ok(PullResult::UpToDate);
    }

    // we only continue if we can ff merge
    if !merge_analysis.0.is_fast_forward() {
        return Ok(PullResult::UnpullableMerge);
    }

    // update fetched branch ref to FETCH_HEAD's oid
    repo.find_reference(&remote_default_branch_name)?
        .set_target(
            fetch_commit.id(),
            format!(
                "maestro_git::sync ff {} to {}",
                remote_default_branch_name,
                fetch_commit.id()
            )
            .as_str(),
        )?;
    // update HEAD ref to fetched branch
    repo.set_head(&remote_default_branch_name)?;
    // update working tree
    repo.checkout_head(None)?;

    let commits = count_commits_from_head(&repo, &start_head_oid)?;
    let from = revparse_short_id(&repo, start_head_oid.to_string().as_str())?;
    let to = revparse_short_id(&repo, fetch_commit.id().to_string().as_str())?;
    let remote = RemoteHost::new(repo.find_remote("origin")?.url().expect("remote url"));

    Ok(PullResult::FastForward {
        commits,
        from,
        to,
        remote,
    })
}

fn revparse_short_id(repo: &git2::Repository, spec: &str) -> Result<String, anyhow::Error> {
    repo.revparse_single(spec)?
        .short_id()?
        .as_str()
        .map(String::from)
        .ok_or_else(|| anyhow!("revparsed obj short id was not utf8"))
}

fn count_commits_from_head(repo: &git2::Repository, to_object: &Oid) -> Result<u16, anyhow::Error> {
    let mut revwalk = repo.revwalk()?;
    revwalk.set_sorting(git2::Sort::TIME | git2::Sort::TOPOLOGICAL)?;
    revwalk.push_head()?;
    let mut commits: u16 = 0;
    while let Some(oid) = revwalk.next().and_then(Result::ok) {
        if &oid == to_object {
            break;
        } else {
            commits += 1
        }
    }
    Ok(commits)
}

fn get_ref_oid(repo: &git2::Repository, ref_name: &str) -> Result<Oid, anyhow::Error> {
    Ok(repo
        .reference_to_annotated_commit(&repo.find_reference(ref_name)?)?
        .id())
}

fn remote_auth_callbacks<'a>() -> RemoteCallbacks<'a> {
    let mut remote_callbacks = git2::RemoteCallbacks::new();
    remote_callbacks
        .credentials(|_, username, _| git2::Cred::ssh_key_from_agent(username.unwrap()));
    remote_callbacks
}

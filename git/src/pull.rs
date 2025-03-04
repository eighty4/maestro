use anyhow::anyhow;
use git2::{Direction, Oid, RemoteCallbacks};
use std::path::Path;

pub enum PullResult {
    DetachedHead,
    Error(String),
    /// A ff merge performed with commit count
    FastForward(u16),
    /// Fetched changes cannot be ff merged
    UnpullableMerge,
    /// Remote had nothing to pull
    UpToDate,
}

// todo rm panic and pull both default branch and current branch
pub fn pull_ff(p: &Path) -> Result<PullResult, anyhow::Error> {
    let repo = git2::Repository::open(p)?;
    let mut remote = repo.find_remote("origin")?;

    if repo.head_detached()? {
        return Ok(PullResult::DetachedHead);
    }

    // connection is required before Remote::default_branch
    // will be reused for Remote::fetch
    let mut remote_connection =
        remote.connect_auth(Direction::Fetch, Some(remote_auth_callbacks()), None)?;
    let remote_ref = remote_connection
        .default_branch()?
        .as_str()
        .map(String::from)
        .ok_or_else(|| anyhow!("default branch ref for remote `origin` was not utf8"))?;

    // will panic for now if HEAD is not default branch
    let start_head_oid = get_ref_oid(&repo, "HEAD")?;
    if start_head_oid != get_ref_oid(&repo, &remote_ref)? {
        panic!("not on {remote_ref} in repo {}", p.to_string_lossy());
    }

    // fetch from remote
    let mut fetch_options = git2::FetchOptions::new();
    fetch_options.remote_callbacks(remote_auth_callbacks());
    remote_connection
        .remote()
        .fetch(&[&remote_ref], Some(&mut fetch_options), None)?;

    // analyze whether FETCH_HEAD is mergability
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
    repo.find_reference(&remote_ref)?.set_target(
        fetch_commit.id(),
        format!(
            "maestro_git::sync ff {} to {}",
            remote_ref,
            fetch_commit.id()
        )
        .as_str(),
    )?;
    // update HEAD ref to fetched branch
    repo.set_head(&remote_ref)?;
    // update working tree
    repo.checkout_head(None)?;

    Ok(PullResult::FastForward(count_commits_from_head(
        &repo,
        &start_head_oid,
    )?))
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

use std::path::Path;

use git2::StatusOptions;

#[derive(Debug, Eq, PartialEq)]
pub enum LocalChanges {
    Clean,
    Present {
        stashes: usize,
        staged: usize,
        unstaged: usize,
        untracked: usize,
    },
}

#[derive(Debug, Eq, PartialEq)]
pub struct RepoState {
    pub changes: LocalChanges,
}

pub fn status(p: &Path) -> Result<RepoState, anyhow::Error> {
    let mut stashes = 0;
    let mut staged = 0;
    let mut unstaged = 0;
    let mut untracked = 0;
    let mut repo = git2::Repository::open(p)?;
    // todo get ref name or sha of HEAD and commit message
    // {
    //     let head_commit =
    //         repo.find_commit(repo.reference_to_annotated_commit(&repo.head()?)?.id())?;
    // }
    repo.stash_foreach(|_, _, _| {
        stashes += 1;
        true
    })?;
    {
        let mut opts = StatusOptions::new();
        // what is a submodule?
        opts.exclude_submodules(true);
        opts.include_ignored(false);
        opts.include_unmodified(false);
        opts.include_unreadable(false);
        opts.include_untracked(true);
        opts.recurse_untracked_dirs(true);
        // not refreshing because we status directly after pull
        opts.no_refresh(true);
        let statuses = repo.statuses(Some(&mut opts))?;
        for entry in statuses.iter() {
            match entry.status() {
                s if s.contains(git2::Status::INDEX_DELETED)
                    || s.contains(git2::Status::INDEX_MODIFIED)
                    || s.contains(git2::Status::INDEX_NEW)
                    || s.contains(git2::Status::INDEX_RENAMED)
                    || s.contains(git2::Status::INDEX_TYPECHANGE) =>
                {
                    staged += 1
                }
                s if s.contains(git2::Status::WT_DELETED)
                    || s.contains(git2::Status::WT_MODIFIED)
                    || s.contains(git2::Status::WT_RENAMED)
                    || s.contains(git2::Status::WT_TYPECHANGE) =>
                {
                    unstaged += 1
                }
                s if s.contains(git2::Status::WT_NEW) => untracked += 1,
                _ => {}
            }
        }
    }
    let changes = if stashes == 0 && staged == 0 && unstaged == 0 && untracked == 0 {
        LocalChanges::Clean
    } else {
        LocalChanges::Present {
            stashes,
            staged,
            unstaged,
            untracked,
        }
    };
    Ok(RepoState { changes })
}

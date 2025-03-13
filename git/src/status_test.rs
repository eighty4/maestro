use std::{fs, process::Command};

use crate::{
    status,
    testing::{assert_cmd, create_test_repo},
    LocalChanges, RepoState,
};

#[test]
fn test_status_local_changes_clean() {
    let test_repo = create_test_repo();

    let result = status(test_repo.path());
    assert!(result.is_ok());
    assert_eq!(
        result.unwrap(),
        RepoState {
            changes: LocalChanges::Clean,
        }
    );
}

#[test]
fn test_status_stashed_local_changes_present() {
    let test_repo = create_test_repo();
    fs::write(test_repo.child("README.md"), "").unwrap();
    assert_cmd(
        Command::new("git")
            .arg("stash")
            .current_dir(test_repo.path()),
    );
    let result = status(test_repo.path());
    assert!(result.is_ok());
    assert_eq!(
        result.unwrap(),
        RepoState {
            changes: LocalChanges::Present {
                stashes: 1,
                staged: 0,
                unstaged: 0,
                untracked: 0,
            },
        },
    );
}

#[test]
fn test_status_staged_local_changes_present() {
    let test_repo = create_test_repo();
    fs::write(test_repo.child("README.md"), "").unwrap();
    assert_cmd(
        Command::new("git")
            .arg("add")
            .arg(test_repo.child("README.md"))
            .current_dir(test_repo.path()),
    );
    let result = status(test_repo.path());
    assert!(result.is_ok());
    assert_eq!(
        result.unwrap(),
        RepoState {
            changes: LocalChanges::Present {
                stashes: 0,
                staged: 1,
                unstaged: 0,
                untracked: 0,
            },
        },
    );
}

#[test]
fn test_status_unstaged_local_changes_present() {
    let test_repo = create_test_repo();
    fs::write(test_repo.child("README.md"), "").unwrap();
    let result = status(test_repo.path());
    assert!(result.is_ok());
    assert_eq!(
        result.unwrap(),
        RepoState {
            changes: LocalChanges::Present {
                stashes: 0,
                staged: 0,
                unstaged: 1,
                untracked: 0,
            },
        },
    );
}

#[test]
fn test_status_untracked_local_changes_present() {
    let test_repo = create_test_repo();
    fs::write(test_repo.child("build.zig"), "").unwrap();
    let result = status(test_repo.path());
    assert!(result.is_ok());
    assert_eq!(
        result.unwrap(),
        RepoState {
            changes: LocalChanges::Present {
                stashes: 0,
                staged: 0,
                unstaged: 0,
                untracked: 1,
            },
        },
    );
}

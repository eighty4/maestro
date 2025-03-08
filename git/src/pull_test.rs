use std::{
    fs::{copy, create_dir, read_dir, remove_dir_all, write},
    io,
    path::PathBuf,
    process::Command,
};

use temp_dir::TempDir;
use tokio::sync::OnceCell;

use crate::{PullResult, pull::pull_ff};

fn create_test_repo() -> TempDir {
    let temp_dir = TempDir::new().unwrap();
    Command::new("git")
        .arg("clone")
        .arg("https://github.com/eighty4/pear.ng")
        .arg(".")
        .current_dir(temp_dir.path());
    temp_dir
}

#[test]
fn pull_ff_detached_head() {
    let test_repo = create_test_repo();
    Command::new("git")
        .arg("checkout")
        .arg("HEAD~1")
        .current_dir(test_repo.path())
        .output()
        .unwrap();

    let result = pull_ff(test_repo.path());
    assert!(result.is_ok());
    assert_eq!(result.unwrap(), PullResult::DetachedHead);
}

#[test]
fn pull_ff_pulled() {
    let test_repo = create_test_repo();
    Command::new("git")
        .arg("reset")
        .arg("--hard")
        .arg("HEAD~2")
        .current_dir(test_repo.path())
        .output()
        .unwrap();

    let result = pull_ff(test_repo.path());
    assert!(result.is_ok());
    assert_eq!(
        result.unwrap(),
        PullResult::FastForward {
            branch: "main".to_string(),
            commits: 2,
            from: "".to_string(),
            to: "".to_string()
        }
    );
}

#[test]
fn pull_ff_unpullable() {
    let test_repo = create_test_repo();
    Command::new("git")
        .arg("reset")
        .arg("--hard")
        .arg("HEAD~2")
        .current_dir(test_repo.path())
        .output()
        .unwrap();
    let conflict_file = "www/package.json";
    write(
        test_repo.join(&conflict_file),
        "If Jimmy eats world, and no one cares",
    )
    .unwrap();
    Command::new("git")
        .arg("add")
        .arg(conflict_file)
        .current_dir(test_repo.path())
        .output()
        .unwrap();

    dbg!("wtf");

    let result = pull_ff(test_repo.path());
    assert!(result.is_ok());
    assert_eq!(result.unwrap(), PullResult::UnpullableMerge);
}

#[test]
fn pull_ff_up_to_date() {
    let test_repo = create_test_repo();
    let result = pull_ff(test_repo.path());
    assert!(result.is_ok());
    assert_eq!(result.unwrap(), PullResult::UpToDate);
}

fn copy_recursive(from_dir: &PathBuf, to_dir: &PathBuf) -> io::Result<()> {
    debug_assert!(from_dir.is_dir());
    if !to_dir.is_dir() {
        create_dir(to_dir).unwrap();
    }
    for dir_entry in read_dir(from_dir).unwrap() {
        let from_path = dir_entry.unwrap().path();
        let to_path = &to_dir.join(from_path.file_name().unwrap());
        if from_path.is_dir() {
            copy_recursive(&from_path, &to_path).unwrap();
        } else {
            copy(&from_path, &to_path).unwrap();
        }
    }
    Ok(())
}

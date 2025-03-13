use std::{
    fs::{read_to_string, write},
    process::Command,
};

use crate::{
    pull::pull_ff,
    testing::{assert_cmd, create_test_repo},
    PullResult, RemoteHost,
};

#[test]
fn pull_ff_detached_head() {
    let test_repo = create_test_repo();
    assert_cmd(
        Command::new("git")
            .arg("checkout")
            .arg("HEAD~1")
            .current_dir(test_repo.path()),
    );
    assert_cmd(
        Command::new("git")
            .arg("checkout")
            .arg("HEAD~1")
            .current_dir(test_repo.path()),
    );

    let result = pull_ff(test_repo.path());
    assert!(result.is_ok());
    assert_eq!(result.unwrap(), PullResult::DetachedHead);
}

#[test]
fn pull_ff_pulled() {
    let test_repo = create_test_repo();
    assert_cmd(
        Command::new("git")
            .arg("reset")
            .arg("--hard")
            .arg("HEAD~2")
            .current_dir(test_repo.path()),
    );

    let result = pull_ff(test_repo.path());
    assert!(result.is_ok());
    assert_eq!(
        result.unwrap(),
        PullResult::FastForward {
            commits: 2,
            from: "e303cea".to_string(),
            to: "fe98a80".to_string(),
            remote: RemoteHost::GitHub {
                owner: "eighty4".to_string(),
                name: "pear.ng".to_string()
            }
        }
    );
}

#[test]
fn pull_ff_unpullable() {
    let test_repo = create_test_repo();
    assert_cmd(
        Command::new("git")
            .arg("reset")
            .arg("--hard")
            .arg("HEAD~2")
            .current_dir(test_repo.path()),
    );
    let conflict_file = "desktop/bin/pear_cli/pear_cli.cc";
    let contents = read_to_string(test_repo.path().join(conflict_file)).unwrap();
    write(
        test_repo.path().join(&conflict_file),
        format!("#include \"net_udp/udp.h\"\n{contents}"),
    )
    .unwrap();
    assert_cmd(
        Command::new("git")
            .arg("add")
            .arg(conflict_file)
            .current_dir(test_repo.path()),
    );
    assert_cmd(
        Command::new("git")
            .arg("commit")
            .arg("-m")
            .arg("conflict")
            .current_dir(test_repo.path()),
    );

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

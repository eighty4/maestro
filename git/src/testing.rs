use std::process::Command;

use temp_dir::TempDir;

pub fn create_test_repo() -> TempDir {
    let temp_dir = TempDir::new().unwrap();
    assert!(Command::new("git")
        .arg("clone")
        .arg("https://github.com/eighty4/pear.ng")
        .arg(".")
        .current_dir(temp_dir.path())
        .output()
        .unwrap()
        .status
        .success());
    temp_dir
}

pub fn assert_cmd(cmd: &mut Command) {
    let output = cmd.output().unwrap();
    assert!(output.status.success());
}

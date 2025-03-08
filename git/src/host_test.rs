use crate::RemoteHost;

#[test]
fn parse_github_ssh_uri() {
    let remote = RemoteHost::new("git@github.com:eighty4/pear.ng.git");
    assert_eq!(
        remote,
        RemoteHost::GitHub {
            owner: "eighty4".to_string(),
            name: "pear.ng".to_string(),
        }
    );
}

#[test]
fn parse_github_ssh_uri_wo_git_ext() {
    let remote = RemoteHost::new("git@github.com:eighty4/pear.ng");
    assert_eq!(
        remote,
        RemoteHost::GitHub {
            owner: "eighty4".to_string(),
            name: "pear.ng".to_string(),
        }
    );
}

#[test]
fn parse_github_https_uri() {
    let remote = RemoteHost::new("https://github.com/eighty4/pear.ng.git");
    assert_eq!(
        remote,
        RemoteHost::GitHub {
            owner: "eighty4".to_string(),
            name: "pear.ng".to_string(),
        }
    );
}

#[test]
fn parse_github_https_uri_wo_git_ext() {
    let remote = RemoteHost::new("https://github.com/eighty4/pear.ng");
    assert_eq!(
        remote,
        RemoteHost::GitHub {
            owner: "eighty4".to_string(),
            name: "pear.ng".to_string(),
        }
    );
}

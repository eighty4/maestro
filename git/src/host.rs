#[derive(Debug, Eq, PartialEq)]
pub enum RemoteHost {
    GitHub { owner: String, name: String },
    Other,
}

impl RemoteHost {
    pub fn new(url: &str) -> Self {
        match url
            .strip_prefix("git@github.com:")
            .map(|ssh_url| ssh_url.strip_suffix(".git").unwrap_or(ssh_url))
            .or_else(|| {
                url.strip_prefix("https://github.com/")
                    .map(|https_url| https_url.strip_suffix(".git").unwrap_or(https_url))
            }) {
            Some(path) => {
                let path_parts: Vec<&str> = path.split("/").take(2).collect();
                Self::GitHub {
                    owner: path_parts[0].to_string(),
                    name: path_parts[1].to_string(),
                }
            }
            None => Self::Other,
        }
    }

    pub fn compare_url(&self, from: &str, to: &str) -> String {
        debug_assert!(self.has_compare_url());
        match self {
            RemoteHost::GitHub { owner, name } => {
                format!("https://github.com/{owner}/{name}/compare/{from}..{to}")
            }
            RemoteHost::Other => panic!(),
        }
    }

    pub fn has_compare_url(&self) -> bool {
        match self {
            RemoteHost::GitHub { .. } => true,
            RemoteHost::Other => false,
        }
    }
}

use maestro_git::{LocalChanges, PullResult, SyncKind, SyncResult};

pub enum SyncIndicator {
    Clean,
    LocalChanges,
    Error,
}

impl From<&SyncResult> for SyncIndicator {
    fn from(sync_result: &SyncResult) -> Self {
        let SyncKind::Pull(pull_result) = &sync_result.kind;
        match pull_result {
            PullResult::FastForward { .. } | PullResult::UpToDate => {
                match sync_result.state.changes {
                    LocalChanges::Clean => SyncIndicator::Clean,
                    LocalChanges::Present { .. } => SyncIndicator::LocalChanges,
                }
            }
            _ => SyncIndicator::Error,
        }
    }
}

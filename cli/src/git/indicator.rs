use maestro_git::{LocalChanges, PullResult, SyncKind, SyncResult};

pub enum SyncIndicator {
    Clean,
    Error,
    LocalChanges,
    Noop,
}

impl From<&SyncResult> for SyncIndicator {
    fn from(sync_result: &SyncResult) -> Self {
        match &sync_result.kind {
            SyncKind::Pull(pull_result) => match pull_result {
                PullResult::FastForward { .. } | PullResult::UpToDate => {
                    match sync_result.state.changes {
                        LocalChanges::Clean => SyncIndicator::Clean,
                        LocalChanges::Present { .. } => SyncIndicator::LocalChanges,
                    }
                }
                _ => SyncIndicator::Error,
            },
            SyncKind::Skipped => SyncIndicator::Noop,
        }
    }
}

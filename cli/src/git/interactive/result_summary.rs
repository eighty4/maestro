use maestro_git::{PullResult, SyncKind, SyncResult};
use ratatui::{
    buffer::Buffer,
    layout::{Constraint, Layout, Rect},
    style::Stylize,
    text::Line,
    widgets::{Paragraph, Widget},
};

use super::LayoutMode;

// keep result sizes odd before compact
pub const RESULT_SIZE_UNFOCUSED: u16 = 3;
pub const RESULT_SIZE_FOCUSED: u16 = RESULT_SIZE_UNFOCUSED;

pub const RESULT_SIZE_COMPACT: u16 = 2;

pub fn placeholder(area: Rect, buf: &mut Buffer) {
    Paragraph::new("...".dark_gray()).render(area, buf)
}

/// Printing an in progress indicator or completed result summary
pub struct SyncResultWidget<'a> {
    focused: bool,
    layout: &'a LayoutMode,
    result: &'a SyncResult,
}

impl<'a> SyncResultWidget<'a> {
    pub fn new(focused: bool, layout: &'a LayoutMode, result: &'a SyncResult) -> Self {
        Self {
            focused,
            layout,
            result,
        }
    }
}

impl Widget for SyncResultWidget<'_> {
    fn render(self, area: Rect, buf: &mut Buffer)
    where
        Self: Sized,
    {
        debug_assert!(
            area.height
                == if self.layout == &LayoutMode::Compact {
                    RESULT_SIZE_COMPACT
                } else if self.focused {
                    RESULT_SIZE_FOCUSED
                } else {
                    RESULT_SIZE_UNFOCUSED
                }
        );
        let indent = "    ";
        let cursor = if self.focused { "●" } else { "○" }.cyan();
        let (line1, line2) = if self.layout == &LayoutMode::Compact {
            let [line1, line2] =
                Layout::vertical([Constraint::Length(1), Constraint::Length(1)]).areas(area);
            (line1, line2)
        } else {
            let [line1, line2] = Layout::vertical([Constraint::Length(1), Constraint::Length(1)])
                .spacing(1)
                .areas(area);
            (line1, line2)
        };
        Paragraph::new(Line::from(vec![
            indent.into(),
            cursor,
            format!(" {}", self.result.repo.label).into(),
        ]))
        .render(line1, buf);
        let SyncKind::Pull(pull_result) = &self.result.kind;

        Paragraph::new(Line::from(vec![
            indent.into(),
            indent.into(),
            match pull_result {
                PullResult::FastForward(_) | PullResult::UpToDate => "✔".green(),
                _ => "✗".red(),
            },
            " ".into(),
            match pull_result {
                PullResult::DetachedHead => "HEAD is detached.".to_string(),
                PullResult::Error(err_msg) => format!("Failed pulling: {err_msg}."),
                PullResult::FastForward(commits) => format!("Pulled {commits} commits."),
                PullResult::UnpullableMerge => {
                    "Unable to ff merge changes from remote.".to_string()
                }
                PullResult::UpToDate => "Already up to date.".to_string(),
            }
            .into(),
        ]))
        .render(line2, buf);
    }
}

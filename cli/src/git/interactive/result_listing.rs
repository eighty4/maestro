use std::cmp::min;
use std::ops::Range;

use super::{
    result_summary::{RESULT_SIZE_COMPACT, RESULT_SIZE_FOCUSED, RESULT_SIZE_UNFOCUSED},
    InterfaceState, LayoutMode, Size, FOOTER_HEIGHT, HEADER_HEIGHT,
};

#[derive(Clone, Default)]
pub struct PagingResults {
    pub repos_count: usize,
    pub page_size: usize,
    pub page_start_index: usize,
}

impl PagingResults {
    pub fn calc(cursor: &usize, repos_count: usize, layout: &LayoutMode, size: &Size) -> Self {
        let content_h = size.h - HEADER_HEIGHT - FOOTER_HEIGHT;
        let page_size: usize = match layout {
            LayoutMode::Break => panic!(),
            LayoutMode::Comfortable => {
                (content_h - RESULT_SIZE_FOCUSED) / (RESULT_SIZE_UNFOCUSED + 1) + 1
            }
            LayoutMode::Compact => {
                (content_h - RESULT_SIZE_COMPACT) / (RESULT_SIZE_COMPACT + 1) + 1
            }
        }
        .into();
        let page_start_index = (cursor / page_size) * page_size;
        Self {
            page_start_index,
            page_size,
            repos_count,
        }
    }

    pub fn can_next_page(&self) -> bool {
        self.page_start_index + self.page_size < self.repos_count
    }

    pub fn can_next_repo(&self, cursor: &usize) -> bool {
        cursor + 1 < self.current_page_range().end
    }

    pub fn can_prev_page(&self) -> bool {
        self.page_start_index + 1 > self.page_size
    }

    pub fn current_page(&self) -> usize {
        self.page_start_index / self.page_size
    }

    pub fn current_page_size(&self) -> usize {
        min(self.repos_count - self.page_start_index, self.page_size)
    }

    pub fn current_page_range(&self) -> Range<usize> {
        self.page_start_index..min(self.page_start_index + self.page_size, self.repos_count)
    }

    pub fn cursor_within_page(&self, cursor: &usize) -> usize {
        cursor % self.page_size
    }

    pub fn page_count(&self) -> usize {
        let divide = self.repos_count / self.page_size;
        if self.repos_count % self.page_size == 0 {
            divide
        } else {
            divide + 1
        }
    }

    pub fn next_page(&self, cursor: &usize) -> InterfaceState {
        debug_assert!(self.can_next_page());
        InterfaceState::ResultListing {
            cursor: min(cursor + self.page_size, self.repos_count - 1),
            page: PagingResults {
                repos_count: self.repos_count,
                page_start_index: self.page_start_index + self.page_size,
                page_size: self.page_size,
            },
        }
    }

    pub fn prev_page(&self, cursor: &usize) -> InterfaceState {
        debug_assert!(self.can_prev_page());
        InterfaceState::ResultListing {
            cursor: cursor - self.page_size,
            page: PagingResults {
                repos_count: self.repos_count,
                page_start_index: self.page_start_index - self.page_size,
                page_size: self.page_size,
            },
        }
    }

    pub fn next_repo(&self, cursor: &usize) -> InterfaceState {
        debug_assert!(self.can_next_repo(cursor));
        InterfaceState::ResultListing {
            cursor: cursor + 1,
            page: self.clone(),
        }
    }

    pub fn prev_repo(&self, cursor: &usize) -> InterfaceState {
        InterfaceState::ResultListing {
            cursor: cursor - 1,
            page: self.clone(),
        }
    }
}

#[cfg(test)]
mod test {
    use crate::git::interactive::Size;

    use super::*;

    #[test]
    fn comfortable_cursor_0() {
        let pr = PagingResults::calc(&0, 6, &LayoutMode::Comfortable, &Size { w: 20, h: 25 });
        assert_eq!(pr.page_size, 4);
        assert_eq!(pr.page_start_index, 0);
        assert_eq!(pr.page_count(), 2);
        assert_eq!(pr.current_page_size(), 4);
        assert_eq!(pr.current_page_range(), 0..4);
    }

    #[test]
    fn comfortable_partial_last_page() {
        let pr = PagingResults::calc(&5, 6, &LayoutMode::Comfortable, &Size { w: 20, h: 25 });
        assert_eq!(pr.page_size, 4);
        assert_eq!(pr.page_start_index, 4);
        assert_eq!(pr.page_count(), 2);
        assert_eq!(pr.current_page_size(), 2);
        assert_eq!(pr.current_page_range(), 4..6);
    }

    #[test]
    fn full_last_page() {
        let pr = PagingResults::calc(&5, 6, &LayoutMode::Compact, &Size { w: 20, h: 15 });
        assert_eq!(pr.page_size, 2);
        assert_eq!(pr.page_start_index, 4);
        assert_eq!(pr.page_count(), 3);
        assert_eq!(pr.current_page_size(), 2);
        assert_eq!(pr.current_page_range(), 4..6);
    }
}

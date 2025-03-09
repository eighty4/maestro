mod result_listing;
mod result_summary;

use anyhow::anyhow;
use crossterm::event::{Event, EventStream, KeyCode, KeyEvent, KeyModifiers};
use futures::{future::FutureExt, StreamExt};
use maestro_git::{PullResult, Sync, SyncKind, SyncResult};
use ratatui::{
    buffer::Buffer,
    layout::{Constraint, Flex, Layout, Rect},
    style::Stylize,
    text::Line,
    widgets::{Paragraph, Widget},
    Frame,
};
use result_listing::PagingResults;
use result_summary::{
    placeholder, SyncResultWidget, RESULT_SIZE_COMPACT, RESULT_SIZE_FOCUSED, RESULT_SIZE_UNFOCUSED,
};

// header/footer heights include symmetric vertical padding of 1
const HEADER_HEIGHT: u16 = 3;
const FOOTER_HEIGHT: u16 = 5;

#[derive(Clone, Default, PartialEq)]
enum LayoutMode {
    Break,
    #[default]
    Comfortable,
    Compact,
}

// todo ls commit history
enum InterfaceState {
    ResultListing { cursor: usize, page: PagingResults },
}

#[derive(Default)]
struct Size {
    w: u16,
    h: u16,
}

pub struct InteractiveSync {
    /// tui frame size (w, h)
    area: Size,
    /// crossterm events as a stream
    events: EventStream,
    /// switch to exit tui loop
    exit: bool,
    /// syncing has finished
    finished: bool,
    /// how to render ui based on terminal size
    layout: LayoutMode,
    /// received sync results
    results: Vec<SyncResult>,
    /// active workspace syncing
    syncing: Sync,
    /// state of interface
    state: InterfaceState,
    /// path to workspace root
    workspace: String,
}

// ctor and init
impl InteractiveSync {
    pub fn new(workspace_root: String, syncing: Sync) -> Self {
        assert!(!syncing.repos.is_empty());
        Self {
            area: Size::default(),
            events: EventStream::new(),
            exit: false,
            finished: false,
            layout: LayoutMode::default(),
            results: Vec::new(),
            state: InterfaceState::ResultListing {
                cursor: 0,
                page: PagingResults::default(),
            },
            syncing,
            workspace: workspace_root,
        }
    }

    pub async fn run(&mut self) -> Result<(), anyhow::Error> {
        let ret = self.event_loop().await;
        ratatui::restore();
        ret
    }
}

// event loop and state mgmt
impl InteractiveSync {
    async fn event_loop(&mut self) -> Result<(), anyhow::Error> {
        let mut terminal = ratatui::init();
        while !self.exit {
            terminal.draw(|frame| {
                let frame_area = frame.area();
                self.update_frame_size(frame_area.width, frame_area.height);
                self.draw(frame);
            })?;
            self.handle_events().await?;
        }
        Ok(())
    }

    fn draw(&self, frame: &mut Frame) {
        frame.render_widget(self, frame.area());
    }

    async fn handle_events(&mut self) -> Result<(), anyhow::Error> {
        let event = self.events.next().fuse();
        tokio::select! {
            maybe_sync_result = self.syncing.rx.recv() => {
                match maybe_sync_result {
                    Some(sync_result) => self.handle_sync_result(sync_result),
                    None => self.finished = true,
                }
            }
            maybe_event = event => {
                match maybe_event {
                    Some(Ok(Event::Resize(w, h))) => self.update_frame_size(w, h),
                    Some(Ok(Event::Key(key_event))) => self.handle_key_event(key_event),
                    Some(Err(err)) => return Err(anyhow!("tui event error: {err}")),
                    _ => {},
                }
            }
        }
        Ok(())
    }

    fn update_frame_size(&mut self, w: u16, h: u16) {
        if self.area.w != w || self.area.h != h {
            self.update_layout(h);
            if self.layout == LayoutMode::Break {
                return;
            }
            self.area = Size { w, h };
            self.state = match &self.state {
                InterfaceState::ResultListing { cursor, .. } => InterfaceState::ResultListing {
                    cursor: *cursor,
                    page: PagingResults::calc(
                        cursor,
                        self.syncing.repos.len(),
                        &self.layout,
                        &self.area,
                    ),
                },
            };
        }
    }

    fn update_layout(&mut self, h: u16) {
        // if 2 results can't fit compactly, break
        const MIN_HEIGHT_BEFORE_BREAK: u16 =
            RESULT_SIZE_COMPACT * 2 + HEADER_HEIGHT + FOOTER_HEIGHT;
        // if 3 results can't fit comfortably, switch to compact
        const MIN_HEIGHT_BEFORE_COMPACT: u16 =
            RESULT_SIZE_FOCUSED + (RESULT_SIZE_UNFOCUSED * 2) + HEADER_HEIGHT + FOOTER_HEIGHT;

        self.layout = if h <= MIN_HEIGHT_BEFORE_BREAK {
            LayoutMode::Break
        } else if h <= MIN_HEIGHT_BEFORE_COMPACT {
            LayoutMode::Compact
        } else {
            LayoutMode::Comfortable
        };
    }

    fn handle_sync_result(&mut self, sync_result: SyncResult) {
        self.results.push(sync_result);
    }

    fn handle_key_event(&mut self, key_event: KeyEvent) {
        match key_event {
            KeyEvent {
                code: KeyCode::Char('c'),
                modifiers: KeyModifiers::CONTROL,
                ..
            } => {
                self.exit = true;
            }
            _ => match &self.state {
                InterfaceState::ResultListing { cursor, page } => match key_event {
                    KeyEvent {
                        code: KeyCode::Char('d'),
                        ..
                    } => {
                        self.maybe_open_compare_url(cursor);
                    }
                    KeyEvent {
                        code: KeyCode::Up, ..
                    } => {
                        if cursor > &page.page_start_index {
                            self.state = page.prev_repo(cursor);
                        }
                    }
                    KeyEvent {
                        code: KeyCode::Down,
                        ..
                    } => {
                        if page.can_next_repo(cursor) {
                            self.state = page.next_repo(cursor);
                        }
                    }
                    KeyEvent {
                        code: KeyCode::Left,
                        ..
                    } => {
                        if page.can_prev_page() {
                            self.state = page.prev_page(cursor);
                        }
                    }
                    KeyEvent {
                        code: KeyCode::Right,
                        ..
                    } => {
                        if page.can_next_page() {
                            self.state = page.next_page(cursor);
                        }
                    }
                    _ => {}
                },
            },
        }
    }

    fn maybe_open_compare_url(&self, cursor: &usize) {
        if let SyncKind::Pull(PullResult::FastForward {
            remote, from, to, ..
        }) = &self.results[*cursor].kind
        {
            if remote.has_compare_url() {
                if let Err(err) = open::that_detached(remote.compare_url(from, to)) {
                    ratatui::restore();
                    eprintln!("{err}");
                    std::process::exit(1);
                }
            }
        }
    }
}

impl Widget for &InteractiveSync {
    fn render(self, area: Rect, buf: &mut Buffer)
    where
        Self: Sized,
    {
        if self.layout == LayoutMode::Break {
            Paragraph::new("Make the terminal taller or run without --interactive")
                .render(area, buf);
            return;
        }
        let [centered_render_area] = Layout::horizontal([Constraint::Percentage(60)])
            .flex(Flex::Center)
            .areas(area);
        let [header_area, content_area, footer_area] = Layout::vertical([
            Constraint::Length(HEADER_HEIGHT),
            Constraint::Fill(1),
            Constraint::Length(FOOTER_HEIGHT),
        ])
        .areas(centered_render_area);
        self.render_header(header_area, buf);
        let InterfaceState::ResultListing { cursor, page } = &self.state;
        self.render_result_listing(cursor, page, content_area, buf);
        self.render_result_listing_footer(cursor, page, footer_area, buf);
    }
}

// rendering sections
impl InteractiveSync {
    fn render_header(&self, area: Rect, buf: &mut Buffer) {
        const SYNC_CMD: &str = "maestro git";
        let [header_cmd_area, header_ws_area] = Layout::horizontal([
            Constraint::Length(SYNC_CMD.len() as u16),
            Constraint::Length(self.workspace.len() as u16),
        ])
        .flex(Flex::SpaceBetween)
        .vertical_margin(1)
        .areas(area);
        Paragraph::new(SYNC_CMD).render(header_cmd_area, buf);
        Paragraph::new(self.workspace.as_str().dark_gray()).render(header_ws_area, buf);
    }

    fn render_result_listing_footer(
        &self,
        cursor: &usize,
        page: &PagingResults,
        padded_area: Rect,
        buf: &mut Buffer,
    ) {
        debug_assert!(padded_area.height == FOOTER_HEIGHT);
        let [line1, line2] = Layout::vertical([Constraint::Length(1), Constraint::Length(1)])
            .flex(Flex::SpaceAround)
            .areas(padded_area);
        let [paging_area, shortcuts_area] =
            Layout::horizontal([Constraint::Fill(1), Constraint::Fill(1)]).areas(line1);
        if page.page_count() > 1 {
            Paragraph::new(format!(
                "Page {} of {}",
                page.current_page(),
                page.page_count()
            ))
            .render(paging_area, buf);
        }
        if self.has_compare_url(cursor) {
            Paragraph::new(Line::from(vec![
                "d".cyan().bold(),
                " compare changes".into(),
            ]))
            .right_aligned()
            .render(shortcuts_area, buf);
        }
        if !self.finished {
            Paragraph::new(format!(
                "Syncing {} repositories",
                self.syncing.repos.len() - self.results.len()
            ))
            .render(line2, buf);
        }
    }

    fn has_compare_url(&self, cursor: &usize) -> bool {
        if self.results.is_empty() || *cursor > self.results.len() - 1 {
            false
        } else if let SyncKind::Pull(PullResult::FastForward { remote, .. }) =
            &self.results[*cursor].kind
        {
            remote.has_compare_url()
        } else {
            false
        }
    }

    fn render_result_listing(
        &self,
        cursor: &usize,
        paging: &PagingResults,
        area: Rect,
        buf: &mut Buffer,
    ) {
        // calculate areas by layout mode
        let result_areas = match &self.layout {
            LayoutMode::Break => panic!(),
            LayoutMode::Comfortable => {
                let mut v_constraints = Vec::with_capacity(paging.page_size);
                let page_cursor = paging.cursor_within_page(cursor);
                for page_i in 0..paging.current_page_size() {
                    if page_i == page_cursor {
                        v_constraints.push(Constraint::Length(RESULT_SIZE_FOCUSED));
                    } else {
                        v_constraints.push(Constraint::Length(RESULT_SIZE_UNFOCUSED));
                    }
                }
                Layout::vertical(v_constraints)
                    // space around if less than a full page
                    .flex(if paging.repos_count < paging.page_size {
                        Flex::SpaceAround
                    } else {
                        Flex::Start
                    })
                    .spacing(1)
                    .split(area)
            }
            LayoutMode::Compact => {
                let mut v_constraints = Vec::with_capacity(paging.page_size);
                for _ in 0..paging.page_size {
                    v_constraints.push(Constraint::Length(RESULT_SIZE_COMPACT));
                }
                Layout::vertical(v_constraints).split(area)
            }
        };
        for (range_i, results_i) in paging.current_page_range().enumerate() {
            let result_area = result_areas[range_i];
            if self.results.len() <= results_i {
                placeholder(result_area, buf);
            } else if !self.syncing.repos.is_empty() {
                SyncResultWidget::new(&results_i == cursor, &self.layout, &self.results[results_i])
                    .render(result_area, buf);
            }
        }
    }
}

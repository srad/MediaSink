use ratatui::text::{Line, Span};

const LIMITS: [usize; 6] = [25, 50, 100, 200, 500, 1000];

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum LatestSortColumn {
    CreatedAt,
    Size,
    Duration,
}

impl LatestSortColumn {
    pub fn next(self) -> Self {
        match self {
            Self::CreatedAt => Self::Size,
            Self::Size => Self::Duration,
            Self::Duration => Self::CreatedAt,
        }
    }

    pub fn previous(self) -> Self {
        match self {
            Self::CreatedAt => Self::Duration,
            Self::Size => Self::CreatedAt,
            Self::Duration => Self::Size,
        }
    }

    pub const fn label(self) -> &'static str {
        match self {
            Self::CreatedAt => "Created at",
            Self::Size => "Filesize",
            Self::Duration => "Video duration",
        }
    }

    pub const fn request_value(self) -> &'static str {
        match self {
            Self::CreatedAt => "created_at",
            Self::Size => "size",
            Self::Duration => "duration",
        }
    }
}

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum LatestSortOrder {
    Asc,
    Desc,
}

impl LatestSortOrder {
    pub fn toggle(self) -> Self {
        match self {
            Self::Asc => Self::Desc,
            Self::Desc => Self::Asc,
        }
    }

    pub const fn label(self) -> &'static str {
        match self {
            Self::Asc => "asc",
            Self::Desc => "desc",
        }
    }
}

#[derive(Debug, Clone, Eq, PartialEq)]
pub struct LatestFilter {
    limit_index: usize,
    pub sort_column: LatestSortColumn,
    pub sort_order: LatestSortOrder,
}

impl Default for LatestFilter {
    fn default() -> Self {
        Self {
            limit_index: 2,
            sort_column: LatestSortColumn::CreatedAt,
            sort_order: LatestSortOrder::Desc,
        }
    }
}

impl LatestFilter {
    pub fn limit(&self) -> usize {
        LIMITS[self.limit_index]
    }

    pub fn next_column(&mut self) {
        self.sort_column = self.sort_column.next();
    }

    pub fn previous_column(&mut self) {
        self.sort_column = self.sort_column.previous();
    }

    pub fn toggle_order(&mut self) {
        self.sort_order = self.sort_order.toggle();
    }

    pub fn next_limit(&mut self) {
        self.limit_index = (self.limit_index + 1) % LIMITS.len();
    }

    pub fn previous_limit(&mut self) {
        self.limit_index = if self.limit_index == 0 {
            LIMITS.len() - 1
        } else {
            self.limit_index - 1
        };
    }

    pub fn reset(&mut self) {
        *self = Self::default();
    }

    pub fn bar_lines(&self) -> Vec<Line<'static>> {
        vec![
            Line::from(vec![
                Span::raw("Column "),
                Span::raw(self.sort_column.label()),
                Span::raw("  Order "),
                Span::raw(self.sort_order.label()),
                Span::raw("  Limit "),
                Span::raw(self.limit().to_string()),
            ]),
            Line::from("[ ] column  o order  -/+ limit  0 reset"),
        ]
    }
}

#[cfg(test)]
mod tests {
    use super::{LatestFilter, LatestSortColumn, LatestSortOrder};

    #[test]
    fn default_matches_frontend_filter_view() {
        let filter = LatestFilter::default();

        assert_eq!(filter.sort_column, LatestSortColumn::CreatedAt);
        assert_eq!(filter.sort_order, LatestSortOrder::Desc);
        assert_eq!(filter.limit(), 100);
    }

    #[test]
    fn cycles_limits_and_resets() {
        let mut filter = LatestFilter::default();
        filter.next_limit();
        filter.next_limit();
        filter.toggle_order();
        filter.next_column();

        filter.reset();

        assert_eq!(filter, LatestFilter::default());
    }
}

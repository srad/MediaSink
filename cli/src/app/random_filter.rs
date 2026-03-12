use ratatui::text::{Line, Span};

const LIMITS: [usize; 6] = [25, 50, 100, 200, 500, 1000];

#[derive(Debug, Clone, Eq, PartialEq)]
pub struct RandomFilter {
    limit_index: usize,
}

impl Default for RandomFilter {
    fn default() -> Self {
        Self { limit_index: 0 }
    }
}

impl RandomFilter {
    pub fn limit(&self) -> usize {
        LIMITS[self.limit_index]
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
                Span::raw("Limit "),
                Span::raw(self.limit().to_string()),
                Span::raw("  Refresh "),
                Span::raw("g"),
            ]),
            Line::from("-/+ limit  0 reset  g refresh"),
        ]
    }
}

#[cfg(test)]
mod tests {
    use super::RandomFilter;

    #[test]
    fn default_matches_frontend_random_view() {
        let filter = RandomFilter::default();

        assert_eq!(filter.limit(), 25);
    }

    #[test]
    fn cycles_limits_and_resets() {
        let mut filter = RandomFilter::default();
        filter.next_limit();
        filter.next_limit();

        assert_eq!(filter.limit(), 100);

        filter.previous_limit();
        assert_eq!(filter.limit(), 50);

        filter.reset();
        assert_eq!(filter.limit(), 25);
    }
}

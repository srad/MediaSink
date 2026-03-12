use std::cmp::min;

pub fn clamp_index(index: usize, len: usize) -> usize {
    if len == 0 { 0 } else { min(index, len - 1) }
}

pub fn visible_window(selected: usize, len: usize, capacity: usize) -> (usize, usize) {
    if len == 0 {
        return (0, 0);
    }
    if len <= capacity {
        return (0, len);
    }

    let max_start = len.saturating_sub(capacity);
    let start = selected
        .saturating_sub(capacity.saturating_sub(1))
        .min(max_start);
    let end = (start + capacity).min(len);
    (start, end)
}

#[cfg(test)]
mod tests {
    use super::{clamp_index, visible_window};

    #[test]
    fn clamp_index_stays_in_bounds() {
        assert_eq!(clamp_index(0, 0), 0);
        assert_eq!(clamp_index(5, 3), 2);
        assert_eq!(clamp_index(1, 3), 1);
    }

    #[test]
    fn visible_window_tracks_bottom_edge_before_scrolling() {
        assert_eq!(visible_window(0, 10, 4), (0, 4));
        assert_eq!(visible_window(1, 10, 4), (0, 4));
        assert_eq!(visible_window(2, 10, 4), (0, 4));
        assert_eq!(visible_window(3, 10, 4), (0, 4));
        assert_eq!(visible_window(4, 10, 4), (1, 5));
        assert_eq!(visible_window(5, 10, 4), (2, 6));
    }

    #[test]
    fn visible_window_clamps_to_last_page() {
        assert_eq!(visible_window(8, 10, 4), (5, 9));
        assert_eq!(visible_window(9, 10, 4), (6, 10));
    }
}

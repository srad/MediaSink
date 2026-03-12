#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum View {
    Streams,
    Channels,
    Channel,
    Latest,
    Random,
    Favourites,
    Similarity,
    Admin,
    Info,
    Processes,
    Monitoring,
    Jobs,
    Logs,
}

impl View {
    pub const fn primary() -> &'static [View] {
        &[
            View::Streams,
            View::Channels,
            View::Latest,
            View::Random,
            View::Favourites,
            View::Similarity,
        ]
    }

    pub const fn label(self) -> &'static str {
        match self {
            Self::Streams => "Streams",
            Self::Channels => "Channels",
            Self::Channel => "Channel",
            Self::Latest => "Latest",
            Self::Random => "Random",
            Self::Favourites => "Favourites",
            Self::Similarity => "Similarity",
            Self::Admin => "Admin",
            Self::Info => "Info",
            Self::Processes => "Processes",
            Self::Monitoring => "Monitoring",
            Self::Jobs => "Jobs",
            Self::Logs => "Logs",
        }
    }

    pub const fn is_primary(self) -> bool {
        matches!(
            self,
            Self::Streams
                | Self::Channels
                | Self::Latest
                | Self::Random
                | Self::Favourites
                | Self::Similarity
        )
    }

    pub const fn auto_refresh(self) -> bool {
        matches!(
            self,
            Self::Channel
                | Self::Admin
                | Self::Info
                | Self::Processes
                | Self::Monitoring
                | Self::Jobs
                | Self::Logs
        )
    }
}

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum StreamTab {
    Live,
    Offline,
    Disabled,
}

impl StreamTab {
    pub const fn label(self) -> &'static str {
        match self {
            Self::Live => "Recording",
            Self::Offline => "Offline",
            Self::Disabled => "Disabled",
        }
    }

    pub const fn next(self) -> Self {
        match self {
            Self::Live => Self::Offline,
            Self::Offline => Self::Disabled,
            Self::Disabled => Self::Live,
        }
    }

    pub const fn previous(self) -> Self {
        match self {
            Self::Live => Self::Disabled,
            Self::Offline => Self::Live,
            Self::Disabled => Self::Offline,
        }
    }
}

#[derive(Debug, Clone, Copy, Eq, PartialEq)]
pub enum SimilarityTab {
    Search,
    Group,
}

impl SimilarityTab {
    pub const fn label(self) -> &'static str {
        match self {
            Self::Search => "Search",
            Self::Group => "Group",
        }
    }

    pub const fn next(self) -> Self {
        match self {
            Self::Search => Self::Group,
            Self::Group => Self::Search,
        }
    }

    pub const fn previous(self) -> Self {
        self.next()
    }
}

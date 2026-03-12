use anyhow::{Context, Result, bail};
use image::{
    DynamicImage, RgbaImage,
    imageops::{FilterType, crop_imm, resize},
};
use ratatui::{buffer::Buffer, layout::Rect, prelude::Color};
use std::cmp::{max, min};

const THUMBNAIL_SUBPIXEL_WIDTH: u16 = 2;
const THUMBNAIL_SUBPIXEL_HEIGHT: u16 = 2;
const THUMBNAIL_SUPERSAMPLE: u16 = 2;

#[derive(Debug, Clone)]
pub struct ThumbnailTarget {
    pub key: String,
    pub label: String,
    pub url: String,
}

#[derive(Debug, Clone)]
pub enum ThumbnailEntry {
    Failed {
        error: String,
    },
    Loading(ThumbnailTarget),
    Ready {
        preview: RenderedThumbnail,
        row: RenderedThumbnail,
    },
}

#[derive(Debug, Clone, Copy)]
struct ThumbnailCell {
    bg: Color,
    fg: Color,
    symbol: char,
}

#[derive(Debug, Clone)]
pub struct RenderedThumbnail {
    cells: Vec<ThumbnailCell>,
    height: u16,
    width: u16,
}

#[derive(Debug, Clone, Copy)]
struct Rgb {
    b: u8,
    g: u8,
    r: u8,
}

impl Rgb {
    fn into_color(self) -> Color {
        Color::Rgb(self.r, self.g, self.b)
    }
}

pub async fn load_thumbnail_image(url: &str, token: Option<&str>) -> Result<DynamicImage> {
    let client = reqwest::Client::builder()
        .redirect(reqwest::redirect::Policy::limited(5))
        .build()
        .context("failed to construct thumbnail client")?;

    let mut request = client.get(url);
    if let Some(token) = token {
        request = request.bearer_auth(token);
    }

    let response = request.send().await.context("thumbnail request failed")?;
    if !response.status().is_success() {
        let status = response.status();
        let body = response.text().await.unwrap_or_default();
        bail!("thumbnail {status}: {body}");
    }

    let bytes = response
        .bytes()
        .await
        .context("failed to read thumbnail response")?;
    image::load_from_memory(&bytes).context("failed to decode thumbnail image")
}

pub fn rendered_video_frame_pixel_dimensions(width: u16, height: u16) -> (u32, u32) {
    rendered_pixel_dimensions(width, height, 1)
}

fn rendered_pixel_dimensions(width: u16, height: u16, supersample: u16) -> (u32, u32) {
    let pixel_width = max(
        1,
        width as usize * THUMBNAIL_SUBPIXEL_WIDTH as usize * supersample.max(1) as usize,
    ) as u32;
    let pixel_height = max(
        1,
        height as usize * THUMBNAIL_SUBPIXEL_HEIGHT as usize * supersample.max(1) as usize,
    ) as u32;
    (pixel_width, pixel_height)
}

pub fn render_thumbnail(image: &DynamicImage, width: u16, height: u16) -> RenderedThumbnail {
    render_thumbnail_with_supersample(image, width, height, THUMBNAIL_SUPERSAMPLE)
}

pub fn render_video_frame(image: &DynamicImage, width: u16, height: u16) -> RenderedThumbnail {
    render_thumbnail_with_supersample(image, width, height, 1)
}

fn render_thumbnail_with_supersample(
    image: &DynamicImage,
    width: u16,
    height: u16,
    supersample: u16,
) -> RenderedThumbnail {
    let supersample = supersample.max(1);
    let (pixel_width, pixel_height) = rendered_pixel_dimensions(width, height, supersample);
    let resized = crop_and_resize_thumbnail(image, pixel_width, pixel_height);
    let mut cells = Vec::with_capacity(width as usize * height as usize);

    for y in 0..height {
        for x in 0..width {
            let px = x as u32 * THUMBNAIL_SUBPIXEL_WIDTH as u32 * supersample as u32;
            let py = y as u32 * THUMBNAIL_SUBPIXEL_HEIGHT as u32 * supersample as u32;
            let samples = [
                average_region_rgb(&resized, px, py, supersample as u32, supersample as u32),
                average_region_rgb(
                    &resized,
                    px + supersample as u32,
                    py,
                    supersample as u32,
                    supersample as u32,
                ),
                average_region_rgb(
                    &resized,
                    px,
                    py + supersample as u32,
                    supersample as u32,
                    supersample as u32,
                ),
                average_region_rgb(
                    &resized,
                    px + supersample as u32,
                    py + supersample as u32,
                    supersample as u32,
                    supersample as u32,
                ),
            ];
            cells.push(best_quadrant_cell(&samples));
        }
    }

    RenderedThumbnail {
        cells,
        height,
        width,
    }
}

pub fn render_placeholder_thumbnail(
    label: &str,
    width: u16,
    height: u16,
    accent: Color,
    background: Color,
) -> RenderedThumbnail {
    let width = width.max(1);
    let height = height.max(1);
    let mut cells = vec![
        ThumbnailCell {
            bg: background,
            fg: accent,
            symbol: ' ',
        };
        width as usize * height as usize
    ];

    for x in 0..width {
        set_cell(&mut cells, width, x, 0, accent, background, '▀');
        set_cell(&mut cells, width, x, height - 1, accent, background, '▄');
    }

    for y in 0..height {
        set_cell(&mut cells, width, 0, y, accent, background, '▌');
        set_cell(&mut cells, width, width - 1, y, accent, background, '▐');
    }

    if width > 4 && height > 2 {
        let initials = placeholder_initials(label);
        let start_x = width.saturating_sub(initials.chars().count() as u16) / 2;
        let center_y = height / 2;
        for (offset, ch) in initials.chars().enumerate() {
            set_cell(
                &mut cells,
                width,
                start_x + offset as u16,
                center_y,
                accent,
                background,
                ch,
            );
        }
    }

    RenderedThumbnail {
        cells,
        height,
        width,
    }
}

pub fn draw_rendered_thumbnail(buffer: &mut Buffer, area: Rect, thumbnail: &RenderedThumbnail) {
    let width = min(area.width, thumbnail.width);
    let height = min(area.height, thumbnail.height);
    for y in 0..height {
        for x in 0..width {
            let index = y as usize * thumbnail.width as usize + x as usize;
            let cell = thumbnail.cells[index];
            if let Some(buffer_cell) = buffer.cell_mut((area.x + x, area.y + y)) {
                buffer_cell
                    .set_char(cell.symbol)
                    .set_fg(cell.fg)
                    .set_bg(cell.bg);
            }
        }
    }
}

fn crop_and_resize_thumbnail(image: &DynamicImage, width: u32, height: u32) -> RgbaImage {
    let source = image.to_rgba8();
    let (src_width, src_height) = source.dimensions();
    if src_width == 0 || src_height == 0 || width == 0 || height == 0 {
        return RgbaImage::new(width.max(1), height.max(1));
    }

    if src_width == width && src_height == height {
        return source;
    }

    let src_ratio = src_width as f32 / src_height as f32;
    let target_ratio = width as f32 / height as f32;
    let (crop_x, crop_y, crop_width, crop_height) = if src_ratio > target_ratio {
        let crop_width = (src_height as f32 * target_ratio)
            .round()
            .clamp(1.0, src_width as f32) as u32;
        ((src_width - crop_width) / 2, 0, crop_width, src_height)
    } else {
        let crop_height = (src_width as f32 / target_ratio)
            .round()
            .clamp(1.0, src_height as f32) as u32;
        (0, (src_height - crop_height) / 2, src_width, crop_height)
    };

    let cropped = crop_imm(&source, crop_x, crop_y, crop_width, crop_height).to_image();
    if crop_width == width && crop_height == height {
        return cropped;
    }
    DynamicImage::ImageRgba8(resize(&cropped, width, height, FilterType::CatmullRom))
        .unsharpen(0.8, 1)
        .to_rgba8()
}

fn rgb_from_rgba(pixel: [u8; 4]) -> Rgb {
    let alpha = pixel[3] as u32;
    let blend =
        |channel: u8| -> u8 { (((channel as u32 * alpha) + (12 * (255 - alpha))) / 255) as u8 };

    Rgb {
        r: blend(pixel[0]),
        g: blend(pixel[1]),
        b: blend(pixel[2]),
    }
}

fn average_region_rgb(image: &RgbaImage, x: u32, y: u32, width: u32, height: u32) -> Rgb {
    let max_x = min(image.width(), x.saturating_add(width));
    let max_y = min(image.height(), y.saturating_add(height));
    if x >= max_x || y >= max_y {
        return Rgb { r: 0, g: 0, b: 0 };
    }

    let mut total_r = 0u32;
    let mut total_g = 0u32;
    let mut total_b = 0u32;
    let mut count = 0u32;

    for sample_y in y..max_y {
        for sample_x in x..max_x {
            let sample = rgb_from_rgba(image.get_pixel(sample_x, sample_y).0);
            total_r += sample.r as u32;
            total_g += sample.g as u32;
            total_b += sample.b as u32;
            count += 1;
        }
    }

    if count == 0 {
        return Rgb { r: 0, g: 0, b: 0 };
    }

    Rgb {
        r: (total_r / count) as u8,
        g: (total_g / count) as u8,
        b: (total_b / count) as u8,
    }
}

fn best_quadrant_cell(samples: &[Rgb; 4]) -> ThumbnailCell {
    let mut best_mask = 0u8;
    let mut best_fg = average_rgb(samples, 0b1111);
    let mut best_bg = best_fg;
    let mut best_error = u32::MAX;

    for mask in 0u8..=0b1111 {
        let fg = if mask == 0 {
            average_rgb(samples, 0b1111)
        } else {
            average_rgb(samples, mask as u16)
        };
        let bg = if mask == 0b1111 {
            average_rgb(samples, 0b1111)
        } else {
            average_rgb(samples, ((!mask) & 0b1111) as u16)
        };

        let error = samples
            .iter()
            .enumerate()
            .map(|(index, sample)| {
                let target = if mask & (1u8 << index) != 0 { fg } else { bg };
                rgb_distance(*sample, target)
            })
            .sum::<u32>();

        if error < best_error {
            best_error = error;
            best_mask = mask;
            best_fg = fg;
            best_bg = bg;
        }
    }

    ThumbnailCell {
        bg: best_bg.into_color(),
        fg: best_fg.into_color(),
        symbol: quadrant_symbol(best_mask),
    }
}

fn average_rgb(samples: &[Rgb], mask: u16) -> Rgb {
    let mut total_r = 0u32;
    let mut total_g = 0u32;
    let mut total_b = 0u32;
    let mut count = 0u32;

    for (index, sample) in samples.iter().enumerate() {
        if mask & (1u16 << index) == 0 {
            continue;
        }
        total_r += sample.r as u32;
        total_g += sample.g as u32;
        total_b += sample.b as u32;
        count += 1;
    }

    if count == 0 {
        return Rgb { r: 0, g: 0, b: 0 };
    }

    Rgb {
        r: (total_r / count) as u8,
        g: (total_g / count) as u8,
        b: (total_b / count) as u8,
    }
}

fn rgb_distance(left: Rgb, right: Rgb) -> u32 {
    let dr = left.r as i32 - right.r as i32;
    let dg = left.g as i32 - right.g as i32;
    let db = left.b as i32 - right.b as i32;
    (dr * dr + dg * dg + db * db) as u32
}

fn quadrant_symbol(mask: u8) -> char {
    match mask {
        0b0000 => ' ',
        0b0001 => '▘',
        0b0010 => '▝',
        0b0011 => '▀',
        0b0100 => '▖',
        0b0101 => '▌',
        0b0110 => '▞',
        0b0111 => '▛',
        0b1000 => '▗',
        0b1001 => '▚',
        0b1010 => '▐',
        0b1011 => '▜',
        0b1100 => '▄',
        0b1101 => '▙',
        0b1110 => '▟',
        0b1111 => '█',
        _ => ' ',
    }
}

fn set_cell(
    cells: &mut [ThumbnailCell],
    width: u16,
    x: u16,
    y: u16,
    fg: Color,
    bg: Color,
    symbol: char,
) {
    let index = y as usize * width as usize + x as usize;
    if let Some(cell) = cells.get_mut(index) {
        *cell = ThumbnailCell { bg, fg, symbol };
    }
}

fn placeholder_initials(label: &str) -> String {
    let parts = label
        .split(|ch: char| !ch.is_alphanumeric())
        .filter(|part| !part.is_empty())
        .collect::<Vec<_>>();

    if parts.is_empty() {
        return "??".to_string();
    }

    let mut initials = String::new();
    for part in parts.iter().take(2) {
        if let Some(ch) = part.chars().next() {
            initials.extend(ch.to_uppercase());
        }
    }

    if initials.len() == 1 {
        if let Some(ch) = parts[0].chars().nth(1) {
            initials.extend(ch.to_uppercase());
        }
    }

    if initials.is_empty() {
        "??".to_string()
    } else {
        initials
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use image::Rgba;

    #[test]
    fn quadrant_symbol_maps_expected_patterns() {
        assert_eq!(quadrant_symbol(0b0011), '▀');
        assert_eq!(quadrant_symbol(0b1100), '▄');
        assert_eq!(quadrant_symbol(0b1001), '▚');
        assert_eq!(quadrant_symbol(0b0110), '▞');
        assert_eq!(quadrant_symbol(0b1111), '█');
    }

    #[test]
    fn best_quadrant_cell_preserves_diagonal_structure() {
        let red = Rgb {
            r: 240,
            g: 40,
            b: 20,
        };
        let blue = Rgb {
            r: 20,
            g: 60,
            b: 220,
        };
        let cell = best_quadrant_cell(&[red, blue, blue, red]);
        assert!(matches!(cell.symbol, '▚' | '▞'));
    }

    #[test]
    fn render_thumbnail_produces_requested_grid_size() {
        let mut image = RgbaImage::new(8, 8);
        for (x, y, pixel) in image.enumerate_pixels_mut() {
            let red = (x * 20) as u8;
            let green = (y * 20) as u8;
            *pixel = Rgba([red, green, 180, 255]);
        }

        let rendered = render_thumbnail(&DynamicImage::ImageRgba8(image), 5, 3);
        assert_eq!(rendered.width, 5);
        assert_eq!(rendered.height, 3);
        assert_eq!(rendered.cells.len(), 15);
    }

    #[test]
    fn placeholder_thumbnail_produces_a_visible_monogram() {
        let rendered =
            render_placeholder_thumbnail("Channel Alpha", 8, 4, Color::Cyan, Color::Blue);
        let symbols = rendered
            .cells
            .iter()
            .map(|cell| cell.symbol)
            .collect::<String>();
        assert!(symbols.contains('C') || symbols.contains('A'));
        assert_eq!(rendered.width, 8);
        assert_eq!(rendered.height, 4);
    }
}

use super::{ThemeBackground, ThemePalette};
use ratatui::{
    Frame,
    buffer::Buffer,
    layout::Rect,
    prelude::{Color, Style},
};

const MATRIX_GLYPHS: &[&str] = &[
    "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "ｦ", "ｧ", "ｨ", "ｩ", "ｪ", "ｫ", "ｬ",
    "ｭ", "ｮ", "ｯ", "ｰ", "ｱ", "ｲ", "ｳ", "ｴ", "ｵ", "ｶ", "ｷ", "ｸ", "ｹ", "ｺ", "ｻ", "ｼ", "ｽ",
    "ｾ", "ｿ", "ﾀ", "ﾁ", "ﾂ", "ﾃ", "ﾄ", "ﾅ", "ﾆ", "ﾇ", "ﾈ", "ﾉ", "ﾊ", "ﾋ", "ﾌ", "ﾍ", "ﾎ",
    "ﾏ", "ﾐ", "ﾑ", "ﾒ", "ﾓ", "ﾔ", "ﾕ", "ﾖ", "ﾗ", "ﾘ", "ﾙ", "ﾚ", "ﾛ", "ﾜ", "ﾝ",
];
const MATRIX_TRAIL: i32 = 22;

pub fn draw_theme_background(
    frame: &mut Frame,
    area: Rect,
    background: ThemeBackground,
    theme: ThemePalette,
    tick: u64,
) {
    match background {
        ThemeBackground::None => {}
        ThemeBackground::MatrixRain => draw_matrix_rain(frame.buffer_mut(), area, theme, tick),
    }
}

fn draw_matrix_rain(buffer: &mut Buffer, area: Rect, _theme: ThemePalette, tick: u64) {
    if area.width == 0 || area.height == 0 {
        return;
    }

    for x_offset in 0..area.width {
        let x = area.x + x_offset;
        let column_seed = mix((x_offset as u64).wrapping_mul(0x9E37_79B9), 0x85EB_CA6B, 0xC2B2_AE35);
        let speed = 0.08 + (column_seed % 9) as f32 * 0.012;
        let phase = (column_seed % (area.height as u64 + MATRIX_TRAIL as u64)) as f32;
        let head_a = ((tick as f32 * speed) + phase)
            % (area.height as f32 + MATRIX_TRAIL as f32)
            - MATRIX_TRAIL as f32;

        let secondary_seed = mix(column_seed, 0x27D4_EB2F, 0x1656_67B1);
        let speed_b = 0.045 + (secondary_seed % 7) as f32 * 0.009;
        let head_b = ((tick as f32 * speed_b) + (secondary_seed % area.height.max(1) as u64) as f32)
            % (area.height as f32 + MATRIX_TRAIL as f32 * 1.4)
            - MATRIX_TRAIL as f32;

        for y_offset in 0..area.height {
            let y = area.y + y_offset;
            let yf = y_offset as f32;
            let dist_a = head_a - yf;
            let dist_b = head_b - yf;
            let ambient_seed = mix(column_seed, y_offset as u64, tick / 2);
            let ambient_on = ambient_seed % 100 < 9;
            if let Some(cell) = buffer.cell_mut((x, y)) {
                if !cell.symbol().trim().is_empty() {
                    continue;
                }
                let (strength, style) = matrix_cell_style(dist_a, dist_b, ambient_on, cell.bg);
                if strength <= 0.0 {
                    continue;
                }
                let glyph = matrix_glyph(mix(column_seed, y_offset as u64, tick / 3));
                cell.set_symbol(glyph);
                cell.set_style(style);
            }
        }
    }
}

fn matrix_glyph(seed: u64) -> &'static str {
    MATRIX_GLYPHS[seed as usize % MATRIX_GLYPHS.len()]
}

fn matrix_cell_style(
    dist_a: f32,
    dist_b: f32,
    ambient_on: bool,
    background: Color,
) -> (f32, Style) {
    let near_a = if (0.0..MATRIX_TRAIL as f32).contains(&dist_a) {
        1.0 - dist_a / MATRIX_TRAIL as f32
    } else {
        0.0
    };
    let near_b = if (0.0..(MATRIX_TRAIL as f32 * 0.8)).contains(&dist_b) {
        0.75 - dist_b / (MATRIX_TRAIL as f32 * 1.3)
    } else {
        0.0
    };
    let strength = near_a.max(near_b);

    let (head, glow, trail, ambient) = matrix_background_palette(background);

    if strength > 0.992 {
        (strength, Style::default().fg(head).bg(background))
    } else if strength > 0.88 {
        (strength, Style::default().fg(glow).bg(background))
    } else if strength > 0.62 {
        (strength, Style::default().fg(trail).bg(background))
    } else if strength > 0.28 || ambient_on {
        (strength.max(0.12), Style::default().fg(ambient).bg(background))
    } else {
        (0.0, Style::default())
    }
}

fn matrix_background_palette(background: Color) -> (Color, Color, Color, Color) {
    match background {
        Color::Rgb(_, _, _) => (
            blend_color(background, Color::Rgb(96, 220, 120), 0.26),
            blend_color(background, Color::Rgb(66, 165, 88), 0.20),
            blend_color(background, Color::Rgb(44, 120, 58), 0.14),
            blend_color(background, Color::Rgb(28, 84, 38), 0.10),
        ),
        _ => (
            Color::Green,
            Color::DarkGray,
            Color::DarkGray,
            Color::Black,
        ),
    }
}

fn blend_color(background: Color, foreground: Color, mix: f32) -> Color {
    match (background, foreground) {
        (Color::Rgb(br, bg, bb), Color::Rgb(fr, fg, fb)) => Color::Rgb(
            blend_channel(br, fr, mix),
            blend_channel(bg, fg, mix),
            blend_channel(bb, fb, mix),
        ),
        _ => foreground,
    }
}

fn blend_channel(background: u8, foreground: u8, mix: f32) -> u8 {
    let mix = mix.clamp(0.0, 1.0);
    let bg = background as f32;
    let fg = foreground as f32;
    (bg + (fg - bg) * mix).round().clamp(0.0, 255.0) as u8
}

fn mix(a: u64, b: u64, c: u64) -> u64 {
    let mut value = a
        .wrapping_mul(0x9E37_79B9_7F4A_7C15)
        .wrapping_add(b.rotate_left(17))
        .wrapping_add(c.rotate_right(9));
    value ^= value >> 33;
    value = value.wrapping_mul(0xFF51_AFD7_ED55_8CCD);
    value ^= value >> 33;
    value = value.wrapping_mul(0xC4CE_B9FE_1A85_EC53);
    value ^ (value >> 33)
}

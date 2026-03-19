import { describe, expect, it } from "vitest";
import {
  buildVisiblePreviewTiles,
  createVideoPreviewManifest,
  getMaxPreviewPixelsPerSecond,
  getMinPreviewPixelsPerSecond,
  getVideoPreviewSource,
  normalizePreviewTimestamps,
  PREVIEW_TILE_TARGET_WIDTH,
} from "../../src/utils/video";

describe("getVideoPreviewSource", () => {
  it("returns null when no preview metadata exists", () => {
    expect(
      getVideoPreviewSource("http://localhost/videos", {
        bitRate: 0,
        bookmark: false,
        channelId: 1,
        channelName: "test",
        createdAt: "2024-01-01T00:00:00Z",
        duration: 120,
        filename: "test.mp4",
        height: 720,
        packets: 0,
        pathRelative: "test.mp4",
        recordingId: 1,
        size: 0,
        videoType: "mp4",
        width: 1280,
      } as any),
    ).toBeNull();
  });

  it("maps preview metadata into the legacy preview source", () => {
    expect(
      getVideoPreviewSource("http://localhost/videos", {
        duration: 120,
        videoPreview: {
          frameCount: 5,
          frameInterval: 2,
          previewPath: "channel/.previews/test",
        },
      } as any),
    ).toEqual({
      duration: 120,
      frameCount: 5,
      frameInterval: 2,
      previewPath: "channel/.previews/test",
      serverPath: "http://localhost/videos",
    });
  });
});

describe("normalizePreviewTimestamps", () => {
  it("sorts, deduplicates, and filters invalid timestamps", () => {
    expect(normalizePreviewTimestamps([10, 0, 10, -1, Number.NaN, 3])).toEqual([0, 3, 10]);
  });
});

describe("createVideoPreviewManifest", () => {
  it("clamps timestamps to the recording duration", () => {
    const manifest = createVideoPreviewManifest("http://localhost/videos", 60, "channel/.previews/test", [0, 30, 90, 120]);

    expect(manifest.timestamps).toEqual([0, 30, 60]);
  });
});

describe("buildVisiblePreviewTiles", () => {
  const manifest = createVideoPreviewManifest("http://localhost/videos", 1_200, "channel/.previews/test", [0, 3, 7, 10, 16, 25, 40, 65, 100, 150, 210, 280, 360, 480, 720, 1_000]);

  it("renders bounded bucketed tiles at low zoom", () => {
    const result = buildVisiblePreviewTiles({
      manifest,
      pixelsPerSecond: 0.5,
      scrollLeft: 0,
      viewportWidth: 600,
    });

    expect(result.timelineWidth).toBe(600);
    expect(result.tiles.length).toBeLessThan(20);
    expect(result.tiles[0]?.left).toBe(0);
    expect(result.tiles.every((tile) => tile.width > 0)).toBe(true);
  });

  it("uses actual frame timestamps at max zoom", () => {
    const pixelsPerSecond = getMaxPreviewPixelsPerSecond(manifest.timestamps);
    const result = buildVisiblePreviewTiles({
      manifest,
      pixelsPerSecond,
      scrollLeft: 0,
      viewportWidth: 960,
    });

    expect(result.usesActualFrames).toBe(true);
    expect(result.tiles[0]?.timestamp).toBe(0);
    expect(result.tiles[1]?.timestamp).toBe(3);
    expect(result.tiles[0]?.width).toBeCloseTo(3 * pixelsPerSecond, 4);
  });

  it("fills the visible window without gaps when bucket sampling", () => {
    const result = buildVisiblePreviewTiles({
      manifest,
      pixelsPerSecond: 2,
      scrollLeft: 120,
      viewportWidth: 800,
    });

    for (let index = 1; index < result.tiles.length; index += 1) {
      const previousTile = result.tiles[index - 1]!;
      const currentTile = result.tiles[index]!;
      expect(currentTile.left).toBeCloseTo(previousTile.left + previousTile.width, 4);
    }
  });
});

describe("preview zoom bounds", () => {
  it("fits the timeline into view at minimum zoom", () => {
    expect(getMinPreviewPixelsPerSecond(1_200, 600)).toBe(0.5);
  });

  it("caps max zoom using the smallest real timestamp gap", () => {
    expect(getMaxPreviewPixelsPerSecond([0, 3, 7])).toBe(PREVIEW_TILE_TARGET_WIDTH / 3);
  });
});

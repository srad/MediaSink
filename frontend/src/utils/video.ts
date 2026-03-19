import type { DbRecording } from "../services/api/v2/MediaSinkClient";

export const PREVIEW_TILE_TARGET_WIDTH = 96;
export const PREVIEW_TILE_OVERSCAN_VIEWPORTS = 2;
const MAX_PREVIEW_FRAMES = 400;

export type VideoPreviewSource = {
  duration: number;
  frameCount: number;
  frameInterval: number;
  previewPath: string;
  serverPath: string;
};

export type VideoPreviewManifest = {
  duration: number;
  previewPath: string;
  serverPath: string;
  timestamps: number[];
};

export type VisiblePreviewTile = {
  index: number;
  key: string;
  left: number;
  src: string;
  timestamp: number;
  width: number;
};

type BuildVisiblePreviewTilesOptions = {
  manifest: VideoPreviewManifest;
  overscanViewports?: number;
  pixelsPerSecond: number;
  scrollLeft: number;
  targetTileWidth?: number;
  viewportWidth: number;
};

type BuildVisiblePreviewTilesResult = {
  tileWidth: number;
  tiles: VisiblePreviewTile[];
  timelineWidth: number;
  usesActualFrames: boolean;
};

const clamp = (value: number, min: number, max: number): number => Math.min(max, Math.max(min, value));

const normalizeFrameCount = (frameCount: number): number => Math.max(0, Math.floor(Number(frameCount) || 0));
const normalizeFrameInterval = (frameInterval: number): number => Math.max(0.1, Number(frameInterval) || 1);

export const normalizePreviewTimestamps = (timestamps: number[]): number[] => {
  return Array.from(
    new Set(
      timestamps
        .map((timestamp) => Number(timestamp))
        .filter((timestamp) => Number.isFinite(timestamp) && timestamp >= 0)
        .map((timestamp) => Math.round(timestamp * 1000) / 1000),
    ),
  ).sort((left, right) => left - right);
};

const findFirstTimestampAtOrAfter = (timestamps: number[], target: number): number => {
  let low = 0;
  let high = timestamps.length;

  while (low < high) {
    const mid = Math.floor((low + high) / 2);
    if (timestamps[mid]! < target) {
      low = mid + 1;
    } else {
      high = mid;
    }
  }

  return low;
};

const findNearestTimestampIndex = (timestamps: number[], target: number): number => {
  if (timestamps.length <= 1) {
    return 0;
  }

  const nextIndex = findFirstTimestampAtOrAfter(timestamps, target);
  if (nextIndex <= 0) {
    return 0;
  }
  if (nextIndex >= timestamps.length) {
    return timestamps.length - 1;
  }

  const previousIndex = nextIndex - 1;
  const previousDistance = Math.abs(timestamps[previousIndex]! - target);
  const nextDistance = Math.abs(timestamps[nextIndex]! - target);

  return previousDistance <= nextDistance ? previousIndex : nextIndex;
};

const getSmallestTimestampGap = (timestamps: number[]): number => {
  let minGap = Number.POSITIVE_INFINITY;

  for (let index = 1; index < timestamps.length; index += 1) {
    const gap = timestamps[index]! - timestamps[index - 1]!;
    if (gap > 0 && gap < minGap) {
      minGap = gap;
    }
  }

  return Number.isFinite(minGap) ? minGap : 1;
};

export const videoCover = (video: DbRecording): string => {
  if (video.videoPreview) {
    return `${video.videoPreview.previewPath}/0.jpg`;
  }

  return `${video.channelName}/.previews/live.jpg`;
};

export const getVideoPreviewSource = (serverPath: string, video: DbRecording): VideoPreviewSource | null => {
  if (!video.videoPreview) {
    return null;
  }

  return {
    duration: Math.max(Number(video.duration) || 0, normalizeFrameInterval(video.videoPreview.frameInterval)),
    frameCount: normalizeFrameCount(video.videoPreview.frameCount),
    frameInterval: normalizeFrameInterval(video.videoPreview.frameInterval),
    previewPath: video.videoPreview.previewPath,
    serverPath,
  };
};

export const createVideoPreviewManifest = (serverPath: string, duration: number, previewPath: string, timestamps: number[]): VideoPreviewManifest => ({
  duration: Math.max(Number(duration) || 0, 1),
  previewPath,
  serverPath,
  timestamps: normalizePreviewTimestamps(
    normalizePreviewTimestamps(timestamps).map((timestamp) => Math.min(timestamp, Math.max(Number(duration) || 0, 1))),
  ),
});

export const mapVideoFrames = (serverPath: string, video: DbRecording): string[] => {
  const source = getVideoPreviewSource(serverPath, video);
  if (!source || source.frameCount === 0) {
    return [];
  }

  const sampleEveryFrames = Math.max(1, Math.ceil(source.frameCount / MAX_PREVIEW_FRAMES));
  const frames: string[] = [];

  for (let frameIndex = 0; frameIndex < source.frameCount; frameIndex += sampleEveryFrames) {
    frames.push(getPreviewFrameUrl(source, frameIndex));
  }

  const lastFrameUrl = getPreviewFrameUrl(source, source.frameCount - 1);
  if (frames[frames.length - 1] !== lastFrameUrl) {
    frames.push(lastFrameUrl);
  }

  return frames;
};

export const getPreviewFrameUrl = (source: VideoPreviewSource, frameIndex: number): string => {
  const safeFrameCount = Math.max(1, source.frameCount);
  const clampedFrameIndex = clamp(Math.floor(frameIndex), 0, safeFrameCount - 1);
  const timestamp = clampedFrameIndex * source.frameInterval;

  return `${source.serverPath}/${source.previewPath}/${timestamp}.jpg`;
};

export const getPreviewFrameUrlForTimestamp = (manifest: VideoPreviewManifest, timestamp: number): string => {
  return `${manifest.serverPath}/${manifest.previewPath}/${timestamp}.jpg`;
};

export const getMinPreviewPixelsPerSecond = (duration: number, viewportWidth: number): number => {
  const safeDuration = Math.max(duration, 1);
  const safeViewportWidth = Math.max(viewportWidth, 1);

  return safeViewportWidth / safeDuration;
};

export const getMaxPreviewPixelsPerSecond = (timestamps: number[], targetTileWidth = PREVIEW_TILE_TARGET_WIDTH): number => {
  const minGap = getSmallestTimestampGap(normalizePreviewTimestamps(timestamps));
  return targetTileWidth / Math.max(minGap, 0.1);
};

export const buildVisiblePreviewTiles = ({
  manifest,
  overscanViewports = PREVIEW_TILE_OVERSCAN_VIEWPORTS,
  pixelsPerSecond,
  scrollLeft,
  targetTileWidth = PREVIEW_TILE_TARGET_WIDTH,
  viewportWidth,
}: BuildVisiblePreviewTilesOptions): BuildVisiblePreviewTilesResult => {
  const timestamps = normalizePreviewTimestamps(manifest.timestamps);
  const safeDuration = Math.max(Number(manifest.duration) || 0, 1);
  const safePixelsPerSecond = Math.max(Number(pixelsPerSecond) || 0, 0.01);
  const safeViewportWidth = Math.max(Number(viewportWidth) || 0, 1);
  const timelineWidth = Math.max(safeDuration * safePixelsPerSecond, safeViewportWidth);

  if (timestamps.length === 0) {
    return {
      tileWidth: targetTileWidth,
      tiles: [],
      timelineWidth,
      usesActualFrames: false,
    };
  }

  const overscanPx = safeViewportWidth * Math.max(overscanViewports, 0);
  const windowStartTime = clamp((scrollLeft - overscanPx) / safePixelsPerSecond, 0, safeDuration);
  const windowEndTime = clamp((scrollLeft + safeViewportWidth + overscanPx) / safePixelsPerSecond, 0, safeDuration);
  const bucketDuration = Math.max(targetTileWidth / safePixelsPerSecond, 0.1);

  const visibleStartIndex = Math.max(0, findFirstTimestampAtOrAfter(timestamps, windowStartTime) - 1);
  const visibleEndIndex = Math.min(timestamps.length - 1, findFirstTimestampAtOrAfter(timestamps, windowEndTime + bucketDuration));
  const visibleFrameCount = visibleEndIndex - visibleStartIndex + 1;
  const desiredBucketCount = Math.max(1, Math.ceil((windowEndTime - windowStartTime) / bucketDuration));

  if (desiredBucketCount >= visibleFrameCount) {
    const tiles: VisiblePreviewTile[] = [];

    for (let index = visibleStartIndex; index <= visibleEndIndex; index += 1) {
      const timestamp = timestamps[index]!;
      const nextTimestamp = index + 1 < timestamps.length ? timestamps[index + 1]! : safeDuration;
      const tileStartTime = index === 0 ? 0 : timestamp;
      const tileEndTime = Math.max(nextTimestamp, tileStartTime + 0.1);

      tiles.push({
        index,
        key: `${timestamp}-${index}`,
        left: tileStartTime * safePixelsPerSecond,
        src: getPreviewFrameUrlForTimestamp(manifest, timestamp),
        timestamp,
        width: Math.max((tileEndTime - tileStartTime) * safePixelsPerSecond, 1),
      });
    }

    return {
      tileWidth: tiles[0]?.width || targetTileWidth,
      tiles,
      timelineWidth,
      usesActualFrames: true,
    };
  }

  const tiles: VisiblePreviewTile[] = [];
  const alignedBucketStart = Math.max(0, Math.floor(windowStartTime / bucketDuration) * bucketDuration);

  for (let bucketStart = alignedBucketStart; bucketStart < windowEndTime; bucketStart += bucketDuration) {
    const bucketEnd = Math.min(bucketStart + bucketDuration, safeDuration);
    const bucketCenter = bucketStart + (bucketEnd - bucketStart) / 2;
    const chosenIndex = findNearestTimestampIndex(timestamps, bucketCenter);
    const chosenTimestamp = timestamps[chosenIndex]!;

    tiles.push({
      index: chosenIndex,
      key: `${bucketStart.toFixed(3)}-${chosenTimestamp}`,
      left: bucketStart * safePixelsPerSecond,
      src: getPreviewFrameUrlForTimestamp(manifest, chosenTimestamp),
      timestamp: chosenTimestamp,
      width: Math.max((bucketEnd - bucketStart) * safePixelsPerSecond, 1),
    });
  }

  return {
    tileWidth: Math.max(bucketDuration * safePixelsPerSecond, 1),
    tiles,
    timelineWidth,
    usesActualFrames: false,
  };
};

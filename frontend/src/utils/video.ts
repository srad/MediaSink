import type { DatabaseRecording } from "../services/api/v1/MediaSinkClient";

export const videoCover = (video: DatabaseRecording): string => {
  if (video.videoPreview) {
    return video.videoPreview?.previewPath + "/0.jpg";
  }
  return video.channelName + "/.previews/live.jpg";
};

export const mapVideoFrames = (serverPath: string, video: DatabaseRecording): string[] => {
  if (video.videoPreview) {
    // intervals allow limiting the number of frames for browser performance reasons.
    const intervals = 2;
    const frameCount = Math.max(1, Math.floor(video.videoPreview.frameCount / intervals));
    return new Array(frameCount).fill(0).map((_, i) => `${serverPath}/${video.videoPreview?.previewPath}/${intervals * i * video.videoPreview!.frameInterval}.jpg`);
  }
  return [];
};

export default {
  durationMinutes: "{0}min",
  bitRate: "Bitrate",
  bitRateMBit: (ctx: any) => `${(ctx.list(0) / 1000 / 1000).toFixed(1)} MBit/s`,
  resolution: "Resolution",
  started: "Started at",
  convert: "Convert to",
};

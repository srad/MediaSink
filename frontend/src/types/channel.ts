export type ChannelUpdate = {
  isPaused: boolean;
  channelId: number;
  channelName: string;
  url: string;
  displayName: string;
  skipStart: number;
  minDuration: number;
};

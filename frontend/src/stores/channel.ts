import type { DatabaseChannel as ChannelResponse, DatabaseChannel, DatabaseRecording, RequestsChannelRequest, RequestsChannelRequest as ChannelRequest, ServicesChannelInfo as ChannelInfo } from "../services/api/v1/MediaSinkClient";
import { defineStore } from "pinia";
import { createClient } from "../services/api/v1/ClientFactory";
import { useJobStore } from "../stores/job";

export type ChannelState = {
  channels: ChannelInfo[];
};

export const sortChannel = (a: ChannelResponse, b: ChannelResponse) => a.channelName!.localeCompare(b.channelName!);

export const useChannelStore = defineStore("channel", {
  state: (): ChannelState => ({
    channels: [],
  }),
  getters: {
    all(): ChannelInfo[] {
      return this.channels;
    },
    notRecordingStreams(): ChannelInfo[] {
      return this.channels.filter((row) => !row.isRecording && !row.isPaused).sort(sortChannel);
    },
    disabledStreams(): ChannelInfo[] {
      return this.channels.filter((row) => row.isPaused).sort(sortChannel);
    },
    recordingStreams(): ChannelInfo[] {
      return this.channels.filter((row) => row.isRecording && !row.isTerminating).sort(sortChannel);
    },
  },
  actions: {
    addRecording(r: DatabaseRecording) {
      const channel = this.channels.find((ch) => ch.channelId === r.channelId);
      if (channel) {
        if (!channel.recordings) {
          channel.recordings = [r];
        } else {
          channel.recordings.push(r);
        }
      }
    },
    async load() {
      const client = createClient();
      const data = await client.channels.channelsList();
      this.channels = (data || []).sort(sortChannel);
    },
    async create(channel: ChannelRequest): Promise<ChannelInfo> {
      const res = await createClient().channels.channelsCreate(channel);
      this.add(res);
      return res;
    },
    async save(id: number, channel: RequestsChannelRequest): Promise<DatabaseChannel> {
      const update = await createClient().channels.channelsPartialUpdate({ id }, channel);
      this.update(update);
      return update;
    },
    online(channelId: number) {
      const channel = this.channels.find((ch) => ch.channelId === channelId);
      if (channel) {
        channel.isOnline = true;
      }
    },
    offline(channelId: number) {
      const channel = this.channels.find((ch) => ch.channelId === channelId);
      if (channel) {
        channel.isOnline = false;
        channel.isRecording = false;
      }
    },
    thumbnail(channelId: number) {
      const channel = this.channels.find((ch) => ch.channelId === channelId);
      if (channel) {
        channel.preview = channel.preview.split("?")[0] + `?time=${Date.now()}`;
      }
    },
    start(channelId: number) {
      const channel = this.channels.find((ch) => ch.channelId === channelId);
      if (channel) {
        channel.isRecording = true;
        channel.isOnline = true;
      }
    },
    add(channel: ChannelInfo) {
      if (!this.channels.some((ch) => ch.channelId === channel.channelId)) {
        this.channels.push(channel);
      }
    },
    update(channel: DatabaseChannel) {
      const ch = this.channels.find((c) => c.channelId === channel.channelId);
      if (ch) {
        Object.assign(ch, channel);
      }
    },
    destroy(channelId: number) {
      const index = this.channels.findIndex((ch) => ch.channelId === channelId);
      if (index !== -1) {
        this.channels.splice(index, 1);
      }
      useJobStore().destroyChannel(channelId);
    },
    pause(channelId: number) {
      const channel = this.channels.find((ch) => ch.channelId === channelId);
      if (channel) {
        channel.isRecording = false;
        channel.isPaused = true;
      }
    },
    resume(channelId: number) {
      const channel = this.channels.find((ch) => ch.channelId === channelId);
      if (channel) {
        channel.isPaused = false;
      }
    },
    fav(id: number) {
      const channel = this.channels.find((ch) => ch.channelId === id);
      if (channel) {
        channel.fav = true;
      }
    },
    unfav(id: number) {
      const channel = this.channels.find((ch) => ch.channelId === id);
      if (channel) {
        channel.fav = false;
      }
    },
    clear() {
      this.channels.length = 0;
    },
    stop() {
      this.channels.forEach((channel) => (channel.isRecording = false));
    },
  },
});

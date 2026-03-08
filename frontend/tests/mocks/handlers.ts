import { http, HttpResponse } from "msw";
import channels from "./channels";
import jobs from "./jobs";
import videos from "./videos";

const lasId = channels.map((channel) => channel.channelId).sort((a, b) => b - a);

export const restHandlers = [
  http.get("http://localhost:3000/channels", () => HttpResponse.json(channels)),

  http.post("http://localhost:3000/channels", ({ request }) => {
    return HttpResponse.json({
      ...request.body,
      channelId: lasId[0] + 1, // Increment only the highest channelId.
    });
  }),

  http.get("http://localhost:3000/channels*", ({ request }) => {
    const url = new URL(request.url);
    const id = Number(url.searchParams.get("id"));
    HttpResponse.json(
      videos.map((x) => {
        x.channelId = id;
        return x;
      }),
    );
  }),

  http.post("http://localhost:3000/jobs/list", () => HttpResponse.json(jobs)),
];

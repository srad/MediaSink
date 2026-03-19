import { mount } from "@vue/test-utils";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { nextTick } from "vue";
import VideoStripe from "../../src/components/VideoStripe.vue";

const preview = {
  duration: 10_000,
  previewPath: "channel/.previews/test",
  serverPath: "http://localhost/videos",
  timestamps: Array.from({ length: 2_500 }, (_, index) => index * 4 + (index % 3)),
};

const baseProps = {
  duration: preview.duration,
  loaded: true,
  markings: [],
  paused: false,
  preview,
  seeked: 0,
  timecode: 0,
};

const originalClientWidth = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "clientWidth");

beforeAll(() => {
  vi.stubGlobal(
    "ResizeObserver",
    class {
      private readonly callback: ResizeObserverCallback;

      constructor(callback: ResizeObserverCallback) {
        this.callback = callback;
      }

      disconnect() {}

      observe() {
        this.callback([], this as unknown as ResizeObserver);
      }

      unobserve() {}
    },
  );

  Object.defineProperty(HTMLElement.prototype, "clientWidth", {
    configurable: true,
    get() {
      return 800;
    },
  });
});

afterAll(() => {
  if (originalClientWidth) {
    Object.defineProperty(HTMLElement.prototype, "clientWidth", originalClientWidth);
  }

  vi.unstubAllGlobals();
});

describe("VideoStripe", () => {
  it("renders a bounded number of preview tiles for long recordings", async () => {
    const wrapper = mount(VideoStripe, {
      props: baseProps,
    });

    await nextTick();

    expect(wrapper.findAll("img").length).toBeLessThan(100);
  });

  it("renders more tiles after zooming in", async () => {
    const wrapper = mount(VideoStripe, {
      attachTo: document.body,
      props: baseProps,
    });

    await nextTick();

    const beforeZoomWidth = Number.parseFloat(wrapper.get("[data-testid='video-stripe-timeline']").element.getAttribute("style")?.match(/width:\s*([0-9.]+)px/)?.[1] || "0");
    wrapper.get("[data-testid='video-stripe-scroll']").element.dispatchEvent(
      new WheelEvent("wheel", {
        bubbles: true,
        cancelable: true,
        clientX: 300,
        deltaX: 0,
        deltaY: -600,
      }),
    );

    await nextTick();
    await new Promise((resolve) => setTimeout(resolve, 0));
    await nextTick();

    const afterZoomWidth = Number.parseFloat(wrapper.get("[data-testid='video-stripe-timeline']").element.getAttribute("style")?.match(/width:\s*([0-9.]+)px/)?.[1] || "0");

    expect(afterZoomWidth).toBeGreaterThan(beforeZoomWidth);
  });

  it("renders grouped highlights as spans when start and end times exist", async () => {
    const wrapper = mount(VideoStripe, {
      props: {
        ...baseProps,
        highlights: [
          {
            endTime: 240,
            intensity: 0.8,
            startTime: 120,
            timestamp: 180,
            type: "motion",
          },
        ],
      },
    });

    await nextTick();

    const marker = wrapper.find(".highlight-marker");
    expect(marker.exists()).toBe(true);
    expect(marker.attributes("title")).toContain("120.0s - 240.0s");
    expect(marker.attributes("style")).toContain("width:");
  });
});

import { describe, it, expect } from "vitest";
import { mount } from "@vue/test-utils";
import { createI18n } from "vue-i18n";
import JobStatus from "../../src/components/JobStatus.vue";

const i18n = createI18n({
  locale: "en",
  messages: {
    en: { menu: { jobs: "Jobs" } },
  },
});

describe("JobStatus", () => {
  it("renders with the correct total count and label", () => {
    const wrapper = mount(JobStatus, {
      global: {
        plugins: [i18n],
      },
      props: {
        totalCount: 5,
      },
    });

    expect(wrapper.text()).toContain("5");
    expect(wrapper.text()).toContain("Jobs");
  });

  it("emits click event when clicked", async () => {
    const wrapper = mount(JobStatus, {
      global: {
        plugins: [i18n],
      },
      props: {
        totalCount: 3,
      },
    });

    await wrapper.find("button").trigger("click");
    expect(wrapper.emitted("click")).toBeTruthy();
  });
});

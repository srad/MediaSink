import { mount } from "@vue/test-utils";
import { describe, it, expect } from "vitest";
import LoadIndicator from "../../src/components/LoadIndicator.vue";

describe("LoadIndicator", () => {
  it('renders spinner when "busy" prop is true', () => {
    const wrapper = mount(LoadIndicator, {
      props: {
        busy: true,
      },
    });

    const spinner = wrapper.find(".spinner-border");
    expect(spinner.exists()).toBe(true);
  });

  it('renders empty text when "empty" prop is true and "busy" prop is false', () => {
    const wrapper = mount(LoadIndicator, {
      props: {
        empty: true,
        busy: false,
        emptyText: "No data available",
      },
    });

    const emptyText = wrapper.find("h3");
    expect(emptyText.text()).toBe("No data available");
  });

  it('renders default empty text when "emptyText" prop is not provided', () => {
    const wrapper = mount(LoadIndicator, {
      props: {
        empty: true,
        busy: false,
      },
    });

    const emptyText = wrapper.find("h3");
    expect(emptyText.text()).toBe("...");
  });

  it('renders slot content when neither "busy" nor "empty" are true', () => {
    const wrapper = mount(LoadIndicator, {
      props: {
        busy: false,
        empty: false,
      },
      slots: {
        default: "This is slot content",
      },
    });

    const slotContent = wrapper.text();
    expect(slotContent).toBe("This is slot content");
  });

  it('does not render spinner when "busy" prop is false', () => {
    const wrapper = mount(LoadIndicator, {
      props: {
        busy: false,
      },
    });

    const spinner = wrapper.find(".spinner-border");
    expect(spinner.exists()).toBe(false);
  });
});

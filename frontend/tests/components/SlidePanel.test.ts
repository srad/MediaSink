import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import SlidePanel from "../../src/SlidePanel.vue";
import { nextTick } from "vue";

describe("SlidePanel.vue", () => {
  it('renders the default label when no label prop is passed', async () => {
    const wrapper = mount(SlidePanel);

    // Check the label for the button
    const buttonText = wrapper.find('.toggle-button').text();

    // Expect an empty string if no label is passed
    expect(buttonText).toBe('Open');
  });

  it('renders the correct label based on the panel state', async () => {
    const wrapper = mount(SlidePanel);

    // Initially, the panel is closed and the label should be "Open"
    let buttonText = wrapper.find('.toggle-button').text();
    expect(buttonText).toContain('Open');

    const button = wrapper.find('.toggle-button');

    // Simulate a click to open the panel
    await button.trigger('click');
    await nextTick();

    // After opening the panel, the label should change to "Close"
    buttonText = wrapper.find('.toggle-button').text();
    expect(buttonText).toContain('Close');

    // Simulate a click to close the panel
    await button.trigger('click');
    await nextTick();

    // After closing the panel, the label should change back to "Open"
    buttonText = wrapper.find('.toggle-button').text();
    expect(buttonText).toContain('Open');
  });

  it("renders custom label when passed as a prop", () => {
    const label = "Test Panel";
    const wrapper = mount(SlidePanel, {
      props: {
        label
      }
    });
    expect(wrapper.text()).toContain(label);
  });

  it("toggles the panel when the button is clicked", async () => {
    const wrapper = mount(SlidePanel);

    const button = wrapper.find(".toggle-button");
    expect(wrapper.vm.isOpen).toBe(false);

    // Simulate button click to open the panel
    await button.trigger("click");
    expect(wrapper.vm.isOpen).toBe(true);

    // Simulate button click to close the panel
    await button.trigger("click");
    expect(wrapper.vm.isOpen).toBe(false);
  });

  it("correctly applies the position classes", () => {
    const wrapperLeft = mount(SlidePanel, {
      props: { position: "left" }
    });
    expect(wrapperLeft.classes()).toContain("position-left");

    const wrapperCenter = mount(SlidePanel, {
      props: { position: "center" }
    });
    expect(wrapperCenter.classes()).toContain("position-center");

    const wrapperRight = mount(SlidePanel, {
      props: { position: "right" }
    });
    expect(wrapperRight.classes()).toContain("position-right");
  });

  it("opens with correct height when content is shown", async () => {
    const wrapper = mount(SlidePanel, {
      slots: {
        default: "<div style=\"height: 200px;\">Content</div>"
      }
    });

    const button = wrapper.find(".toggle-button");
    expect(wrapper.vm.isOpen).toBe(false);
    expect(wrapper.find(".panel-content").attributes().style).toContain("max-height: 0");

    // Simulate opening the panel
    await button.trigger("click");
    expect(wrapper.vm.isOpen).toBe(true);

    const contentStyle = wrapper.find(".panel-content").attributes().style;
    expect(contentStyle).toContain("max-height");
  });

  it("applies custom width and opacity correctly", () => {
    const wrapper = mount(SlidePanel, {
      props: {
        width: "400px",
        opacity: 0.8
      }
    });

    const panelStyle = wrapper.find(".slide-panel").attributes().style;
    expect(panelStyle).toContain("width: 400px");
    expect(panelStyle).toContain("opacity: 0.8");
  });

  it("has correct default styles and classes", () => {
    const wrapper = mount(SlidePanel);

    // Test default style and class
    const panel = wrapper.find(".slide-panel");
    expect(panel.attributes().style).toContain("height: 35px");
    expect(panel.classes()).toContain("slide-panel");
  });

  it('toggles the panel state when the button is clicked', async () => {
    const wrapper = mount(SlidePanel);

    // Ensure isOpen is initially false
    const { isOpen } = wrapper.vm;
    expect(isOpen).toBe(false);

    const button = wrapper.find('.toggle-button');

    // Simulate the first button click to open the panel
    await button.trigger('click');

    // Wait for the next DOM update
    await nextTick();

    // Check if the panel state changed to open (isOpen should be true)
    expect(wrapper.vm.isOpen).toBe(true);

    // Simulate the second button click to close the panel
    await button.trigger('click');

    // Wait for the next DOM update again
    await nextTick();

    // Check if the panel state changed back to closed (isOpen should be false)
    expect(wrapper.vm.isOpen).toBe(false);
  });
});

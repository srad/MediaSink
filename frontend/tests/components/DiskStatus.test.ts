import DiskStatus from "../../src/components/DiskStatus.vue";
import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";

describe("DiskStatus", () => {
  const pcentValues = [0, 25, 50, 75, 100]; // Define the range of pcent values to test

  pcentValues.forEach((pcent) => {
    it(`renders correctly with pcent value: ${pcent}`, () => {
      const wrapper = mount(DiskStatus, {
        props: { pcent },
      });

      // Check if the percentage is rendered correctly
      expect(wrapper.text()).toContain(`${pcent}%`);

      // Check if the progress bar width is set correctly
      const progressBar = wrapper.find('.progress-bar');
      expect(progressBar.attributes('style')).toContain(`width: ${pcent}%`);
    });
  });
});

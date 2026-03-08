import { mount } from "@vue/test-utils";
import { describe, it, expect } from "vitest";
import AppAlert from "../../src/components/AppAlert.vue";
import { AlertType } from "../../src/types/alert";

describe("AppAlert", () => {
  it("applies the correct alert class based on the alertType prop", () => {
    const alertTypes = [
      { type: AlertType.Info, expectedClass: "alert-info" },
      { type: AlertType.Warn, expectedClass: "alert-warning" },
      { type: AlertType.Error, expectedClass: "alert-danger" },
      { type: AlertType.Success, expectedClass: "alert-success" },
      { type: AlertType.Dark, expectedClass: "alert-dark" },
      { type: AlertType.Default, expectedClass: "alert-primary" },
      { type: AlertType.Light, expectedClass: "alert-light" },
      { type: AlertType.Secondary, expectedClass: "alert-secondary" },
    ];

    alertTypes.forEach(({ type, expectedClass }) => {
      const wrapper = mount(AppAlert, {
        props: {
          alertType: type,
        },
      });

      // Check if the expected alert class is applied
      expect(wrapper.classes()).toContain(expectedClass);
    });
  });

  it("applies the correct border class based on the alertType prop", () => {
    const alertTypes = [
      { type: AlertType.Info, expectedBorderClass: "border-info" },
      { type: AlertType.Warn, expectedBorderClass: "border-warning" },
      { type: AlertType.Error, expectedBorderClass: "border-danger" },
      { type: AlertType.Success, expectedBorderClass: "border-success" },
      { type: AlertType.Dark, expectedBorderClass: "border-dark" },
      { type: AlertType.Default, expectedBorderClass: "border-primary" },
      { type: AlertType.Light, expectedBorderClass: "border-dark-subtle" },
      { type: AlertType.Secondary, expectedBorderClass: "border-secondary" },
    ];

    alertTypes.forEach(({ type, expectedBorderClass }) => {
      const wrapper = mount(AppAlert, {
        props: {
          alertType: type,
        },
      });

      // Check if the expected border class is applied
      expect(wrapper.classes()).toContain(expectedBorderClass);
    });
  });

  it("applies custom className when passed as a prop", () => {
    const customClass = "my-custom-class";
    const wrapper = mount(AppAlert, {
      props: {
        alertType: AlertType.Info,
        className: customClass,
      },
    });

    // Check if the custom class is applied
    expect(wrapper.classes()).toContain(customClass);
  });

  it("renders slot content correctly", () => {
    const wrapper = mount(AppAlert, {
      props: {
        alertType: AlertType.Info,
      },
      slots: {
        default: "This is an alert message",
      },
    });

    // Check if the slot content is rendered
    expect(wrapper.text()).toBe("This is an alert message");
  });
});

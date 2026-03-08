import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it } from "vitest";
import AppToaster from "../../src/components/AppToaster.vue";
import { Toast, ToastKind, useToastStore } from "../../src/stores/toast";
import { createPinia, setActivePinia } from "pinia";

describe("AppToaster", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    const store = useToastStore();
    store.$reset(); // Reset the store before each test
  });

  const toasts: Toast[] = [
    {
      id: 1,
      created: new Date(),
      title: "Info Toast",
      message: "This is an info toast",
      hide: false,
      countdown: 50,
      kind: ToastKind.Info, // Use the mocked ToastKind enum
    },
    {
      id: 2,
      created: new Date(),
      title: "Success Toast",
      message: "This is a success toast",
      hide: false,
      countdown: 70,
      kind: ToastKind.Success, // Use the mocked ToastKind enum
    },
  ];

  it("closes toast on button click", async () => {
    const wrapper = mount(AppToaster, {
      props: { toasts },
    });

    // Get the close button for the first toast
    const elId = `#toast_${toasts[0].id}`;
    const closeButton = wrapper.find(elId + " .btn-close");

    await closeButton.trigger("click");

    const emittedEvents = wrapper.emitted();
    expect(emittedEvents).toHaveProperty("destroy");
    expect(emittedEvents.destroy).toHaveLength(1); // Check if it was emitted once
  });
});

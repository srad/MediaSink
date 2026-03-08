import { ToastKind, useToastStore } from "../../src/stores/toast";
import { describe, expect, it, beforeEach, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";

describe("Toast Store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    const store = useToastStore();
    store.$reset(); // Reset the store before each test
  });

  it("should add an info toast", () => {
    const store = useToastStore();
    const title = "Info Title";
    const message = "This is an info message";

    store.info({ title, message });

    expect(store.toasts).toHaveLength(1);
    expect(store.toasts[0].title).toBe(title);
    expect(store.toasts[0].message).toBe(message);
    expect(store.toasts[0].kind).toBe("info");
  });

  it("should add an error toast", () => {
    const store = useToastStore();
    const title = "Error Title";
    const message = "This is an error message";

    store.error({ title, message });

    expect(store.toasts).toHaveLength(1);
    expect(store.toasts[0].title).toBe(title);
    expect(store.toasts[0].message).toBe(message);
    expect(store.toasts[0].kind).toBe("error");
  });

  it("should add a warning toast", () => {
    const store = useToastStore();
    const title = "Warning Title";
    const message = "This is a warning message";

    store.warn({ title, message });

    expect(store.toasts).toHaveLength(1);
    expect(store.toasts[0].title).toBe(title);
    expect(store.toasts[0].message).toBe(message);
    expect(store.toasts[0].kind).toBe("warning");
  });

  it("should add a success toast", () => {
    const store = useToastStore();
    const title = "Success Title";
    const message = "This is a success message";

    store.success({ title, message });

    expect(store.toasts).toHaveLength(1);
    expect(store.toasts[0].title).toBe(title);
    expect(store.toasts[0].message).toBe(message);
    expect(store.toasts[0].kind).toBe("success");
  });

  it("should add a toast with custom kind", () => {
    const store = useToastStore();
    const title = "Custom Title";
    const message = "This is a custom message";
    const kind = ToastKind.Warning;

    store.add({ title, message, kind });

    expect(store.toasts).toHaveLength(1);
    expect(store.toasts[0].title).toBe(title);
    expect(store.toasts[0].message).toBe(message);
    expect(store.toasts[0].kind).toBe(kind);
  });

  it("should destroy a toast", () => {
    const store = useToastStore();
    const title = "Info Title";
    const message = "This is an info message";

    store.info({ title, message });
    const toast = store.toasts[0];

    store.destroy(toast);

    expect(store.toasts).toHaveLength(0);
  });

  it("should hide a toast", () => {
    const store = useToastStore();
    const title = "Info Title";
    const message = "This is an info message";

    store.info({ title, message });
    const toast = store.toasts[0];

    store.hide(toast);

    expect(store.toasts[0].hide).toBe(true);
  });

  it("should have getter all", () => {
    const store = useToastStore();
    const title = "Info Title";
    const message = "This is an info message";

    store.info({ title, message });
    const toast = store.toasts[0];

    expect(store.all).toEqual([toast]);
  });

  // The setInterval is just not working in the test
  it("should count down and hide a toast", async () => {
    const store = useToastStore();
    const title = "Info Title";
    const message = "This is an info message";

    vi.useFakeTimers();

    store.info({ title, message });
    const toast = store.toasts[0];

    expect(toast.countdown).toBe(100);

    await vi.advanceTimersByTimeAsync(3100); // >3 seconds

    expect(toast.hide).toBe(true);
    expect(toast.countdown).toBeLessThanOrEqual(0);

    vi.useRealTimers();
  });

});

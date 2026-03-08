import { defineStore } from "pinia";

export type ToastState = {
  toasts: Toast[];
  total: number;
};

export enum ToastKind {
  Success = "success",
  Error = "error",
  Warning = "warning",
  Info = "info",
}

export type Toast = {
  id: number;
  title: string;
  message: string;
  hide: boolean;
  created: Date;
  countdown: number;
  kind: ToastKind;
};

export const useToastStore = defineStore("toast", {
  state: (): ToastState => ({
    toasts: [],
    total: 0, // Is just incremented for each new incoming toast.
  }),
  actions: {
    info({ title, message }: { title: string; message: string }) {
      this.add({ title, message, kind: ToastKind.Info });
    },
    error({ title, message }: { title: string; message: string }) {
      this.add({ title, message, kind: ToastKind.Error });
    },
    warn({ title, message }: { title: string; message: string }) {
      this.add({ title, message, kind: ToastKind.Warning });
    },
    success({ title, message }: { title: string; message: string }) {
      this.add({ title, message, kind: ToastKind.Success });
    },
    add({ title, message, kind }: { title: string; message: string; kind?: ToastKind }) {
      // The animation can also be implemented with pure CSS, but that free us from this state update logic.
      const toast: Toast = {
        id: ++this.total,
        title,
        message,
        kind: kind || ToastKind.Info,
        hide: false,
        created: new Date(),
        countdown: 100,
      };
      const i = this.toasts.push(toast) - 1;

      const toastDisplayDurationMs = 3000;
      const id = setInterval(() => {
        const dtMS = new Date().getTime() - toast.created.getTime();
        if (!this.toasts[i]) clearInterval(id);
        this.toasts[i]!.countdown = 100 - (dtMS / toastDisplayDurationMs) * 100;
        if (this.toasts[i]!.countdown <= 0) {
          clearInterval(id);
          this.toasts[i]!.hide = true;
        }
      }, toastDisplayDurationMs / 10);
    },
    destroy(toast: Toast) {
      const i = this.toasts.findIndex((x: Toast) => x === toast);
      if (i !== -1) {
        this.toasts.splice(i, 1);
      }
    },
    hide(toast: Toast) {
      const i = this.toasts.findIndex((x: Toast) => x === toast);
      if (i !== -1) {
        this.toasts[i] = { ...this.toasts[i]!, hide: true };
      }
    },
  },
  getters: {
    all: (state: ToastState): Toast[] => {
      return state.toasts;
    },
  },
});

<template>
  <modal :scroll-top="validations.length > 0" :show="showModal" @close="emit('close')">
    <template v-slot:header>
      <div class="d-flex justify-content-between">
        <h5 class="modal-title">{{ formState.myTitle }}</h5>
      </div>
    </template>
    <template v-slot:body>
      <form :class="{ saving: 'disabled' }">
        <AppAlert style="font-size: 0.9rem" :alert-type="AlertType.Error" v-if="validations.length > 0">
          <CheckList :items="validations" />
        </AppAlert>

        <div>
          <label :for="`${id}_url`" class="form-label fw-bold">URL</label>
          <div class="input-group mb-3">
            <input :id="`${id}_url`" type="url" required autocomplete="off" ref="streamUrl" class="form-control" :name="`${id}_url`" v-model="formState.myUrl" @input="recommendChannelName" />
            <button class="btn btn-outline-secondary" type="button" name="button-addon1" @click="paste">Paste</button>
          </div>
        </div>

        <div class="mb-3">
          <label :for="`${id}_display`" class="form-label fw-bold">Display name</label>
          <input :id="`${id}_display`" pattern="^[^\s\\]+(\s{1}[^\s\\]+)*$" type="text" required autocapitalize="off" autocomplete="off" class="form-control" :name="`${id}_display`" v-model="formState.myDisplayName" />
          <div class="fs-6 my-2">Displayed as stream name. Can be changed at any time. No leading and trailing white spaces allowed, or double white spaces.</div>
        </div>

        <div class="mb-3">
          <label :for="`${id}_channel`" class="form-label fw-bold">Channel name</label>
          <input :id="`${id}_channel`" pattern="^[_a-z0-9]+$" type="text" required autocapitalize="off" autocomplete="off" class="form-control" :name="`${id}_channel`" :disabled="channelDisabled" v-model="formState.myChannelName" />
          <div v-if="!channelDisabled" class="fs-6 my-2">
            Only letters
            <span class="badge bg-info">a-z</span>
            , numbers
            <span class="badge bg-info">a-z</span>
            , and underscores
            <span class="badge bg-info">_</span>
            is allowed as channel name. This will also be the parent folder name for all recordings of this service.
          </div>
          <div v-else class="fs-6 my-2">This field is the file system folder name and cannot be changed.</div>
        </div>

        <div class="mb-3">
          <label :for="`${id}_minDuration`" class="form-label fw-bold">Minimum recording duration (minutes)</label>
          <input :id="`${id}_minDuration`" type="number" required min="0" class="form-control" :name="`${id}_minDuration`" v-model="formState.myMinDuration" />
          <div class="fs-6 my-2">Under this duration (min) a recording is discarded (considered too short)</div>
        </div>

        <div class="mb-3">
          <label :for="`${id}_skip`" class="form-label fw-bold">Skip start (seconds)</label>
          <input :id="`${id}_skip`" type="number" required min="0" class="form-control" :name="`${id}_skip`" v-model="formState.mySkipStart" />
          <div class="fs-6 my-2">Some broadcasters have certain number of seconds ads at the video start. Define how many seconds at start should be skipped when recording, i.e. for Twitch 15s.</div>
        </div>

        <div class="mb-3">
          <div class="form-check form-switch">
            <input :id="`${id}_isPaused`" type="checkbox" :checked="formState.myIsPaused" class="form-check-input" :name="`${id}_isPaused`" v-model="formState.myIsPaused" />
            <label class="form-check-label" :for="`${id}_isPaused`">Pause Recording</label>
          </div>
          <div class="fs-6 my-2">Do not record as long as paused.</div>
        </div>
      </form>
    </template>

    <template v-slot:footer>
      <button class="btn btn-primary" @click="save" :disabled="isSaving">
        <span class="spinner-border spinner-border-sm text-light" role="status" v-if="isSaving">
          <span class="visually-hidden">Loading...</span>
        </span>
        Save
      </button>
    </template>
  </modal>
</template>

<script setup lang="ts">
import Modal from "./ModalWindow.vue";
import { computed,  reactive, ref, watch  } from "vue";
import { randomString } from "../../utils/math";
import AppAlert from "../AppAlert.vue";
import { createValidator, type ValidationMessage } from "../../utils/validator";
import CheckList from "../CheckList.vue";
import { AlertType } from "../../types/alert";
import type { ChannelUpdate } from "../../types/channel";

// --------------------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------------------

const props = defineProps<{
  channelDisabled?: boolean;
  clear: boolean;
  isPaused: boolean;
  show: boolean;
  saving: boolean;
  channelId?: number;
  channelName?: string;
  displayName?: string;
  url?: string;
  skipStart?: number;
  minDuration?: number;
  title: string;
}>();

// --------------------------------------------------------------------------------------
// Declarations
// --------------------------------------------------------------------------------------

const id = randomString();
const channelParser = /^[a-z_0-9]+$/i;

const formState = reactive({
  myIsPaused: props.isPaused,
  myTitle: props.title,
  myUrl: props.url || "",
  myDisplayName: props.displayName || "",
  myChannelName: props.channelName || "",
  mySkipStart: props.skipStart || 0,
  myMinDuration: props.minDuration || 15});

const isSaving = ref(false);

const validatorChannelName = "channelName";
const validatorChannelDisplayName = "channelDisplayName";
const validatorChannelUrl = "channelUrl";

const validator = createValidator([
  {
    name: validatorChannelName,
    validator: (val: string) => /^[_a-z0-9]+$/i.test(val),
    validMessage: "Channel name valid",
    invalidMessage: "Channel name invalid"},
  {
    name: validatorChannelDisplayName,
    validator: (val: string) => /^[^\s\\]+(\s[^\s\\]+)*$/i.test(val),
    validMessage: "Channel display name valid",
    invalidMessage: "Channel display name invalid"},
  {
    name: validatorChannelUrl,
    validator: (val: string) => {
      let url;

      try {
        url = new URL(val);
      } catch {
        return false;
      }

      return url.protocol === "http:" || url.protocol === "https:";
    },
    validMessage: "URL valid",
    invalidMessage: "URL invalid"},
]);

const streamUrl = ref<HTMLInputElement | null>(null);
const showModal = computed(() => props.show);

const validations = ref<{ message: string; checked: boolean }[]>([]);

// --------------------------------------------------------------------------------------
// Emits
// --------------------------------------------------------------------------------------

const emit = defineEmits<{
  (e: "save", value: ChannelUpdate): void;
  (e: "close"): void;
}>();

// --------------------------------------------------------------------------------------
// Watchers
// --------------------------------------------------------------------------------------

watch(
  () => props.clear,
  (val) => {
    if (val) {
      validations.value = [];
      formState.myChannelName = "";
      formState.myUrl = "";
      formState.myDisplayName = "";
      formState.mySkipStart = 0;
      formState.myMinDuration = 20;
    }
  },
);

watch(showModal, (val) => {
  if (val) {
    isSaving.value = false;
    streamUrl.value?.focus();
  }
});

// Listen for form changes.
// Reason for all this: This is an optimisation to only have one modal instance.
watch(
  () => props,
  (newProps) => {
    formState.myIsPaused = newProps.isPaused;
    formState.myTitle = newProps.title;
    formState.myUrl = newProps.url || "";
    formState.myDisplayName = newProps.displayName || "";
    formState.myChannelName = newProps.channelName || "";
    formState.mySkipStart = newProps.skipStart || 0;
    formState.myMinDuration = newProps.minDuration || 15;
  },
  { deep: true },
);

watch(
  () => props.saving,
  (val: boolean) => (isSaving.value = val),
);

// --------------------------------------------------------------------------------------
// Methods
// --------------------------------------------------------------------------------------

const paste = async () => {
  formState.myUrl = await navigator.clipboard.readText();
  recommendChannelName();
};

const recommendChannelName = () => {
  if (props.channelDisabled) {
    return;
  }

  const find = formState.myUrl
    .toLowerCase()
    .split("/")
    .find((s) => channelParser.test(s));

  if (find) {
    if (formState.myChannelName === "") {
      formState.myChannelName = find;
    }
    if (formState.myDisplayName === "") {
      formState.myDisplayName = find;
    }
  }
};

const formValid = (): { validations: ValidationMessage[]; isValid: boolean } => {
  const results = validator.validateAll([
    { name: validatorChannelName, value: formState.myChannelName },
    { name: validatorChannelDisplayName, value: formState.myDisplayName },
    { name: validatorChannelUrl, value: formState.myUrl },
  ]);

  const isValid = results.reduce((previousValue, currentValue) => previousValue && currentValue.isValid, true);

  return {
    validations: results,
    isValid};
};

const save = () => {
  try {
    isSaving.value = true;
    validations.value = [];
    const validationResult = formValid();

    if (!validationResult.isValid) {
      validations.value = validationResult.validations.map(({ message, isValid }) => ({
        message: message,
        checked: isValid}));
      return;
    }

    // Valid
    emit("save", {
      isPaused: formState.myIsPaused,
      channelId: props.channelId!,
      channelName: formState.myChannelName,
      url: formState.myUrl,
      displayName: formState.myDisplayName,
      skipStart: formState.mySkipStart,
      minDuration: formState.myMinDuration});
  } catch (error) {
    console.error(error);
    alert(error);
  } finally {
    isSaving.value = false;
  }
};
</script>

<style scoped></style>


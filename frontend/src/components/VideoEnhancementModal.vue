<template>
  <ModalWindow :show="show" :scroll-top="true" :show-footer="true">
    <template #header>Enhance video</template>
    <template #body>
      <AppAlert style="font-size: 0.9rem" :alert-type="AlertType.Error" v-if="validations.length > 0">
        <CheckList :items="validations" />
      </AppAlert>
      <div class="enhancement-form">
        <div class="mb-3">
          <label class="form-label">Target Resolution</label>
          <select v-model="targetResolution" class="form-select" required>
            <option value="">Select resolution</option>
            <option v-for="res in enhancementDescriptions?.resolutions" :key="res.resolution" :value="res.resolution">{{ res.resolution }} - {{ res.dimensions }} ({{ res.useCase }})</option>
          </select>
          <small v-if="enhancementDescriptions?.resolutions" class="form-text text-muted">
            {{ enhancementDescriptions.resolutions.find((resolution) => resolution.resolution === targetResolution)?.description }}
          </small>
        </div>

        <div class="mb-3">
          <label class="form-label">Encoding Preset</label>
          <select v-model="encodingPreset" class="form-select" required>
            <option value="">Select preset</option>
            <option v-for="preset in enhancementDescriptions?.presets" :key="preset.preset" :value="preset.preset">{{ preset.label }} - {{ preset.encodeSpeed }} ({{ preset.description }})</option>
          </select>
        </div>

        <div class="mb-3">
          <label class="form-label">Quality (CRF) - {{ crf ?? "Not set" }}</label>
          <div class="d-flex gap-2 align-items-center">
            <input v-model.number="crf" type="range" step="1" class="form-range" :min="getCrfMinValue()" :max="getCrfMaxValue()" />
            <button v-if="enhancementDescriptions?.crfValues" type="button" class="btn btn-sm btn-outline-secondary" @click="setCrfRecommended">Recommended</button>
          </div>
          <small class="form-text text-muted">
            {{ getCrfDescription() }}
          </small>
        </div>

        <div v-if="enhancementDescriptions?.filters?.denoiseStrength" class="mb-3">
          <label class="form-label">Denoise Strength - {{ denoiseStrength }}</label>
          <div class="d-flex gap-2 align-items-center">
            <input v-model.number="denoiseStrength" type="range" step="0.1" class="form-range" :min="enhancementDescriptions.filters.denoiseStrength.minValue" :max="enhancementDescriptions.filters.denoiseStrength.maxValue" />
            <button type="button" class="btn btn-sm btn-outline-secondary" @click="denoiseStrength = enhancementDescriptions.filters.denoiseStrength.recommended">Recommended</button>
          </div>
          <small class="form-text text-muted">
            {{ enhancementDescriptions.filters.denoiseStrength.description }}
          </small>
        </div>

        <div v-if="enhancementDescriptions?.filters?.sharpenStrength" class="mb-3">
          <label class="form-label">Sharpen Strength - {{ sharpenStrength }}</label>
          <div class="d-flex gap-2 align-items-center">
            <input v-model.number="sharpenStrength" type="range" step="0.1" class="form-range" :min="enhancementDescriptions.filters.sharpenStrength.minValue" :max="enhancementDescriptions.filters.sharpenStrength.maxValue" />
            <button type="button" class="btn btn-sm btn-outline-secondary" @click="sharpenStrength = enhancementDescriptions.filters.sharpenStrength.recommended">Recommended</button>
          </div>
          <small class="form-text text-muted">
            {{ enhancementDescriptions.filters.sharpenStrength.description }}
          </small>
        </div>

        <div v-if="enhancementDescriptions?.filters?.applyNormalize" class="mb-3 form-check">
          <input v-model="applyNormalize" type="checkbox" class="form-check-input" id="normalizeCheck" />
          <label class="form-check-label" for="normalizeCheck">
            {{ enhancementDescriptions.filters.applyNormalize.name }}
          </label>
          <small class="form-text text-muted d-block">
            {{ enhancementDescriptions.filters.applyNormalize.description }}
          </small>
        </div>

        <div v-if="estimatedSize" class="alert alert-info" role="alert">Estimated file size: {{ formatFileSize(estimatedSize) }}</div>
      </div>
    </template>
    <template #footer>
      <div class="d-flex gap-2">
        <button type="button" class="btn btn-secondary" @click="onCancel">Cancel</button>
        <busy-button :busy="busy" @click="onConfirm" caption="Confirm" button-type="button" position="left" />
      </div>
    </template>
  </ModalWindow>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { DatabaseRecording as RecordingResponse, RequestsEnhanceRequestEncodingPresetEnum, RequestsEnhanceRequestTargetResolutionEnum, RequestsEstimateEnhancementRequestEncodingPresetEnum, RequestsEstimateEnhancementRequestTargetResolutionEnum, ResponsesEnhancementDescriptions } from "../services/api/v1/MediaSinkClient";
import { createClient } from "../services/api/v1/ClientFactory";
import { createValidator, type ValidationMessage } from "../utils/validator";
import AppAlert from "./AppAlert.vue";
import CheckList from "./CheckList.vue";
import { AlertType } from "../types/alert";
import ModalWindow from "../components/modals/ModalWindow.vue";
import BusyButton from "../components/controls/BusyButton.vue";

// --------------------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------------------

const props = defineProps<{
  show: boolean;
  recording: RecordingResponse;
}>();

// --------------------------------------------------------------------------------------
// Emits
// --------------------------------------------------------------------------------------

const emit = defineEmits<{
  (e: "close"): void;
}>();

// --------------------------------------------------------------------------------------
// State
// --------------------------------------------------------------------------------------

const targetResolution = ref("");
const encodingPreset = ref("");
const crf = ref<number | undefined>(undefined);
const denoiseStrength = ref(1);
const sharpenStrength = ref(0);
const applyNormalize = ref(false);

const enhancementDescriptions = ref<ResponsesEnhancementDescriptions | null>(null);
const estimatedSize = ref<number | null>(null);
const estimating = ref(false);
const busy = ref(false);

// Validation
const validatorTargetResolution = "targetResolution";
const validatorEncodingPreset = "encodingPreset";

const validator = createValidator([
  {
    name: validatorTargetResolution,
    validator: (val: string) => {
      if (!val) return false;
      return enhancementDescriptions.value?.resolutions?.some((r) => r.resolution === val) ?? false;
    },
    validMessage: "Target resolution valid",
    invalidMessage: "Target resolution invalid",
  },
  {
    name: validatorEncodingPreset,
    validator: (val: string) => {
      if (!val) return false;
      return enhancementDescriptions.value?.presets?.some((p) => p.preset === val) ?? false;
    },
    validMessage: "Encoding preset valid",
    invalidMessage: "Encoding preset invalid",
  },
]);

const validations = ref<{ message: string; checked: boolean }[]>([]);

// --------------------------------------------------------------------------------------
// Watchers
// --------------------------------------------------------------------------------------

watch(targetResolution, () => onEnhancementChange());
watch(encodingPreset, () => onEnhancementChange());
watch(crf, () => onEnhancementChange());
watch(denoiseStrength, () => onEnhancementChange());
watch(sharpenStrength, () => onEnhancementChange());
watch(applyNormalize, () => onEnhancementChange());

watch(
  () => props.show,
  async (newShow) => {
    if (newShow) {
      await loadEnhancementOptions();
    }
  },
);

// --------------------------------------------------------------------------------------
// Helper Methods
// --------------------------------------------------------------------------------------

const getCrfMinValue = (): number => {
  if (!enhancementDescriptions.value?.crfValues || enhancementDescriptions.value.crfValues.length === 0) {
    return 0;
  }
  return Math.min(...enhancementDescriptions.value.crfValues.map((c) => c.value));
};

const getCrfMaxValue = (): number => {
  if (!enhancementDescriptions.value?.crfValues || enhancementDescriptions.value.crfValues.length === 0) {
    return 51;
  }
  return Math.max(...enhancementDescriptions.value.crfValues.map((c) => c.value));
};

const getCrfDescription = (): string => {
  if (!enhancementDescriptions.value?.crfValues) return "Lower values = higher quality but larger file size";
  const selected = enhancementDescriptions.value.crfValues.find((c) => c.value === crf.value);
  if (selected) {
    return `${selected.description} (Quality: ${selected.quality}, Compression ratio: ${selected.approxRatio})`;
  }
  return "Lower values = higher quality but larger file size";
};

const setCrfRecommended = () => {
  if (!enhancementDescriptions.value?.crfValues) return;
  const recommended = enhancementDescriptions.value.crfValues.find((c) => c.label.toLowerCase().includes("recommended"));
  if (recommended) {
    crf.value = recommended.value;
  }
};

const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return "0 Bytes";
  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
};

const resetForm = () => {
  applyNormalize.value = false;
  crf.value = undefined;
  denoiseStrength.value = 1;
  encodingPreset.value = "";
  sharpenStrength.value = 0;
  targetResolution.value = "";
  estimatedSize.value = null;
  validations.value = [];
};

const formValid = (): { validations: ValidationMessage[]; isValid: boolean } => {
  const results = validator.validateAll([
    { name: validatorTargetResolution, value: targetResolution.value },
    { name: validatorEncodingPreset, value: encodingPreset.value },
  ]);

  const isValid = results.reduce((previousValue: boolean, currentValue: ValidationMessage) => previousValue && currentValue.isValid, true);

  return {
    validations: results,
    isValid,
  };
};

// --------------------------------------------------------------------------------------
// API Methods
// --------------------------------------------------------------------------------------

const loadEnhancementOptions = async () => {
  try {
    const client = createClient();
    enhancementDescriptions.value = await client.videos.enhanceDescriptionsList();

    // Initialize form fields with recommended values from API
    if (enhancementDescriptions.value?.filters) {
      const filters = enhancementDescriptions.value.filters;
      applyNormalize.value = filters.applyNormalize.recommended;
      denoiseStrength.value = filters.denoiseStrength.recommended;
      sharpenStrength.value = filters.sharpenStrength.recommended;
    }

    // Initialize CRF with the recommended value from API
    setCrfRecommended();

    // Trigger estimation with recommended values
    await onEnhancementChange();
  } catch (ex) {
    alert("Failed to load enhancement options: " + ex);
  }
};

const onEnhancementChange = async () => {
  try {
    estimating.value = true;
    const client = createClient();
    const estimate = await client.videos.estimateEnhancementCreate(
      { id: props.recording.recordingId },
      {
        applyNormalize: applyNormalize.value,
        crf: crf.value,
        denoiseStrength: denoiseStrength.value,
        encodingPreset: encodingPreset.value as RequestsEstimateEnhancementRequestEncodingPresetEnum,
        sharpenStrength: sharpenStrength.value,
        targetResolution: targetResolution.value as RequestsEstimateEnhancementRequestTargetResolutionEnum,
      },
    );
    estimatedSize.value = estimate.estimatedFileSize || 0;
  } catch (ex) {
    console.error("Failed to estimate file size:", ex);
    estimatedSize.value = null;
  } finally {
    estimating.value = false;
  }
};

const enhance = async () => {
  try {
    busy.value = true;
    validations.value = [];

    // Validate form
    const validationResult = formValid();

    if (!validationResult.isValid) {
      validations.value = validationResult.validations.map(({ message, isValid }) => ({
        message: message,
        checked: isValid,
      }));
      busy.value = false;
      return;
    }

    const client = createClient();
    await client.videos.enhanceCreate(
      { id: props.recording.recordingId },
      {
        applyNormalize: applyNormalize.value,
        crf: crf.value,
        denoiseStrength: denoiseStrength.value,
        encodingPreset: encodingPreset.value as RequestsEnhanceRequestEncodingPresetEnum,
        recordingId: props.recording.recordingId,
        sharpenStrength: sharpenStrength.value,
        targetResolution: targetResolution.value as RequestsEnhanceRequestTargetResolutionEnum,
      },
    );

    // Reset form and close
    resetForm();
    emit("close");
  } catch (ex) {
    alert("Failed to enhance video: " + (ex as { error?: string }).error);
  } finally {
    busy.value = false;
  }
};

// --------------------------------------------------------------------------------------
// Event Handlers
// --------------------------------------------------------------------------------------

const onConfirm = () => {
  enhance();
};

const onCancel = () => {
  validations.value = [];
  resetForm();
  emit("close");
};
</script>

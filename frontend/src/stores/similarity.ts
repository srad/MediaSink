import { defineStore } from "pinia";
import { ref } from "vue";
import type {
  DbRecording,
  ResponsesSimilarityGroupsResponse,
  ResponsesVisualSearchResponse,
} from "@/services/api/v1/MediaSinkClient";

export const useSimilarityStore = defineStore(
  "similarity",
  () => {
    const tab = ref<"search" | "group">("search");

    // Search
    const searchSource = ref<"upload" | "library">("upload");
    const searchSimilarity = ref(0.8);
    const searchLimit = ref(50);
    const searchResults = ref<ResponsesVisualSearchResponse | null>(null);
    const selectedRecording = ref<DbRecording | null>(null);
    const previewDataUrl = ref<string | null>(null);

    // Group
    const groupSimilarity = ref(0.8);
    const pairLimit = ref(20000);
    const includeSingletons = ref(false);
    const groupResults = ref<ResponsesSimilarityGroupsResponse | null>(null);

    return {
      tab,
      searchSource,
      searchSimilarity,
      searchLimit,
      searchResults,
      selectedRecording,
      previewDataUrl,
      groupSimilarity,
      pairLimit,
      includeSingletons,
      groupResults,
    };
  },
  { persist: true } as any,
);

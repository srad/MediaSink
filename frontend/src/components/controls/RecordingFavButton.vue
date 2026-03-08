<template>
  <div>
    <FavButton v-if="!busy" :data="{}" :faved="fav" @fav="bookmark" @unfav="bookmark" />
    <span v-else class="spinner-border spinner-border-sm" aria-hidden="true"></span>
  </div>
</template>

<script setup lang="ts">
import FavButton from "./FavButton.vue";
import { ref } from "vue";
import { createClient } from "../../services/api/v1/ClientFactory";

const props = defineProps<{
  bookmarked: boolean;
  recordingId: number;
}>();

const busy = ref(false);
const fav = ref(props.bookmarked);

const bookmark = () => {
  const client = createClient();
  busy.value = true;
  const fn = fav.value ? client.videos.unfavPartialUpdate : client.videos.favPartialUpdate;
  fn({ id: props.recordingId })
    .then(() => (fav.value = !fav.value))
    .catch((res) => alert(res.error))
    .finally(() => (busy.value = false));
};
</script>

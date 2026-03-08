<template>
  <div>
    <FavButton v-if="!busy" :data="{}" :faved="fav" @fav="(data, event) => bookmark(event)" @unfav="(data, event) => bookmark(event)" @click.stop />
    <span v-else class="spinner-border spinner-border-sm" aria-hidden="true"></span>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import FavButton from "./FavButton.vue";
import { createClient } from "../../services/api/v1/ClientFactory";

// --------------------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------------------

const props = defineProps<{
  bookmarked: boolean;
  channelId: number;
}>();

// --------------------------------------------------------------------------------------
// Declarations
// --------------------------------------------------------------------------------------

const busy = ref(false);
const fav = ref(props.bookmarked);

// --------------------------------------------------------------------------------------
// Watchers
// --------------------------------------------------------------------------------------

watch(
  () => props.bookmarked,
  (val) => (fav.value = val),
);

// --------------------------------------------------------------------------------------
// Methods
// --------------------------------------------------------------------------------------

const bookmark = (event: Event) => {
  event.stopPropagation();
  busy.value = true;
  const client = createClient();
  const fn = fav.value ? client.channels.unfavPartialUpdate : client.channels.favPartialUpdate;

  fn({ id: props.channelId })
    .then(() => (fav.value = !fav.value))
    .catch((res) => alert(res.error))
    .finally(() => (busy.value = false));
};
</script>

<template>
  <div ref="dropdownRef" class="user-dropdown d-flex align-content-center">
    <button class="dropdown-toggle text-secondary fs-5" @click="toggleDropdown">
      <i class="bi bi-person-circle"></i>
    </button>

    <ul class="dropdown-menu dropdown-menu-end border-primary bg-secondary" :class="{ show: isOpen }" @mouseleave="isOpen = false" style="width: 180px">
      <li class="dropdown-item rounded-top-2">
        <a href="#" class="rounded-top-2 nav-link bg-transparent">
          <DarkModelToggleButton :checkbox="true" />
        </a>
      </li>
      <li class="dropdown-item" v-for="link in links" :key="link.link">
        <RouterLink :to="link.link" class="nav-link d-flex bg-transparent justify-content-between" @click="isOpen = false">
          <span>{{ link.title }}</span>
          <i class="bi" :class="link.icon"></i>
        </RouterLink>
      </li>
      <li class="dropdown-item rounded-bottom-2">
        <a href="#" class="d-flex justify-content-between bg-transparent nav-link rounded-bottom-2" @click="emit('logout')">
          <span>Logout</span>
          <i class="bi bi-door-open-fill"></i>
        </a>
      </li>
    </ul>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, onBeforeUnmount } from "vue";
import DarkModelToggleButton from "../DarkModelToggleButton.vue";

const emit = defineEmits<{
  (e: "logout"): void;
}>();

const links = [
  { title: "Admin", icon: "bi bi-sliders", link: "/admin" },
  { title: "System Info", icon: "bi bi-info-circle-fill", link: "/info" },
  { title: "Processes", icon: "bi bi-cpu-fill", link: "/processes" },
  { title: "Monitoring", icon: "bi bi-binoculars-fill", link: "/monitoring" },
  { title: "Favourites", icon: "bi bi-heart-fill", link: "/bookmarks" },
];

// Dropdown state
const isOpen = ref(false);
const dropdownRef = ref<HTMLElement | null>(null);

function toggleDropdown() {
  isOpen.value = !isOpen.value;
}

function handleClickOutside(event: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    isOpen.value = false;
  }
}

onMounted(() => {
  document.addEventListener("click", handleClickOutside);
});

onBeforeUnmount(() => {
  document.removeEventListener("click", handleClickOutside);
});
</script>

<style scoped>
.user-dropdown {
  position: relative;
  display: inline-block;
}

.dropdown-toggle {
  background: none;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
}

.avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  margin-right: 0.5rem;
}

.dropdown-menu {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 0.5rem;
  background: white;
  border: 1px solid #ddd;
  border-radius: 6px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  list-style: none;
  padding: 0;
  min-width: 150px;
  z-index: 1000;
}

.dropdown-menu li a {
  display: block;
  padding: 0.5rem 1rem;
  color: #333;
  text-decoration: none;
}

.dropdown-menu li a:hover {
  background-color: #f8f9fa;
}

.divider {
  height: 1px;
  margin: 0.5rem 0;
  background-color: #eee;
}
</style>

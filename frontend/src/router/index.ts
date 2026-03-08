import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "../stores/auth";

// Lazy load views with code splitting for better performance
// Login/Register load eagerly since they're shown first
import LoginView from "../views/LoginView.vue";
import RegisterView from "../views/RegisterView.vue";
import StreamsView from "../views/StreamsView.vue";

const routes = [
  { path: "", redirect: "/streams" },
  { path: "/", redirect: "/streams/live/tab" },
  { path: "/login", name: "Login", component: LoginView, meta: { title: "Login", layout: "auth" } },
  { path: "/register", name: "Register", component: RegisterView, meta: { title: "Register", layout: "auth" } },
  {
    path: "/channel/:id/:channelName",
    name: "Channel",
    component: () => import("@/views/ChannelView.vue"),
    props: true,
    meta: { title: "Channel" },
  },
  { path: "/channels", name: "Channels", component: () => import("../views/ChannelsView.vue"), meta: { title: "Channels" } },
  { path: "/admin", name: "Admin", component: () => import("../views/AdminView.vue"), meta: { title: "Admin" } },
  { path: "/info", name: "Info", component: () => import("../views/InfoView.vue"), meta: { title: "Info" } },
  { path: "/processes", name: "Processes", component: () => import("../views/ProcessView.vue"), meta: { title: "Processes" } },
  { path: "/monitoring", name: "Monitoring", component: () => import("../views/MonitoringView.vue"), meta: { title: "Monitoring" } },

  { path: "/jobs", redirect: "/jobs/processing" },
  { path: "/jobs/:tab", name: "Job", component: () => import("../views/JobView.vue"), meta: { title: "Jobs" } },
  {
    path: "/streams/:tab/tab/:tag?",
    name: "Streams",
    component: StreamsView,
    meta: { title: "Stream" },
  },
  { path: "/filter", name: "Filter", component: () => import("../views/FilterView.vue"), meta: { title: "Latest" } },
  { path: "/random", name: "Random", component: () => import("../views/RandomView.vue"), meta: { title: "Random" } },
  { path: "/recordings/:id", name: "Video", component: () => import("../views/VideoView.vue"), meta: { layout: "fullscreen" } },
  { path: "/logs", name: "Log", component: () => import("../views/LogsView.vue"), meta: { title: "Logs" } },
  { path: "/bookmarks", name: "Bookmark", component: () => import("../views/BookmarksView.vue"), meta: { title: "Favourites" } },
  { path: "/similarity", name: "Similarity", component: () => import("../views/SimilarityView.vue"), meta: { title: "Similarity" } },
  { path: "/:pathMatch(.*)*", redirect: "/streams/live/tab" },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    return new Promise((resolve) => {
      setTimeout(() => {
        const contentElement = document.querySelector("body");
        if (contentElement) {
          contentElement.scrollTo({ top: 0, behavior: "instant" });
        }
        resolve(savedPosition || { top: 0 });
      }, 100); // Small delay to ensure the content has loaded
    });
  },
});

router.afterEach((to) => {
  //@ts-expect-error Vue nonsense error
  document.title = to.meta.title || "StreamSink";
});

router.beforeEach((to, from, next) => {
  const authStore = useAuthStore();
  const publicPages = ["/login", "/register"];
  const isLoggedIn = authStore.isLoggedIn;
  const isPublicPage = publicPages.includes(to.path);

  // If not logged in and trying to access protected page, redirect to login
  if (!isLoggedIn && !isPublicPage) {
    next("/login");
    return;
  }

  // If logged in and trying to access public page (login/register), redirect to home
  if (isLoggedIn && isPublicPage) {
    next("/");
    return;
  }

  // Allow navigation
  next();
});

export default router;

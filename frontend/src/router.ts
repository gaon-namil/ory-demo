import { createRouter, createWebHistory } from "vue-router";
import Home from "./pages/Home.vue";
import AppPage from "./pages/AppPage.vue";

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", component: Home },
    { path: "/app", component: AppPage },
  ],
});
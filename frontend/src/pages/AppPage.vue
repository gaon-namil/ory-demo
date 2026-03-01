<script setup lang="ts">
import { onMounted, ref } from "vue";

const status = ref<number>(0);
const body = ref<string>("");

const API = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

function logout() {
  window.location.href = `${API}/api/logout`;
}

onMounted(async () => {
  const res = await fetch(`${API}/api/me`, { credentials: "include" });
  status.value = res.status;
  body.value = await res.text();
});
</script>

<template>
  <div style="padding:24px;font-family:system-ui;">
    <h1>/app</h1>
    <p>GET /api/me => {{ status }}</p>
    <pre>{{ body }}</pre>
    <button @click="logout" style="margin-right:8px;">Logout</button>
    <a href="/">Back</a>
  </div>
</template>

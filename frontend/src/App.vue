<script setup lang="ts">
function login() {
  window.location.href = "http://localhost:8080/api/login";
}
</script>

<template>
  <div style="padding:24px;font-family:system-ui;">
    <h1>Vue Frontend OK</h1>
    <button @click="login">Login</button>
  </div>
</template>
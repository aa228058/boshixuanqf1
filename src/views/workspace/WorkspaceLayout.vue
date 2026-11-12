<template>
  <div class="h-full min-h-0">
    <router-view />
  </div>
</template>

<script setup lang="ts">
// Workspace routes are nested under the global Layout.vue, which already owns
// the single responsive MobileTabBar. Do not render a second copy here.
</script>

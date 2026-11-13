<template>
  <div class="h-full min-h-0">
    <!-- Route params are part of the page identity: task detail must remount when switching task IDs
       so its task-scoped SSE stream, pagination requests and timers are disposed together. -->
    <router-view :key="String($route.params.accountId ?? '') + ':' + String($route.params.id ?? $route.name ?? '')" />
  </div>
</template>

<script setup lang="ts">
// Workspace routes are nested under the global Layout.vue, which already owns
// the single responsive MobileTabBar. Do not render a second copy here.
</script>

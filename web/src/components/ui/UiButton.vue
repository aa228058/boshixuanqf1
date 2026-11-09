<template>
  <t-button
    v-bind="$attrs"
    :theme="tdesignTheme"
    :variant="tdesignVariant"
    :size="size"
    :class="[
      'ui-t-button',
      `ui-t-button--${tone}`,
      size === 'small' && 'ui-t-button--sm',
    ]"
  >
    <slot />
  </t-button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  tone?: 'primary' | 'secondary' | 'danger' | 'text'
  size?: 'small' | 'medium' | 'large'
}>(), {
  tone: 'primary',
  size: 'medium',
})

const tdesignTheme = computed(() => {
  if (props.tone === 'danger') return 'danger'
  if (props.tone === 'secondary' || props.tone === 'text') return 'default'
  return 'primary'
})

const tdesignVariant = computed(() => {
  if (props.tone === 'secondary') return 'outline'
  if (props.tone === 'text') return 'text'
  return 'base'
})
</script>

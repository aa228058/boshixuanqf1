<template>
  <t-button
    v-bind="$attrs"
    :theme="tdesignTheme"
    :variant="tdesignVariant"
    :size="tdesignSize"
    :class="[
      'ui-t-button',
      semanticTone && `ui-t-button--${semanticTone}`,
      isCompact && 'ui-t-button--sm',
    ]"
  >
    <slot />
  </t-button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  tone?: 'primary' | 'secondary' | 'danger' | 'text'
  theme?: 'default' | 'primary' | 'danger' | 'warning' | 'success'
  variant?: 'base' | 'outline' | 'dashed' | 'text'
  size?: 'sm' | 'small' | 'medium' | 'large' | 'extra-small'
}>()

const semanticTone = computed(() => {
  if (props.tone) return props.tone
  if (props.theme === 'danger') return 'danger'
  if (props.variant === 'text') return 'text'
  if (props.theme === 'primary') return 'primary'
  if (props.theme === 'default' && props.variant === 'outline') return 'secondary'
  return null
})

const tdesignTheme = computed(() => {
  if (props.tone === 'danger' || props.tone === 'primary') return props.tone
  if (props.tone === 'secondary' || props.tone === 'text') return 'default'
  return props.theme ?? 'default'
})

const tdesignVariant = computed(() => {
  if (props.tone === 'secondary') return 'outline'
  if (props.tone === 'text') return 'text'
  if (props.tone === 'primary' || props.tone === 'danger') return 'base'
  return props.variant ?? 'base'
})

const isCompact = computed(() => props.size === 'sm' || props.size === 'small' || props.size === 'extra-small')
const tdesignSize = computed(() => isCompact.value ? 'small' : props.size ?? 'medium')
</script>

<template>
  <!-- 统一分组卡: 与更多页列表卡一致 — 表面色 + 细边框 + 20px 圆角, 内容区溢出裁切 -->
  <section class="rounded-[20px] overflow-hidden min-w-0"
           style="background: var(--t-surface-1); border: 1px solid var(--t-line);">
    <slot />
  </section>
</template>

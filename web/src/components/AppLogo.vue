<template>
  <!-- 线性描边目录树 Logo（与产品原型同款，单色蓝） -->
  <svg :width="size" :height="size" viewBox="0 0 48 48" fill="none" :stroke="color"
    stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round">
    <rect x="4" y="4" width="40" height="40" rx="10" />
    <circle cx="24" cy="15.5" r="3.2" />
    <path d="M24 18.7V28.8M13 22h22M13 22v6.8M35 22v6.8" />
    <circle cx="13" cy="32" r="3.2" />
    <circle cx="24" cy="32" r="3.2" />
    <circle cx="35" cy="32" r="3.2" />
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ size?: number; color?: string }>(), { size: 30, color: '#2F54EB' })
</script>

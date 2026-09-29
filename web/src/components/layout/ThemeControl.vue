<script setup lang="ts">
import { MonitorIcon, MoonIcon, SunIcon } from '@lucide/vue'
import { useTheme } from '@/features/theme'

const { preference, setPreference } = useTheme()
</script>

<template>
  <div class="theme-control" role="group" aria-label="外观主题">
    <button type="button" title="跟随系统" aria-label="跟随系统主题" :aria-pressed="preference === 'system'" @click="setPreference('system')"><MonitorIcon /></button>
    <button type="button" title="浅色模式" aria-label="浅色模式" :aria-pressed="preference === 'light'" @click="setPreference('light')"><SunIcon /></button>
    <button type="button" title="暗色模式" aria-label="暗色模式" :aria-pressed="preference === 'dark'" @click="setPreference('dark')"><MoonIcon /></button>
  </div>
</template>

<style scoped>
.theme-control{display:inline-grid;grid-template-columns:repeat(3,30px);gap:2px;padding:3px;border:1px solid var(--border);border-radius:6px;background:var(--muted);flex:none}
.theme-control button{display:grid;place-items:center;width:30px;height:30px;border-radius:4px;color:var(--muted-foreground);cursor:pointer}
.theme-control button:hover{color:var(--foreground)}
.theme-control button[aria-pressed=true]{background:var(--background);color:var(--primary);box-shadow:0 1px 4px #00000018}
.theme-control svg{width:16px;height:16px}
.theme-control button:focus-visible{outline:2px solid var(--ring);outline-offset:2px}
</style>

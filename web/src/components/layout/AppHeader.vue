<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { ArrowRightIcon, KeyRoundIcon, LogInIcon, ShieldCheckIcon, UserRoundIcon } from '@lucide/vue'
import { useAuthStore } from '@/features/auth/store'
import ThemeControl from '@/components/layout/ThemeControl.vue'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'

const auth = useAuthStore()
const route = useRoute()

onMounted(() => auth.ensureLoaded())
</script>

<template>
  <header class="sticky top-0 z-20 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/85">
    <div class="mx-auto flex h-16 max-w-7xl items-center justify-between gap-3 px-4 sm:px-6">
      <RouterLink class="flex min-w-0 items-center gap-2 font-semibold" to="/" aria-label="Glosc AI Password 首页">
        <span class="flex size-9 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground">
          <ShieldCheckIcon />
        </span>
        <span class="truncate"><span class="sm:hidden">Glosc</span><span class="hidden sm:inline">Glosc AI Password</span></span>
      </RouterLink>

      <nav class="flex items-center gap-2" aria-label="主导航">
        <Button variant="ghost" size="sm" as-child class="hidden md:inline-flex"><RouterLink to="/">首页</RouterLink></Button>
        <ThemeControl />

        <Spinner v-if="auth.loading && !auth.resolved" class="mx-2" aria-label="正在确认登录状态" />

        <template v-else-if="auth.isAuthenticated">
          <Button variant="outline" size="sm" as-child class="hidden sm:inline-flex"><RouterLink to="/profile"><UserRoundIcon data-icon="inline-start" />账号</RouterLink></Button>
          <Button size="sm" as-child><RouterLink to="/vault"><KeyRoundIcon data-icon="inline-start" class="hidden sm:block" /><span class="hidden sm:inline">打开密码库</span><span class="sm:hidden">密码库</span><ArrowRightIcon class="hidden sm:block" data-icon="inline-end" /></RouterLink></Button>
        </template>

        <Button v-else-if="route.name !== 'login'" size="sm" as-child><RouterLink to="/login"><LogInIcon data-icon="inline-start" class="hidden sm:block" />登录</RouterLink></Button>
      </nav>
    </div>
  </header>
</template>

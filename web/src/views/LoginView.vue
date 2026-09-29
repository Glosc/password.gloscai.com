<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { AlertCircleIcon, ArrowLeftIcon, KeyRoundIcon, LogInIcon, ShieldCheckIcon } from '@lucide/vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import { useAuthStore } from '@/features/auth/store'
import { notifyError } from '@/lib/message'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

// The callback redirects here with ?error=<reason> when a login fails.
const failureReasons: Record<string, string> = {
  access_denied: '你取消了本次授权。',
  expired_state: '登录请求已过期，请重新发起登录。',
  invalid_response: '身份服务返回的信息不完整，请重试。',
  exchange_failed: '登录暂时没有完成，请稍后重试。',
  userinfo_failed: '暂时无法读取账号信息，请稍后重试。',
  internal_error: '登录服务暂时不可用，请稍后重试。',
}

const redirectTo = computed(() => {
  const target = route.query.redirect_to
  // Require a same-site root-relative path. `startsWith('/')` alone also
  // accepts `//evil.com/...`, a protocol-relative URL that browsers resolve
  // against a different origin — reject that case explicitly to match the
  // strictness of the backend's sso.safeRedirect.
  const isSameSitePath = typeof target === 'string' && target.startsWith('/') && !target.startsWith('//')
  return isSameSitePath ? target : '/vault'
})

const failure = computed(() => {
  const reason = route.query.error
  if (typeof reason !== 'string' || reason === '') {
    return ''
  }
  return failureReasons[reason] ?? '登录未完成，请重新尝试。'
})

onMounted(async () => {
  if (failure.value) {
    notifyError(failure.value)
  }
  await auth.ensureLoaded()
  if (auth.isAuthenticated) {
    router.replace(redirectTo.value)
  }
})
</script>

<template>
  <div class="min-h-screen bg-background">
    <AppHeader />

    <main class="mx-auto flex max-w-md flex-col gap-6 px-4 py-12 sm:px-6 sm:py-20">
      <RouterLink to="/" class="inline-flex w-fit items-center gap-2 text-sm text-muted-foreground hover:text-foreground"><ArrowLeftIcon class="size-4" />返回首页</RouterLink>
      <Card class="rounded-md">
        <CardHeader>
          <span class="mb-3 flex size-10 items-center justify-center rounded-md bg-primary/10 text-primary"><ShieldCheckIcon class="size-6" /></span>
          <CardTitle class="text-2xl">登录你的密码库</CardTitle>
          <CardDescription>
            使用 Glosc AI 账号继续。登录后，再用你的安全 Key 解锁密码库。
          </CardDescription>
        </CardHeader>

        <CardContent class="flex flex-col gap-4">
          <Alert v-if="failure" variant="destructive">
            <AlertCircleIcon />
            <AlertTitle>登录未完成</AlertTitle>
            <AlertDescription>{{ failure }}</AlertDescription>
          </Alert>

          <Alert v-if="auth.unavailable">
            <AlertCircleIcon />
            <AlertTitle>登录暂不可用</AlertTitle>
            <AlertDescription>
              现在无法登录，请稍后再试。
            </AlertDescription>
          </Alert>

          <Button
            size="lg"
            class="w-full"
            :disabled="auth.loading || auth.unavailable"
            @click="auth.login(redirectTo)"
          >
            <Spinner v-if="auth.loading" data-icon="inline-start" />
            <LogInIcon v-else data-icon="inline-start" />
            继续使用 Glosc AI 账号
          </Button>
        </CardContent>

        <CardFooter class="flex-col items-start gap-3 border-t pt-5 text-xs text-muted-foreground">
          <span class="flex items-center gap-2"><LogInIcon class="size-4" />1. 登录 Glosc AI 账号</span>
          <span class="flex items-center gap-2"><KeyRoundIcon class="size-4" />2. 在本机输入安全 Key 解锁</span>
        </CardFooter>
      </Card>
      <p class="text-center text-xs leading-6 text-muted-foreground">安全 Key 不会上传。请勿向任何人透露安全 Key 或恢复码。</p>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { ArrowRightIcon, DownloadIcon, KeyRoundIcon, LockKeyholeIcon, ShieldCheckIcon, WandSparklesIcon } from '@lucide/vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import { useAuthStore } from '@/features/auth/store'
import { Button } from '@/components/ui/button'

const auth = useAuthStore()
const destination = computed(() => auth.isAuthenticated ? '/vault' : '/login?redirect_to=/vault')

onMounted(() => auth.ensureLoaded())
</script>

<template>
  <div class="home-page">
    <AppHeader />

    <main>
      <section class="home-hero" aria-labelledby="hero-title">
        <div class="hero-scrim" />
        <div class="hero-content">
          <span class="hero-eyebrow"><ShieldCheckIcon /> 你的个人密码库</span>
          <h1 id="hero-title">Glosc AI Password</h1>
          <p class="hero-lead">重要的账号，值得认真保护。</p>
          <p class="hero-description">集中保存网站密码，生成更难猜的密码。使用 Glosc AI 账号登录，再由你自己的安全 Key 解锁密码库。</p>
          <Button size="lg" as-child class="hero-button"><RouterLink :to="destination"><KeyRoundIcon data-icon="inline-start" />打开我的密码库<ArrowRightIcon data-icon="inline-end" /></RouterLink></Button>
        </div>
        <span class="hero-image-note">界面示意 · 非真实账号数据</span>
      </section>

      <section id="features" class="feature-band" aria-labelledby="feature-title">
        <div class="section-inner">
          <div class="section-heading"><span>日常使用</span><h2 id="feature-title">一个地方，管好每个登录信息</h2></div>
          <div class="feature-grid">
            <article><span class="feature-icon"><LockKeyholeIcon /></span><h3>安全保存</h3><p>添加、搜索和管理网站登录信息；密码默认遮蔽，需要时再查看或复制。</p></article>
            <article><span class="feature-icon"><WandSparklesIcon /></span><h3>生成强密码</h3><p>按长度和字符类型生成随机密码，告别多个网站重复使用同一密码。</p></article>
            <article><span class="feature-icon"><DownloadIcon /></span><h3>轻松迁移</h3><p>从常见浏览器和密码管理器导入登录信息，也可导出加密备份。</p></article>
          </div>
        </div>
      </section>

      <section id="security" class="security-band" aria-labelledby="security-title">
        <div class="section-inner security-layout">
          <div><span class="section-kicker">安全设计</span><h2 id="security-title">登录确认身份，安全 Key 保护密码。</h2><p>你的凭据在浏览器中加密后才会保存到服务端。安全 Key 不会上传；每台设备登录后都需要你亲自解锁。</p></div>
          <div class="security-steps"><div><strong>01</strong><span>使用 Glosc AI 账号登录</span></div><div><strong>02</strong><span>用安全 Key 解锁个人密码库</span></div><div><strong>03</strong><span>离线保存一次性展示的恢复码</span></div></div>
        </div>
      </section>

      <section class="closing-band"><div class="section-inner closing-layout"><div><h2>从现在开始，整理你的密码。</h2><p>如果安全 Key 和恢复码都丢失，旧密码库无法找回。请妥善保管它们。</p></div><Button size="lg" as-child><RouterLink :to="destination">进入密码库<ArrowRightIcon data-icon="inline-end" /></RouterLink></Button></div></section>
    </main>

    <footer class="home-footer"><div class="section-inner"><span>Glosc AI Password</span><span>由你掌控的个人密码库</span></div></footer>
  </div>
</template>

<style scoped>
.home-page{min-height:100vh;background:var(--background);color:var(--foreground)}
.home-hero{position:relative;display:flex;align-items:center;min-height:min(690px,74vh);background-image:url('/images/vault-hero.jpg');background-size:cover;background-position:center;color:#111e22;overflow:hidden}
.hero-scrim{position:absolute;inset:0;background:#eff4f0aa}
.hero-content{position:relative;z-index:1;width:min(100%,1280px);margin:auto;padding:72px max(24px,calc((100vw - 1280px)/2 + 24px))}
.hero-content>*{max-width:545px}
.hero-eyebrow{display:inline-flex;align-items:center;gap:9px;margin-bottom:28px;color:#087479;font-size:13px;font-weight:700}
.hero-eyebrow svg{width:18px;height:18px}
.hero-content h1{font-size:clamp(38px,4.6rem,72px);font-weight:750;line-height:1.08;overflow-wrap:anywhere}
.hero-lead{margin-top:20px;font-size:clamp(25px,2rem,34px);font-weight:650;line-height:1.25}
.hero-description{margin-top:20px;color:#334d52;font-size:16px;line-height:1.85}
.hero-button{margin-top:32px}
.hero-image-note{position:absolute;right:22px;bottom:17px;z-index:1;color:#3e5558;font-size:11px}
.section-inner{width:min(100% - 48px,1180px);margin:auto}
.feature-band{padding:78px 0 88px;background:var(--background)}
.section-heading{text-align:center}.section-heading>span,.section-kicker{color:var(--primary);font-size:13px;font-weight:700}.section-heading h2,.security-band h2,.closing-band h2{margin-top:10px;font-size:clamp(25px,2.4rem,38px);font-weight:680;line-height:1.25}
.feature-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:0;margin-top:58px}
.feature-grid article{padding:0 36px;border-right:1px solid var(--border)}.feature-grid article:first-child{padding-left:0}.feature-grid article:last-child{padding-right:0;border-right:0}
.feature-icon{display:grid;place-items:center;width:44px;height:44px;border-radius:7px;background:var(--accent);color:var(--primary)}.feature-icon svg{width:23px;height:23px}.feature-grid h3{margin-top:21px;font-size:18px;font-weight:650}.feature-grid p{margin-top:10px;color:var(--muted-foreground);font-size:14px;line-height:1.8}
.security-band{padding:82px 0;background:var(--secondary)}.security-layout{display:grid;grid-template-columns:1fr 1fr;gap:100px;align-items:center}.security-layout p{max-width:540px;margin-top:20px;color:var(--muted-foreground);line-height:1.9}.security-steps{border-top:1px solid var(--border)}.security-steps div{display:flex;gap:22px;align-items:center;min-height:72px;border-bottom:1px solid var(--border)}.security-steps strong{color:var(--primary);font-size:13px}.security-steps span{font-weight:550}
.closing-band{padding:68px 0}.closing-layout{display:flex;align-items:center;justify-content:space-between;gap:28px}.closing-band h2{margin:0}.closing-band p{margin-top:12px;color:var(--muted-foreground);font-size:14px}.closing-layout>a{flex:none}
.home-footer{padding:24px 0;border-top:1px solid var(--border);color:var(--muted-foreground);font-size:12px}.home-footer .section-inner{display:flex;justify-content:space-between;gap:16px}
:global(.dark .home-hero){color:#f5fbfa}:global(.dark .hero-scrim){background:#101d20bb}:global(.dark .hero-eyebrow){color:#9cded4}:global(.dark .hero-description),:global(.dark .hero-image-note){color:#d4e2df}
@media(max-width:900px){.home-hero{min-height:650px;background-position:63% center}.hero-scrim{background:#edf4f0d9}.hero-content{padding:72px 24px}.security-layout{gap:44px}}
@media(max-width:640px){.home-hero{min-height:min(660px,calc(100svh - 140px));align-items:flex-start;background-position:68% bottom}.hero-scrim{background:#edf4f0d9}.hero-content{padding:54px 24px 72px}.hero-content h1{font-size:42px}.hero-lead{font-size:27px}.hero-description{font-size:15px}.hero-image-note{font-size:10px}.feature-band,.security-band{padding:58px 0}.feature-grid{grid-template-columns:1fr;margin-top:38px}.feature-grid article,.feature-grid article:first-child,.feature-grid article:last-child{padding:20px 0;border-right:0;border-bottom:1px solid var(--border)}.feature-grid article:last-child{border-bottom:0}.feature-grid h3{margin-top:12px}.security-layout{grid-template-columns:1fr;gap:30px}.closing-layout{align-items:flex-start;flex-direction:column}.home-footer .section-inner{flex-wrap:wrap}:global(.dark .hero-scrim){background:#101d20d9}}
@media(max-width:640px) and (max-height:650px){.hero-content{padding-top:30px;padding-bottom:40px}.hero-eyebrow{margin-bottom:14px}.hero-content h1{font-size:36px}.hero-lead{margin-top:12px;font-size:23px}.hero-description{margin-top:12px}.hero-button{margin-top:18px}}
</style>

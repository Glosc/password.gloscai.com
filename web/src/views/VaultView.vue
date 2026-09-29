<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import {
  ArrowLeftIcon, CheckIcon, CopyIcon, DownloadIcon, EyeIcon, EyeOffIcon,
  KeyRoundIcon, LockKeyholeIcon, LogOutIcon, MenuIcon, MoreHorizontalIcon,
  PlusIcon, RefreshCwIcon, SearchIcon, SettingsIcon, ShieldCheckIcon,
  StarIcon, Trash2Icon, UploadIcon, WandSparklesIcon, XIcon,
} from '@lucide/vue'
import { Button } from '@/components/ui/button'
import ThemeControl from '@/components/layout/ThemeControl.vue'
import { useAuthStore } from '@/features/auth/store'
import { useVaultStore } from '@/features/vault/store'
import type { VaultEntry } from '@/features/vault/store'
import type { Credential, GeneratorOptions } from '@/features/vault/crypto'
import { generatePassword, makeBackup, randomSecret } from '@/features/vault/crypto'
import { downloadFile, exportCSV, previewImport } from '@/features/vault/transfer'
import type { ImportPreview } from '@/features/vault/transfer'
import { confirmAction, notifyError, notifySuccess } from '@/lib/message'

const router = useRouter()
const auth = useAuthStore()
const vault = useVaultStore()
const query = ref('')
const favoritesOnly = ref(false)
const selectedID = ref('')
const reveal = ref(false)
const mobileMenu = ref(false)
const panel = ref<'none' | 'edit' | 'generator' | 'transfer' | 'settings'>('none')
const transferTab = ref<'import' | 'export'>('import')
const editingID = ref('')
const formPasswordVisible = ref(false)
const form = reactive<Credential>(emptyCredential())
const secret = ref('')
const setupSecretVisible = ref(false)
const confirmSecret = ref('')
const recoveryInput = ref('')
const newSecret = ref('')
const newSecretVisible = ref(false)
const recoveryMode = ref(false)
const displayedRecovery = ref('')
const recoveryAcknowledged = ref(false)
const generated = ref('')
const generatorFromEdit = ref(false)
const generator = reactive<GeneratorOptions>({ length: 24, lower: true, upper: true, digits: true, symbols: true })
const importFile = ref<File | null>(null)
const importPassphrase = ref('')
const importPreview = ref<ImportPreview | null>(null)
const exportPassphrase = ref('')
const working = ref(false)

function emptyCredential(): Credential {
  return { title: '', url: '', username: '', password: '', note: '', favorite: false }
}

const filtered = computed(() => vault.entries.filter(entry => {
  if (favoritesOnly.value && !entry.favorite) return false
  const term = query.value.trim().toLocaleLowerCase()
  return !term || [entry.title, entry.url, entry.username].some(value => value.toLocaleLowerCase().includes(term))
}).sort((a, b) => b.updated_at.localeCompare(a.updated_at)))
const selected = computed(() => vault.entries.find(item => item.id === selectedID.value) || null)

watch(() => auth.isAuthenticated, signedIn => { if (!signedIn) vault.lock() })
watch(selectedID, () => { reveal.value = false })
let idleTimer: ReturnType<typeof setTimeout> | undefined
function resetIdleTimer() {
  clearTimeout(idleTimer)
  if (vault.unlocked) idleTimer = setTimeout(() => vault.lock(), 10 * 60 * 1000)
}
watch(() => vault.unlocked, unlocked => {
  resetIdleTimer()
  if (!unlocked) { displayedRecovery.value = ''; panel.value = 'none'; reveal.value = false }
})
onMounted(async () => {
  window.addEventListener('pointerdown', resetIdleTimer)
  window.addEventListener('keydown', resetIdleTimer)
  try { await vault.loadState() }
  catch (error) { showError(error) }
})
onUnmounted(() => {
  clearTimeout(idleTimer)
  window.removeEventListener('pointerdown', resetIdleTimer)
  window.removeEventListener('keydown', resetIdleTimer)
  vault.lock()
})

function showError(error: unknown) { notifyError(error instanceof Error ? error.message : '操作失败') }
function formatDate(date: string): string { return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium' }).format(new Date(date)) }
function iconLetter(title: string): string { return title.trim().slice(0, 1).toUpperCase() || '?' }
function safeURL(value: string): string | null {
  try {
    const url = new URL(value)
    return ['https:', 'http:'].includes(url.protocol) ? url.href : null
  }
  catch { return null }
}

async function copy(value: string, label: string) {
  try { await navigator.clipboard.writeText(value); notifySuccess(`${label}已复制`) }
  catch { notifyError('复制失败，请检查浏览器剪贴板权限') }
}

function showPassword() {
  reveal.value = true
  setTimeout(() => { reveal.value = false }, 20_000)
}

async function signOut() {
  vault.lock()
  await auth.logout()
  await router.replace('/login')
}

function showRecovery(code: string) {
  displayedRecovery.value = code
  recoveryAcknowledged.value = false
  panel.value = 'none'
}

async function setup() {
  if (secret.value.length < 12 || secret.value !== confirmSecret.value) { notifyError('安全 Key 至少 12 位，且两次输入必须一致'); return }
  try {
    const code = await vault.create(secret.value)
    secret.value = ''; confirmSecret.value = ''
    showRecovery(code)
  }
  catch (error) { showError(error) }
}

async function unlock() {
  try {
    if (recoveryMode.value) {
      if (newSecret.value.length < 12) { notifyError('新安全 Key 至少 12 位'); return }
      const code = await vault.recover(recoveryInput.value.trim(), newSecret.value)
      recoveryInput.value = ''; newSecret.value = ''
      showRecovery(code)
    }
    else {
      await vault.unlock(secret.value)
      secret.value = ''
    }
  }
  catch (error) { showError(error) }
}

async function destroyVault() {
  if (!await confirmAction('旧密码库及全部凭据将永久删除。只有安全 Key 和恢复码都已丢失时才应继续。', '销毁密码库', { danger: true, confirmText: '永久删除' })) return
  try { await vault.destroy(); recoveryMode.value = false; notifySuccess('旧密码库已删除') }
  catch (error) { showError(error) }
}

function edit(entry?: VaultEntry) {
  Object.assign(form, entry || emptyCredential())
  formPasswordVisible.value = false
  editingID.value = entry?.id || ''
  panel.value = 'edit'
}

async function saveEntry() {
  if (!form.title.trim() || !form.password) { notifyError('请输入名称和密码'); return }
  try {
    const value = { ...form, title: form.title.trim() }
    if (editingID.value) await vault.update(editingID.value, value)
    else await vault.add(value)
    panel.value = 'none'
    notifySuccess(editingID.value ? '已保存修改' : '密码已添加')
  }
  catch (error) { showError(error) }
}

async function removeEntry(entry: VaultEntry) {
  if (!await confirmAction(`确定删除“${entry.title}”吗？`, '删除密码', { danger: true })) return
  try { await vault.remove(entry.id); selectedID.value = ''; notifySuccess('密码已删除') }
  catch (error) { showError(error) }
}

function openGenerator() {
  generatorFromEdit.value = panel.value === 'edit'
  generated.value = generatePassword(generator)
  panel.value = 'generator'
}
function regenerate() {
  try { generated.value = generatePassword(generator) }
  catch (error) { showError(error) }
}
function useGenerated() {
  if (!generatorFromEdit.value) { Object.assign(form, emptyCredential()); editingID.value = '' }
  form.password = generated.value
  panel.value = 'edit'
}

function openTransfer(tab: 'import' | 'export') {
  transferTab.value = tab
  importPreview.value = null
  importFile.value = null
  panel.value = 'transfer'
  mobileMenu.value = false
}
function selectFile(event: Event) {
  importFile.value = (event.target as HTMLInputElement).files?.[0] || null
  importPreview.value = null
}
async function previewFile() {
  if (!importFile.value) return
  working.value = true
  try { importPreview.value = await previewImport(importFile.value, importPassphrase.value) }
  catch (error) { showError(error) }
  finally { working.value = false }
}
async function importNow() {
  if (!importPreview.value?.entries.length) return
  working.value = true
  try {
    const count = await vault.importEntries(importPreview.value.entries)
    notifySuccess(`已导入 ${count} 条凭据`)
    importPreview.value = null; importFile.value = null; importPassphrase.value = ''; panel.value = 'none'
  }
  catch (error) { showError(error) }
  finally { working.value = false }
}
async function exportEncrypted() {
  if (exportPassphrase.value.length < 12) { notifyError('备份口令至少 12 位'); return }
  working.value = true
  try {
    const backup = await makeBackup(exportPassphrase.value, vault.entries)
    downloadFile(`glosc-vault-${new Date().toISOString().slice(0, 10)}.json`, JSON.stringify(backup), 'application/json')
    exportPassphrase.value = ''
    notifySuccess('加密备份已下载')
  }
  catch (error) { showError(error) }
  finally { working.value = false }
}
async function exportPlain() {
  if (!await confirmAction('CSV 包含所有明文密码。请只保存在可信设备上，并在迁移后安全删除。', '导出明文密码', { danger: true, confirmText: '导出明文 CSV' })) return
  downloadFile(`glosc-passwords-${new Date().toISOString().slice(0, 10)}.csv`, exportCSV(vault.entries), 'text/csv;charset=utf-8')
}
async function rotateKey() {
  if (newSecret.value.length < 12) { notifyError('新安全 Key 至少 12 位'); return }
  working.value = true
  try {
    const code = await vault.changeKey(newSecret.value)
    newSecret.value = ''
    showRecovery(code)
    notifySuccess('全部凭据已用新 Key 重新加密')
  }
  catch (error) { showError(error) }
  finally { working.value = false }
}
</script>

<template>
  <div class="vault-app">
    <header class="topbar">
      <Button variant="ghost" size="icon" class="mobile-menu" aria-label="打开菜单" @click="mobileMenu = !mobileMenu"><MenuIcon /></Button>
      <RouterLink to="/" class="brand" title="返回首页"><span class="brand-mark"><ShieldCheckIcon /></span><strong>Glosc AI</strong><span class="brand-divider" /> <span class="brand-sub">Password</span></RouterLink>
      <div class="top-actions">
        <span class="account-name">{{ auth.displayName }}</span>
        <ThemeControl />
        <Button v-if="vault.unlocked" variant="ghost" size="icon" title="锁定密码库" aria-label="锁定密码库" @click="vault.lock()"><LockKeyholeIcon /></Button>
        <Button variant="ghost" size="icon" title="退出登录" aria-label="退出登录" @click="signOut"><LogOutIcon /></Button>
      </div>
    </header>

    <div v-if="vault.loading" class="center-state"><div class="spinner" /><span>正在打开密码库...</span></div>
    <div v-else-if="!vault.state" class="center-state"><ShieldCheckIcon class="size-9 text-teal-700" /><p>无法读取密码库</p><Button variant="outline" @click="vault.loadState()">重试</Button></div>

    <main v-else-if="!vault.state.initialized || !vault.unlocked" class="access-screen">
      <div class="access-heading"><span class="access-icon"><LockKeyholeIcon /></span><h1>{{ !vault.state.initialized ? '创建你的密码库' : '解锁密码库' }}</h1><p>{{ !vault.state.initialized ? 'Glosc AI 账号负责身份验证，安全 Key 由你独自保管。' : '输入你的安全 Key，读取已加密的密码。' }}</p></div>
      <form v-if="!vault.state.initialized" class="access-form" @submit.prevent="setup">
        <label>安全 Key<div class="input-action"><input v-model="secret" :type="setupSecretVisible ? 'text' : 'password'" autocomplete="new-password" minlength="12" placeholder="至少 12 位的长口令或随机 Key" required><Button type="button" variant="ghost" size="icon" :title="setupSecretVisible ? '隐藏 Key' : '显示 Key'" aria-label="切换显示安全 Key" @click="setupSecretVisible = !setupSecretVisible"><EyeOffIcon v-if="setupSecretVisible" /><EyeIcon v-else /></Button><Button v-if="setupSecretVisible" type="button" variant="ghost" size="icon" title="复制 Key" aria-label="复制安全 Key" @click="copy(secret, '安全 Key')"><CopyIcon /></Button></div></label>
        <Button type="button" variant="outline" @click="secret = randomSecret(); confirmSecret = secret; setupSecretVisible = true"><WandSparklesIcon />生成随机 Key</Button>
        <label>确认安全 Key<input v-model="confirmSecret" type="password" autocomplete="new-password" minlength="12" required></label>
        <p class="security-note">安全 Key 不会上传。丢失后只能用离线恢复码找回数据。</p>
        <Button type="submit" size="lg" :disabled="vault.busy"><ShieldCheckIcon />创建密码库</Button>
      </form>
      <form v-else class="access-form" @submit.prevent="unlock">
        <template v-if="!recoveryMode"><label>安全 Key<input v-model="secret" type="password" autocomplete="current-password" autofocus required></label><Button type="submit" size="lg" :disabled="vault.busy"><KeyRoundIcon />解锁</Button><button type="button" class="text-link" @click="recoveryMode = true">改用恢复码</button></template>
        <template v-else><label>恢复码<input v-model="recoveryInput" type="text" autocomplete="off" required></label><label>设置新安全 Key<input v-model="newSecret" :type="newSecretVisible ? 'text' : 'password'" autocomplete="new-password" minlength="12" required></label><Button type="button" variant="outline" @click="newSecret = randomSecret(); newSecretVisible = true"><WandSparklesIcon />生成随机 Key</Button><Button v-if="newSecretVisible" type="button" variant="outline" @click="copy(newSecret, '新安全 Key')"><CopyIcon />复制新 Key</Button><Button type="submit" size="lg" :disabled="vault.busy"><RefreshCwIcon />恢复并更换 Key</Button><button type="button" class="text-link" @click="recoveryMode = false">返回安全 Key 解锁</button></template>
        <button type="button" class="reset-link" @click="destroyVault">Key 和恢复码都丢失了？清空并重建</button>
      </form>
    </main>

    <div v-else class="workspace">
      <aside class="sidebar" :class="{ 'sidebar-open': mobileMenu }">
        <div class="side-group">
          <button class="side-item" :class="{ active: !favoritesOnly }" @click="favoritesOnly = false; mobileMenu = false"><LockKeyholeIcon />所有密码<span>{{ vault.entries.length }}</span></button>
          <button class="side-item" :class="{ active: favoritesOnly }" @click="favoritesOnly = true; mobileMenu = false"><StarIcon />收藏夹<span>{{ vault.entries.filter(item => item.favorite).length }}</span></button>
        </div>
        <div class="side-separator" />
        <div class="side-group">
          <button class="side-item" @click="openGenerator"><WandSparklesIcon />密码生成器</button>
          <button class="side-item" @click="openTransfer('import')"><UploadIcon />导入密码</button>
          <button class="side-item" @click="openTransfer('export')"><DownloadIcon />导出密码</button>
        </div>
        <div class="side-bottom"><button class="side-item" @click="panel = 'settings'; mobileMenu = false"><SettingsIcon />安全设置</button><div class="side-account"><span class="account-avatar">{{ auth.displayName.slice(0, 1) }}</span><span class="truncate">{{ auth.displayName }}</span></div></div>
      </aside>

      <section class="list-pane">
        <div class="pane-heading"><div><h1>{{ favoritesOnly ? '收藏夹' : '密码库' }}</h1><p>{{ favoritesOnly ? '你标记的重要登录信息' : '你的登录信息，安全存放' }}</p></div><Button @click="edit()"><PlusIcon />添加密码</Button></div>
        <div class="search-wrap"><SearchIcon /><input v-model="query" type="search" placeholder="搜索名称、网站或用户名" aria-label="搜索密码"></div>
        <div class="list-head"><span>网站 / 应用</span><span>用户名</span><span>更新日期</span><span class="text-right">操作</span></div>
        <div v-if="!filtered.length" class="empty-list"><LockKeyholeIcon /><strong>{{ query || favoritesOnly ? '没有匹配的密码' : '还没有保存密码' }}</strong><span>{{ query ? '试试其他关键词' : '添加第一条登录信息' }}</span></div>
        <div v-else class="entry-list">
          <button v-for="entry in filtered" :key="entry.id" class="entry-row" :class="{ selected: selectedID === entry.id }" @click="selectedID = entry.id">
            <span class="entry-name"><span class="site-avatar">{{ iconLetter(entry.title) }}</span><span class="truncate">{{ entry.title }}</span><StarIcon v-if="entry.favorite" class="favorite-icon" /></span>
            <span class="entry-username truncate">{{ entry.username || '—' }}</span><span class="entry-date">{{ formatDate(entry.updated_at) }}</span><span class="entry-more"><MoreHorizontalIcon /></span>
          </button>
        </div>
      </section>

      <section class="detail-pane" :class="{ 'detail-mobile-open': selected }">
        <template v-if="selected">
          <div class="detail-top"><Button variant="ghost" size="icon" class="mobile-back" aria-label="返回列表" @click="selectedID = ''"><ArrowLeftIcon /></Button><span class="site-avatar large">{{ iconLetter(selected.title) }}</span><div class="detail-title"><h2>{{ selected.title }}</h2><span class="truncate">{{ selected.url || '未填写网站地址' }}</span></div><Button variant="ghost" size="icon" :title="selected.favorite ? '取消收藏' : '收藏'" :aria-label="selected.favorite ? '取消收藏' : '收藏'" @click="vault.update(selected.id, { ...selected, favorite: !selected.favorite }).catch(showError)"><StarIcon :class="{ 'favorite-icon': selected.favorite }" /></Button></div>
          <div class="detail-body"><div class="detail-field"><label>网站地址</label><div class="detail-value"><span class="truncate">{{ selected.url || '—' }}</span><a v-if="safeURL(selected.url)" :href="safeURL(selected.url)!" target="_blank" rel="noopener noreferrer" title="打开网站">↗</a></div></div><div class="detail-field"><label>用户名</label><div class="detail-value"><span class="truncate">{{ selected.username || '—' }}</span><Button variant="ghost" size="icon-sm" title="复制用户名" aria-label="复制用户名" @click="copy(selected.username, '用户名')"><CopyIcon /></Button></div></div><div class="detail-field"><label>密码</label><div class="detail-value"><span class="password-value">{{ reveal ? selected.password : '••••••••••••' }}</span><Button variant="ghost" size="icon-sm" :title="reveal ? '隐藏密码' : '显示密码 20 秒'" :aria-label="reveal ? '隐藏密码' : '显示密码'" @click="reveal ? reveal = false : showPassword()"><EyeOffIcon v-if="reveal" /><EyeIcon v-else /></Button><Button variant="ghost" size="icon-sm" title="复制密码" aria-label="复制密码" @click="copy(selected.password, '密码')"><CopyIcon /></Button></div></div><div class="detail-field"><label>备注</label><p class="note-value">{{ selected.note || '—' }}</p></div><div class="detail-field"><label>最后更新</label><p class="muted-value">{{ formatDate(selected.updated_at) }}</p></div></div>
          <div class="detail-actions"><Button class="w-full" @click="edit(selected)">编辑密码</Button><div class="detail-action-row"><Button variant="outline" class="flex-1" @click="copy(selected.password, '密码')"><CopyIcon />复制密码</Button><Button variant="destructive" title="删除密码" aria-label="删除密码" @click="removeEntry(selected)"><Trash2Icon /></Button></div></div>
        </template>
        <div v-else class="detail-placeholder"><ShieldCheckIcon /><p>选择一条密码查看详情</p></div>
      </section>
    </div>

    <div v-if="displayedRecovery" class="modal-backdrop"><div class="modal-panel recovery-panel"><span class="modal-symbol"><ShieldCheckIcon /></span><h2>保存你的恢复码</h2><p>这是找回密码库的唯一备用方式。它只显示这一次，请保存在离线安全位置。</p><div class="secret-code">{{ displayedRecovery }}</div><Button variant="outline" class="w-full" @click="copy(displayedRecovery, '恢复码')"><CopyIcon />复制恢复码</Button><label class="check-line"><input v-model="recoveryAcknowledged" type="checkbox">我已安全保存恢复码</label><Button class="w-full" :disabled="!recoveryAcknowledged" @click="displayedRecovery = ''"><CheckIcon />继续使用</Button></div></div>

    <div v-if="panel !== 'none' && !displayedRecovery" class="modal-backdrop" @click.self="panel = 'none'"><div class="modal-panel"><div class="modal-heading"><h2>{{ panel === 'edit' ? (editingID ? '编辑密码' : '添加密码') : panel === 'generator' ? '密码生成器' : panel === 'settings' ? '安全设置' : '导入 / 导出' }}</h2><Button variant="ghost" size="icon" aria-label="关闭" @click="panel = 'none'"><XIcon /></Button></div>
      <form v-if="panel === 'edit'" class="modal-form" @submit.prevent="saveEntry"><label>名称<input v-model="form.title" placeholder="例如 GitHub" maxlength="160" required></label><label>网站地址<input v-model="form.url" type="url" placeholder="https://example.com"></label><label>用户名<input v-model="form.username" autocomplete="off" placeholder="邮箱或用户名"></label><label>密码<div class="input-action"><input v-model="form.password" :type="formPasswordVisible ? 'text' : 'password'" autocomplete="off" required><Button type="button" variant="ghost" size="icon" :title="formPasswordVisible ? '隐藏密码' : '显示密码'" aria-label="切换显示密码" @click="formPasswordVisible = !formPasswordVisible"><EyeOffIcon v-if="formPasswordVisible" /><EyeIcon v-else /></Button><Button type="button" variant="ghost" size="icon" title="生成强密码" aria-label="生成强密码" @click="openGenerator"><WandSparklesIcon /></Button></div></label><label>备注<textarea v-model="form.note" rows="3" maxlength="5000" placeholder="可选"></textarea></label><label class="check-line"><input v-model="form.favorite" type="checkbox">加入收藏夹</label><div class="modal-footer"><Button type="button" variant="outline" @click="panel = 'none'">取消</Button><Button type="submit" :disabled="vault.busy">保存</Button></div></form>
      <div v-else-if="panel === 'generator'" class="modal-form"><div class="generated-value"><span>{{ generated }}</span><Button variant="ghost" size="icon" title="复制密码" aria-label="复制密码" @click="copy(generated, '密码')"><CopyIcon /></Button><Button variant="ghost" size="icon" title="重新生成" aria-label="重新生成" @click="regenerate"><RefreshCwIcon /></Button></div><label>长度：{{ generator.length }}<input v-model.number="generator.length" type="range" min="12" max="128" @input="regenerate"></label><div class="generator-options"><label><input v-model="generator.lower" type="checkbox" @change="regenerate">小写字母</label><label><input v-model="generator.upper" type="checkbox" @change="regenerate">大写字母</label><label><input v-model="generator.digits" type="checkbox" @change="regenerate">数字</label><label><input v-model="generator.symbols" type="checkbox" @change="regenerate">符号</label></div><div class="modal-footer"><Button variant="outline" @click="panel = 'none'">关闭</Button><Button @click="useGenerated"><CheckIcon />填入密码</Button></div></div>
      <div v-else-if="panel === 'transfer'" class="modal-form"><div class="segment"><button :class="{ active: transferTab === 'import' }" @click="transferTab = 'import'">导入</button><button :class="{ active: transferTab === 'export' }" @click="transferTab = 'export'">导出</button></div><template v-if="transferTab === 'import'"><label>选择文件<input type="file" accept=".csv,.json,.1pux" @change="selectFile"></label><p class="helper">支持浏览器、Bitwarden、1Password 登录条目及 Glosc 加密备份。文件只在本机解析。</p><label v-if="importFile?.name.toLowerCase().endsWith('.json')">备份口令（仅 Glosc 备份需要）<input v-model="importPassphrase" type="password"></label><Button variant="outline" :disabled="!importFile || working" @click="previewFile"><EyeIcon />预览导入</Button><div v-if="importPreview" class="preview-box"><strong>{{ importPreview.source }} · {{ importPreview.entries.length }} 条可导入</strong><span>{{ importPreview.skipped }} 条跳过（非登录项或缺少密码）</span><ul><li v-for="item in importPreview.entries.slice(0, 5)" :key="`${item.title}-${item.username}`">{{ item.title }} · {{ item.username || '无用户名' }}</li></ul></div><div class="modal-footer"><Button variant="outline" @click="panel = 'none'">取消</Button><Button :disabled="!importPreview?.entries.length || working" @click="importNow"><UploadIcon />确认导入</Button></div></template><template v-else><label>加密备份口令<input v-model="exportPassphrase" type="password" minlength="12" autocomplete="new-password" placeholder="至少 12 位"></label><p class="helper">加密备份可在新的 Glosc 部署中恢复，不依赖当前服务器密钥。</p><Button :disabled="working" @click="exportEncrypted"><DownloadIcon />下载加密备份</Button><div class="side-separator" /><Button variant="outline" @click="exportPlain"><DownloadIcon />导出明文 CSV</Button><p class="danger-note">明文 CSV 含所有密码。请在迁移后安全删除。</p></template></div>
      <div v-else class="modal-form"><p class="helper">更换安全 Key 会在浏览器中重新加密每条凭据，并产生新的恢复码。</p><label>新安全 Key<input v-model="newSecret" :type="newSecretVisible ? 'text' : 'password'" autocomplete="new-password" minlength="12" placeholder="至少 12 位"></label><Button variant="outline" @click="newSecret = randomSecret(); newSecretVisible = true"><WandSparklesIcon />生成随机 Key</Button><Button v-if="newSecretVisible" variant="outline" @click="copy(newSecret, '新安全 Key')"><CopyIcon />复制新 Key</Button><Button :disabled="working" @click="rotateKey"><RefreshCwIcon />更换安全 Key</Button><div class="side-separator" /><Button variant="destructive" @click="destroyVault"><Trash2Icon />销毁整个密码库</Button></div>
    </div></div>
  </div>
</template>

<style scoped>
.vault-app{min-height:100vh;background:#f8faf9;color:#182427;font-size:14px}.topbar{height:62px;display:flex;align-items:center;justify-content:space-between;padding:0 28px;background:#fff;border-bottom:1px solid #e4eaeb}.brand,.top-actions{display:flex;align-items:center;gap:12px}.brand strong{font-size:19px;letter-spacing:0}.brand-mark{display:grid;place-items:center;width:30px;height:30px;color:#0c7778}.brand-mark svg{width:26px;height:26px}.brand-divider{height:22px;border-left:1px solid #dce4e5;margin:0 3px}.brand-sub{color:#67787b}.account-name{color:#526368;font-size:13px}.mobile-menu,.mobile-back{display:none}.workspace{display:grid;grid-template-columns:225px minmax(460px,1fr) minmax(300px,350px);height:calc(100vh - 62px);min-height:590px}.sidebar{display:flex;flex-direction:column;padding:24px 12px;background:#fbfcfc;border-right:1px solid #e5ebec}.side-group{display:grid;gap:4px}.side-item{display:flex;align-items:center;gap:13px;width:100%;height:42px;padding:0 14px;border-radius:5px;text-align:left;color:#405257}.side-item:hover,.side-item.active{background:#e5f0f0;color:#086c70}.side-item svg{width:18px;height:18px}.side-item span:last-child:not(.truncate){margin-left:auto;font-size:12px}.side-separator{height:1px;background:#e6ecec;margin:17px 9px}.side-bottom{margin-top:auto}.side-account{display:flex;align-items:center;gap:10px;padding:18px 13px 4px;border-top:1px solid #e6ecec;color:#526367}.account-avatar{display:grid;place-items:center;width:29px;height:29px;border-radius:50%;background:#d9e8e8;color:#176c70}.list-pane{min-width:0;overflow:auto;background:#fff;border-right:1px solid #e6ecec}.pane-heading{display:flex;align-items:center;justify-content:space-between;gap:14px;padding:28px 28px 18px}.pane-heading h1{font-size:26px;font-weight:650;line-height:1.2}.pane-heading p{margin-top:7px;color:#718084;font-size:13px}.search-wrap{display:flex;align-items:center;gap:10px;height:41px;margin:0 28px 20px;padding:0 12px;border:1px solid #dce5e7;border-radius:5px;color:#75858a}.search-wrap svg{width:18px;height:18px}.search-wrap input{min-width:0;flex:1;outline:none;background:transparent}.list-head,.entry-row{display:grid;grid-template-columns:minmax(170px,1.5fr) minmax(100px,1.1fr) 100px 40px;align-items:center;gap:10px}.list-head{height:38px;padding:0 28px;background:#f7f9f9;color:#718084;font-size:12px}.entry-row{width:100%;min-height:62px;padding:0 28px;border-bottom:1px solid #edf0f1;text-align:left}.entry-row:hover,.entry-row.selected{background:#eaf4f3}.entry-name{display:flex;align-items:center;gap:10px;min-width:0;font-weight:550}.site-avatar{display:grid;place-items:center;flex:none;width:30px;height:30px;border:1px solid #dfe7e7;border-radius:5px;background:#fff;color:#187579;font-weight:700}.site-avatar.large{width:51px;height:51px;font-size:22px}.favorite-icon{width:15px;height:15px;color:#e0a62a;fill:#e0a62a}.entry-username,.entry-date{color:#526368;font-size:13px}.entry-more{display:flex;justify-content:flex-end;color:#77888b}.entry-more svg{width:17px}.empty-list,.detail-placeholder{display:flex;flex-direction:column;align-items:center;justify-content:center;gap:10px;min-height:320px;color:#829195}.empty-list svg,.detail-placeholder svg{width:30px;height:30px;color:#66a5a5}.empty-list strong{color:#34464a}.detail-pane{display:flex;flex-direction:column;min-width:0;background:#fbfcfc}.detail-top{display:flex;align-items:center;gap:14px;padding:30px 23px 23px}.detail-title{flex:1;min-width:0}.detail-title h2{font-size:19px;font-weight:650}.detail-title span{display:block;margin-top:5px;color:#7b898d;font-size:12px}.detail-body{overflow:auto;padding:0 23px}.detail-field{margin:17px 0}.detail-field label{display:block;margin-bottom:8px;color:#718084;font-size:12px}.detail-value,.note-value{display:flex;align-items:center;gap:5px;min-height:40px;padding:0 10px;border:1px solid #e0e7e8;border-radius:5px;background:#fff}.detail-value>span:first-child{min-width:0;flex:1}.detail-value a{font-size:20px;color:#0e787b}.password-value{font-family:ui-monospace,monospace;overflow:hidden;text-overflow:ellipsis}.note-value{align-items:start;white-space:pre-wrap;padding:10px;min-height:75px}.muted-value{color:#4b5e62}.detail-actions{margin-top:auto;padding:19px 23px 25px;border-top:1px solid #e8eeee}.detail-action-row{display:flex;gap:8px;margin-top:9px}.center-state{display:flex;align-items:center;justify-content:center;gap:13px;min-height:55vh;color:#687b7e}.spinner{width:20px;height:20px;border:2px solid #cde4e4;border-top-color:#087579;border-radius:50%;animation:spin .7s linear infinite}@keyframes spin{to{transform:rotate(360deg)}}.access-screen{display:flex;flex-direction:column;align-items:center;max-width:430px;margin:10vh auto;padding:0 20px}.access-heading{text-align:center}.access-icon,.modal-symbol{display:grid;place-items:center;width:54px;height:54px;margin:0 auto 20px;border-radius:8px;background:#e4f2f1;color:#087579}.access-icon svg,.modal-symbol svg{width:27px;height:27px}.access-heading h1{font-size:25px;font-weight:650}.access-heading p{margin-top:9px;color:#718084;line-height:1.7}.access-form,.modal-form{display:grid;gap:17px;width:100%;margin-top:28px}.access-form label,.modal-form label:not(.check-line){display:grid;gap:8px;color:#405256;font-weight:550}.access-form input,.modal-form input:not([type=checkbox]):not([type=range]):not([type=file]),.modal-form textarea{width:100%;height:40px;padding:0 11px;border:1px solid #dbe4e5;border-radius:5px;outline:none;background:#fff;color:#192729}.modal-form textarea{height:auto;padding:10px}.access-form input:focus,.modal-form input:focus,.modal-form textarea:focus{border-color:#198689;box-shadow:0 0 0 2px #d9eeee}.security-note,.helper{color:#718084;font-size:12px;line-height:1.6}.text-link,.reset-link{justify-self:center;color:#087579}.reset-link{margin-top:13px;color:#9a5555;font-size:12px}.modal-backdrop{position:fixed;inset:0;z-index:30;display:grid;place-items:center;padding:18px;background:#172d30a8}.modal-panel{width:min(100%,500px);max-height:min(90vh,850px);overflow:auto;padding:24px;border:1px solid #dfe8e9;border-radius:7px;background:#fff;box-shadow:0 20px 60px #0d2e3240}.modal-heading{display:flex;align-items:center;justify-content:space-between}.modal-heading h2,.recovery-panel h2{font-size:19px;font-weight:650}.modal-form{margin-top:21px}.modal-footer{display:flex;justify-content:flex-end;gap:9px;margin-top:7px}.input-action{display:flex;align-items:center;border:1px solid #dbe4e5;border-radius:5px}.input-action input{border:0!important;box-shadow:none!important}.generated-value,.secret-code{display:flex;align-items:center;gap:6px;min-height:50px;padding:8px 10px;border:1px solid #dbe5e6;border-radius:5px;background:#f8fbfb;font-family:ui-monospace,monospace;overflow-wrap:anywhere}.generated-value span{flex:1;min-width:0}.generator-options{display:grid;grid-template-columns:1fr 1fr;gap:10px}.generator-options label,.check-line{display:flex!important;align-items:center;gap:8px;font-weight:450!important}.generator-options input,.check-line input{accent-color:#087579}.modal-form input[type=range]{width:100%;accent-color:#087579}.preview-box{display:grid;gap:8px;padding:13px;border:1px solid #dce8e8;border-radius:5px;background:#f8fbfb;font-size:12px}.preview-box span{color:#718084}.preview-box ul{list-style:disc;padding-left:18px}.segment{display:grid;grid-template-columns:1fr 1fr;padding:3px;border:1px solid #e1e8e9;border-radius:5px;background:#f5f8f8}.segment button{height:32px;border-radius:4px}.segment button.active{background:#fff;color:#087579;box-shadow:0 1px 3px #13363a1a}.danger-note{font-size:12px;color:#a45656}.recovery-panel{text-align:center}.recovery-panel p{margin:9px 0 20px;color:#68797c;line-height:1.6}.recovery-panel .secret-code{justify-content:center;margin-bottom:12px;font-size:15px}.recovery-panel .check-line{justify-content:center;margin:20px 0}.truncate{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
@media(max-width:1120px){.workspace{grid-template-columns:195px minmax(350px,1fr) minmax(280px,320px)}.entry-row,.list-head{grid-template-columns:minmax(140px,1fr) minmax(90px,1fr) 30px}.entry-date,.list-head span:nth-child(3){display:none}.pane-heading,.search-wrap{margin-left:18px;margin-right:18px}.pane-heading{padding-left:0;padding-right:0}.entry-row,.list-head{padding-left:18px;padding-right:18px}}
@media(max-width:800px){.topbar{padding:0 15px}.mobile-menu{display:inline-flex}.brand{margin-right:auto}.brand-divider,.brand-sub,.account-name{display:none}.workspace{display:block;height:calc(100vh - 62px);min-height:0}.sidebar{position:fixed;z-index:15;top:62px;bottom:0;left:0;width:240px;transform:translateX(-100%);transition:transform .2s;box-shadow:8px 0 20px #152c3024}.sidebar-open{transform:translateX(0)}.list-pane{height:100%;border-right:0}.detail-pane{position:fixed;z-index:12;inset:62px 0 0;display:none;overflow:auto}.detail-mobile-open{display:flex}.mobile-back{display:inline-flex}.detail-top{padding:19px}.detail-body{padding:0 19px}.detail-actions{padding:16px 19px}.list-head,.entry-row{grid-template-columns:minmax(150px,1.4fr) minmax(100px,1fr) 24px}}
@media(max-width:470px){.pane-heading{align-items:start}.pane-heading h1{font-size:23px}.pane-heading p{display:none}.pane-heading button{font-size:12px}.list-head,.entry-row{grid-template-columns:minmax(135px,1fr) 28px}.entry-username,.list-head span:nth-child(2){display:none}.modal-panel{padding:18px}}
:global(.dark .vault-app){background:var(--background);color:var(--foreground)}
:global(.dark .topbar),:global(.dark .list-pane){background:var(--background);border-color:var(--border)}
:global(.dark .sidebar),:global(.dark .detail-pane){background:var(--sidebar);border-color:var(--border)}
:global(.dark .brand-mark),:global(.dark .detail-value a),:global(.dark .text-link){color:var(--primary)}
:global(.dark .brand-divider),:global(.dark .side-account),:global(.dark .detail-actions){border-color:var(--border)}
:global(.dark .side-separator){background:var(--border)}
:global(.dark .brand-sub),:global(.dark .account-name),:global(.dark .side-account),:global(.dark .pane-heading p),:global(.dark .entry-username),:global(.dark .entry-date),:global(.dark .entry-more),:global(.dark .detail-title span),:global(.dark .detail-field label),:global(.dark .muted-value),:global(.dark .center-state),:global(.dark .access-heading p),:global(.dark .security-note),:global(.dark .helper),:global(.dark .preview-box span),:global(.dark .recovery-panel p){color:var(--muted-foreground)}
:global(.dark .side-item){color:var(--foreground)}
:global(.dark .side-item:hover),:global(.dark .side-item.active),:global(.dark .entry-row:hover),:global(.dark .entry-row.selected){background:var(--accent);color:var(--accent-foreground)}
:global(.dark .account-avatar),:global(.dark .access-icon),:global(.dark .modal-symbol){background:var(--accent);color:var(--primary)}
:global(.dark .search-wrap),:global(.dark .site-avatar),:global(.dark .detail-value),:global(.dark .note-value),:global(.dark .input-action){background:var(--card);border-color:var(--border);color:var(--foreground)}
:global(.dark .search-wrap input){color:var(--foreground)}
:global(.dark .list-head),:global(.dark .segment){background:var(--muted);color:var(--muted-foreground)}
:global(.dark .entry-row){border-color:var(--border)}
:global(.dark .site-avatar){color:var(--primary)}
:global(.dark .empty-list),:global(.dark .detail-placeholder),:global(.dark .empty-list strong){color:var(--muted-foreground)}
:global(.dark .empty-list svg),:global(.dark .detail-placeholder svg),:global(.dark .center-state svg){color:var(--primary)}
:global(.dark .access-form label),:global(.dark .modal-form label:not(.check-line)){color:var(--foreground)}
:global(.dark .access-form input),:global(.dark .modal-form input:not([type=checkbox]):not([type=range]):not([type=file])),:global(.dark .modal-form textarea){background:var(--card);border-color:var(--border);color:var(--foreground)}
:global(.dark .access-form input:focus),:global(.dark .modal-form input:focus),:global(.dark .modal-form textarea:focus){border-color:var(--ring);box-shadow:0 0 0 2px color-mix(in oklab,var(--ring) 25%,transparent)}
:global(.dark .modal-panel){background:var(--card);border-color:var(--border);box-shadow:0 20px 60px #0009}
:global(.dark .generated-value),:global(.dark .secret-code),:global(.dark .preview-box){background:var(--muted);border-color:var(--border)}
:global(.dark .segment button.active){background:var(--card);color:var(--primary)}
:global(.dark .reset-link),:global(.dark .danger-note){color:var(--destructive)}
:global(.dark .generator-options input),:global(.dark .check-line input),:global(.dark .modal-form input[type=range]){accent-color:var(--primary)}
:global(.dark .spinner){border-color:var(--border);border-top-color:var(--primary)}
:global(.dark .sidebar){box-shadow:none}
</style>

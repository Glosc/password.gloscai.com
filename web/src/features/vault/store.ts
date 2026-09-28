import { computed, shallowRef, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  addEncryptedItems, deleteEncryptedItem, getVaultState, initializeVault,
  listEncryptedItems, resetVault, rotateVault, updateEncryptedItem,
} from '@/api/vault'
import type { VaultState } from '@/api/vault'
import type { Credential } from './crypto'
import { createMetadata, decryptCredential, encryptCredential, unlockMetadata } from './crypto'

export interface VaultEntry extends Credential { id: string; created_at: string; updated_at: string }

export const useVaultStore = defineStore('vault', () => {
  const state = ref<VaultState | null>(null)
  const entries = ref<VaultEntry[]>([])
  const key = shallowRef<CryptoKey | null>(null)
  const loading = ref(false)
  const busy = ref(false)
  const unlocked = computed(() => key.value !== null)

  function lock() {
    key.value = null
    entries.value = []
  }

  async function loadState() {
    loading.value = true
    try {
      state.value = await getVaultState()
      lock()
    }
    finally { loading.value = false }
  }

  async function create(secret: string): Promise<string> {
    busy.value = true
    try {
      const created = await createMetadata(secret)
      const result = await initializeVault(created.metadata)
      state.value = { initialized: true, metadata: created.metadata, version: result.version }
      key.value = created.key
      return created.recoveryCode
    }
    finally { busy.value = false }
  }

  async function unlock(secret: string, recovery = false) {
    if (!state.value?.metadata) throw new Error('密码库未初始化')
    busy.value = true
    try {
      const unlockedKey = await unlockMetadata(state.value.metadata, secret, recovery)
      const encrypted = await listEncryptedItems()
      const decrypted = await Promise.all(encrypted.map(async item => ({
        id: item.id, created_at: item.created_at, updated_at: item.updated_at,
        ...await decryptCredential(unlockedKey, item.id, item.payload),
      })))
      entries.value = decrypted
      key.value = unlockedKey
    }
    finally { busy.value = false }
  }

  async function add(value: Credential) {
    if (!key.value) throw new Error('请先解锁')
    busy.value = true
    try {
      const id = crypto.randomUUID()
      const payload = await encryptCredential(key.value, id, value)
      const result = await addEncryptedItems([{ id, payload }])
      const now = new Date().toISOString()
      entries.value.unshift({ ...value, id, created_at: now, updated_at: now })
      if (state.value) state.value.version = result.version
    }
    finally { busy.value = false }
  }

  async function update(id: string, value: Credential) {
    if (!key.value) throw new Error('请先解锁')
    busy.value = true
    try {
      const payload = await encryptCredential(key.value, id, value)
      const result = await updateEncryptedItem(id, payload)
      const index = entries.value.findIndex(item => item.id === id)
      if (index >= 0) entries.value[index] = { ...entries.value[index]!, ...value, updated_at: new Date().toISOString() }
      if (state.value) state.value.version = result.version
    }
    finally { busy.value = false }
  }

  async function remove(id: string) {
    busy.value = true
    try {
      const result = await deleteEncryptedItem(id)
      entries.value = entries.value.filter(item => item.id !== id)
      if (state.value) state.value.version = result.version
    }
    finally { busy.value = false }
  }

  async function importEntries(values: Credential[]): Promise<number> {
    if (!key.value) throw new Error('请先解锁')
    busy.value = true
    let imported = 0
    try {
      for (let offset = 0; offset < values.length; offset += 250) {
        const batch = values.slice(offset, offset + 250)
        const items = await Promise.all(batch.map(async value => {
          const id = crypto.randomUUID()
          return { id, payload: await encryptCredential(key.value!, id, value), value }
        }))
        const result = await addEncryptedItems(items.map(({ id, payload }) => ({ id, payload })))
        const now = new Date().toISOString()
        entries.value.push(...items.map(({ id, value }) => ({ ...value, id, created_at: now, updated_at: now })))
        if (state.value) state.value.version = result.version
        imported += items.length
      }
      return imported
    }
    catch (error) {
      if (imported) throw new Error(`已导入 ${imported} 条，后续批次失败：${error instanceof Error ? error.message : '未知错误'}`)
      throw error
    }
    finally { busy.value = false }
  }

  async function changeKey(newSecret: string): Promise<string> {
    if (!key.value || !state.value?.metadata || !state.value.version) throw new Error('请先解锁')
    busy.value = true
    try {
      const oldKey = key.value
      const created = await createMetadata(newSecret)
      const encrypted = await listEncryptedItems()
      const rotated = await Promise.all(encrypted.map(async item => {
        const plain = await decryptCredential(oldKey, item.id, item.payload)
        return { id: item.id, payload: await encryptCredential(created.key, item.id, plain) }
      }))
      const result = await rotateVault(state.value.version, created.metadata, rotated)
      state.value = { initialized: true, metadata: created.metadata, version: result.version }
      key.value = created.key
      return created.recoveryCode
    }
    finally { busy.value = false }
  }

  async function recover(recoveryCode: string, newSecret: string): Promise<string> {
    await unlock(recoveryCode, true)
    try { return await changeKey(newSecret) }
    catch (error) { lock(); throw error }
  }

  async function destroy() {
    busy.value = true
    try {
      await resetVault()
      lock()
      state.value = { initialized: false }
    }
    finally { busy.value = false }
  }

  return { state, entries, loading, busy, unlocked, lock, loadState, create, unlock, add, update, remove, importEntries, changeKey, recover, destroy }
})

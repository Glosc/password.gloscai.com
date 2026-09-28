import { request } from './client'
import type { VaultMetadata } from '@/features/vault/crypto'

interface DataResponse<T> { data: T }
export interface VaultState { initialized: boolean; metadata?: VaultMetadata; version?: number }
export interface EncryptedItem { id: string; payload: string; created_at: string; updated_at: string }
interface Version { version: number }

const writeHeaders = { 'X-Vault-Request': '1' }

export async function getVaultState(): Promise<VaultState> {
  return (await request<DataResponse<VaultState>>('/vault')).data
}

export async function initializeVault(metadata: VaultMetadata): Promise<Version> {
  return (await request<DataResponse<Version>>('/vault', { method: 'POST', headers: writeHeaders, body: JSON.stringify({ metadata }) })).data
}

export async function listEncryptedItems(): Promise<EncryptedItem[]> {
  return (await request<DataResponse<EncryptedItem[]>>('/vault/items')).data
}

export async function addEncryptedItems(items: { id: string; payload: string }[]): Promise<Version> {
  return (await request<DataResponse<Version>>('/vault/items', { method: 'POST', headers: writeHeaders, body: JSON.stringify({ items }) })).data
}

export async function updateEncryptedItem(id: string, payload: string): Promise<Version> {
  return (await request<DataResponse<Version>>(`/vault/items/${id}`, { method: 'PUT', headers: writeHeaders, body: JSON.stringify({ payload }) })).data
}

export async function deleteEncryptedItem(id: string): Promise<Version> {
  return (await request<DataResponse<Version>>(`/vault/items/${id}`, { method: 'DELETE', headers: writeHeaders, body: JSON.stringify({}) })).data
}

export async function rotateVault(expectedVersion: number, metadata: VaultMetadata, items: { id: string; payload: string }[]): Promise<Version> {
  return (await request<DataResponse<Version>>('/vault/keys', { method: 'PUT', headers: writeHeaders, body: JSON.stringify({ expected_version: expectedVersion, metadata, items }) })).data
}

export async function resetVault(): Promise<void> {
  await request<void>('/vault', { method: 'DELETE', headers: writeHeaders, body: JSON.stringify({ confirm: 'DELETE' }) })
}

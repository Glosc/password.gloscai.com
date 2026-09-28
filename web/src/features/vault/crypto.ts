import { argon2id } from 'hash-wasm'

export interface Credential {
  title: string
  url: string
  username: string
  password: string
  note: string
  favorite: boolean
}

export interface CipherBlob { iv: string; data: string }
export interface VaultMetadata {
  v: 1
  salt: string
  recoverySalt: string
  keyWrap: CipherBlob
  recoveryWrap: CipherBlob
}

const encoder = new TextEncoder()
const decoder = new TextDecoder()

function randomBytes(length: number): Uint8Array<ArrayBuffer> {
  return crypto.getRandomValues(new Uint8Array(length))
}

function toBase64(bytes: Uint8Array): string {
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary)
}

function fromBase64(value: string): Uint8Array<ArrayBuffer> {
  return Uint8Array.from(atob(value), char => char.charCodeAt(0))
}

async function importKey(bytes: Uint8Array): Promise<CryptoKey> {
  return crypto.subtle.importKey('raw', Uint8Array.from(bytes), 'AES-GCM', false, ['encrypt', 'decrypt'])
}

async function deriveKey(secret: string, salt: string): Promise<CryptoKey> {
  const derived = await argon2id({
    password: secret,
    salt: fromBase64(salt),
    parallelism: 1,
    iterations: 3,
    memorySize: 32768,
    hashLength: 32,
    outputType: 'binary',
  })
  return importKey(derived)
}

async function encrypt(key: CryptoKey, bytes: Uint8Array, aad: string): Promise<CipherBlob> {
  const iv = randomBytes(12)
  const encrypted = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv, additionalData: encoder.encode(aad) },
    key,
    Uint8Array.from(bytes),
  )
  return { iv: toBase64(iv), data: toBase64(new Uint8Array(encrypted)) }
}

async function decrypt(key: CryptoKey, blob: CipherBlob, aad: string): Promise<Uint8Array<ArrayBuffer>> {
  return new Uint8Array(await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv: fromBase64(blob.iv), additionalData: encoder.encode(aad) },
    key,
    fromBase64(blob.data),
  ))
}

export function randomSecret(): string {
  return toBase64(randomBytes(32)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

export async function createMetadata(userSecret: string): Promise<{ metadata: VaultMetadata; recoveryCode: string; key: CryptoKey }> {
  const dataKey = randomBytes(32)
  const salt = toBase64(randomBytes(16))
  const recoverySalt = toBase64(randomBytes(16))
  const recoveryCode = randomSecret()
  const [userKey, recoveryKey] = await Promise.all([
    deriveKey(userSecret, salt), deriveKey(recoveryCode, recoverySalt),
  ])
  const [keyWrap, recoveryWrap] = await Promise.all([
    encrypt(userKey, dataKey, 'vault:key:v1'),
    encrypt(recoveryKey, dataKey, 'vault:recovery:v1'),
  ])
  return {
    metadata: { v: 1, salt, recoverySalt, keyWrap, recoveryWrap },
    recoveryCode,
    key: await importKey(dataKey),
  }
}

export async function unlockMetadata(metadata: VaultMetadata, secret: string, recovery = false): Promise<CryptoKey> {
  if (metadata.v !== 1) throw new Error('不支持的密码库版本')
  try {
    const key = await deriveKey(secret, recovery ? metadata.recoverySalt : metadata.salt)
    const bytes = await decrypt(key, recovery ? metadata.recoveryWrap : metadata.keyWrap, recovery ? 'vault:recovery:v1' : 'vault:key:v1')
    return importKey(bytes)
  }
  catch {
    throw new Error(recovery ? '恢复码不正确' : '安全 Key 不正确')
  }
}

export async function encryptCredential(key: CryptoKey, id: string, value: Credential): Promise<string> {
  return JSON.stringify(await encrypt(key, encoder.encode(JSON.stringify(value)), `vault:item:${id}`))
}

export async function decryptCredential(key: CryptoKey, id: string, payload: string): Promise<Credential> {
  const value = JSON.parse(decoder.decode(await decrypt(key, JSON.parse(payload) as CipherBlob, `vault:item:${id}`))) as Credential
  if (typeof value.title !== 'string' || typeof value.password !== 'string') throw new Error('凭据数据已损坏')
  return { title: value.title, url: value.url || '', username: value.username || '', password: value.password, note: value.note || '', favorite: !!value.favorite }
}

export interface EncryptedBackup { format: 'glosc-vault-backup'; v: 1; salt: string; blob: CipherBlob }

export async function makeBackup(passphrase: string, entries: Credential[]): Promise<EncryptedBackup> {
  const salt = toBase64(randomBytes(16))
  const key = await deriveKey(passphrase, salt)
  return { format: 'glosc-vault-backup', v: 1, salt, blob: await encrypt(key, encoder.encode(JSON.stringify(entries)), 'vault:backup:v1') }
}

export async function openBackup(backup: EncryptedBackup, passphrase: string): Promise<Credential[]> {
  if (backup.format !== 'glosc-vault-backup' || backup.v !== 1) throw new Error('不支持的备份格式')
  try {
    const key = await deriveKey(passphrase, backup.salt)
    const values = JSON.parse(decoder.decode(await decrypt(key, backup.blob, 'vault:backup:v1')))
    if (!Array.isArray(values)) throw new Error('invalid backup')
    return values as Credential[]
  }
  catch { throw new Error('备份口令不正确或文件已损坏') }
}

export interface GeneratorOptions { length: number; lower: boolean; upper: boolean; digits: boolean; symbols: boolean }

function randomIndex(limit: number): number {
  const cutoff = 256 - 256 % limit
  while (true) {
    const value = randomBytes(1)[0]!
    if (value < cutoff) return value % limit
  }
}

export function generatePassword(options: GeneratorOptions): string {
  const groups = [
    options.lower && 'abcdefghijkmnopqrstuvwxyz',
    options.upper && 'ABCDEFGHJKLMNPQRSTUVWXYZ',
    options.digits && '23456789',
    options.symbols && '!@#$%^&*()-_=+[]{}',
  ].filter((value): value is string => !!value)
  if (!groups.length || !Number.isInteger(options.length) || options.length < 12 || options.length > 128 || options.length < groups.length) {
    throw new Error('请选择字符类型，并将长度设为 12 到 128 位')
  }
  const alphabet = groups.join('')
  const result = groups.map(group => group[randomIndex(group.length)]!)
  while (result.length < options.length) result.push(alphabet[randomIndex(alphabet.length)]!)
  for (let i = result.length - 1; i > 0; i--) {
    const j = randomIndex(i + 1)
    ;[result[i], result[j]] = [result[j]!, result[i]!]
  }
  return result.join('')
}

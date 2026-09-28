import JSZip from 'jszip'
import Papa from 'papaparse'
import type { Credential, EncryptedBackup } from './crypto'
import { openBackup } from './crypto'

export interface ImportPreview { entries: Credential[]; skipped: number; source: string }

function text(value: unknown): string { return typeof value === 'string' ? value.trim() : '' }
function record(value: unknown): Record<string, unknown> {
  return value && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : {}
}

function normalize(value: unknown): Credential | null {
  const item = record(value)
  const title = text(item.title) || text(item.url) || text(item.username)
  if (!title || !text(item.password)) return null
  return { title, url: text(item.url), username: text(item.username), password: String(item.password), note: text(item.note), favorite: !!item.favorite }
}

function fromCSV(source: string): ImportPreview {
  const parsed = Papa.parse<Record<string, string>>(source.replace(/^\uFEFF/, ''), { header: true, skipEmptyLines: 'greedy', transformHeader: header => header.trim().toLowerCase() })
  if (parsed.errors.length || !parsed.meta.fields?.length) throw new Error('CSV 格式无效')
  const entries: Credential[] = []
  let skipped = 0
  for (const row of parsed.data) {
    const get = (...keys: string[]) => keys.map(key => row[key]).find(Boolean) || ''
    const type = get('type')
    if (type && !['login', '1'].includes(type.toLowerCase())) { skipped++; continue }
    const entry = normalize({
      title: get('name', 'title'),
      url: get('url', 'login_uri', 'website', 'website url', 'login_url'),
      username: get('username', 'login_username', 'user name'),
      password: get('password', 'login_password'),
      note: get('note', 'notes', 'comments'),
      favorite: get('favorite') === '1',
    })
    if (entry) entries.push(entry)
    else skipped++
  }
  return { entries, skipped, source: 'CSV' }
}

function fromBitwardenJSON(value: unknown): ImportPreview {
  const items = record(value).items
  if (!Array.isArray(items)) throw new Error('Bitwarden JSON 格式无效')
  const entries: Credential[] = []
  let skipped = 0
  for (const raw of items) {
    const item = record(raw)
    const login = record(item.login)
    if (item.type !== 1 && item.type !== 'login') { skipped++; continue }
    const uris = Array.isArray(login.uris) ? login.uris : []
    const entry = normalize({
      title: text(item.name), url: text(record(uris[0]).uri),
      username: text(login.username), password: text(login.password),
      note: text(item.notes), favorite: item.favorite === true,
    })
    if (entry) entries.push(entry)
    else skipped++
  }
  return { entries, skipped, source: 'Bitwarden JSON' }
}

function fromOnePassword(value: unknown): ImportPreview {
  const accounts = record(value).accounts
  if (!Array.isArray(accounts)) throw new Error('1PUX 格式无效')
  const entries: Credential[] = []
  let skipped = 0
  for (const accountRaw of accounts) {
    const vaults = record(accountRaw).vaults
    if (!Array.isArray(vaults)) continue
    for (const vaultRaw of vaults) {
      const items = record(vaultRaw).items
      if (!Array.isArray(items)) continue
      for (const itemRaw of items) {
        const item = record(itemRaw)
        const overview = record(item.overview)
        const details = record(item.details)
        const fields = Array.isArray(details.fields) ? details.fields : []
        const find = (designation: string) => {
          const field = fields.map(record).find(candidate => text(candidate.designation).toLowerCase() === designation)
          return text(field?.value)
        }
        const urls = Array.isArray(overview.urls) ? overview.urls : []
        const entry = item.categoryUuid === '001' ? normalize({
          title: text(overview.title),
          url: text(record(urls[0]).url),
          username: find('username'), password: find('password'),
          note: text(details.notesPlain), favorite: overview.faveIndex === 1,
        }) : null
        if (entry) entries.push(entry)
        else skipped++
      }
    }
  }
  return { entries, skipped, source: '1Password 1PUX' }
}

export async function previewImport(file: File, backupPassphrase = ''): Promise<ImportPreview> {
  if (file.size > 20 * 1024 * 1024) throw new Error('文件不能超过 20 MB')
  const name = file.name.toLowerCase()
  let preview: ImportPreview
  if (name.endsWith('.csv')) preview = fromCSV(await file.text())
  else if (name.endsWith('.1pux')) {
    const zip = await JSZip.loadAsync(file)
    const entry = zip.file('export.data')
    if (!entry) throw new Error('1PUX 缺少 export.data')
    const info = entry as unknown as { _data?: { uncompressedSize?: number } }
    if ((info._data?.uncompressedSize || 0) > 20 * 1024 * 1024) throw new Error('1PUX 内容过大')
    preview = fromOnePassword(JSON.parse(await entry.async('string')))
  }
  else if (name.endsWith('.json')) {
    const content = JSON.parse(await file.text()) as unknown
    if (record(content).format === 'glosc-vault-backup') {
      if (!backupPassphrase) throw new Error('请输入备份口令后预览')
      const entries = await openBackup(content as EncryptedBackup, backupPassphrase)
      const valid = entries.map(normalize).filter((item): item is Credential => item !== null)
      preview = { entries: valid, skipped: entries.length - valid.length, source: 'Glosc 加密备份' }
    }
    else preview = fromBitwardenJSON(content)
  }
  else throw new Error('请选择 CSV、Bitwarden JSON、1PUX 或 Glosc 加密备份')
  if (preview.entries.length > 5000) throw new Error('单次最多导入 5000 条登录凭据')
  return preview
}

function safeCSV(value: string): string {
  const escaped = /^[=+\-@\t\r]/.test(value) ? `'${value}` : value
  return `"${escaped.replace(/"/g, '""')}"`
}

export function exportCSV(entries: Credential[]): string {
  return ['name,url,username,password,note', ...entries.map(entry => [entry.title, entry.url, entry.username, entry.password, entry.note].map(safeCSV).join(','))].join('\r\n')
}

export function downloadFile(name: string, contents: string, type: string): void {
  const url = URL.createObjectURL(new Blob([contents], { type }))
  const link = document.createElement('a')
  link.href = url
  link.download = name
  link.click()
  setTimeout(() => URL.revokeObjectURL(url), 60_000)
}

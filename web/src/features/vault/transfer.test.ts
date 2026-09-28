import JSZip from 'jszip'
import { describe, expect, it } from 'vitest'
import { exportCSV, previewImport } from './transfer'

describe('credential transfer', () => {
  it('maps browser and Bitwarden CSV fields', async () => {
    const browser = new File(['name,url,username,password,note\nExample,https://example.com,alice,secret,hello'], 'chrome.csv')
    expect((await previewImport(browser)).entries[0]?.username).toBe('alice')
    const bitwarden = new File(['type,name,login_uri,login_username,login_password\nlogin,GitHub,https://github.com,bob,secret\nnote,Ignore,,,'], 'bitwarden.csv')
    const result = await previewImport(bitwarden)
    expect(result.entries).toHaveLength(1)
    expect(result.skipped).toBe(1)
  })

  it('maps Bitwarden JSON and 1Password 1PUX logins', async () => {
    const bw = new File([JSON.stringify({ items: [{ type: 1, name: 'Mail', login: { username: 'a', password: 'secret', uris: [{ uri: 'https://mail.test' }] } }, { type: 2, name: 'Note' }] })], 'bitwarden.json')
    expect((await previewImport(bw)).entries[0]?.title).toBe('Mail')
    const zip = new JSZip()
    zip.file('export.data', JSON.stringify({ accounts: [{ vaults: [{ items: [{ categoryUuid: '001', overview: { title: 'Forum', urls: [{ url: 'https://forum.test' }] }, details: { fields: [{ designation: 'username', value: 'bob' }, { designation: 'password', value: 'hidden' }] } }] }] }] }))
    const file = new File([Uint8Array.from(await zip.generateAsync({ type: 'uint8array' }))], 'export.1pux')
    expect((await previewImport(file)).entries[0]?.password).toBe('hidden')
  })

  it('rejects malformed input and neutralizes spreadsheet formulas', async () => {
    await expect(previewImport(new File(['not json'], 'bad.json'))).rejects.toThrow()
    expect(exportCSV([{ title: '=SUM(1,1)', url: '', username: '', password: '+cmd', note: '', favorite: false }])).toContain("\"'=SUM(1,1)\"")
    expect(exportCSV([{ title: '=SUM(1,1)', url: '', username: '', password: '+cmd', note: '', favorite: false }])).toContain("\"'+cmd\"")
  })
})

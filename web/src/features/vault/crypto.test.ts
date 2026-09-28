import { describe, expect, it } from 'vitest'
import {
  createMetadata, decryptCredential, encryptCredential, generatePassword,
  makeBackup, openBackup, unlockMetadata,
} from './crypto'

const credential = { title: 'Example', url: 'https://example.com', username: 'alice', password: 'p@ssword', note: 'private', favorite: true }
const id = '65c68698-970c-42eb-a232-9803cd79c674'

describe('vault cryptography', () => {
  it('unlocks with the user key or recovery code, but not a wrong key', async () => {
    const created = await createMetadata('a sufficiently long private phrase')
    const payload = await encryptCredential(created.key, id, credential)
    expect(await decryptCredential(await unlockMetadata(created.metadata, 'a sufficiently long private phrase'), id, payload)).toEqual(credential)
    expect(await decryptCredential(await unlockMetadata(created.metadata, created.recoveryCode, true), id, payload)).toEqual(credential)
    await expect(unlockMetadata(created.metadata, 'incorrect user key')).rejects.toThrow('不正确')
    await expect(decryptCredential(created.key, crypto.randomUUID(), payload)).rejects.toThrow()
  })

  it('re-encrypts each item under a new key and makes the old key useless', async () => {
    const old = await createMetadata('old private phrase 123')
    const next = await createMetadata('new private phrase 456')
    const oldPayload = await encryptCredential(old.key, id, credential)
    const plain = await decryptCredential(old.key, id, oldPayload)
    const newPayload = await encryptCredential(next.key, id, plain)
    expect(await decryptCredential(next.key, id, newPayload)).toEqual(credential)
    await expect(decryptCredential(old.key, id, newPayload)).rejects.toThrow()
  })

  it('makes a portable backup with an independent passphrase', async () => {
    const backup = await makeBackup('backup phrase that is long', [credential])
    expect(await openBackup(backup, 'backup phrase that is long')).toEqual([credential])
    await expect(openBackup(backup, 'wrong')).rejects.toThrow()
  })
})

describe('password generator', () => {
  it('respects length and selected character groups', () => {
    for (let index = 0; index < 20; index++) {
      const password = generatePassword({ length: 32, lower: true, upper: true, digits: true, symbols: true })
      expect(password).toHaveLength(32)
      expect(password).toMatch(/[a-z]/)
      expect(password).toMatch(/[A-Z]/)
      expect(password).toMatch(/[0-9]/)
      expect(password).toMatch(/[^a-zA-Z0-9]/)
    }
    expect(() => generatePassword({ length: 8, lower: true, upper: false, digits: false, symbols: false })).toThrow()
    expect(() => generatePassword({ length: 20, lower: false, upper: false, digits: false, symbols: false })).toThrow()
  })
})

import { describe, expect, it } from 'vitest'
import {
  AUTH_TEXT_MAX_LENGTH,
  createAuthDraft,
  validateAuthDraft,
} from '@/utils/auth'

describe('validateAuthDraft', () => {
  it('accepts a valid login payload and trims email', () => {
    const result = validateAuthDraft(
      'login',
      createAuthDraft({
        email: '  user@example.com  ',
        password: 'Valid1!A',
      }),
    )

    expect(result.values.email).toBe('user@example.com')
    expect(result.fieldErrors).toEqual({})
  })

  it('rejects passwords that do not satisfy the SSO password policy', () => {
    const result = validateAuthDraft(
      'login',
      createAuthDraft({
        email: 'user@example.com',
        password: 'NoSymbol12',
      }),
    )

    expect(result.fieldErrors.password).toBe('passwordPolicy')
  })

  it('requires matching confirmation and names during signup', () => {
    const result = validateAuthDraft(
      'signup',
      createAuthDraft({
        email: 'user@example.com',
        password: 'Valid1!A',
        confirmPassword: 'Mismatch1!',
        firstName: '',
        lastName: '',
      }),
    )

    expect(result.fieldErrors.firstName).toBe('required')
    expect(result.fieldErrors.lastName).toBe('required')
    expect(result.fieldErrors.confirmPassword).toBe('mismatch')
  })

  it('rejects emails and names that exceed SSO field limits', () => {
    const tooLong = 'a'.repeat(AUTH_TEXT_MAX_LENGTH + 1)
    const result = validateAuthDraft(
      'signup',
      createAuthDraft({
        email: `${tooLong}@example.com`,
        password: 'Valid1!A',
        confirmPassword: 'Valid1!A',
        firstName: tooLong,
        lastName: tooLong,
      }),
    )

    expect(result.fieldErrors.email).toBe('maxLength')
    expect(result.fieldErrors.firstName).toBe('maxLength')
    expect(result.fieldErrors.lastName).toBe('maxLength')
  })
})

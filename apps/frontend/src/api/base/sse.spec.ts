import { describe, expect, it } from 'vitest'
import { extractSseDataMessages } from './sse'

describe('extractSseDataMessages', () => {
  it('extracts data payloads and keeps incomplete events buffered', () => {
    const parsed = extractSseDataMessages(
      ': heartbeat\n\n' +
        'data: {"resource":"rooms"}\n\n' +
        'data: {"resource":"servers"}',
    )

    expect(parsed.messages).toEqual(['{"resource":"rooms"}'])
    expect(parsed.remaining).toBe('data: {"resource":"servers"}')
  })

  it('joins multi-line data payloads', () => {
    const parsed = extractSseDataMessages('data: {"a":1,\ndata: "b":2}\n\n')

    expect(parsed.messages).toEqual(['{"a":1,\n"b":2}'])
    expect(parsed.remaining).toBe('')
  })
})

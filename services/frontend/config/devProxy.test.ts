// feat:if backend
import { describe, expect, it } from 'vitest'

import { parseBackendPort, resolveBackendPort, resolveProxyTarget } from './devProxy'

describe('parseBackendPort', () => {
  it.each([
    ['export MY_PROJECT_API_FIBER_PORT="9001"', '9001'],
    ['MY_PROJECT_API_FIBER_PORT=9001', '9001'],
    ['export MY_PROJECT_API_FIBER_PORT=9001', '9001'],
    ['# export MY_PROJECT_API_FIBER_PORT="9001"', undefined],
  ])('reads %j as %j', (line, want) => {
    expect(parseBackendPort(line)).toBe(want)
  })

  it('finds the port among other variables', () => {
    const file = [
      'export MY_PROJECT_API_POSTGRESQL_HOST="postgresql"',
      'export MY_PROJECT_API_FIBER_PORT="9001"',
      'export MY_PROJECT_API_JWT_SECRET="secret"',
    ].join('\n')

    expect(parseBackendPort(file)).toBe('9001')
  })

  it('reports nothing when the variable is absent', () => {
    expect(parseBackendPort('export MY_PROJECT_API_FIBER_HOST="0.0.0.0"')).toBeUndefined()
  })
})

describe('resolveBackendPort', () => {
  it('prefers the environment over the env file', () => {
    expect(resolveBackendPort('.', { MY_PROJECT_API_FIBER_PORT: '7777' })).toBe('7777')
  })

  it('falls back to 8000 when nothing declares a port', () => {
    expect(resolveBackendPort('/nonexistent', {})).toBe('8000')
  })
})

describe('resolveProxyTarget', () => {
  it('targets the backend container when PROXY_TARGET is set', () => {
    const env = {
      MY_PROJECT_API_PROXY_TARGET: 'http://backend:8000',
      MY_PROJECT_API_FIBER_PORT: '7777',
    }

    expect(resolveProxyTarget('.', env)).toBe('http://backend:8000')
  })

  it('targets localhost on the resolved port otherwise', () => {
    expect(resolveProxyTarget('.', { MY_PROJECT_API_FIBER_PORT: '7777' })).toBe(
      'http://localhost:7777',
    )
  })
})
// feat:end

// feat:if backend
// Dev-server proxy configuration, kept out of vite.config.ts so it can be
// tested. The SPA always calls a relative /api/v1 base (same origin), so it
// works unchanged behind the prod nginx proxy; in dev the vite server proxies
// /api to the backend.
import fs from 'node:fs'
import path from 'node:path'

/** Where the backend port is declared for host runs. */
export const API_ENV_FILE = '../../env/local/api.env'

const PORT_LINE = /^\s*(?:export\s+)?MY_PROJECT_API_FIBER_PORT\s*=\s*"?([^"\s]+)"?\s*$/m

/**
 * Reads the backend port out of an env file's contents. Exported separately
 * from the file read so the parsing rule can be tested without a filesystem:
 * the line may or may not carry `export`, and may or may not be quoted.
 */
export function parseBackendPort(envFileContents: string): string | undefined {
  return envFileContents.match(PORT_LINE)?.[1]
}

function portFromEnvFile(dir: string): string | undefined {
  try {
    return parseBackendPort(fs.readFileSync(path.resolve(dir, API_ENV_FILE), 'utf8'))
  } catch {
    return undefined
  }
}

/**
 * Resolves the backend port, most explicit source first: the environment, then
 * env/local/api.env, then the default. Changing the port in one place must
 * move both the published host port and the port the dev server proxies to.
 */
export function resolveBackendPort(
  dir: string,
  env: NodeJS.ProcessEnv = process.env,
): string {
  return env.MY_PROJECT_API_FIBER_PORT ?? portFromEnvFile(dir) ?? '8000'
}

/**
 * Resolves the proxy target. PROXY_TARGET is set when the frontend itself runs
 * in Docker and must reach the backend container rather than localhost.
 */
export function resolveProxyTarget(
  dir: string,
  env: NodeJS.ProcessEnv = process.env,
): string {
  return env.MY_PROJECT_API_PROXY_TARGET ?? `http://localhost:${resolveBackendPort(dir, env)}`
}
// feat:end

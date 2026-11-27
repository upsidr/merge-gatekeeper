// Proxy support for the built-in fetch.
//
// Node's fetch ignores HTTPS_PROXY / HTTP_PROXY / NO_PROXY unless
// NODE_USE_ENV_PROXY=1 is set when the process starts. v1 (Go) honoured them,
// so runners that can only reach GitHub through a proxy would lose access in
// v2. http.setGlobalProxyFromEnv() (Node >= 24.14) turns the same support on
// after startup, which is the only point an action can hook into.
import http from "node:http";

const PROXY_VARS = ["HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"];

/**
 * Routes fetch through the proxy configured in the environment, if any.
 * NO_PROXY / no_proxy is honoured by Node.
 *
 * @param {NodeJS.ProcessEnv} env
 * @param {{ httpModule?: { setGlobalProxyFromEnv?: (env?: object) => unknown }, log?: (s: string) => void, warn?: (s: string) => void }} [deps]
 * @returns {"none" | "already-enabled" | "enabled" | "unsupported"}
 */
export function useProxyFromEnv(env, { httpModule = http, log = () => {}, warn = () => {} } = {}) {
  const configured = PROXY_VARS.filter((k) => env[k]);
  if (configured.length === 0) return "none";

  // The proxy URL can contain credentials, so only variable names are logged.
  const names = configured.join(", ");
  if (env.NODE_USE_ENV_PROXY === "1") {
    log(`Using the proxy from ${names} (NODE_USE_ENV_PROXY=1).`);
    return "already-enabled";
  }
  if (typeof httpModule.setGlobalProxyFromEnv === "function") {
    httpModule.setGlobalProxyFromEnv(env);
    log(`Using the proxy from ${names}.`);
    return "enabled";
  }
  warn(
    `${names} is set, but this Node.js (${process.version}) cannot apply it to fetch. ` +
      "Update the Actions runner, or set NODE_USE_ENV_PROXY=1 in the job environment.",
  );
  return "unsupported";
}

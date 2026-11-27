import assert from "node:assert/strict";
import http from "node:http";
import { describe, test } from "node:test";

import { useProxyFromEnv } from "../src/proxy.js";

function fakeHttp() {
  const calls = [];
  return { calls, httpModule: { setGlobalProxyFromEnv: (env) => calls.push(env) } };
}

describe("useProxyFromEnv", () => {
  test("does nothing without proxy variables", () => {
    const { calls, httpModule } = fakeHttp();
    assert.equal(useProxyFromEnv({}, { httpModule }), "none");
    assert.equal(calls.length, 0);
  });

  for (const name of ["HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"]) {
    test(`enables the proxy from ${name}`, () => {
      const { calls, httpModule } = fakeHttp();
      const env = { [name]: "http://proxy:3128" };
      assert.equal(useProxyFromEnv(env, { httpModule }), "enabled");
      assert.deepEqual(calls, [env]);
    });
  }

  test("leaves it to Node when NODE_USE_ENV_PROXY=1", () => {
    const { calls, httpModule } = fakeHttp();
    assert.equal(useProxyFromEnv({ HTTPS_PROXY: "http://p", NODE_USE_ENV_PROXY: "1" }, { httpModule }), "already-enabled");
    assert.equal(calls.length, 0);
  });

  test("warns when this Node cannot apply the proxy", () => {
    const warnings = [];
    const res = useProxyFromEnv({ HTTPS_PROXY: "http://p" }, { httpModule: {}, warn: (m) => warnings.push(m) });
    assert.equal(res, "unsupported");
    assert.match(warnings[0], /NODE_USE_ENV_PROXY=1/);
  });

  test("never logs the proxy URL, which may contain credentials", () => {
    const { httpModule } = fakeHttp();
    const logs = [];
    useProxyFromEnv({ HTTPS_PROXY: "http://user:secret@proxy:3128" }, { httpModule, log: (m) => logs.push(m) });
    assert.equal(logs.length, 1);
    assert.doesNotMatch(logs[0], /secret|proxy:3128/);
  });
});

describe("fetch goes through the proxy (real Node behaviour)", { skip: typeof http.setGlobalProxyFromEnv !== "function" }, () => {
  test("HTTPS_PROXY is used for the API, and NO_PROXY bypasses it", async () => {
    // A local proxy that records CONNECT requests and refuses them, so no
    // network is needed. Node tunnels through the proxy with CONNECT.
    const seen = [];
    const proxy = http.createServer().on("connect", (req, socket) => {
      seen.push(req.url);
      socket.end("HTTP/1.1 502 Bad Gateway\r\n\r\n");
    });
    await new Promise((r) => proxy.listen(0, "127.0.0.1", r));
    const proxyUrl = `http://127.0.0.1:${proxy.address().port}`;

    let restore;
    try {
      const env = { HTTPS_PROXY: proxyUrl, NO_PROXY: "bypassed.invalid" };
      const realHttp = { setGlobalProxyFromEnv: (e) => (restore = http.setGlobalProxyFromEnv(e)) };
      assert.equal(useProxyFromEnv(env, { httpModule: realHttp }), "enabled");

      await assert.rejects(fetch("https://api.example.invalid/zen", { signal: AbortSignal.timeout(5000) }));
      assert.deepEqual(seen, ["api.example.invalid:443"]);

      // NO_PROXY: goes direct (and fails DNS), without touching the proxy.
      await assert.rejects(fetch("https://bypassed.invalid/zen", { signal: AbortSignal.timeout(5000) }));
      assert.deepEqual(seen, ["api.example.invalid:443"]);
    } finally {
      restore?.();
      proxy.closeAllConnections();
      proxy.close();
    }
  });
});

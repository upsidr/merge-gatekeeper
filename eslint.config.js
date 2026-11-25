// ESLint config. ESLint runs in CI only, via `npx eslint@<pinned>` (see
// .github/workflows/build-ci.yaml), so the action itself stays dependency-free.
// This file must not import any package: npx does not make them resolvable
// from here. Rules are therefore listed explicitly instead of extending a
// shared preset.

// Globals available in the node24 runtime that this code uses.
const nodeGlobals = Object.fromEntries(
  [
    "AbortController",
    "AbortSignal",
    "DOMException",
    "Response",
    "URL",
    "clearTimeout",
    "console",
    "fetch",
    "process",
    "setTimeout",
  ].map((name) => [name, "readonly"]),
);

export default [
  {
    files: ["**/*.js"],
    languageOptions: {
      ecmaVersion: 2024,
      sourceType: "module",
      globals: nodeGlobals,
    },
    linterOptions: {
      reportUnusedDisableDirectives: "error",
    },
    rules: {
      // Correctness
      "no-undef": "error",
      "no-unused-vars": ["error", { argsIgnorePattern: "^_", caughtErrors: "none" }],
      "no-unreachable": "error",
      "no-dupe-keys": "error",
      "no-duplicate-case": "error",
      "no-self-compare": "error",
      "no-constant-binary-expression": "error",
      "no-unsafe-finally": "error",
      "no-unsafe-optional-chaining": "error",
      "no-async-promise-executor": "error",
      "no-promise-executor-return": "error",
      "require-atomic-updates": "off",
      "use-isnan": "error",
      "valid-typeof": "error",
      eqeqeq: ["error", "always", { null: "ignore" }],

      // Security: no dynamic code execution.
      "no-eval": "error",
      "no-implied-eval": "error",
      "no-new-func": "error",

      // Style that prevents bugs
      "no-var": "error",
      "prefer-const": "error",
      "no-shadow": "error",
      "no-restricted-syntax": [
        "error",
        {
          selector: "CallExpression[callee.name='require']",
          message: "Use ES module imports.",
        },
      ],
      "no-restricted-imports": [
        "error",
        {
          // Keep the action dependency-free: only Node built-ins and relative files.
          patterns: [{ regex: "^(?!node:|\\.{1,2}/)", message: "Only node: built-ins and relative imports are allowed (no dependencies)." }],
        },
      ],
    },
  },
];

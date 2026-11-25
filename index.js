// Entry point for the Merge Gatekeeper JavaScript action (runs.using: node24).
// No dependencies: everything uses Node built-ins.
import { readConfig, run } from "./src/run.js";

// Workflow commands need newlines and "%" escaped to survive in a single line.
const escapeData = (s) => String(s).replace(/%/g, "%25").replace(/\r/g, "%0D").replace(/\n/g, "%0A");

try {
  const result = await run(readConfig(process.env));
  if (!result.ok) {
    console.log(`::error::${escapeData(result.message)}`);
    process.exitCode = 1;
  }
} catch (err) {
  console.log(`::error::${escapeData(err?.message ?? err)}`);
  process.exitCode = 1;
}

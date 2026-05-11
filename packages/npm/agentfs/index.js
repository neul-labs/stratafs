"use strict";

const fs = require("fs");
const os = require("os");
const path = require("path");
const { spawnSync } = require("child_process");

function getPlatform() {
  const platform = os.platform();
  if (platform === "darwin") return "darwin";
  if (platform === "win32") return "windows";
  return "linux";
}

function getBinaryPath() {
  const binaryName = getPlatform() === "windows" ? "agentfs.exe" : "agentfs";
  const binaryDir = path.join(os.homedir(), ".agentfs", "bin");
  return path.join(binaryDir, binaryName);
}

function main() {
  const binaryPath = getBinaryPath();

  if (!fs.existsSync(binaryPath)) {
    console.error("AgentFS binary not found. Running postinstall to download...");
    require("./install.js");
  }

  if (!fs.existsSync(binaryPath)) {
    console.error("Failed to locate AgentFS binary.", binaryPath);
    process.exit(1);
  }

  const args = process.argv.slice(2);
  const result = spawnSync(binaryPath, args, {
    stdio: "inherit",
    shell: false,
  });

  if (result.error) {
    console.error("Failed to run AgentFS:", result.error.message);
    process.exit(1);
  }

  process.exit(result.status || 0);
}

main();

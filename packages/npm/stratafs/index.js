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
  const binaryName = getPlatform() === "windows" ? "stratafs.exe" : "stratafs";
  const binaryDir = path.join(os.homedir(), ".stratafs", "bin");
  return path.join(binaryDir, binaryName);
}

function main() {
  const binaryPath = getBinaryPath();

  if (!fs.existsSync(binaryPath)) {
    console.error("StrataFS binary not found. Running postinstall to download...");
    require("./install.js");
  }

  if (!fs.existsSync(binaryPath)) {
    console.error("Failed to locate StrataFS binary.", binaryPath);
    process.exit(1);
  }

  const args = process.argv.slice(2);
  const result = spawnSync(binaryPath, args, {
    stdio: "inherit",
    shell: false,
  });

  if (result.error) {
    console.error("Failed to run StrataFS:", result.error.message);
    process.exit(1);
  }

  process.exit(result.status || 0);
}

main();

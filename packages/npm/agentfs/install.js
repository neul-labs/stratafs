"use strict";

const fs = require("fs");
const os = require("os");
const path = require("path");
const { execSync } = require("child_process");

const VERSION = process.env.AGENTFS_VERSION || "0.2.0";
const GITHUB_RELEASES_URL = "https://github.com/dipankarsarkar/agentfs/releases/download";

function getPlatform() {
  const platform = os.platform();
  if (platform === "darwin") return "darwin";
  if (platform === "win32") return "windows";
  return "linux";
}

function getArch() {
  const arch = os.arch();
  if (arch === "x64") return "amd64";
  if (arch === "arm64") return "arm64";
  return arch;
}

function getBinaryDir() {
  const dir = path.join(os.homedir(), ".agentfs", "bin");
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
  return dir;
}

function getBinaryPath() {
  const binaryName = getPlatform() === "windows" ? "agentfs.exe" : "agentfs";
  return path.join(getBinaryDir(), binaryName);
}

async function downloadFile(url, dest) {
  const response = await fetch(url, { redirect: "follow" });
  if (!response.ok) {
    throw new Error(`Download failed with status ${response.status}: ${url}`);
  }
  const buffer = await response.arrayBuffer();
  fs.writeFileSync(dest, Buffer.from(buffer));
}

async function downloadBinary() {
  const platform = getPlatform();
  const arch = getArch();
  const binaryDir = getBinaryDir();
  const binaryPath = getBinaryPath();

  if (fs.existsSync(binaryPath)) {
    return binaryPath;
  }

  const archiveName = `agentfs-v${VERSION}-${platform}-${arch}`;
  const archiveExt = platform === "windows" ? "zip" : "tar.gz";
  const url = `${GITHUB_RELEASES_URL}/v${VERSION}/${archiveName}.${archiveExt}`;
  const archivePath = path.join(binaryDir, `${archiveName}.${archiveExt}`);

  console.log(`Downloading AgentFS ${VERSION} for ${platform}/${arch}...`);

  await downloadFile(url, archivePath);

  if (archiveExt === "zip") {
    execSync(`unzip -o "${archivePath}" -d "${binaryDir}"`, { stdio: "inherit" });
    const extractedBinary = path.join(binaryDir, `${platform}-${arch}`, "agentfs.exe");
    if (fs.existsSync(extractedBinary)) {
      fs.renameSync(extractedBinary, binaryPath);
    }
  } else {
    execSync(`tar -xzf "${archivePath}" -C "${binaryDir}"`, { stdio: "inherit" });
    const extractedBinary = path.join(binaryDir, `${platform}-${arch}`, "agentfs");
    if (fs.existsSync(extractedBinary)) {
      fs.renameSync(extractedBinary, binaryPath);
    }
  }

  fs.chmodSync(binaryPath, 0o755);
  fs.unlinkSync(archivePath);

  const extractedDir = path.join(binaryDir, `${platform}-${arch}`);
  if (fs.existsSync(extractedDir)) {
    fs.rmSync(extractedDir, { recursive: true, force: true });
  }

  return binaryPath;
}

async function main() {
  try {
    await downloadBinary();
    console.log("AgentFS binary installed successfully.");
  } catch (err) {
    console.error("Failed to install AgentFS binary:", err.message);
    process.exit(0);
  }
}

main();

"""Download and cache the StrataFS native binary."""

import hashlib
import os
import platform
import stat
import sys
import tarfile
import urllib.request
import zipfile
from pathlib import Path

DEFAULT_VERSION = "0.2.0"
GITHUB_RELEASES_URL = "https://github.com/neul-labs/stratafs/releases/download"


def get_platform():
    """Return the platform name used in release assets."""
    system = platform.system().lower()
    if system == "darwin":
        return "darwin"
    if system == "windows":
        return "windows"
    return "linux"


def get_arch():
    """Return the architecture name used in release assets."""
    machine = platform.machine().lower()
    if machine in ("amd64", "x86_64"):
        return "amd64"
    if machine in ("arm64", "aarch64"):
        return "arm64"
    return machine


def get_binary_dir():
    """Return the directory where the binary should be cached."""
    cache_dir = os.path.expanduser("~/.stratafs/bin")
    os.makedirs(cache_dir, exist_ok=True)
    return cache_dir


def get_binary_path():
    """Return the path to the cached StrataFS binary."""
    binary_name = "stratafs.exe" if get_platform() == "windows" else "stratafs"
    return os.path.join(get_binary_dir(), binary_name)


def download_binary(version=None):
    """Download the StrataFS binary for the current platform."""
    if version is None:
        version = os.environ.get("STRATAFS_VERSION", DEFAULT_VERSION)

    plat = get_platform()
    arch = get_arch()
    binary_dir = get_binary_dir()
    binary_path = get_binary_path()

    archive_name = f"stratafs-v{version}-{plat}-{arch}"
    if plat == "windows":
        archive_ext = "zip"
    else:
        archive_ext = "tar.gz"

    url = f"{GITHUB_RELEASES_URL}/v{version}/{archive_name}.{archive_ext}"
    archive_path = os.path.join(binary_dir, f"{archive_name}.{archive_ext}")

    if os.path.exists(binary_path):
        return binary_path

    print(f"Downloading StrataFS {version} for {plat}/{arch}...")
    print(f"URL: {url}")

    try:
        urllib.request.urlretrieve(url, archive_path)
    except Exception as exc:
        raise RuntimeError(
            f"Failed to download StrataFS binary from {url}: {exc}"
        ) from exc

    if archive_ext == "zip":
        with zipfile.ZipFile(archive_path, "r") as zf:
            for member in zf.namelist():
                if member.endswith("stratafs.exe"):
                    zf.extract(member, binary_dir)
                    extracted = os.path.join(binary_dir, member)
                    os.rename(extracted, binary_path)
                    break
    else:
        with tarfile.open(archive_path, "r:gz") as tf:
            for member in tf.getmembers():
                if member.name.endswith("stratafs"):
                    tf.extract(member, binary_dir)
                    extracted = os.path.join(binary_dir, member.name)
                    os.rename(extracted, binary_path)
                    break

    os.chmod(binary_path, os.stat(binary_path).st_mode | stat.S_IEXEC)

    # Cleanup archive
    os.remove(archive_path)

    return binary_path


def ensure_binary():
    """Ensure the binary is available, downloading if necessary."""
    binary_path = get_binary_path()
    if not os.path.exists(binary_path):
        binary_path = download_binary()
    return binary_path

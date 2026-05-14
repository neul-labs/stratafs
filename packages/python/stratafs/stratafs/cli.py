"""CLI entry point that delegates to the native StrataFS binary."""

import os
import sys

from stratafs.download import ensure_binary


def main():
    binary = ensure_binary()
    os.execv(binary, [binary] + sys.argv[1:])


if __name__ == "__main__":
    main()

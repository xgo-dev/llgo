#!/usr/bin/env python3

"""Configure the runner's Ubuntu mirror lists and bounded APT downloads."""

import pathlib
import sys


def configure(apt_dir, architecture):
    if architecture in ("amd64", "i386"):
        # Ubuntu's ordinary HTTP transport retains APT's signed Release and
        # package-hash verification. Keep HTTPS as an independent transport
        # fallback when the TLS endpoint is unavailable on a runner.
        # mirror+file priorities are supported by all Ubuntu CI APT versions.
        archive = ("http://archive.ubuntu.com/ubuntu/\tpriority:1\n"
                   "https://archive.ubuntu.com/ubuntu/\tpriority:2")
        security = ("http://security.ubuntu.com/ubuntu/\tpriority:1\n"
                    "https://security.ubuntu.com/ubuntu/\tpriority:2")
    else:
        archive = security = "https://ports.ubuntu.com/ubuntu-ports/"

    # The runner's mirror+file source selected a malformed https:/ URL and
    # then stalled on another mirror for the entire 45-minute Wasm job.
    # Preserve source suites, components and signing keys; replace only the
    # two Ubuntu mirror lists when the runner already uses them.
    for name, url in (("apt-mirrors.txt", archive),
                      ("apt-mirrors-security.txt", security)):
        path = apt_dir / name
        if path.exists():
            path.write_text(url + "\n")

    (apt_dir / "apt.conf.d" / "80-llgo-downloads").write_text('''\
Acquire::ForceIPv4 "true";
Acquire::Retries "3";
Acquire::http::Timeout "30";
Acquire::https::Timeout "30";
APT::Update::Error-Mode "any";
''')


if __name__ == "__main__":
    configure(pathlib.Path(sys.argv[1]), sys.argv[2])

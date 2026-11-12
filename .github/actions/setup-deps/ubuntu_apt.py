#!/usr/bin/env python3

"""Configure the runner's Ubuntu mirror lists and bounded APT downloads."""

import pathlib
import sys


def configure(apt_dir, architecture):
    if architecture in ("amd64", "i386"):
        azure = "https://azure.archive.ubuntu.com/ubuntu/"
        archive = (azure, "https://archive.ubuntu.com/ubuntu/")
        security = (azure, "https://security.ubuntu.com/ubuntu/")
    else:
        archive = security = ("https://ports.ubuntu.com/ubuntu-ports/",)

    # The runner's mirror+file source selected a malformed https:/ URL and
    # then stalled on another mirror for the entire 45-minute Wasm job.
    # Preserve source suites, components and signing keys; replace only the
    # two Ubuntu mirror lists when the runner already uses them. APT tries
    # another mirror on download failures; explicit priorities avoid randomly
    # selecting an unavailable archive endpoint before the Azure mirror.
    for name, mirrors in (("apt-mirrors.txt", archive),
                          ("apt-mirrors-security.txt", security)):
        path = apt_dir / name
        if path.exists():
            path.write_text("".join(
                f"{url}\tpriority:{priority}\n"
                for priority, url in enumerate(mirrors, 1)))

    (apt_dir / "apt.conf.d" / "80-llgo-downloads").write_text('''\
Acquire::ForceIPv4 "true";
Acquire::Retries "3";
Acquire::http::Timeout "30";
Acquire::https::Timeout "30";
APT::Update::Error-Mode "any";
''')


if __name__ == "__main__":
    configure(pathlib.Path(sys.argv[1]), sys.argv[2])

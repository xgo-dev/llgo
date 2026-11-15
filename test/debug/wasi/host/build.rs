// Copyright 2026 The XGo Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0.

fn main() {
    let compiler = std::env::var_os("RUSTC").expect("Cargo compiler identity");
    let output = std::process::Command::new(compiler)
        .arg("--version")
        .output()
        .expect("query Rust compiler");
    assert!(output.status.success());
    println!(
        "cargo:rustc-env=LLGO_HOST_RUST_VERSION={}",
        String::from_utf8(output.stdout).unwrap().trim()
    );
    println!("cargo:rerun-if-changed=Cargo.lock");
    let lock = std::fs::read_to_string("Cargo.lock").expect("read locked engine identity");
    let mut package = "";
    let mut versions = 0;
    for line in lock.lines() {
        if line == "[[package]]" {
            package = "";
        } else if line == "name = \"wasmtime\"" {
            package = "LLGO_WASMTIME_VERSION";
        } else if line == "name = \"wasi-common\"" {
            package = "LLGO_WASI_VERSION";
        } else if !package.is_empty() {
            if let Some(version) = line
                .strip_prefix("version = \"")
                .and_then(|s| s.strip_suffix('"'))
            {
                println!("cargo:rustc-env={package}={version}");
                versions += 1;
                if versions == 2 {
                    return;
                }
            }
        }
    }
    panic!("missing engine/WASI identity in Cargo.lock");
}

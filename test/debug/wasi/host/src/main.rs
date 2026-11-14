// Copyright 2026 The XGo Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0.

use std::sync::atomic::{AtomicI32, Ordering};
use std::sync::{Arc, OnceLock};
use wasi_common::{WasiCtx, sync::WasiCtxBuilder};
use wasmtime::{
    Caller, Config, Engine, InstancePre, Linker, Module, OptLevel, SharedMemory, Store,
};
use wasmtime::{Error, Result};

fn trap(error: anyhow::Error) -> wasmtime::Error {
    if let Some(exit) = error.downcast_ref::<wasi_common::I32Exit>() {
        std::process::exit(exit.0);
    }
    wasmtime::Error::msg(format!("{error:?}"))
}

macro_rules! wasi {
    ($linker:ident, $name:ident, ($($arg:ident: $ty:ty),*), $result:ty) => {
        $linker.func_wrap("wasi_snapshot_preview1", stringify!($name),
            |mut caller: Caller<'_, State>, $($arg: $ty),*| -> wasmtime::Result<$result> {
                let memory = caller.get_export("memory").and_then(|m| m.into_shared_memory())
                    .ok_or_else(|| wasmtime::Error::msg("missing shared memory export"))?;
                let state = caller.data_mut();
                let mut memory = wiggle_legacy::GuestMemory::Shared(memory.data());
                let future = wasi_common::snapshots::preview_1::wasi_snapshot_preview1::$name(
                    &mut state.wasi, &mut memory, $($arg),*);
                wiggle_legacy::run_in_dummy_executor(future).map_err(trap)?.map(<$result>::from).map_err(trap)
            })?;
    }
}

include!("preview1.rs");

#[derive(Clone)]
struct State {
    wasi: WasiCtx,
    shared: Arc<Shared>,
}
struct Shared {
    pre: OnceLock<InstancePre<State>>,
    tid: AtomicI32,
}

fn main() -> Result<()> {
    let mut arguments = std::env::args().skip(1);
    let path = arguments
        .next()
        .ok_or_else(|| Error::msg("usage: llgo-wasi-debug module.wasm [arguments...]"))?;
    if path == "--version" {
        println!(
            "llgo-wasi-debug {} (Wasmtime {}; WASI {}; {})",
            env!("CARGO_PKG_VERSION"),
            env!("LLGO_WASMTIME_VERSION"),
            env!("LLGO_WASI_VERSION"),
            env!("LLGO_HOST_RUST_VERSION")
        );
        return Ok(());
    }
    let argv = std::iter::once(path.clone())
        .chain(arguments)
        .collect::<Vec<_>>();
    let mut config = Config::new();
    config
        .debug_info(true)
        .cranelift_opt_level(OptLevel::None)
        .wasm_threads(true)
        .shared_memory(true)
        .wasm_exceptions(true);
    let engine = Engine::new(&config)?;
    let module = Module::from_file(&engine, path)?;
    let shared = Arc::new(Shared {
        pre: OnceLock::new(),
        tid: AtomicI32::new(0),
    });
    let mut wasi = WasiCtxBuilder::new();
    wasi.inherit_stdio()
        .args(&argv)
        .map_err(|error| Error::msg(format!("{error:?}")))?;
    // Match LLGo's normal WASI emulator: the working directory and /tmp are
    // shared process resources, not independently reopened in each worker.
    for (host, guest) in [
        (std::env::current_dir()?, "/"),
        (std::env::temp_dir(), "/tmp"),
    ] {
        let directory =
            wasi_common::sync::Dir::open_ambient_dir(host, wasi_common::sync::ambient_authority())?;
        wasi.preopened_dir(directory, guest)
            .map_err(|error| Error::msg(format!("{error:?}")))?;
    }
    let mut store = Store::new(
        &engine,
        State {
            wasi: wasi.build(),
            shared: shared.clone(),
        },
    );
    let mut linker = Linker::<State>::new(&engine);
    add_preview1(&mut linker)?;
    let ty = module
        .imports()
        .find_map(|import| {
            if import.module() == "env" && import.name() == "memory" {
                import.ty().memory().cloned()
            } else {
                None
            }
        })
        .ok_or_else(|| Error::msg("expected shared env.memory"))?;
    linker.define(&store, "env", "memory", SharedMemory::new(&engine, ty)?)?;
    linker.func_wrap(
        "wasi",
        "thread-spawn",
        |caller: Caller<'_, State>, argument: i32| -> i32 {
            let state = caller.data().clone();
            let shared = state.shared.clone();
            let Some(id) = shared
                .tid
                .fetch_update(Ordering::Relaxed, Ordering::Relaxed, |id| {
                    if id < 0x1ffffffe { Some(id + 1) } else { None }
                })
                .ok()
                .map(|id| id + 1)
            else {
                return -1;
            };
            let pre = shared.pre.get().unwrap().clone();
            match std::thread::Builder::new().spawn(move || {
                let mut store = Store::new(pre.module().engine(), state);
                let result = (|| -> Result<()> {
                    let instance = pre.instantiate(&mut store)?;
                    instance
                        .get_typed_func::<(i32, i32), ()>(&mut store, "wasi_thread_start")?
                        .call(&mut store, (id, argument))?;
                    Ok(())
                })();
                if let Err(error) = result {
                    eprintln!("worker: {error:?}");
                    std::process::exit(2);
                }
            }) {
                Ok(_) => id,
                Err(_) => -1,
            }
        },
    )?;
    let pre = linker.instantiate_pre(&module)?;
    assert!(shared.pre.set(pre.clone()).is_ok());
    let instance = pre.instantiate(&mut store)?;
    instance
        .get_typed_func::<(), ()>(&mut store, "_start")?
        .call(&mut store, ())?;
    Ok(())
}

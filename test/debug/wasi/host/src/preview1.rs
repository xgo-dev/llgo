// Copyright 2026 The XGo Authors. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// WASI snapshot-preview1 core signatures, checked against wasi-common's ABI.
fn add_preview1(linker: &mut Linker<State>) -> wasmtime::Result<()> {
    wasi!(linker, args_get, (arg0: i32, arg1: i32), i32);
    wasi!(linker, args_sizes_get, (arg0: i32, arg1: i32), i32);
    wasi!(linker, environ_get, (arg0: i32, arg1: i32), i32);
    wasi!(linker, environ_sizes_get, (arg0: i32, arg1: i32), i32);
    wasi!(linker, clock_res_get, (arg0: i32, arg1: i32), i32);
    wasi!(linker, clock_time_get, (arg0: i32, arg1: i64, arg2: i32), i32);
    wasi!(linker, fd_advise, (arg0: i32, arg1: i64, arg2: i64, arg3: i32), i32);
    wasi!(linker, fd_allocate, (arg0: i32, arg1: i64, arg2: i64), i32);
    wasi!(linker, fd_close, (arg0: i32), i32);
    wasi!(linker, fd_datasync, (arg0: i32), i32);
    wasi!(linker, fd_fdstat_get, (arg0: i32, arg1: i32), i32);
    wasi!(linker, fd_fdstat_set_flags, (arg0: i32, arg1: i32), i32);
    wasi!(linker, fd_fdstat_set_rights, (arg0: i32, arg1: i64, arg2: i64), i32);
    wasi!(linker, fd_filestat_get, (arg0: i32, arg1: i32), i32);
    wasi!(linker, fd_filestat_set_size, (arg0: i32, arg1: i64), i32);
    wasi!(linker, fd_filestat_set_times, (arg0: i32, arg1: i64, arg2: i64, arg3: i32), i32);
    wasi!(linker, fd_pread, (arg0: i32, arg1: i32, arg2: i32, arg3: i64, arg4: i32), i32);
    wasi!(linker, fd_prestat_get, (arg0: i32, arg1: i32), i32);
    wasi!(linker, fd_prestat_dir_name, (arg0: i32, arg1: i32, arg2: i32), i32);
    wasi!(linker, fd_pwrite, (arg0: i32, arg1: i32, arg2: i32, arg3: i64, arg4: i32), i32);
    wasi!(linker, fd_read, (arg0: i32, arg1: i32, arg2: i32, arg3: i32), i32);
    wasi!(linker, fd_readdir, (arg0: i32, arg1: i32, arg2: i32, arg3: i64, arg4: i32), i32);
    wasi!(linker, fd_renumber, (arg0: i32, arg1: i32), i32);
    wasi!(linker, fd_seek, (arg0: i32, arg1: i64, arg2: i32, arg3: i32), i32);
    wasi!(linker, fd_sync, (arg0: i32), i32);
    wasi!(linker, fd_tell, (arg0: i32, arg1: i32), i32);
    wasi!(linker, fd_write, (arg0: i32, arg1: i32, arg2: i32, arg3: i32), i32);
    wasi!(linker, path_create_directory, (arg0: i32, arg1: i32, arg2: i32), i32);
    wasi!(linker, path_filestat_get, (arg0: i32, arg1: i32, arg2: i32, arg3: i32, arg4: i32), i32);
    wasi!(linker, path_filestat_set_times, (arg0: i32, arg1: i32, arg2: i32, arg3: i32, arg4: i64, arg5: i64, arg6: i32), i32);
    wasi!(linker, path_link, (arg0: i32, arg1: i32, arg2: i32, arg3: i32, arg4: i32, arg5: i32, arg6: i32), i32);
    wasi!(linker, path_open, (arg0: i32, arg1: i32, arg2: i32, arg3: i32, arg4: i32, arg5: i64, arg6: i64, arg7: i32, arg8: i32), i32);
    wasi!(linker, path_readlink, (arg0: i32, arg1: i32, arg2: i32, arg3: i32, arg4: i32, arg5: i32), i32);
    wasi!(linker, path_remove_directory, (arg0: i32, arg1: i32, arg2: i32), i32);
    wasi!(linker, path_rename, (arg0: i32, arg1: i32, arg2: i32, arg3: i32, arg4: i32, arg5: i32), i32);
    wasi!(linker, path_symlink, (arg0: i32, arg1: i32, arg2: i32, arg3: i32, arg4: i32), i32);
    wasi!(linker, path_unlink_file, (arg0: i32, arg1: i32, arg2: i32), i32);
    wasi!(linker, poll_oneoff, (arg0: i32, arg1: i32, arg2: i32, arg3: i32), i32);
    wasi!(linker, proc_exit, (arg0: i32), ());
    wasi!(linker, proc_raise, (arg0: i32), i32);
    wasi!(linker, sched_yield, (), i32);
    wasi!(linker, random_get, (arg0: i32, arg1: i32), i32);
    wasi!(linker, sock_accept, (arg0: i32, arg1: i32, arg2: i32), i32);
    wasi!(linker, sock_recv, (arg0: i32, arg1: i32, arg2: i32, arg3: i32, arg4: i32, arg5: i32), i32);
    wasi!(linker, sock_send, (arg0: i32, arg1: i32, arg2: i32, arg3: i32, arg4: i32), i32);
    wasi!(linker, sock_shutdown, (arg0: i32, arg1: i32), i32);
    Ok(())
}

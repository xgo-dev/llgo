"""Own a bounded native command tree without a launch-before-assignment race."""

import ctypes
import json
import os
import subprocess
import sys


class BasicLimits(ctypes.Structure):
    _fields_ = [("process_time", ctypes.c_int64), ("job_time", ctypes.c_int64),
                ("flags", ctypes.c_uint32), ("minimum", ctypes.c_size_t),
                ("maximum", ctypes.c_size_t), ("active", ctypes.c_uint32),
                ("affinity", ctypes.c_size_t), ("priority", ctypes.c_uint32),
                ("scheduling", ctypes.c_uint32)]


class ExtendedLimits(ctypes.Structure):
    _fields_ = [("basic", BasicLimits), ("io", ctypes.c_uint64 * 6),
                ("process_memory", ctypes.c_size_t), ("job_memory", ctypes.c_size_t),
                ("peak_process", ctypes.c_size_t), ("peak_job", ctypes.c_size_t)]


# Only this bootstrap runs before assignment. It cannot start the command or
# its descendants until the parent signals the inherited gate. Standard input
# is unchanged; the gate is a separate, explicitly inherited Windows handle.
BOOTSTRAP = """
import ctypes,json,subprocess,sys
k = ctypes.WinDLL('kernel32', use_last_error=True)
k.WaitForSingleObject.argtypes = [ctypes.c_void_p, ctypes.c_uint32]
k.CloseHandle.argtypes = [ctypes.c_void_p]
gate = int(sys.argv[1])
status = k.WaitForSingleObject(gate, 0xffffffff)
k.CloseHandle(gate)
if status != 0:
    raise SystemExit(125)
raise SystemExit(subprocess.call(json.loads(sys.argv[2])))
"""


class WindowsJob:
    def __init__(self, command):
        self.kernel = ctypes.WinDLL("kernel32", use_last_error=True)
        handle = ctypes.c_void_p
        signatures = {
            "CreateJobObjectW": ([handle, ctypes.c_wchar_p], handle),
            "CreateEventW": ([handle, ctypes.c_int, ctypes.c_int, ctypes.c_wchar_p], handle),
            "SetInformationJobObject": ([handle, ctypes.c_int, handle, ctypes.c_uint32], ctypes.c_int),
            "AssignProcessToJobObject": ([handle, handle], ctypes.c_int),
            "SetEvent": ([handle], ctypes.c_int),
            "TerminateJobObject": ([handle, ctypes.c_uint32], ctypes.c_int),
            "CloseHandle": ([handle], ctypes.c_int),
        }
        for name, (arguments, result) in signatures.items():
            function = getattr(self.kernel, name)
            function.argtypes, function.restype = arguments, result
        self.job = self.kernel.CreateJobObjectW(None, None)
        if not self.job:
            raise ctypes.WinError(ctypes.get_last_error())
        self.process = None
        gate = None
        try:
            limits = ExtendedLimits()
            # Closing this parent's last job handle also kills descendants
            # if Git Bash or the CI runner terminates the Python supervisor.
            limits.basic.flags = 0x2000  # JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE.
            self.check(self.kernel.SetInformationJobObject(
                self.job, 9, ctypes.byref(limits), ctypes.sizeof(limits)))
            gate = self.kernel.CreateEventW(None, True, False, None)
            self.check(gate)
            os.set_handle_inheritable(gate, True)
            startup = subprocess.STARTUPINFO()
            startup.lpAttributeList = {"handle_list": [gate]}
            self.process = subprocess.Popen(
                [sys.executable, "-c", BOOTSTRAP, str(gate), json.dumps(command)],
                stdout=subprocess.PIPE, stderr=subprocess.STDOUT, bufsize=0,
                startupinfo=startup, close_fds=True)
            self.check(self.kernel.AssignProcessToJobObject(self.job, int(self.process._handle)))
            self.check(self.kernel.SetEvent(gate))
            self.check(self.kernel.CloseHandle(gate))
            gate = None
        except BaseException as original:
            errors = []
            operations = []
            if self.process is not None:
                operations += [self.process.kill, lambda: self.process.wait(timeout=2),
                               self.process.stdout.close]
            operations.append(self.close)
            if gate:
                operations.append(lambda: self.check(self.kernel.CloseHandle(gate)))
            for operation in operations:
                try:
                    operation()
                except Exception as error:
                    errors.append(str(error))
            if errors:
                raise OSError(f"{original}; cleanup failures: {'; '.join(errors)}") from original
            raise

    @staticmethod
    def check(success):
        if not success:
            raise ctypes.WinError(ctypes.get_last_error())

    def wait(self, timeout):
        return self.process.wait(timeout=timeout)

    def terminate(self):
        self.check(self.kernel.TerminateJobObject(self.job, 124))

    def close(self):
        if self.job:
            self.check(self.kernel.CloseHandle(self.job))
            self.job = None

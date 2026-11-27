import { spawn } from "node:child_process";

export function spawnBrowser(command, args) {
  // Each invocation owns a separate process group. A Chrome launcher can exit
  // before its renderer processes, which may still hold the stderr pipe open.
  const processGroup = process.platform !== "win32";
  const child = spawn(command, args, {
    detached: processGroup,
    stdio: ["ignore", "ignore", "pipe"],
  });
  let stopped = false;
  return {
    child,
    stop() {
      if (stopped) return;
      stopped = true;
      try {
        if (child.pid !== undefined) {
          if (processGroup) {
            try {
              process.kill(-child.pid, "SIGKILL");
            } catch (error) {
              if (error.code !== "ESRCH") throw error;
            }
          } else {
            child.kill("SIGKILL");
          }
        }
      } finally {
        // Do not wait for a descendant to close inherited descriptors or for
        // the launcher to emit another event after it has already exited.
        child.stderr.destroy();
        child.unref();
      }
    },
  };
}

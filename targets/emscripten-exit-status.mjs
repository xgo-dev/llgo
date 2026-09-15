const nodeProcess = globalThis.process;

export async function runEmscriptenModule(factory, options) {
	let exitStatus = 0;

	const recordExit = status => {
		// A fatal exit may be followed by a normal exit while Asyncify unwinds.
		// Preserve the first nonzero status as the process result.
		if (status !== 0) {
			exitStatus = status;
		}
		nodeProcess.exitCode = exitStatus;
	};
	const consumeExitStatus = error => {
		if (error?.name !== "ExitStatus" || !Number.isInteger(error.status)) {
			return false;
		}
		recordExit(error.status);
		return true;
	};
	const uninstall = () => {
		nodeProcess.off("uncaughtException", handleUncaughtException);
		nodeProcess.off("unhandledRejection", handleUnhandledRejection);
	};
	const handleUncaughtException = error => {
		if (!consumeExitStatus(error)) {
			uninstall();
			throw error;
		}
	};
	const handleUnhandledRejection = reason => {
		if (!consumeExitStatus(reason)) {
			uninstall();
			throw reason;
		}
	};

	// A short program rejects the module factory directly. With Asyncify, the
	// same ExitStatus can arrive after that factory has already resolved, either
	// as an uncaught exception or an unhandled rejection. Keep these handlers
	// installed through process shutdown so all three paths share one contract.
	nodeProcess.on("uncaughtException", handleUncaughtException);
	nodeProcess.on("unhandledRejection", handleUnhandledRejection);

	try {
		await factory({ ...options, onExit: recordExit });
	} catch (error) {
		if (!consumeExitStatus(error)) {
			uninstall();
			throw error;
		}
	}
	nodeProcess.exitCode = exitStatus;
}

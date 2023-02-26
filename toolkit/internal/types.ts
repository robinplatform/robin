export interface Robin {
	isDaemonProcess: boolean;
	isLambdaProcess: boolean;

	startupHandlers: (() => Promise<void>)[];
}

function createRobin(): Robin {
	return {
		isDaemonProcess: process.env.ROBIN_PROCESS_TYPE === 'daemon',
		isLambdaProcess: process.env.ROBIN_PROCESS_TYPE === 'lambda',

		startupHandlers: [],
	};
}

function getRobin() {
	const proc = process as unknown as { Robin: Robin };
	proc.Robin = proc.Robin || createRobin();
	return proc.Robin;
}

export const Robin = getRobin();

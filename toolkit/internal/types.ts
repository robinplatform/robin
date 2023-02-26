import * as os from 'os';

export interface Robin {
	isDaemonProcess: boolean;
	isLambdaProcess: boolean;

	daemonTarget: string;

	startupHandlers: (() => Promise<void>)[];

	getAppRpcMethods(): Record<
		string,
		Record<string, AppRpcMethod<unknown, unknown>>
	>;
}

export interface AppRpcMethod<Input, Output> {
	(input: Input): Promise<Output>;
}

function createRobin(): Robin {
	const daemonTarget = process.env.ROBIN_DAEMON_TARGET;
	if (!daemonTarget) {
		throw new Error('ROBIN_DAEMON_TARGET must be set');
	}

	const robin: Robin = {
		isDaemonProcess: process.env.ROBIN_PROCESS_TYPE === 'daemon',
		isLambdaProcess: process.env.ROBIN_PROCESS_TYPE === 'lambda',

		daemonTarget,

		startupHandlers: [],

		getAppRpcMethods: () => {
			const { serverRpcMethods } = require(Robin.daemonTarget);
			if (typeof serverRpcMethods !== 'object' || serverRpcMethods === null) {
				throw new Error(
					`Invalid serverRpcMethods loaded from from ${Robin.daemonTarget}`,
				);
			}
			return serverRpcMethods;
		},
	};
	console.log('Starting daemon runner', {
		argv: process.argv,
		daemonTarget,
		platform: os.platform(),
		arch: os.arch(),
	});
	return robin;
}

function getRobin() {
	const proc = process as unknown as { Robin: Robin };
	proc.Robin = proc.Robin || createRobin();
	return proc.Robin;
}

export const Robin = getRobin();

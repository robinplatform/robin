import * as http from 'http';

import { Robin } from './types';

if (!Robin.isDaemonProcess) {
	throw new Error('This file should only be run in a daemon process');
}

const serverRpcMethods = Robin.getAppRpcMethods();

let lastRequest = Date.now();

async function handleRequest(
	req: http.IncomingMessage,
	res: http.ServerResponse,
) {
	if (req.url !== '/api/RunAppMethod' && req.url !== '/api/health') {
		res.writeHead(404);
		res.end();
		return;
	}

	lastRequest = Date.now();

	if (req.url === '/api/health') {
		res.writeHead(200, { 'Content-Type': 'application/json' });
		res.end(`{"ok":true}`);
		return;
	}

	try {
		const { serverFile, methodName, data } = await new Promise<{
			serverFile: string;
			methodName: string;
			data: unknown;
		}>((resolve, reject) => {
			let body = '';
			req.on('data', (chunk) => (body += chunk));
			req.on('end', () => {
				try {
					const data = JSON.parse(body);

					if (
						!data ||
						typeof data !== 'object' ||
						typeof data.serverFile !== 'string' ||
						typeof data.methodName !== 'string' ||
						!data.data
					) {
						throw new Error('Invalid request body');
					}

					resolve(data);
				} catch (err) {
					reject(err);
				}
			});
			req.on('error', reject);
		});

		const fileMethods = serverRpcMethods[serverFile];
		if (!fileMethods) {
			throw new Error(`No methods found for file ${serverFile}`);
		}

		const method = fileMethods[methodName];
		if (!method) {
			throw new Error(`No method found for ${serverFile}.${methodName}`);
		}

		const result = await method(data);
		res.end(JSON.stringify(result));
	} catch (err) {
		if (!res.headersSent) {
			res.writeHead(500, { 'Content-Type': 'application/json' });
		}
		res.end(
			JSON.stringify({
				type: 'error',
				error: String((err as Error)?.stack ?? err),
			}),
		);
	}
}

async function main() {
	try {
		// Run startup handlers
		try {
			for (const handler of Robin.startupHandlers) {
				await handler();
			}
		} catch (err) {
			throw Object.assign(
				new Error('Robin app daemon crashed during startup handlers'),
				{ cause: err },
			);
		}

		// Start the server
		const server = http.createServer(handleRequest);
		await new Promise<void>((resolve, reject) => {
			server.on('error', reject);
			server.listen(process.env.PORT, () => resolve());
		});
		console.log(`Started listening on :${process.env.PORT}`);

		// Start a timer to automatically exit after 5 seconds of inactivity
		setInterval(() => {
			if (Date.now() - lastRequest > 5000) {
				console.log('No requests in 5 seconds, exiting');
				process.exit(0);
			}
		}, 1000);
	} catch (err) {
		console.error(err);
		process.exit(1);
	}
}

main();

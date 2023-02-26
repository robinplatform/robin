// TODO: Slim down this file (we don't need express for two routes).

import bodyParser from 'body-parser';
import express from 'express';
import morgan from 'morgan';
import { z } from 'zod';

import { Robin } from './types';

if (!Robin.isDaemonProcess) {
	throw new Error('This file should only be run in a daemon process');
}

interface RpcMethod<Input, Output> {
	(input: Input): Promise<Output>;
}

const { serverRpcMethods } = require(process.env.ROBIN_DAEMON_TARGET!) as {
	serverRpcMethods: Record<string, Record<string, RpcMethod<unknown, unknown>>>;
};

const app = express();

let lastRequest = Date.now();
app.use((_, __, next) => {
	lastRequest = Date.now();
	next();
});

app.get('/api/health', (_, res) => {
	res.json({ ok: true });
});
app.use(morgan('dev'));
app.use(bodyParser.json());
app.post('/api/RunAppMethod', async (req, res) => {
	try {
		const { serverFile, methodName, data } = z
			.object({
				serverFile: z.string(),
				methodName: z.string(),
				data: z.unknown(),
			})
			.parse(req.body);

		const fileMethods = serverRpcMethods[serverFile];
		if (!fileMethods) {
			throw new Error(`No methods found for file ${serverFile}`);
		}

		const method = fileMethods[methodName];
		if (!method) {
			throw new Error(`No method found for ${serverFile}.${methodName}`);
		}

		const result = await method(data);
		res.json({ type: 'success', result });
	} catch (err) {
		res.statusCode = 500;
		res.json({ type: 'error', error: String((err as Error)?.stack ?? err) });
	}
});

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
		await new Promise<void>((resolve, reject) => {
			app.on('error', reject);
			app.listen(process.env.PORT, () => resolve());
		});
		console.log(`Started listening on :${process.env.PORT}`);

		// Start a timer to automatically exit after 5 minutes of inactivity
		setInterval(() => {
			if (Date.now() - lastRequest > 1 * 60 * 1000) {
				console.log('No requests in 5 minutes, exiting');
				process.exit(0);
			}
		}, 1 * 60 * 1000);
	} catch (err) {
		console.error(err);
		process.exit(1);
	}
}

main();

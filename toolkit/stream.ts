let _ws: Promise<WebSocket> | null = null;
let inFlight: Record<string, Stream> = {};

async function getWs(): Promise<WebSocket> {
	if (_ws !== null) return _ws;

	let resolve: (ws: WebSocket) => void = () => {};

	_ws = new Promise<WebSocket>((res) => (resolve = res));

	const ws = new WebSocket('ws://localhost:9010/api/websocket');
	ws.onclose = (evt) => {
		console.log('close', evt.code);
	};

	ws.onmessage = (evt) => {
		const data = JSON.parse(evt.data);
		const stream = inFlight[data.id];
		if (!stream) {
			return;
		}

		switch (data.kind) {
			case 'error':
				stream.onerror(data.error);
				break;

			case 'methodDone':
				stream.onclose();
				break;

			default:
				stream.onmessage(data);
		}
	};

	ws.onerror = (evt) => {
		console.error('Robin WS error', evt);
	};

	ws.onopen = () => {
		resolve(ws);
	};

	return _ws;
}

// This is a low-level primitive that can be used to implement higher-level
// streaming requests.
export class Stream {
	onmessage: (a: unknown) => void = () => {};
	onerror: (a: unknown) => void = () => {};

	private started = false;
	private closed = false;

	private closeHandler: () => void = () => {
		this.closed = true;
	};

	constructor(readonly method: string, readonly id: string) {}

	set onclose(f: () => void) {
		this.closeHandler = () => {
			f();
			this.closed = true;
		};
	}

	get onclose(): () => void {
		return this.closeHandler;
	}

	async start(data: unknown) {
		if (this.started) {
			throw new Error('Already started');
		}

		this.started = true;
		inFlight[this.id] = this;

		const ws = await getWs();
		ws.send(
			JSON.stringify({
				kind: 'call',
				method: this.method,
				id: this.id,
				data,
			}),
		);
	}

	async close() {
		if (!this.started) {
			throw new Error(`hasn't started yet`);
		}

		if (this.closed) {
			return;
		}

		this.closed = true;

		const ws = await getWs();
		ws.send(
			JSON.stringify({
				kind: 'cancel',
				method: this.method,
				id: this.id,
			}),
		);
	}
}

let _ws: Promise<WebSocket> | null = null;
let inFlight: Record<string, Stream> = {};

const STARTED = 1;
const SERVER_STARTED = 2;
const CLOSED = 4;

const stage = Symbol('Robin stream stage');

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

			case 'methodStarted':
				stream[stage] = SERVER_STARTED;
				console.log('methodStarted:', stream.id);
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
	[stage] = 0;

	constructor(readonly method: string, readonly id: string) {}

	private closeHandler: () => void = () => {
		console.log(`Stream(${this.method}, ${this.id}) closed`);
		this[stage] = CLOSED;
	};

	onmessage: (a: unknown) => void = (a) => {
		console.log(`Stream(${this.method}, ${this.id}) message:`, a);
	};
	onerror: (a: unknown) => void = (a) => {
		console.error(`Stream(${this.method}, ${this.id}) error:`, a);
	};

	set onclose(f: () => void) {
		this.closeHandler = () => {
			f();
			this[stage] = CLOSED;
		};
	}

	get onclose(): () => void {
		return this.closeHandler;
	}

	async start(data: unknown) {
		if (this[stage] >= STARTED) {
			throw new Error('Already started');
		}

		this[stage] = STARTED;
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
		if (this[stage] < STARTED) {
			throw new Error(`hasn't started yet`);
		}

		if (this[stage] >= CLOSED) {
			return;
		}

		this[stage] = CLOSED;

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

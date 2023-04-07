let _ws: Promise<WebSocket> | null = null;
const inFlight: Map<string, Stream> = new Map();

async function getWs(): Promise<WebSocket> {
	if (_ws !== null) {
		return _ws;
	}

	console.log('making new websocket');
	let resolveWs: (ws: WebSocket) => void = () => {};

	const newWs = new Promise<WebSocket>((res) => (resolveWs = res));
	_ws = newWs;

	const ws = new WebSocket(
		`ws://${window.location.hostname}:9010/api/websocket`,
	);
	ws.onclose = (evt) => {
		console.log('close', evt.code);

		// We only want to over-write if this specific websocket is the one that
		// is currently active; otherwise, we can just die peacefully.
		//
		// It's possible that this will always be true, but at this very moment
		// I think the extra condition to ensure no shenanigans is fine.
		if (_ws === newWs) {
			// Gain "exclusive write access" in a sense; nobody else will write to the
			// `_ws` variable, and any time someone tries to get the websocket, they are
			// instead directed to a promise that doesn't resolve until we say so.
			// This must happen first, so that user-code called in stream.onclose
			// doesn't get a handle to the old websocket instead of the new one.
			let resolveWsCloseRetry: (ws: WebSocket) => void = () => {};
			_ws = new Promise<WebSocket>((res) => (resolveWsCloseRetry = res));

			// We spread before operating on the values, so that when calling the onclose handler,
			// we don't modify the `inFlight` map while still iterating over it.
			[...inFlight.values()].forEach((stream) =>
				stream.onclose('WebsocketClosed'),
			);

			// Wait 2 seconds before retrying; then, before retrying, make sure to
			// set `_ws` to null so that getWs() actually overwrites the variable again
			// and we don't infinite loop.
			setTimeout(() => {
				_ws = null;
				getWs().then(resolveWsCloseRetry);
			}, 2000);
		}
	};

	ws.onmessage = (evt) => {
		const data = JSON.parse(evt.data);
		const stream = inFlight.get(data.id);
		if (!stream) {
			return;
		}

		try {
			switch (data.kind) {
				case 'error':
					stream.onerror(data.data, 'MethodError');
					break;

				case 'methodDone':
					stream.onclose('MethodDone');
					break;

				default:
					stream.onmessage(data);
			}
		} catch (e) {
			stream.onerror(e, 'HandlerException');
		}
	};

	ws.onerror = (evt) => {
		console.error('Robin WS error', evt);
	};

	ws.onopen = () => {
		console.log('websocket opened');
		resolveWs(ws);
	};

	return _ws;
}

type StreamCloseCause = 'CalledCloseMethod' | 'WebsocketClosed' | 'MethodDone';
type StreamErrorSource = 'HandlerException' | 'MethodError';

// This is a low-level primitive that can be used to implement higher-level
// streaming requests.
export class Stream {
	private started = false;
	private closed = false;
	readonly id: string;

	constructor(readonly method: string) {
		this.id = `${this.method}-${Math.random()}`;
		this.onclose = () => {};
	}

	private closeHandler: (cause: StreamCloseCause) => void = () => {};

	onmessage: (a: unknown) => void = (a) => {
		console.log(`Stream(${this.method}, ${this.id}) message:`, a);
	};
	onerror: (a: unknown, source: StreamErrorSource) => void = (a, source) => {
		console.error(`Stream(${this.method}, ${this.id}) ${source}:`, a);
	};

	set onclose(f: (cause: StreamCloseCause) => void) {
		this.closeHandler = (cause: StreamCloseCause) => {
			this.closed = true;
			inFlight.delete(this.id);

			f(cause);
		};
	}

	get onclose(): (cause: StreamCloseCause) => void {
		return this.closeHandler;
	}

	async start(data: unknown) {
		if (this.started) {
			throw new Error('Already started');
		}

		this.started = true;
		inFlight.set(this.id, this);

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

	// Don't like this name, but I'd rather be explicit about what this function actually does.
	newStreamWithSameHandlers(): Stream {
		const newStream = new Stream(this.method);
		newStream.onmessage = this.onmessage;
		newStream.onerror = this.onerror;
		newStream.closeHandler = this.closeHandler;

		return newStream;
	}

	async close() {
		if (!this.started) {
			throw new Error(`hasn't started yet`);
		}

		if (this.closed) {
			return;
		}

		this.onclose('CalledCloseMethod');

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

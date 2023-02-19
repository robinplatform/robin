let _ws: Promise<WebSocket> | null = null;
let inFlight: Record<string, Stream> = {};

export class Stream {
	onmessage: (a: any) => void = () => {};
	onerror: (a: any) => void = () => {};

	private started = false;

	constructor(
		private readonly ws: WebSocket,
		readonly method: string,
		readonly id: string,
	) {}

	start(data: any) {
		if (this.started) {
			throw new Error('Already started');
		}

		this.started = true;

		this.ws.send(
			JSON.stringify({
				kind: 'call',
				method: this.method,
				id: this.id,
				data,
			}),
		);
	}

	static async callStreamRpc(method: string, id: string): Promise<Stream> {
		const ws = await this.getWs();

		const stream = new Stream(ws, method, id);
		inFlight[id] = stream;

		return stream;
	}

	static async getWs(): Promise<WebSocket> {
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

			if (data.kind === 'error') {
				stream.onerror(data.error);
			} else {
				stream.onmessage(data);
			}
		};

		ws.onerror = (evt) => {
			console.log('error', evt);
		};

		ws.onopen = () => {
			console.log('opened');
			resolve(ws);
		};

		return _ws;
	}
}

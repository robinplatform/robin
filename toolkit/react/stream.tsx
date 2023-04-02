import React from 'react';
import { z } from 'zod';
import { Stream } from '../stream';
import stableStringify from 'json-stable-stringify';

const PubsubData = z.object({
	// Kind is not sent from the server; instead, its used by the frontend to tell the stream
	// to set the state properly
	kind: z.union([z.literal('state'), z.literal('user')]).optional(),

	messageId: z.number(),
	data: z.unknown(),
});

// Subscribe to a topic and track the messages received in relation
// to state.
export function useTopicQuery<State, Output>({
	topicId,
	fetchState,
	reducer,
	resultType,
	skip,
}: {
	resultType: z.Schema<Output>;
	topicId?: { category: string; key: string };
	fetchState: () => Promise<{ state: State; counter: number }>;
	reducer: (s: State, o: Output) => State;
	skip?: boolean;
}) {
	/* TODO: This code is quite bad. I'm not convinced it is correct if e.g. the socket
	   closes, or if something crashes, and I have not tested it in that situation.
	*/
	type StreamState =
		| { kind: 'empty'; seenMessages: z.infer<typeof PubsubData>[] }
		| {
				kind: 'state';
				seenMessages?: undefined;
				counter: number;
				state: State;
		  };

	const { state, dispatch: rawDispatch } = useStreamMethod({
		methodName: 'SubscribeTopic',
		resultType: PubsubData,
		data: { id: topicId },
		skip: skip || !topicId,
		initialState: { kind: 'empty', seenMessages: [] },
		onConnection: () => {
			// This needs to run AFTER the connection completes so that there's no messages
			// dropped after the state is fetched. If we did this the other way around,
			// we might connect slowly and fetch state fast, and drop a message by the time
			// we get the state, causing a mismatch between what we *should* have and what we do have.
			//
			// On an unrelated note, I don't have much confidence that the underlying stream is properly implemented,
			// and so in theory it is enough to just run this after the connection finishes but I can't
			// say for certain that this code works correctly in all edge cases.
			fetchState().then((data) =>
				rawDispatch({
					kind: 'state',
					messageId: data.counter,
					data: data.state,
				}),
			);
		},
		reducer: (prev: StreamState, packet): StreamState => {
			if (packet.kind === 'state') {
				// The >= seems to be required. Sometimes we'll get two fetches in a row,
				// and the first will cause the saved messages to be applied, only for the
				// second fetch to arrive. If you use > instead of >=, the second fetch
				// might overwrite the first, even though the state counter is literally
				// the same.
				if (prev.kind === 'state' && prev.counter >= packet.messageId) {
					return prev;
				}

				const seenMessages = (prev.seenMessages ?? []).filter(
					(msg) => msg.messageId > packet.messageId,
				);
				const state = seenMessages
					.flatMap((msg) => {
						const res = resultType.safeParse(msg.data);
						if (res.success) {
							return [res.data];
						}

						return [];
					})
					.reduce((prev, data) => reducer(prev, data), packet.data as State);

				const maxMessageId = seenMessages.reduce(
					(max, msg) => Math.max(msg.messageId, max),
					packet.messageId,
				);

				return {
					kind: 'state',
					counter: maxMessageId,
					state,
				};
			}

			if (prev.kind === 'empty') {
				return {
					kind: 'empty',
					seenMessages: [...prev.seenMessages, packet],
				};
			} else {
				const res = resultType.safeParse(packet.data);
				if (res.success) {
					return {
						kind: 'state',
						counter: packet.messageId,
						state: reducer(prev.state, res.data),
					};
				}

				console.warn('Failed to parse data:', JSON.stringify(packet));
				return prev;
			}
		},
	});

	if (state.kind === 'empty') {
		return { state: undefined };
	}

	return { state: state.state };
}

export function useStreamMethod<State, Output>({
	methodName,
	data: initialData,
	initialState,
	reducer,
	resultType,
	onConnection,
	skip,
}: {
	resultType: z.Schema<Output>;
	methodName: string;
	data: object;
	initialState: State;
	reducer: (s: State, o: Output) => State;
	onConnection?: () => void;
	skip?: boolean;
}) {
	// enforce reducer stability
	const reducerRef = React.useRef(reducer);
	reducerRef.current = reducer;

	// enforce callback stability
	const onConnRef = React.useRef(onConnection);
	onConnRef.current = onConnection;

	const cb = React.useCallback(
		(s: State, o: Output) => reducerRef.current(s, o),
		[],
	);

	const [state, dispatch] = React.useReducer(cb, initialState);

	React.useEffect(() => {
		if (skip) {
			return;
		}

		const id = `${methodName}-${Math.random()}`;

		const stream = new Stream(methodName, id);

		stream.onmessage = (message) => {
			const { kind, data } = message as { kind: string; data: string };
			if (kind !== 'methodOutput') {
				return;
			}

			const res = resultType.safeParse(data);
			if (!res.success) {
				// TODO: handle the error
				stream.onerror(res.error);
				return;
			}

			dispatch(res.data);
		};

		console.log('initialData', JSON.stringify(initialData));
		stream.start(initialData).then(() => onConnRef.current?.());

		return () => {
			stream.close();
		};

		// initialData JSON is here, so that when you change the information
		// in the parameters, you get a new stream.
	}, [skip, methodName, stableStringify(initialData)]);

	return { state, dispatch };
}

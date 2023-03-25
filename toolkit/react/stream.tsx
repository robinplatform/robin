import React from 'react';
import { z } from 'zod';
import { Stream } from '../stream';
import stableStringify from 'json-stable-stringify';

export function useStreamMethod<State, Output>({
	methodName,
	data: initialData,
	initialState,
	reducer,
	resultType,
	skip,
}: {
	resultType: z.Schema<Output>;
	methodName: string;
	data: object;
	initialState: State;
	reducer: (s: State, o: Output) => State;
	skip?: boolean;
}) {
	// enforce reducer stability
	const reducerRef = React.useRef(reducer);
	reducerRef.current = reducer;

	const cb = React.useCallback(
		(s: State, o: Output) => reducerRef.current(s, o),
		[],
	);

	const [state, dispatch] = React.useReducer(cb, initialState);

	// Skip is an extra dependency so that the stream gets reset when
	// the stream is turned off/on again.
	// initialData JSON is also here, so that when you change the information
	// in the parameters, you get a new stream.
	const id = React.useMemo(
		() => `${methodName}-${Math.random()}`,
		[methodName, stableStringify(initialData), skip],
	);

	React.useEffect(() => {
		if (skip) {
			return;
		}

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

		stream.start(initialData);

		return () => {
			stream.close();
		};
	}, [skip, methodName, id, stableStringify(initialData)]);

	return { state, dispatch };
}

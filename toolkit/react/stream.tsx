import {
	QueryClientProvider,
	QueryClient,
	useQuery,
	UseQueryOptions,
} from '@tanstack/react-query';
import React from 'react';
import { z } from 'zod';
import { Stream } from '../stream';

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

	const id = React.useMemo(
		() => `${methodName}-${Math.random()}`,
		[methodName, JSON.stringify(initialData), skip],
	);

	const stream = React.useMemo(
		() => new Stream(methodName, id),
		[methodName, id],
	);

	// TODO: use stable stringify for the dep check
	React.useEffect(() => {
		if (skip) {
			return;
		}

		stream.onmessage = (message) => {
			const { kind, data } = message as { kind: string; data: string };
			if (kind !== 'methodOutput') {
				return;
			}

			const res = resultType.safeParse(JSON.parse(data));
			if (!res.success) {
				// TODO: handle the error
				console.log('Robin stream parse error', res.error);
				return;
			}

			dispatch(res.data);
		};

		stream.start(initialData);

		return () => {
			stream.close();
		};
	}, [skip, stream, JSON.stringify(initialData)]);

	return { state, dispatch };
}

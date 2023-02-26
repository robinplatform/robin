import {
	useQuery,
	UseQueryOptions,
	useMutation,
	UseMutationOptions,
} from '@tanstack/react-query';
import { z } from 'zod';
import axios from 'axios';
import React from 'react';

export function useRpcQuery<R>(
	options: Omit<
		UseQueryOptions<R, unknown, R, [string, unknown]>,
		'queryKey'
	> & {
		method: string;
		data: unknown;
		result: z.Schema<R>;
	},
) {
	return useQuery<R, unknown, R, [string, unknown]>({
		onError: (err) => console.log(`Error in method ${options.method}`, err),

		...options,
		queryKey: [options.method, options.data],
		queryFn: async ({ queryKey }) => {
			const rpcMethodName = queryKey[0] as string;
			const rpcMethodData = queryKey[1] as unknown;

			const { data } = await axios.post(
				`/api/internal/rpc/${rpcMethodName}`,
				rpcMethodData,
			);
			return options.result.parse(data);
		},
	});
}

export function useRpcMutation<Output>(
	options: Omit<
		UseMutationOptions<Output, unknown, unknown, [string]>,
		'mutationKey'
	> & {
		method: string;
		result: z.Schema<Output>;
	},
) {
	const rpcMethodName = options.method;

	const resultShapeRef = React.useRef(options.result);
	resultShapeRef.current = options.result;

	const mutationFn = React.useCallback(
		async (input: unknown) => {
			const { data } = await axios.post(
				`/api/internal/rpc/${rpcMethodName}`,
				input,
			);
			return resultShapeRef.current.parse(data);
		},
		[rpcMethodName],
	);

	return useMutation<Output, unknown, unknown, [string]>({
		onError: (err) => console.log(`Error in method ${options.method}`, err),

		...options,
		mutationKey: [options.method],
		mutationFn: mutationFn,
	});
}

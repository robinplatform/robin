import {
	useQuery,
	UseQueryOptions,
	useMutation,
	UseMutationOptions,
} from '@tanstack/react-query';
import { z } from 'zod';
import axios from 'axios';
import React from 'react';

export function useRpcQuery<R>({
	method,
	data,
	result,
	pathPrefix = '/api/internal/rpc',
	...options
}: Omit<UseQueryOptions<R, unknown, R, [string, unknown]>, 'queryKey'> & {
	pathPrefix?: string;
	method: string;
	data: unknown;
	result: z.Schema<R>;
}) {
	return useQuery<R, unknown, R, [string, unknown]>({
		onError: (err) => console.log(`Error in method ${method}`, err),

		...options,
		queryKey: [method, data],
		queryFn: async ({ queryKey }) => {
			const rpcMethodName = queryKey[0] as string;
			const rpcMethodData = queryKey[1] as unknown;

			const { data } = await axios.post(
				`${pathPrefix}/${rpcMethodName}`,
				rpcMethodData,
			);
			return result.parse(data);
		},
	});
}

export function useRpcMutation<Output>({
	method,
	result,
	pathPrefix = '/api/internal/rpc',

	...options
}: Omit<
	UseMutationOptions<Output, unknown, unknown, [string]>,
	'mutationKey'
> & {
	method: string;
	result: z.Schema<Output>;
	pathPrefix?: string;
}) {
	const resultShapeRef = React.useRef(result);
	resultShapeRef.current = result;

	const mutationFn = React.useCallback(
		async (input: unknown) => {
			const { data } = await axios.post(`${pathPrefix}/${method}`, input);
			return resultShapeRef.current.parse(data);
		},
		[pathPrefix, method],
	);

	return useMutation<Output, unknown, unknown, [string]>({
		onError: (err) => console.log(`Error in method ${method}`, err),

		...options,
		mutationKey: [method],
		mutationFn: mutationFn,
	});
}

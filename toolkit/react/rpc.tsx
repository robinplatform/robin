import {
	QueryClientProvider,
	QueryClient,
	useQuery,
	UseQueryOptions,
	UseMutationOptions,
	useMutation,
} from '@tanstack/react-query';
import React from 'react';
import { z } from 'zod';

import { runAppMethod } from '../';

const globalQueryClient = new QueryClient({
	defaultOptions: {
		queries: {
			refetchOnWindowFocus: true,
			retry: 1,
		},
	},
});

export function ReactQueryProvider({
	children,
}: {
	children: React.ReactNode;
}) {
	return (
		<QueryClientProvider client={globalQueryClient}>
			{children}
		</QueryClientProvider>
	);
}

interface RpcMethod<Input, Output> {
	getQueryKey: (data: Input) => string[];
	(data: Input): Promise<Output>;
}

export function useRpcQuery<Input, Output>(
	method: (data: Input) => Promise<Output>,
	data: Input,
	overrides?: Omit<
		UseQueryOptions<Output, unknown, Output, unknown[]>,
		'queryKey' | 'queryFn'
	>,
) {
	const rpcMethod = method as RpcMethod<Input, Output>;
	if (typeof rpcMethod.getQueryKey !== 'function') {
		throw new Error(
			`Invalid RPC method passed to useRpcQuery. Make sure you are importing from a '.server.ts' file.`,
		);
	}

	return useQuery({
		queryKey: rpcMethod.getQueryKey(data) as unknown[],
		queryFn: () => method(data),
		...overrides,
	});
}

export function useRpcMutation<Input, Output>(
	method: (data: Input) => Promise<Output>,
	overrides?: Omit<
		UseMutationOptions<Output, unknown, Input, unknown[]>,
		'mutationKey' | 'mutationFn'
	>,
) {
	const rpcMethod = method as RpcMethod<Input, Output>;
	if (typeof rpcMethod.getQueryKey !== 'function') {
		throw new Error(
			`Invalid RPC method passed to useRpcQuery. Make sure you are importing from a '.server.ts' file.`,
		);
	}

	return useMutation({
		mutationKey: rpcMethod.getQueryKey(
			undefined as unknown as Input,
		) as unknown[],
		mutationFn: method,
		...overrides,
	});
}

function useRemoteAppMethod<Output>(
	methodName: string,
	data: object,
	{
		resultType,
		...overrides
	}: Omit<
		UseQueryOptions<Output, unknown, Output, unknown[]>,
		'queryKey' | 'queryFn'
	> & { resultType: z.Schema<Output> },
) {
	return useRpcQuery(runAppMethod, { methodName, data, resultType }, overrides);
}

export function createReactRpcBridge<
	Shape extends Record<
		string,
		{ input: z.Schema<unknown>; output: z.Schema<unknown> }
	>,
>(bridge: Shape) {
	return {
		useRpcQuery: (
			methodName: string & keyof Shape,
			data: z.infer<Shape[keyof Shape]['input']>,
			overrides?: Omit<
				UseQueryOptions<
					z.infer<Shape[keyof Shape]['output']>,
					unknown,
					z.infer<Shape[keyof Shape]['output']>,
					unknown[]
				>,
				'queryKey' | 'queryFn'
			>,
		) => {
			return useRemoteAppMethod(methodName, data as unknown as object, {
				resultType: bridge[methodName].output,
				...overrides,
			});
		},
	};
}

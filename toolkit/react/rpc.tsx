import {
	QueryClientProvider,
	QueryClient,
	useQuery,
	UseQueryOptions,
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

export function useRemoteAppMethod<Output>(
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

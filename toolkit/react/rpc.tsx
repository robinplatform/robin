import {
	QueryClientProvider,
	QueryClient,
	useQuery,
	UseQueryOptions,
} from '@tanstack/react-query';
import React from 'react';

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
	queryKeyPrefix: string[];
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
	if (!Array.isArray(rpcMethod.queryKeyPrefix)) {
		throw new Error(
			`Invalid RPC method passed to useRpcQuery. Make sure you are importing from a '.server.ts' file.`,
		);
	}

	return useQuery({
		queryKey: [...rpcMethod.queryKeyPrefix, data] as unknown[],
		queryFn: () => method(data),
		...overrides,
	});
}

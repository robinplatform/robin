import {
	QueryClientProvider,
	QueryClient,
	useQuery,
} from '@tanstack/react-query';
import React from 'react';

const globalQueryClient = new QueryClient({
	defaultOptions: {
		queries: {
			refetchOnWindowFocus: true,
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
	serverFile: string;
	methodName: string;
	(data: Input): Promise<Output>;
}

export function useRpcQuery<Input, Output>(
	method: (data: Input) => Promise<Output>,
	data: Input,
) {
	const rpcMethod = method as RpcMethod<Input, Output>;
	if (
		typeof rpcMethod.serverFile !== 'string' ||
		typeof rpcMethod.methodName !== 'string'
	) {
		throw new Error(
			`Invalid RPC method passed to useRpcQuery. Make sure you are importing from a '.server.ts' file.`,
		);
	}

	return useQuery({
		queryKey: [rpcMethod.serverFile, rpcMethod.methodName, data],
		queryFn: () => method(data),
	});
}

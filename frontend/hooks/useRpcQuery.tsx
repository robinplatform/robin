import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { z } from 'zod';
import axios from 'axios';

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

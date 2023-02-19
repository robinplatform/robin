import type { AppProps } from 'next/app';
import React from 'react';
import { QueryClient, QueryClientProvider, useQuery } from 'react-query';
import { ReactQueryDevtools } from 'react-query/devtools';
import { getConfig } from '@robin/toolkit';

import './globals.scss';
import '@robin/toolkit/dist/global-styles.css';

const queryClient = new QueryClient({
	defaultOptions: {
		queries: {},
	},
});

function QueryDevtools() {
	const { data: config } = useQuery({
		queryKey: ['getConfig'],
		queryFn: getConfig,
	});
	if (!config?.showReactQueryDebugger) {
		return null;
	}
	return <ReactQueryDevtools />;
}

export default function Robin({ Component, pageProps }: AppProps) {
	return (
		<QueryClientProvider client={queryClient}>
			<QueryDevtools />

			<main className={"robin-text-white"} style={{ width: '100%', height: '100%' }}>
				<Component {...pageProps} />
			</main>
		</QueryClientProvider>
	);
}

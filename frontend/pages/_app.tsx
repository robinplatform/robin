import type { AppProps } from 'next/app';
import React from 'react';
import { QueryClient, QueryClientProvider, useQuery } from 'react-query';
import { ReactQueryDevtools } from 'react-query/devtools';
import { getConfig } from '@robinplatform/toolkit';

import 'tippy.js/dist/tippy.css';
import './globals.scss';
import '@robinplatform/toolkit/dist/global-styles.css';
import { Sidebar } from '../components/Sidebar';

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
	return <ReactQueryDevtools position="top-right" />;
}

export default function Robin({ Component, pageProps }: AppProps) {
	return (
		<QueryClientProvider client={queryClient}>
			<QueryDevtools />
			<Sidebar />

			<main className={'robin-text-white full'}>
				<Component {...pageProps} />
			</main>
		</QueryClientProvider>
	);
}

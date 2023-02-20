import type { AppProps } from 'next/app';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from 'react-hot-toast';

import 'normalize.css/normalize.css';
import 'tippy.js/dist/tippy.css';
import './globals.scss';
import '@robinplatform/toolkit/styles.css';
import { Sidebar } from '../components/Sidebar';

const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			refetchOnWindowFocus: true,
		},
	},
});

export default function Robin({ Component, pageProps }: AppProps) {
	return (
		<QueryClientProvider client={queryClient}>
			<Sidebar />
			<Toaster />

			<main className={'robin-text-white full'}>
				<Component {...pageProps} />
			</main>
		</QueryClientProvider>
	);
}

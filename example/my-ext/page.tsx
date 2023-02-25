import { renderApp } from '@robinplatform/toolkit/react';
import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { QueryClientProvider, QueryClient } from '@tanstack/react-query';
import { getSelfSource } from './page.server';

function Page() {
	const { data, error } = useRpcQuery(getSelfSource, {
		filename: './package.json',
	});

	return (
		<pre>
			<code>{error ? String(error) : data ? String(data) : 'Loading ...'}</code>
		</pre>
	);
}

const queryClient = new QueryClient();

renderApp(
	<QueryClientProvider client={queryClient}>
		<Page />
	</QueryClientProvider>,
);

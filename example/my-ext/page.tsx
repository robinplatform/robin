import { renderApp } from '@robinplatform/toolkit/react';
import React from 'react';
import {
	QueryClientProvider,
	QueryClient,
	useQuery,
} from '@tanstack/react-query';
import { getSelfSource } from './page.server';

function Page() {
	const { data, error } = useQuery({
		queryKey: ['my-query'],
		queryFn: getSelfSource,
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

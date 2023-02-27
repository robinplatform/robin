import React from 'react';
import ReactDOM from 'react-dom';

import { ReactQueryProvider } from './rpc';

export function renderApp(
	children: React.ReactNode,
	{ reactQueryEnabled }: { reactQueryEnabled?: boolean } = {},
) {
	const withReactQuery = reactQueryEnabled
		? (children: React.ReactNode) => (
				<ReactQueryProvider>{children}</ReactQueryProvider>
		  )
		: (children: React.ReactNode) => children;

	ReactDOM.render(
		<React.StrictMode>{withReactQuery(children)}</React.StrictMode>,
		document.getElementById('root'),
	);
}

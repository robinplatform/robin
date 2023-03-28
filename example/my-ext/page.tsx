import { getAppSettings } from '@robinplatform/toolkit';
import { renderApp } from '@robinplatform/toolkit/react';
import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { getSelfSource } from './page.server';
import '@robinplatform/toolkit/styles.css';
import './ext.scss';
import { z } from 'zod';
import { Pogo } from './pogo';

function Main() {
	const { data: settings, error: errFetchingSettings } = useRpcQuery(
		getAppSettings,
		z.object({ filename: z.string().optional() }),
	);
	const { data, error: errFetchingFile } = useRpcQuery(
		getSelfSource,
		{
			filename: String(settings?.filename ?? './package.json'),
		},
		{
			enabled: !!settings,
		},
	);

	const error = errFetchingSettings || errFetchingFile;

	return (
		<pre
			style={{
				margin: '1rem',
				padding: '1rem',
				background: '#e3e3e3',
				borderRadius: 'var(--robin-border-radius)',
			}}
		>
			<code>{error ? String(error) : data ? data.data : 'Loading ...'}</code>
		</pre>
	);
}

const AppPages = {
	Main,
	Pogo,
} as const;

function App() {
	const [page, setPage] = React.useState<keyof typeof AppPages>('Pogo');
	const Component = AppPages[page];

	return (
		<div>
			<div>
				{Object.keys(AppPages).map((key) => (
					<button
						key={key}
						onClick={() => setPage(key as keyof typeof AppPages)}
					>
						{key}
					</button>
				))}
			</div>

			<Component />
		</div>
	);
}

renderApp(<App />);

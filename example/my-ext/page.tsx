import { getAppSettings } from '@robinplatform/toolkit';
import { renderApp } from '@robinplatform/toolkit/react';
import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import '@robinplatform/toolkit/styles.css';
import './ext.scss';
import { z } from 'zod';

function Page() {
	const { data: settings, error: errFetchingSettings } = useRpcQuery(
		getAppSettings,
		z.object({ filename: z.string().optional() }),
	);

	const error = errFetchingSettings;

	return (
		<div>
			<div>
				LOCATION: {String(window.location.href)}
				<a href="./blahblah">My Link</a>
			</div>

			<pre
				style={{
					margin: '1rem',
					padding: '1rem',
					background: '#e3e3e3',
					borderRadius: 'var(--robin-border-radius)',
				}}
			>
				<code>
					{error
						? JSON.stringify(error)
						: settings
						? JSON.stringify(settings, undefined, 2)
						: 'Loading ...'}
				</code>
			</pre>
		</div>
	);
}

renderApp(<Page />);

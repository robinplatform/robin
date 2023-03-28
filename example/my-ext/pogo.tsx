import { getAppSettings } from '@robinplatform/toolkit';
import { renderApp } from '@robinplatform/toolkit/react';
import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { getCommunityDays, getSelfSource } from './page.server';
import '@robinplatform/toolkit/styles.css';
import './ext.scss';

// I'm not handling errors in this file, because... oh well. Whatever. Meh.

export function Pogo() {
	const { data: commDays } = useRpcQuery(getCommunityDays, {});

	return (
		<pre
			style={{
				margin: '1rem',
				padding: '1rem',
				background: '#e3e3e3',
				borderRadius: 'var(--robin-border-radius)',
			}}
		>
			<div>{JSON.stringify(commDays, undefined, 2)}</div>
		</pre>
	);
}

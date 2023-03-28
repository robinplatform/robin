import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { getUpcomingCommDays } from './page.server';
import '@robinplatform/toolkit/styles.css';
import './ext.scss';

// I'm not handling errors in this file, because... oh well. Whatever. Meh.

export function Pogo() {
	const { data: events } = useRpcQuery(getUpcomingCommDays, {});

	const upcomingEvents = React.useMemo(() => {
		const now = new Date();
		return events?.filter((day) => {
			return new Date(day.end) > now;
		});
	}, [events]);

	return (
		<pre
			style={{
				margin: '1rem',
				padding: '1rem',
				background: '#e3e3e3',
				borderRadius: 'var(--robin-border-radius)',
			}}
		>
			<div>{JSON.stringify(upcomingEvents, undefined, 2)}</div>
		</pre>
	);
}

import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { getUpcomingCommDays } from './pogo.server';
import '@robinplatform/toolkit/styles.css';

// I'm not handling errors in this file, because... oh well. Whatever. Meh.

// "PoGo" is an abbreviation for Pokemon Go which is well-known in the
// PoGo community.
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

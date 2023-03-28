import { useRpcQuery, useRpcMutation } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { getUpcomingCommDays, refreshDex } from './pogo.server';
import '@robinplatform/toolkit/styles.css';
import { fetchDb } from './db.server';

// I'm not handling errors in this file, because... oh well. Whatever. Meh.

// "PoGo" is an abbreviation for Pokemon Go which is well-known in the
// PoGo community.
export function Pogo() {
	const { data: db, refetch: refetchDb } = useRpcQuery(fetchDb, {});
	const { data: events } = useRpcQuery(getUpcomingCommDays, {});
	const { mutate, data: pokedex } = useRpcMutation(refreshDex, {
		onSuccess: () => refetchDb(),
	});

	const upcomingEvents = React.useMemo(() => {
		const now = new Date();
		return events?.filter((day) => {
			return new Date(day.end) > now;
		});
	}, [events]);

	return (
		<div>
			<button onClick={() => mutate({})}>Refresh Pokedex</button>

			<pre
				style={{
					margin: '1rem',
					padding: '1rem',
					background: '#e3e3e3',
					borderRadius: 'var(--robin-border-radius)',
				}}
			>
				<div>{JSON.stringify(db, undefined, 2)}</div>
			</pre>
		</div>
	);
}

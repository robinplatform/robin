import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { ScrollWindow } from '../components/ScrollWindow';
import { SelectPage } from '../components/SelectPage';
import { getUpcomingEventsRpc, PogoEvent } from '../server/leekduck.server';

function Event({ event }: { event: PogoEvent }) {
	return (
		<div
			className={'robin-rounded robin-pad'}
			style={{ border: '1px solid black' }}
		>
			{JSON.stringify(event)}
		</div>
	);
}

export function EventPlanner() {
	const { data: upcomingEvents } = useRpcQuery(getUpcomingEventsRpc, {});

	return (
		<div className={'col full robin-rounded robin-gap robin-pad'}>
			<div className={'row robin-gap'}>
				<SelectPage />
			</div>

			<ScrollWindow
				className="full"
				style={{ backgroundColor: 'white' }}
				innerClassName="col robin-gap"
			>
				{upcomingEvents?.map((event) => (
					<Event key={event.eventID} event={event} />
				))}
			</ScrollWindow>
		</div>
	);
}

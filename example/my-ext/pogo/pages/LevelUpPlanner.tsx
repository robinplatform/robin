import { useRpcMutation, useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { ScrollWindow } from '../components/ScrollWindow';
import {
	SelectPage,
	SelectPokemon,
	usePageState,
} from '../components/PageState';
import {
	MegaEvolveEvent,
	megaLevelPlanForPokemonRpc,
} from '../server/planner.server';
import { addPlannedEventRpc } from '../server/db.server';

// Include cancel or not
// Specify locks/planned actions

// iterate forwards over lock points,
// iterate backwards in time from each lock point
// at the last lock point, iterate forwards in time

// TODO: add something to allow for checking the cost of daily level-ups
// TODO: add data that shows remaining mega energy

function DateText({ date }: { date: Date }) {
	return (
		<div
			style={{
				position: 'absolute',
				right: '2rem',
				top: '0',
				bottom: '0',
				width: '10rem',
				display: 'flex',
				flexDirection: 'column',
				justifyContent: 'center',
				textAlign: 'right',
			}}
		>
			{date.toDateString()}
		</div>
	);
}

function EventText({ event }: { event: MegaEvolveEvent }) {
	return (
		<div
			style={{
				position: 'absolute',
				left: '2rem',
				top: '0',
				bottom: '0',
				width: '12rem',
			}}
		>
			Evolve for {event.megaEnergySpent} to level up
		</div>
	);
}

function SmallDot() {
	return (
		<div
			style={{
				position: 'absolute',
				top: '0',
				bottom: '0',
				height: '1rem',
				width: '1rem',
				borderRadius: '1rem',
				backgroundColor: 'blue',
			}}
		/>
	);
}

function BigDot() {
	return (
		<div
			style={{
				position: 'absolute',
				top: '-0.5rem',
				left: '-0.5rem',
				height: '2rem',
				width: '2rem',
				borderRadius: '2rem',
				backgroundColor: 'blue',
			}}
		/>
	);
}

function DayBox({ children }: { children: React.ReactNode }) {
	return (
		<div
			style={{
				position: 'relative',
				height: '1rem',
				width: '1rem',
			}}
		>
			{children}
		</div>
	);
}

function AddEventButton({
	pokemonId,
	date,
}: {
	pokemonId: string;
	date: Date;
}) {
	const { mutate: addPlannedEvents } = useRpcMutation(addPlannedEventRpc, {});

	return (
		<button
			style={{
				position: 'absolute',
				left: '2rem',
				top: '0',
			}}
			onClick={() =>
				addPlannedEvents({ pokemonId, isoDate: date.toISOString() })
			}
		>
			Mega
		</button>
	);
}

export function LevelUpPlanner() {
	const { pokemon: selectedMonId = '' } = usePageState();

	const { data: days } = useRpcQuery(megaLevelPlanForPokemonRpc, {
		id: selectedMonId,
	});

	return (
		<div className={'col full robin-rounded robin-gap robin-pad'}>
			<div className={'row robin-gap'}>
				<SelectPage />

				<SelectPokemon />
			</div>

			<ScrollWindow
				className={'full'}
				style={{ background: 'white' }}
				innerClassName={'col robin-pad'}
				innerStyle={{
					alignItems: 'center',
					gap: '1.5rem',
				}}
			>
				{days?.map(({ date, eventsToday }) => (
					<DayBox key={date}>
						<DateText date={new Date(date)} />

						{new Date(date).toDateString() === new Date().toDateString() ? (
							<BigDot />
						) : (
							<SmallDot />
						)}

						{eventsToday.map((e, index) => (
							<EventText key={`${index}`} event={e} />
						))}

						{eventsToday.length === 0 && (
							<AddEventButton pokemonId={selectedMonId} date={new Date(date)} />
						)}
					</DayBox>
				))}
			</ScrollWindow>
		</div>
	);
}

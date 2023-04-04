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
import { addPlannedEventRpc, deletePlannedEventRpc } from '../server/db.server';

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

function EventInfo({
	pokemonId,
	date,
	event,
	refetch,
}: {
	pokemonId: string;
	date: Date;
	event?: MegaEvolveEvent;
	refetch: () => void;
}) {
	const { mutate: addPlannedEvent } = useRpcMutation(addPlannedEventRpc, {
		onSuccess: () => refetch(),
	});
	const { mutate: deletePlannedEvent } = useRpcMutation(deletePlannedEventRpc, {
		onSuccess: () => refetch(),
	});

	if (!event) {
		return (
			<div
				style={{
					position: 'absolute',
					left: '2rem',
					top: '0',
					width: '12rem',
				}}
			>
				<button
					onClick={() =>
						addPlannedEvent({ pokemonId, isoDate: date.toISOString() })
					}
				>
					Mega
				</button>
			</div>
		);
	}

	const { id, title, megaEnergyAvailable } = event;

	if (!id) {
		return (
			<div
				style={{
					position: 'absolute',
					left: '2rem',
					top: '0',
					width: '12rem',
					color: 'gray',
				}}
			>
				{title}
			</div>
		);
	}

	return (
		<div
			style={{
				position: 'absolute',
				left: '2rem',
				top: '0',
				width: '12rem',
				color: 'black',
			}}
		>
			{title}
			<button onClick={() => deletePlannedEvent({ id })}>X</button>
			Remaining: {megaEnergyAvailable}
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

export function LevelUpPlanner() {
	const { pokemon: selectedMonId = '' } = usePageState();

	const { data: days, refetch } = useRpcQuery(megaLevelPlanForPokemonRpc, {
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

						<EventInfo
							pokemonId={selectedMonId}
							date={new Date(date)}
							event={eventsToday[0]}
							refetch={refetch}
						/>
					</DayBox>
				))}
			</ScrollWindow>
		</div>
	);
}

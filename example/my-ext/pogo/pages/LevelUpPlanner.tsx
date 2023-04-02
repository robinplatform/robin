import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { useSelectOption } from '../components/EditableField';
import { ScrollWindow } from '../components/ScrollWindow';
import {
	SelectPage,
	SelectPokemon,
	usePageState,
} from '../components/PageState';
import {
	computeEvolve,
	nextMegaDeadline,
	Pokemon,
	PokemonMegaValues,
	Species,
} from '../domain-utils';
import { arrayOfN, DAY_MS } from '../math';
import { fetchDbRpc } from '../server/db.server';
import {
	MegaEvolveEvent,
	megaLevelPlanForPokemonRpc,
} from '../server/planner.server';

// Include cancel or not
// Specify locks/planned actions

// iterate forwards over lock points,
// iterate backwards in time from each lock point
// at the last lock point, iterate forwards in time

// TODO: add something to allow for checking the cost of daily level-ups
// TODO: move the calculations to the server
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

export function LevelUpPlanner() {
	const { pokemon: selectedMonId } = usePageState();

	const { data: plan } = useRpcQuery(megaLevelPlanForPokemonRpc, {
		id: selectedMonId ?? '',
	});

	const days = React.useMemo(() => {
		const now = new Date();

		if (!plan || plan.length === 0) {
			return undefined;
		}

		const timeToLastEvent =
			new Date(plan[plan.length - 1].date).getTime() - now.getTime();
		const daysToDisplay = Math.ceil(timeToLastEvent / DAY_MS) + 4;

		return arrayOfN(daysToDisplay)
			.map((i) => new Date(Date.now() + (i - 2) * DAY_MS))
			.map((date) => {
				const eventsToday = plan.filter(
					(e) => new Date(e.date).toDateString() === date.toDateString(),
				);

				return {
					date,
					eventsToday,
				};
			});
	}, [plan]);

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
					<DayBox key={`${date.toISOString()}`}>
						<DateText date={date} />

						{date.toDateString() === new Date().toDateString() ? (
							<BigDot />
						) : (
							<SmallDot />
						)}

						{eventsToday.map((e, index) => (
							<EventText key={`${index}`} event={e} />
						))}
					</DayBox>
				))}
			</ScrollWindow>
		</div>
	);
}

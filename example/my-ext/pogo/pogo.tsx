import { useRpcQuery, useRpcMutation } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { getUpcomingCommDays, refreshDexRpc } from './pogo.server';
import { ScrollWindow } from './ScrollWindow';
import '@robinplatform/toolkit/styles.css';
import { addPokemonRpc, fetchDb, Pokemon } from './db.server';
import { TypeColors } from './typings';
import { create } from 'zustand';

// I'm not handling errors in this file, because... oh well. Whatever. Meh.

const useCurrentSecond = create<{ now: Date }>((set) => {
	const now = new Date();

	function updateTime() {
		set({ now: new Date() });
		setTimeout(updateTime, 1000);
	}

	setTimeout(updateTime, 1000 - now.getMilliseconds());

	return { now };
});

// This is a component because otherwise it'd be very easy to re-render things that shouldn't need
// to re-render.
function CountdownText({
	deadline,
	doneText,
}: {
	deadline: Date;
	doneText: string;
}) {
	const { now } = useCurrentSecond();

	if (now > deadline) {
		return <>{doneText}</>;
	}

	const difference = deadline.getTime() - now.getTime();
	const seconds = difference / 1000;
	const minutes = seconds / 60;
	const hours = minutes / 60;
	const days = hours / 24;
	const weeks = days / 7;

	if (weeks > 2) {
		return <>{`${Math.floor(weeks)} weeks, ${days % 7} days`}</>;
	}

	if (days >= 1) {
		return <>{`${Math.floor(days)} days, ${hours % 24} hours`}</>;
	}

	if (hours >= 1) {
		return <>{`${Math.floor(hours)} hours, ${minutes % 60} minutes`}</>;
	}

	return <>{`${Math.floor(minutes)} minutes, ${seconds % 60} seconds`}</>;
}

function SelectPokemon({
	submit,
	buttonText,
}: {
	submit: (data: { pokemonId: number }) => unknown;
	buttonText: string;
}) {
	const { data: db } = useRpcQuery(fetchDb, {});
	const [selected, setSelected] = React.useState<number>(NaN);

	return (
		<div className={'row robin-gap'}>
			<select
				value={`${selected}`}
				onChange={(evt) => setSelected(Number.parseInt(evt.target.value))}
			>
				<option>---</option>

				{Object.entries(db?.pokedex ?? {}).map(([id, dexEntry]) => {
					return (
						<option key={id} value={dexEntry.number}>
							{dexEntry.name}
						</option>
					);
				})}
			</select>

			<button
				disabled={Number.isNaN(selected)}
				onClick={() => submit({ pokemonId: selected })}
			>
				{buttonText}
			</button>
		</div>
	);
}

function megaLevelFromCount(count: number): 0 | 1 | 2 | 3 {
	switch (true) {
		case count >= 30:
			return 3;

		case count >= 7:
			return 2;

		case count >= 1:
			return 1;

		default:
			return 0;
	}
}

function nextMegaDeadline(count: number, lastMega: Date): Date {
	const date = new Date(lastMega);
	switch (megaLevelFromCount(count)) {
		case 0:
			break;
		case 1:
			date.setDate(lastMega.getDate() + 7);
		case 2:
			date.setDate(lastMega.getDate() + 5);
		case 3:
			date.setDate(lastMega.getDate() + 3);
	}

	return date;
}

function PokemonInfo({ pokemon }: { pokemon: Pokemon }) {
	const { data: db } = useRpcQuery(fetchDb, {});
	const dexEntry = db?.pokedex[pokemon.pokemonId];
	if (!dexEntry) {
		return null;
	}

	return (
		<div
			key={pokemon.id}
			className={'robin-rounded col robin-pad'}
			style={{ backgroundColor: 'white', border: '1px solid black' }}
		>
			<div className={'row robin-gap'}>
				<h3>{dexEntry.name}</h3>

				<div className={'row'} style={{ gap: '0.5rem' }}>
					{dexEntry.megaType.map((t) => (
						<div
							key={t}
							className={'robin-rounded'}
							style={{
								padding: '0.25rem 0.5rem 0.25rem 0.5rem',
								backgroundColor: TypeColors[t.toLowerCase()],
								color: 'white',
							}}
						>
							{t}
						</div>
					))}
				</div>
			</div>

			<div>
				<p>Mega Level: {megaLevelFromCount(pokemon.megaCount)}</p>

				{!!pokemon.megaCount && (
					<p>
						Next Free Mega:{' '}
						<CountdownText
							doneText="now"
							deadline={nextMegaDeadline(
								pokemon.megaCount,
								new Date(pokemon.lastMega),
							)}
						/>
					</p>
				)}
			</div>
		</div>
	);
}

// "PoGo" is an abbreviation for Pokemon Go which is well-known in the
// PoGo community.
export function Pogo() {
	const { data: db, refetch: refetchDb } = useRpcQuery(fetchDb, {});
	const { data: events } = useRpcQuery(getUpcomingCommDays, {});
	const { mutate: refreshDex } = useRpcMutation(refreshDexRpc, {
		onSuccess: () => refetchDb(),
	});
	const { mutate: addPokemon } = useRpcMutation(addPokemonRpc, {
		onSuccess: () => refetchDb(),
	});

	const upcomingEvents = React.useMemo(() => {
		const now = new Date();
		return events?.filter((day) => {
			return new Date(day.end) > now;
		});
	}, [events]);

	return (
		<div className={'col full robin-rounded robin-gap robin-pad'}>
			<div>
				<button onClick={() => refreshDex({})}>Refresh Pokedex</button>
			</div>

			<ScrollWindow className={'full'} style={{ backgroundColor: 'white' }}>
				<pre style={{ wordBreak: 'break-all' }}>
					{JSON.stringify(db?.pokedex, undefined, 2)}
				</pre>
			</ScrollWindow>

			<ScrollWindow
				className={'full'}
				style={{ backgroundColor: 'white' }}
				innerClassName={'col robin-gap robin-pad'}
				innerStyle={{ gap: '0.5rem', paddingRight: '0.5rem' }}
			>
				<div
					className={'robin-rounded robin-pad'}
					style={{ backgroundColor: 'Gray' }}
				>
					<SelectPokemon submit={addPokemon} buttonText={'Add Pokemon'} />
				</div>

				{Object.entries(db?.pokemon ?? {}).map(([id, pokemon]) => (
					<PokemonInfo key={id} pokemon={pokemon} />
				))}
			</ScrollWindow>
		</div>
	);
}

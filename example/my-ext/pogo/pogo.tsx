import { useRpcQuery, useRpcMutation } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { getUpcomingCommDays, refreshDexRpc } from './pogo.server';
import { ScrollWindow } from './ScrollWindow';
import '@robinplatform/toolkit/styles.css';
import {
	addPokemonRpc,
	fetchDb,
	Pokemon,
	setPokemonMegaCountRpc,
} from './db.server';
import {
	megaLevelFromCount,
	MegaRequirements,
	nextMegaDeadline,
	TypeColors,
} from './domain-utils';
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
		return <>{`${Math.floor(weeks)} weeks, ${Math.floor(days % 7)} days`}</>;
	}

	if (days >= 1) {
		return <>{`${Math.floor(days)} days, ${Math.floor(hours % 24)} hours`}</>;
	}

	if (hours >= 1) {
		return (
			<>{`${Math.floor(hours)} hours, ${Math.floor(minutes % 60)} minutes`}</>
		);
	}

	return (
		<>{`${Math.floor(minutes)} minutes, ${Math.floor(seconds % 60)} seconds`}</>
	);
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

function PokemonInfo({ pokemon }: { pokemon: Pokemon }) {
	const { data: db, refetch: refreshDb } = useRpcQuery(fetchDb, {});
	const { mutate: setMegaCount, isLoading: setMegaCountLoading } =
		useRpcMutation(setPokemonMegaCountRpc, {
			onSuccess: () => refreshDb(),
		});

	const dexEntry = db?.pokedex[pokemon.pokemonId];
	if (!dexEntry) {
		return null;
	}

	const megaLevel = megaLevelFromCount(pokemon.megaCount);

	return (
		<div
			key={pokemon.id}
			className={'robin-rounded col robin-pad'}
			style={{ backgroundColor: 'white', border: '1px solid black' }}
		>
			<div className={'row'} style={{ justifyContent: 'space-between' }}>
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

				<div className={'row'}>
					<button onClick={() => {}}>Evolve!</button>
				</div>
			</div>

			<div>
				<div className={'row'}>
					<div className={'row robin-gap'} style={{ minWidth: '20rem' }}>
						<p>Mega Level: {megaLevel}</p>
						{megaLevel < 3 && (
							<p>
								Level up in:{' '}
								{MegaRequirements[megaLevel + 1] - pokemon.megaCount} evolutions
							</p>
						)}
					</div>

					<div className={'row'}>
						<button
							disabled={setMegaCountLoading}
							onClick={() =>
								setMegaCount({ id: pokemon.id, count: pokemon.megaCount + 1 })
							}
						>
							+
						</button>
						<button
							disabled={setMegaCountLoading}
							onClick={() =>
								setMegaCount({ id: pokemon.id, count: pokemon.megaCount - 1 })
							}
						>
							-
						</button>
					</div>
				</div>

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

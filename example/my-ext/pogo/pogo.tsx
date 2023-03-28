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
	MegaWaitTime,
	TypeColors,
} from './domain-utils';
import { CountdownTimer } from './CountdownTimer';

// I'm not handling errors in this file, because... oh well. Whatever. Meh.

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
			style={{
				backgroundColor: 'white',
				border: '1px solid black',
				gap: '0.5rem',
			}}
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

					{!!pokemon.megaCount && (
						<CountdownTimer
							doneText="now"
							disableEditing={setMegaCountLoading}
							setDeadline={(deadline) =>
								setMegaCount({
									id: pokemon.id,
									count: pokemon.megaCount,
									lastMega: new Date(
										deadline.getTime() - MegaWaitTime[megaLevel],
									).toISOString(),
								})
							}
							deadline={
								new Date(
									new Date(pokemon.lastMega).getTime() +
										MegaWaitTime[megaLevel],
								)
							}
						/>
					)}
				</div>

				<div className={'row'}>
					<button
						onClick={() =>
							setMegaCount({
								id: pokemon.id,
								count: pokemon.megaCount + 1,
								lastMega: new Date().toISOString(),
							})
						}
					>
						Evolve!
					</button>
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

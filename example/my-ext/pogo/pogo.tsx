import { useRpcQuery, useRpcMutation } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { refreshDexRpc } from './pogo.server';
import { ScrollWindow } from './ScrollWindow';
import '@robinplatform/toolkit/styles.css';
import {
	addPokemonRpc,
	deletePokemonRpc,
	evolvePokemonRpc,
	fetchDb,
	setNameRpc,
	setPokemonEvolveTimeRpc,
	setPokemonMegaCountRpc,
	setPokemonMegaEnergyRpc,
} from './db.server';
import {
	megaCostForSpecies,
	megaLevelFromCount,
	MegaRequirements,
	MegaWaitTime,
	Pokemon,
	Species,
	TypeColors,
	TypeTextColors,
} from './domain-utils';
import { CountdownTimer, useCurrentSecond } from './CountdownTimer';
import { EditField } from './EditableField';

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

function EvolvePokemonButton({
	dexEntry,
	pokemon,
	refreshDb,
}: {
	dexEntry: Species;
	pokemon: Pokemon;
	refreshDb: () => void;
}) {
	const { now } = useCurrentSecond();
	const megaLevel = megaLevelFromCount(pokemon.megaCount);
	const megaCost = megaCostForSpecies(
		dexEntry,
		megaLevel,
		now.getTime() - new Date(pokemon.lastMega ?? 0).getTime(),
	);
	const { mutate: megaEvolve, isLoading: megaEvolveLoading } = useRpcMutation(
		evolvePokemonRpc,
		{ onSuccess: () => refreshDb() },
	);

	return (
		<button
			disabled={megaEvolveLoading || megaCost > dexEntry.megaEnergyAvailable}
			onClick={() => megaEvolve({ id: pokemon.id })}
		>
			Evolve for {megaCost}
		</button>
	);
}

function PokemonInfo({ pokemon }: { pokemon: Pokemon }) {
	const { data: db, refetch: refreshDb } = useRpcQuery(fetchDb, {});
	const { mutate: setMegaCount, isLoading: setMegaCountLoading } =
		useRpcMutation(setPokemonMegaCountRpc, { onSuccess: () => refreshDb() });
	const { mutate: setMegaEvolveTime, isLoading: setMegaEvolveTimeLoading } =
		useRpcMutation(setPokemonEvolveTimeRpc, { onSuccess: () => refreshDb() });
	const { mutate: setEnergy, isLoading: setEneryLoading } = useRpcMutation(
		setPokemonMegaEnergyRpc,
		{ onSuccess: () => refreshDb() },
	);
	const { mutate: deletePokemon, isLoading: deletePokemonLoading } =
		useRpcMutation(deletePokemonRpc, { onSuccess: () => refreshDb() });
	const { mutate: setName, isLoading: setNameLoading } = useRpcMutation(
		setNameRpc,
		{ onSuccess: () => refreshDb() },
	);

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
			<div
				className={'row'}
				style={{ justifyContent: 'space-between', height: '3rem' }}
			>
				<div className={'row robin-gap'}>
					<EditField
						disabled={setNameLoading}
						value={pokemon.name ?? dexEntry.name}
						setValue={(value) => setName({ id: pokemon.id, name: value })}
						parseFunc={(val) => {
							if (!val.trim()) {
								return undefined;
							}

							return val;
						}}
					>
						<h3>{pokemon.name ?? dexEntry.name}</h3>
					</EditField>

					<div className={'row'} style={{ gap: '0.5rem' }}>
						{dexEntry.megaType.map((t) => (
							<div
								key={t}
								className={'robin-rounded'}
								style={{
									padding: '0.25rem 0.5rem 0.25rem 0.5rem',
									backgroundColor: TypeColors[t.toLowerCase()],
									color: TypeTextColors[t.toLowerCase()],
								}}
							>
								{t}
							</div>
						))}
					</div>

					{!!pokemon.megaCount && (
						<CountdownTimer
							doneText="now"
							disableEditing={setMegaEvolveTimeLoading}
							setDeadline={(deadline) =>
								setMegaEvolveTime({
									id: pokemon.id,
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
					<EvolvePokemonButton
						dexEntry={dexEntry}
						pokemon={pokemon}
						refreshDb={refreshDb}
					/>
				</div>
			</div>

			<div className={'row'}>
				<div className={'row robin-gap'} style={{ minWidth: '20rem' }}>
					<p>Mega Level: {megaLevel}</p>
					{megaLevel < 3 && (
						<p>
							{pokemon.megaCount} done,{' '}
							{MegaRequirements[megaLevel + 1] - pokemon.megaCount} to level{' '}
							{megaLevel + 1}
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

			<div className={'row'} style={{ gap: '0.5rem' }}>
				Mega Energy:{' '}
				<EditField
					disabled={setEneryLoading}
					value={dexEntry.megaEnergyAvailable}
					setValue={(value) =>
						setEnergy({ pokemonId: dexEntry.number, megaEnergy: value })
					}
					parseFunc={(val) => {
						const parsed = Number.parseInt(val);
						if (Number.isNaN(parsed)) {
							return undefined;
						}

						return parsed;
					}}
				>
					<p style={{ width: '10rem' }}>{dexEntry.megaEnergyAvailable}</p>
				</EditField>
			</div>

			<div className={'row'} style={{ justifyContent: 'flex-end' }}>
				<button
					disabled={deletePokemonLoading}
					onClick={() => deletePokemon({ id: pokemon.id })}
				>
					Delete
				</button>
			</div>
		</div>
	);
}

// "PoGo" is an abbreviation for Pokemon Go which is well-known in the
// PoGo community.
export function Pogo() {
	const { data: db, refetch: refetchDb } = useRpcQuery(fetchDb, {});
	const { mutate: refreshDex } = useRpcMutation(refreshDexRpc, {
		onSuccess: () => refetchDb(),
	});
	const { mutate: addPokemon } = useRpcMutation(addPokemonRpc, {
		onSuccess: () => refetchDb(),
	});

	return (
		<div className={'col full robin-rounded robin-gap robin-pad'}>
			<div>
				{Object.keys(db?.pokedex ?? {}).length === 0 && (
					<div>Pokedex is empty!</div>
				)}
				<button onClick={() => refreshDex({})}>Refresh Pokedex</button>
			</div>

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

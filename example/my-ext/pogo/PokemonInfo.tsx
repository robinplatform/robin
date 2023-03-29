import { useRpcMutation, useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import { useCurrentSecond, CountdownTimer } from './components/CountdownTimer';
import { EditField } from './components/EditableField';
import {
	Species,
	Pokemon,
	megaLevelFromCount,
	megaCostForSpecies,
	TypeColors,
	TypeTextColors,
	MegaWaitTime,
	MegaRequirements,
} from './domain-utils';
import {
	evolvePokemonRpc,
	fetchDbRpc,
	setPokemonMegaCountRpc,
	setPokemonEvolveTimeRpc,
	setPokemonMegaEnergyRpc,
	deletePokemonRpc,
	setNameRpc,
} from './server/db.server';
import React from 'react';

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
		now.getTime() - new Date(pokemon.lastMegaEnd ?? 0).getTime(),
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

function MegaIndicator({ pokemon }: { pokemon: Pokemon }) {
	const { now } = useCurrentSecond();
	const { data: db } = useRpcQuery(fetchDbRpc, {});
	const currentMega = db?.currentMega;
	if (!currentMega) {
		return null;
	}

	if (currentMega.id !== pokemon.id) {
		return null;
	}

	if (new Date(pokemon.lastMegaEnd) < now) {
		return null;
	}

	return <div style={{ fontSize: '1.5rem', fontWeight: 'bold' }}>M</div>;
}

export function PokemonInfo({ pokemon }: { pokemon: Pokemon }) {
	const { data: db, refetch: refreshDb } = useRpcQuery(fetchDbRpc, {});
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
			className={'robin-rounded robin-pad'}
			style={{
				backgroundColor: 'white',
				border: '1px solid black',
			}}
		>
			<div style={{ position: 'relative' }}>
				<div
					style={{
						position: 'absolute',
						top: '-1.75rem',
						left: '-1.75rem',
					}}
				>
					<MegaIndicator pokemon={pokemon} />
				</div>
			</div>

			<div className={'col'} style={{ gap: '0.5rem' }}>
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
							<div className={'col'} style={{ position: 'relative' }}>
								{pokemon.name && pokemon.name !== dexEntry.name ? (
									<>
										<h3>{pokemon.name}</h3>
										<p
											style={{
												opacity: '50%',
												fontSize: '0.75rem',
												position: 'absolute',
												left: '-0.5rem',
												bottom: '-0.5rem',
												zIndex: 1000,
												backgroundColor: 'lightgray',
												borderRadius: '4px',
												padding: '2px',
											}}
										>
											{dexEntry.name}
										</p>
									</>
								) : (
									<h3>{dexEntry.name}</h3>
								)}
							</div>
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
										new Date(pokemon.lastMegaEnd).getTime() +
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
		</div>
	);
}

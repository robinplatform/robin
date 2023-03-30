import { useRpcQuery, useRpcMutation } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { refreshDexRpc, searchPokemonRpc } from '../server/pogo.server';
import { ScrollWindow } from '../components/ScrollWindow';
import '@robinplatform/toolkit/styles.css';
import { addPokemonRpc, fetchDbRpc } from '../server/db.server';
import { PokemonInfo } from '../components/PokemonInfo';
import { SelectPage } from '../components/SelectPage';
import { useSelectOption } from '../components/EditableField';

// TODO: planner for upcoming events
// TODO: put POGO thingy into its own package on NPM, and debug why packages sorta dont work right now

function SelectPokemon({
	submit,
	buttonText,
}: {
	submit: (data: { pokemonId: number }) => unknown;
	buttonText: string;
}) {
	const { data: { pokedex = {} } = {} } = useRpcQuery(fetchDbRpc, {});
	const { selected, ...selectMon } = useSelectOption(pokedex);

	return (
		<div className={'row robin-gap'}>
			<select {...selectMon}>
				<option>---</option>

				{Object.entries(pokedex).map(([id, dexEntry]) => {
					return (
						<option key={id} value={id}>
							{dexEntry.name}
						</option>
					);
				})}
			</select>

			<button
				disabled={!selected}
				onClick={() => selected && submit({ pokemonId: selected.number })}
			>
				{buttonText}
			</button>
		</div>
	);
}

const Sorts = ['name', 'pokemonId', 'megaTime', 'megaLevelUp'] as const;
export function PokemonManager() {
	const [sortIndex, setSortIndex] = React.useState<number>(0);
	const sort = Sorts[sortIndex] ?? 'name';
	const { data: pokemon, refetch: refetchQuery } = useRpcQuery(
		searchPokemonRpc,
		{ sort },
	);
	const { data: db, refetch: refetchDb } = useRpcQuery(fetchDbRpc, {});
	const { mutate: refreshDex } = useRpcMutation(refreshDexRpc, {
		onSuccess: () => refetchDb(),
	});
	const { mutate: addPokemon } = useRpcMutation(addPokemonRpc, {
		onSuccess: () => {
			refetchQuery();
			refetchDb();
		},
	});

	return (
		<div className={'col full robin-rounded robin-gap robin-pad'}>
			<div className={'row robin-gap'}>
				<SelectPage />

				{db && Object.keys(db.pokedex).length === 0 && (
					<div>Pokedex is empty!</div>
				)}
				<button onClick={() => refreshDex({})}>Refresh Pokedex</button>

				<div>
					Sort by:{' '}
					<select
						value={sortIndex}
						onChange={(evt) => {
							const index = Number.parseInt(evt.target.value);
							setSortIndex(index);
						}}
					>
						{Sorts.map((sort, index) => {
							return (
								<option key={sort} value={index}>
									{sort}
								</option>
							);
						})}
					</select>
				</div>
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

				{!!db &&
					pokemon?.map((id) => {
						const pokemon = db.pokemon[id];
						if (!pokemon) {
							return null;
						}

						return <PokemonInfo key={id} pokemon={pokemon} />;
					})}
			</ScrollWindow>
		</div>
	);
}

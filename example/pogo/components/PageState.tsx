import { create } from 'zustand';
import React from 'react';
import { useRpcQuery } from '@robinplatform/toolkit/react/rpc';
import { fetchDbRpc } from '../server/db.server';

const DefaultPage = 'levelup' as const;

// I'm not handling errors in this file, because... oh well. Whatever. Meh.
const PageTypes = ['pokemon', 'planner', 'tables', 'levelup'] as const;
type PageType = typeof PageTypes[number];
export const usePageState = create<{
	pokemon: string | undefined;
	page: PageType;
	setPage: (a: PageType) => void;
	setPokemon: (p: string | undefined) => void;
}>((set, get) => {
	return {
		pokemon: undefined,
		page: DefaultPage,
		setPage: (a) => set({ page: a }),
		setPokemon: (pokemon) => set({ pokemon }),
	};
});

export function SelectPage() {
	const { page, setPage } = usePageState();

	return (
		<div className={'col'}>
			<p>Page:</p>
			<select
				value={page}
				onChange={(evt) => setPage(evt.target.value as PageType)}
			>
				{PageTypes.map((page) => (
					<option key={page} value={page}>
						{page}
					</option>
				))}
			</select>
		</div>
	);
}

export function SelectPokemon() {
	const { data: db } = useRpcQuery(fetchDbRpc, {});
	const { pokemon: selectedPokemon, setPokemon } = usePageState();
	const pokemon = React.useMemo(
		() => Object.values(db?.pokemon ?? {}),
		[db?.pokemon],
	);

	return (
		<div className={'col'}>
			<p>Pokemon:</p>
			<select
				value={selectedPokemon}
				onChange={(evt) =>
					evt.target.value
						? setPokemon(evt.target.value)
						: setPokemon(undefined)
				}
			>
				<option value={''}>Select pokemon...</option>

				{pokemon.map((mon) => (
					<option key={mon.id} value={mon.id}>
						{mon.name && mon.name !== db?.pokedex?.[mon.pokedexId]?.name
							? `${mon.name} (${db?.pokedex?.[mon.pokedexId]?.name})`
							: db?.pokedex?.[mon.pokedexId]?.name}
					</option>
				))}
			</select>
		</div>
	);
}

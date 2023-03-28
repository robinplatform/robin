import { useRpcQuery, useRpcMutation } from '@robinplatform/toolkit/react/rpc';
import React from 'react';
import { getUpcomingCommDays, refreshDexRpc } from './pogo.server';
import { ScrollWindow } from './ScrollWindow';
import '@robinplatform/toolkit/styles.css';
import { addPokemonRpc, fetchDb } from './db.server';

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
		<div style={{ display: 'flex' }}>
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
				innerClassName={'robin-gap robin-pad'}
			>
				<div className={'robin-rounded'} style={{ backgroundColor: 'Gray' }}>
					<SelectPokemon submit={addPokemon} buttonText={'Add Pokemon'} />
				</div>

				{!db
					? []
					: Object.entries(db.pokemon).map(([id, pokemon]) => {
							const dexEntry = db.pokedex[pokemon.pokemonId];
							if (!dexEntry) {
								return null;
							}

							return (
								<div
									key={pokemon.id}
									className={'robin-rounded'}
									style={{ backgroundColor: 'Gray' }}
								>
									{dexEntry.name}

									{dexEntry.megaType}

									{JSON.stringify(dexEntry)}

									{pokemon.lastMega}
									{pokemon.megaCount}
								</div>
							);
					  })}

				<pre style={{ wordBreak: 'break-all' }}>
					{JSON.stringify(db?.pokemon, undefined, 2)}
				</pre>
			</ScrollWindow>
		</div>
	);
}

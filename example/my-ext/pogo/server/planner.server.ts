import {
	PokemonMegaValues,
	Species,
	Pokemon,
	nextMegaDeadline,
	computeEvolve,
} from '../domain-utils';
import { getDB } from './db.server';

export type MegaEvolveEvent = PokemonMegaValues & {
	date: string;
};

function naiveFreeMegaEvolve(
	now: Date,
	dexEntry: Species,
	pokemon: Pick<Pokemon, 'lastMegaEnd' | 'lastMegaStart' | 'megaCount'>,
): MegaEvolveEvent[] {
	let { megaCount, lastMegaEnd, lastMegaStart } = pokemon;
	const out: MegaEvolveEvent[] = [];

	let currentState = { megaCount, lastMegaEnd, lastMegaStart };
	while (currentState.megaCount < 30) {
		const deadline = nextMegaDeadline(
			currentState.megaCount,
			new Date(currentState.lastMegaEnd),
		);

		// Move time forwards until the deadline; however, if the deadline is in the past,
		// because its been a while since the last mega, don't accidentally go back in time.
		now = new Date(Math.max(now.getTime(), deadline.getTime()));

		const newState = computeEvolve(now, dexEntry, currentState);

		out.push({
			date: now.toISOString(),
			...newState,
		});

		currentState = {
			megaCount: newState.megaCount,
			lastMegaEnd: newState.lastMegaEnd,
			lastMegaStart: newState.lastMegaStart,
		};
	}

	return out;
}

export async function megaLevelPlanForPokemonRpc({
	id,
}: {
	id: string;
}): Promise<MegaEvolveEvent[]> {
	const db = getDB();
	const pokemon = db.pokemon[id];
	const dexEntry = db.pokedex[pokemon?.pokemonId ?? -1];

	// rome-ignore lint/complexity/useSimplifiedLogicExpression: idiotic rule
	if (!pokemon || !dexEntry) {
		return [];
	}

	const now = new Date();
	return naiveFreeMegaEvolve(now, dexEntry, pokemon);
}

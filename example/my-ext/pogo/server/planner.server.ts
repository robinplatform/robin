import {
	PokemonMegaValues,
	Species,
	Pokemon,
	nextMegaDeadline,
	computeEvolve,
} from '../domain-utils';
import { DAY_MS, arrayOfN } from '../math';
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

export type PlannerDay = {
	date: string;
	eventsToday: MegaEvolveEvent[];
};

export async function megaLevelPlanForPokemonRpc({
	id,
}: {
	id: string;
}): Promise<PlannerDay[]> {
	const db = getDB();
	const pokemon = db.pokemon[id];
	const dexEntry = db.pokedex[pokemon?.pokemonId ?? -1];

	// rome-ignore lint/complexity/useSimplifiedLogicExpression: idiotic rule
	if (!pokemon || !dexEntry) {
		return [];
	}

	const now = new Date();
	const plan = naiveFreeMegaEvolve(now, dexEntry, pokemon);

	const timeToLastEvent =
		plan.length === 0
			? 0
			: new Date(plan[plan.length - 1].date).getTime() - now.getTime();
	const daysToDisplay = Math.max(0, Math.ceil(timeToLastEvent / DAY_MS)) + 4;

	return arrayOfN(daysToDisplay)
		.map((i) => new Date(Date.now() + (i - 2) * DAY_MS))
		.map((date) => {
			const eventsToday = plan.filter(
				(e) => new Date(e.date).toDateString() === date.toDateString(),
			);

			return {
				date: date.toISOString(),
				eventsToday,
			};
		});
}

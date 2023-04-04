import {
	PokemonMegaValues,
	Species,
	Pokemon,
	nextMegaDeadline,
	computeEvolve,
} from '../domain-utils';
import { DAY_MS, arrayOfN, dateString } from '../math';
import { getDB } from './db.server';

// iterate forwards over lock points,
// iterate backwards in time from each lock point
// at the last lock point, iterate forwards in time

// TODO: add something to allow for checking the cost of daily level-ups
// TODO: add data that shows remaining mega energy

export type MegaEvolveEvent = PokemonMegaValues & {
	id?: string;
	date: string;
	megaEnergyAvailable: number;
};

function naiveFreeMegaEvolve(
	now: Date,
	dexEntry: Species,
	state: Pick<Pokemon, 'lastMegaEnd' | 'lastMegaStart' | 'megaCount'> & {
		megaEnergyAvailable: number;
	},
): MegaEvolveEvent[] {
	let { megaCount, lastMegaEnd, lastMegaStart, megaEnergyAvailable } = state;
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
			megaEnergyAvailable,
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
	const dexEntry = db.pokedex[pokemon?.pokedexId ?? -1];

	// rome-ignore lint/complexity/useSimplifiedLogicExpression: idiotic rule
	if (!pokemon || !dexEntry) {
		return [];
	}

	const now = new Date();

	const plans = db.evolvePlans.filter((plan) => {
		const date = new Date(plan.date);
		if (dateString(date) < dateString(now)) {
			return false;
		}

		return plan.pokemonId === id;
	});

	plans.sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime());

	let currentState = {
		date: now,
		lastMegaEnd: pokemon.lastMegaEnd,
		lastMegaStart: pokemon.lastMegaStart,
		megaCount: pokemon.megaCount,
		megaEnergyAvailable: dexEntry.megaEnergyAvailable,
	};
	const events: MegaEvolveEvent[] = [];
	for (const plan of plans) {
		const planDate = new Date(plan.date);

		const newState = computeEvolve(planDate, dexEntry, currentState);
		const megaEnergyAvailable =
			currentState.megaEnergyAvailable - newState.megaEnergySpent;
		events.push({
			date: plan.date,
			megaEnergyAvailable,
			id: plan.id,
			...newState,
		});

		currentState = {
			...newState,
			megaEnergyAvailable,
			date: new Date(Math.max(planDate.getTime(), currentState.date.getTime())),
		};
	}

	events.push(
		...naiveFreeMegaEvolve(currentState.date, dexEntry, currentState),
	);

	const timeToLastEvent =
		events.length === 0
			? 0
			: new Date(events[events.length - 1].date).getTime() - now.getTime();
	const daysToDisplay = Math.max(0, Math.ceil(timeToLastEvent / DAY_MS)) + 4;

	return arrayOfN(daysToDisplay)
		.map((i) => new Date(Date.now() + (i - 2) * DAY_MS))
		.map((date) => {
			const eventsToday = events.filter(
				(e) => new Date(e.date).toDateString() === date.toDateString(),
			);

			return {
				date: date.toISOString(),
				eventsToday,
			};
		});
}

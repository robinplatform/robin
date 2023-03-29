import { z } from 'zod';
import { lerp } from './math';

export type Species = z.infer<typeof Species>;
export const Species = z.object({
	number: z.number(),
	name: z.string(),

	megaEnergyAvailable: z.number(),
	initialMegaCost: z.number(),
	megaLevel1Cost: z.number(),
	megaLevel2Cost: z.number(),
	megaLevel3Cost: z.number(),

	megaType: z.array(z.string()),
});

export type Pokemon = z.infer<typeof Pokemon>;
export const Pokemon = z.object({
	id: z.string(),
	pokemonId: z.number(),
	lastMega: z.string(),
	megaCount: z.number(),
});

export const TypeColors: Record<string, string> = {
	normal: 'gray',
	steel: 'darkslategray',
	ground: 'chocolate',
	dark: 'darkslateblue',
	fire: 'orange',
	grass: 'green',
	poison: 'purple',
	flying: 'aqua',
	bug: 'chartreuse',
	psychic: 'lightcoral',
	fairy: 'lightpink',
	rock: 'peru',
	water: 'royalblue',
	dragon: 'SlateBlue',
	fighting: 'Maroon',
	electric: 'gold',
	ice: 'deepskyblue',
};

export const TypeTextColors: Record<string, string> = {
	normal: 'white',
	steel: 'white',
	ground: 'white',
	dark: 'white',
	fire: 'white',
	grass: 'white',
	poison: 'white',
	flying: 'black',
	bug: 'black',
	psychic: 'black',
	fairy: 'black',
	rock: 'white',
	water: 'white',
	dragon: 'white',
	fighting: 'white',
	electric: 'black',
	ice: 'white',
};

export const MegaRequirements = {
	1: 1,
	2: 7,
	3: 30,
} as const;

export const MegaWaitDays = {
	1: 7,
	2: 5,
	3: 3,
} as const;

export const MegaWaitTime = {
	1: 7 * 24 * 60 * 60 * 1000,
	2: 5 * 24 * 60 * 60 * 1000,
	3: 3 * 24 * 60 * 60 * 1000,
} as const;

export function megaLevelFromCount(count: number): 0 | 1 | 2 | 3 {
	switch (true) {
		case count >= 30:
			return 3;

		case count >= 7:
			return 2;

		case count >= 1:
			return 1;

		default:
			return 0;
	}
}

export function nextMegaDeadline(count: number, lastMega: Date): Date {
	const date = new Date(lastMega);
	const offset = MegaWaitTime[megaLevelFromCount(count)] ?? 0;
	date.setTime(date.getTime() + offset);

	return date;
}

export function megaCostForSpecies(
	dexEntry: Species,
	megaLevel: 0 | 1 | 2 | 3,
	timeSinceLastMega: number,
): number {
	if (megaLevel === 0) {
		return dexEntry.initialMegaCost;
	}

	let megaCost = 0;
	switch (megaLevel) {
		case 1:
			megaCost = dexEntry.megaLevel1Cost;
			break;
		case 2:
			megaCost = dexEntry.megaLevel2Cost;
			break;
		case 3:
			megaCost = dexEntry.megaLevel3Cost;
			break;
	}

	const megaCostProrated = lerp(
		0,
		megaCost,
		Math.min(1, Math.max(0, 1 - timeSinceLastMega / MegaWaitTime[megaLevel])),
	);
	return Math.ceil(megaCostProrated);
}

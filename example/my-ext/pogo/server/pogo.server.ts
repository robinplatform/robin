import { z } from 'zod';
import _ from 'lodash';
import fetch from 'node-fetch';
import { getDB, PogoDb, withDb } from './db.server';
import { getMegaPokemon } from './pogoapi.server';
import { nextMegaDeadline, Pokemon, Species } from '../domain-utils';

// Going to start by making a mega evolution planner.

function leekDuckGET<T>(path: string, shape: z.ZodSchema<T>): () => Promise<T> {
	return async () => {
		const resp = await fetch(
			`https://raw.githubusercontent.com/bigfoott/ScrapedDuck/data${path}`,
		);
		const data = await resp.json();
		return shape.parse(data);
	};
}

export const getUpcomingCommDays = leekDuckGET(
	'/events.min.json',
	z.array(
		z.object({
			eventID: z.string(),
			name: z.string(),
			eventType: z.string(),
			heading: z.string(),
			link: z.string(),
			image: z.string(),
			start: z.string(),
			end: z.string(),
			extraData: z
				.object({
					spotlight: z.object({
						name: z.string(),
						canBeShiny: z.boolean(),
						image: z.string(),
						bonus: z.string(),
					}),

					spawns: z.array(
						z.object({
							name: z.string(),
							image: z.string(),
						}),
					),

					bonuses: z.array(
						z.object({
							text: z.string(),
							image: z.string(),
						}),
					),

					bonusDisclaimers: z.array(z.string()),

					shinies: z.array(
						z.object({
							name: z.string(),
							image: z.string(),
						}),
					),

					specialresearch: z.array(
						z.object({
							name: z.string(),
							step: z.number(),
							tasks: z.array(
								z.object({
									text: z.string(),
									reward: z.object({
										text: z.string(),
										image: z.string(),
									}),
								}),
							),

							rewards: z.array(
								z.object({
									text: z.string(),
									image: z.string(),
								}),
							),
						}),
					),
				})
				.partial()
				.nullable(),
		}),
	),
);

export async function refreshDexRpc() {
	const pokemon = await getMegaPokemon();

	withDb((db) => {
		for (const entry of pokemon) {
			const prev: Species | undefined = db.pokedex[entry.pokemon_id];
			db.pokedex[entry.pokemon_id] = {
				megaEnergyAvailable: prev?.megaEnergyAvailable ?? 0,

				number: entry.pokemon_id,
				name: entry.pokemon_name,
				megaType: entry.type,

				initialMegaCost: entry.first_time_mega_energy_required,
				megaLevel1Cost: entry.mega_energy_required,
				megaLevel2Cost: entry.mega_energy_required / 2,
				megaLevel3Cost: entry.mega_energy_required / 4,
			};
		}
	});

	// We get a crash during JSON parsing if we don't return something here.
	return {};
}

const compareNames = (db: PogoDb, a: Pokemon, b: Pokemon) => {
	const aName = a.name ?? db.pokedex[a.pokemonId]?.name ?? '';
	const bName = b.name ?? db.pokedex[b.pokemonId]?.name ?? '';

	return aName.localeCompare(bName);
};

const compareMegaTimes = (nowTime: number, a: Pokemon, b: Pokemon) => {
	const aDeadline = nextMegaDeadline(
		a.megaCount,
		new Date(a.lastMegaEnd),
	).getTime();
	const bDeadline = nextMegaDeadline(
		b.megaCount,
		new Date(b.lastMegaEnd),
	).getTime();

	return Math.max(aDeadline, nowTime) - Math.max(bDeadline, nowTime);
};

export async function searchPokemonRpc({
	sort,
}: {
	sort: 'name' | 'pokemonId' | 'megaTime' | 'megaLevelUp';
}) {
	const db = getDB();

	const out = Object.values(db.pokemon);
	const now = new Date();
	const nowTime = now.getTime();
	switch (sort) {
		case 'name':
			out.sort((a, b) => compareNames(db, a, b));
			break;
		case 'pokemonId':
			out.sort((a, b) => {
				return a.pokemonId - b.pokemonId;
			});
			break;
		case 'megaTime':
			out.sort((a, b) => {
				const diff = compareMegaTimes(nowTime, a, b);
				if (diff !== 0) {
					return diff;
				}

				return compareNames(db, a, b);
			});
		case 'megaLevelUp':
			out.sort((a, b) => {
				if (a.megaCount < 30 && b.megaCount >= 30) {
					return -1;
				}
				if (a.megaCount >= 30 && b.megaCount < 30) {
					return 1;
				}

				const diff = compareMegaTimes(nowTime, a, b);
				if (diff !== 0) {
					return diff;
				}

				return compareNames(db, a, b);
			});
	}

	return out.map((pokemon) => pokemon.id);
}

import { z } from 'zod';
import _ from 'lodash';
import fetch from 'node-fetch';
import { withDb } from './db.server';
import { getMegaPokemon, getRegisteredPokemon } from './pogoapi.server';

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

export async function refreshDex() {
	const pokemon = await getMegaPokemon();

	withDb((db) => {
		for (const entry of pokemon) {
			db.pokedex[entry.pokemon_id] = {
				...db.pokedex[entry.pokemon_id],
				number: entry.pokemon_id,
				name: entry.pokemon_name,
				initialMegaEnergy: entry.first_time_mega_energy_required,
				megaEnergy: entry.mega_energy_required,
				megaType: entry.type,
			};
		}
	});

	return {};
}

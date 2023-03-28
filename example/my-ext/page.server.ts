import { onAppStart } from '@robinplatform/toolkit/daemon';
import { z } from 'zod';
import * as fs from 'fs';
import _ from 'lodash';
import * as path from 'path';
import fetch from 'node-fetch';

onAppStart(async () => {
	console.log('Server started for my-ext');
});

// Simple code to perform GET endpoint calls
// This is being done server-side instead of client-side
// because the PoGo API has some interesting behaviors
// like built-in endpoint hashing, so we want to eventually
// cache stuff.
function pogoApiEndpointGET<T>(
	path: string,
	shape: z.ZodSchema<T>,
): () => Promise<T> {
	return async () => {
		// TODO: handle caching, etc.

		const resp = await fetch(`https://pogoapi.net/api${path}`);
		const data = await resp.json();
		return shape.parse(data);
	};
}

export const getCommunityDays = pogoApiEndpointGET(
	'/v1/community_days.json',
	z.array(
		z.object({
			bonuses: z.array(z.string()),
			boosted_pokemon: z.array(z.string()),
			community_day_number: z.number(),
			end_date: z.string(),
			start_date: z.string(),
			event_moves: z.array(
				z.object({
					move: z.string(),
					move_type: z.string(),
					pokemon: z.string(),
				}),
			),
		}),
	),
);

export async function getSelfSource({
	filename,
}: {
	filename: string;
}): Promise<{ data: string }> {
	// Random lodash code to test modules
	_.each([1, 2, 3], () => {});

	return {
		data: await fs.promises.readFile(
			path.resolve(process.env.ROBIN_PROJECT_PATH ?? '', filename),
			'utf8',
		),
	};
}

// export async function getData({ filename });

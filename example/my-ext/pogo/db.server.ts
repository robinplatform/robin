import { z } from 'zod';
import * as fs from 'fs';
import _, { last } from 'lodash';
import * as path from 'path';
import produce from 'immer';
import * as os from 'os';
import { onAppStart } from '@robinplatform/toolkit/daemon';

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

export type PogoDb = z.infer<typeof PogoDb>;
const PogoDb = z.object({
	pokedex: z.record(z.coerce.number(), Species),
	pokemon: z.record(z.string(), Pokemon),
});

const EmptyDb: PogoDb = {
	pokedex: {},
	pokemon: {},
};

const DB_FILE = path.join(os.homedir(), '.a1liu-robin-pogo-db');
let DB: PogoDb = EmptyDb;

onAppStart(async () => {
	try {
		const text = await fs.promises.readFile(DB_FILE, 'utf8');
		const data = JSON.parse(text);
		DB = PogoDb.parse(data);
	} catch (e) {
		console.log('Failed to read from JSON', e);
		// TODO: better error handling
		DB = EmptyDb;
	}
});

export async function withDb(mut: (db: PogoDb) => void) {
	const newDb = produce(DB, mut);

	if (newDb !== DB) {
		console.log('DB access caused mutation');
		DB = newDb;
		await fs.promises.writeFile(DB_FILE, JSON.stringify(DB));
	}

	return newDb;
}

export async function fetchDb() {
	return withDb((db) => {});
}

export async function addPokemonRpc({ pokemonId }: { pokemonId: number }) {
	const id = `${pokemonId}-${Math.random()}`;
	await withDb((db) => {
		db.pokemon[id] = {
			id,
			pokemonId,
			lastMega: new Date().toISOString(),
			megaCount: 0,
		};
	});

	return {};
}

export async function setPokemonMegaCountRpc({
	id,
	count,
	lastMega,
}: {
	id: string;
	count: number;
	lastMega?: string;
}) {
	await withDb((db) => {
		const pokemon = db.pokemon[id];
		if (!pokemon) {
			return;
		}

		pokemon.megaCount = Math.max(count, 0);
		if (lastMega) {
			pokemon.lastMega = lastMega;
		}
	});

	return {};
}

export async function setPokemonMegaEnergyRpc({
	pokemonId,
	megaEnergy,
}: {
	pokemonId: number;
	megaEnergy: number;
}) {
	await withDb((db) => {
		const dexEntry = db.pokedex[pokemonId];
		if (!dexEntry) {
			return;
		}

		dexEntry.megaEnergyAvailable = Math.max(megaEnergy, 0);
	});

	return {};
}

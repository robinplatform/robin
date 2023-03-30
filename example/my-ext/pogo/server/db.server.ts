import { z } from 'zod';
import * as fs from 'fs';
import * as path from 'path';
import produce from 'immer';
import * as os from 'os';
import { onAppStart } from '@robinplatform/toolkit/daemon';
import {
	computeEvolve,
	isCurrentMega,
	megaCostForSpecies,
	megaLevelFromCount,
	Pokemon,
	Species,
} from '../domain-utils';
import { HOUR_MS } from '../math';

export type PogoDb = z.infer<typeof PogoDb>;
const PogoDb = z.object({
	pokedex: z.record(z.coerce.number(), Species),
	pokemon: z.record(z.string(), Pokemon),
	currentMega: z
		.object({
			id: z.string(),
		})
		.optional(),
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

let dbAccessActive = false;
let dbDirty = false;
const mutexQueue: ((u: unknown) => void)[] = [];

export async function withDb(mut: (db: PogoDb) => void) {
	if (dbAccessActive) {
		await new Promise((res) => mutexQueue.push(res));
	} else {
		dbAccessActive = true;
	}

	const newDb = produce(DB, mut);

	if (newDb !== DB) {
		console.log('DB access caused mutation');
		dbDirty = true;
		DB = newDb;
	}

	const waiter = mutexQueue.shift();
	if (waiter) {
		waiter(null);
		return newDb;
	}

	if (dbDirty) {
		await fs.promises.writeFile(DB_FILE, JSON.stringify(DB));
		dbDirty = false;
	}

	dbAccessActive = false;

	return newDb;
}

export async function fetchDbRpc() {
	return DB;
}

export function getDB() {
	return DB;
}

export async function addPokemonRpc({ pokemonId }: { pokemonId: number }) {
	const id = `${pokemonId}-${Math.random()}`;
	const now = new Date().toISOString();
	await withDb((db) => {
		db.pokemon[id] = {
			id,
			pokemonId,
			megaCount: 0,

			// This causes some strange behavior but... it's probably fine.
			lastMegaStart: now,
			lastMegaEnd: now,
		};
	});

	return {};
}

export async function evolvePokemonRpc({ id }: { id: string }) {
	await withDb((db) => {
		const pokemon = db.pokemon[id];
		const dexEntry = db.pokedex[pokemon.pokemonId];

		// rome-ignore lint/complexity/useSimplifiedLogicExpression: I'm not fucking applying demorgan's law to this
		if (!pokemon || !dexEntry) return;

		const now = new Date();

		if (isCurrentMega(db.currentMega?.id, pokemon, now)) {
			console.log('Tried to evolve the currently evolved pokemon');
			return;
		}

		const megaLevel = megaLevelFromCount(pokemon.megaCount);
		const megaCost = megaCostForSpecies(
			dexEntry,
			megaLevel,
			new Date().getTime() - new Date(pokemon.lastMegaEnd).getTime(),
		);

		const nextData = computeEvolve(now, {
			megaCost,
			megaCount: pokemon.megaCount,
			megaEnergyAvailable: dexEntry.megaEnergyAvailable,
			lastMegaStart: pokemon.lastMegaStart,
			lastMegaEnd: pokemon.lastMegaEnd,
		});

		dexEntry.megaEnergyAvailable = nextData.megaEnergyAvailable;
		pokemon.lastMegaStart = nextData.lastMegaStart;
		pokemon.lastMegaEnd = nextData.lastMegaEnd;
		pokemon.megaCount = nextData.megaCount;

		const currentMega = db.pokemon[db.currentMega?.id ?? ''];
		if (currentMega) {
			const prevMegaEnd = new Date(currentMega.lastMegaEnd);
			currentMega.lastMegaEnd = new Date(
				Math.min(now.getTime(), prevMegaEnd.getTime()),
			).toISOString();
		}

		db.currentMega = { id };
	});

	return {};
}

export async function setPokemonMegaEndRpc({
	id,
	newMegaEnd,
}: {
	id: string;
	newMegaEnd: string;
}) {
	await withDb((db) => {
		const pokemon = db.pokemon[id];
		if (!pokemon) return;

		pokemon.lastMegaEnd = newMegaEnd;
		const newMegaDate = new Date(newMegaEnd);

		const newMegaDateEightHoursBefore = new Date(
			newMegaDate.getTime() - 8 * HOUR_MS,
		);

		const lastMegaStartDate = new Date(pokemon.lastMegaStart);
		if (newMegaDate < lastMegaStartDate) {
			pokemon.lastMegaStart = newMegaEnd;
		}
		if (newMegaDateEightHoursBefore > lastMegaStartDate) {
			pokemon.lastMegaStart = newMegaEnd;
		}
	});

	return {};
}

export async function setPokemonMegaCountRpc({
	id,
	count,
}: {
	id: string;
	count: number;
}) {
	await withDb((db) => {
		const pokemon = db.pokemon[id];
		if (!pokemon) return;

		pokemon.megaCount = Math.min(Math.max(count, 0), 30);
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
		if (!dexEntry) return;

		dexEntry.megaEnergyAvailable = Math.max(megaEnergy, 0);
	});

	return {};
}

export async function deletePokemonRpc({ id }: { id: string }) {
	await withDb((db) => {
		// rome-ignore lint/performance/noDelete: fucking idiot rule
		delete db.pokemon[id];
	});

	return {};
}

export async function setNameRpc({ id, name }: { id: string; name: string }) {
	await withDb((db) => {
		const pokemon = db.pokemon[id];
		if (!pokemon) return;

		pokemon.name = name;
	});

	return {};
}

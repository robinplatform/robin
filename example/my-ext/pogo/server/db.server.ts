import { z } from 'zod';
import * as fs from 'fs';
import * as path from 'path';
import produce from 'immer';
import * as os from 'os';
import { onAppStart, Topic } from '@robinplatform/toolkit/daemon';
import {
	computeEvolve,
	isCurrentMega,
	Pokemon,
	Species,
} from '../domain-utils';
import { HOUR_MS } from '../math';
import { Low } from 'lowdb';
import { JSONFile } from 'lowdb/node';

class Mutex {
	private lastLockWaiter = Promise.resolve();

	// Uses an implicit chain of promises and control flow stuffs. IDK if I like it, but it is VERY short.
	// https://stackoverflow.com/questions/51086688/mutex-in-javascript-does-this-look-like-a-correct-implementation
	async lock(): Promise<() => void> {
		const waitFor = this.lastLockWaiter;

		let unlock: () => void;
		this.lastLockWaiter = new Promise((res) => {
			unlock = () => res();
		});

		return waitFor.then((r) => unlock);
	}

	async withLock<T>(f: () => Promise<T>): Promise<T> {
		const unlock = await this.lock();
		try {
			return await f();
		} finally {
			unlock();
		}
	}
}

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

const DB_FILE = path.join(os.homedir(), '.a1liu-robin-pogo-db');
const DB = new Low<PogoDb>(new JSONFile(DB_FILE));
const EmptyDb: PogoDb = {
	pokedex: {},
	pokemon: {},
};
DB.data = EmptyDb;

onAppStart(async () => {
	try {
		await DB.read();
		DB.data = PogoDb.parse(DB.data);
	} catch (e) {
		console.log('Failed to read from JSON', e);
	}
});

let dbModifiedTopic = undefined as unknown as Topic<{}>;

onAppStart(async () => {
	dbModifiedTopic = await Topic.createTopic(['pogo'], 'db');
});

export async function withDb(mut: (db: PogoDb) => void) {
	const newDb = produce(DB.data, mut);
	if (newDb !== DB.data) {
		console.log('DB access caused mutation');

		// TODO: don't do this on literally every write. Maybe do it once a second.
		await fs.promises.writeFile(DB_FILE, JSON.stringify(newDb));

		await dbModifiedTopic.publish({}).catch((e) => console.error('err', e));
		DB.data = newDb;
	}

	return newDb;
}

export async function setDbValueRpc({ db }: { db: PogoDb }) {
	return await withDb((prev) => {
		prev.pokedex = db.pokedex;
		prev.pokemon = db.pokemon;
		prev.currentMega = db.currentMega;
	});
}

export async function fetchDbRpc(): Promise<PogoDb> {
	return DB.data ?? EmptyDb;
}

export function getDB(): PogoDb {
	return DB.data ?? EmptyDb;
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

		const nextData = computeEvolve(now, dexEntry, pokemon);

		dexEntry.megaEnergyAvailable -= Math.min(
			dexEntry.megaEnergyAvailable,
			nextData.megaEnergySpent,
		);

		pokemon.lastMegaStart = nextData.lastMegaStart;
		pokemon.lastMegaEnd = nextData.lastMegaEnd;
		pokemon.megaCount = nextData.megaCount;

		// If there's a pokemon who is set as "currentMega", and their mega evolution is
		// still active, we should update their mega time. If they're not still active,
		// we can safely ignore their mega time.
		//
		// It might be possible to write this condition a little cleaner, but for now,
		// this is fine.
		const currentMega = db.pokemon[db.currentMega?.id ?? ''];
		if (currentMega && currentMega.id !== pokemon.id) {
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

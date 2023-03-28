import { z } from 'zod';
import * as fs from 'fs';
import _ from 'lodash';
import * as path from 'path';
import produce from 'immer';
import * as os from 'os';
import { onAppStart } from '@robinplatform/toolkit/daemon';

export type Species = z.infer<typeof Species>;
export const Species = z.object({
	number: z.number(),
	name: z.string(),
	megaEnergy: z.number(),
	initialMegaEnergy: z.number(),
	megaType: z.array(z.string()),
});

export type Pokemon = z.infer<typeof Pokemon>;
export const Pokemon = z.object({
	id: z.number(),
});

export type PogoDb = z.infer<typeof PogoDb>;
const PogoDb = z.object({
	pokedex: z.record(z.coerce.number(), Species),
	pokemon: z.array(Pokemon),
});

const EmptyDb: PogoDb = {
	pokedex: {},
	pokemon: [],
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

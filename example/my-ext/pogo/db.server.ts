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
});

export type Pokemon = z.infer<typeof Pokemon>;
export const Pokemon = z.object({
	pokemon: z.number(),
});

export type PogoDb = z.infer<typeof PogoDb>;
const PogoDb = z.object({
	pokedex: z.array(Species),
	pokemon: z.array(Pokemon),
});

const EmptyDb: PogoDb = {
	pokedex: [],
	pokemon: [],
};

const DB_FILE = path.join(os.homedir(), '.robin-pogo-db');
let DB: PogoDb = EmptyDb;

onAppStart(async () => {
	try {
		const text = await fs.promises.readFile(DB_FILE, 'utf8');
		const data = JSON.parse(text);
		DB = PogoDb.parse(data);
	} catch (e) {
		// TODO: better error handling
		DB = EmptyDb;
	}
});

export async function withDb(mut: (PogoDb) => void) {
	const newDb = produce(DB, mut);

	if (newDb !== DB) {
		console.log('DB access caused mutation');
		DB = newDb;
		await fs.promises.writeFile(DB_FILE, JSON.stringify(DB));
	}
}

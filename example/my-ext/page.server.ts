import * as fs from 'fs';

export async function getSelfSource({ filename }: { filename: string }) {
	return fs.promises.readFile(filename, 'utf8');
}

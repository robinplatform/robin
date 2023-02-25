import * as fs from 'fs';
import _ from 'lodash';

export async function getSelfSource({ filename }: { filename: string }) {
	// Random lodash code to test modules
	_.each([1, 2, 3], () => {});

	return fs.promises.readFile(filename, 'utf8');
}

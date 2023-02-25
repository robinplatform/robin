import * as fs from 'fs';

export async function getSelfSource() {
    return fs.promises.readFile(__filename, 'utf8');
}

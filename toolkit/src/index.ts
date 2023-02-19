import { Stream } from './stream';

// Setting this up as a function so that there's no
// weirdness with object mutation.
export const getDefaultFetchSettings = () =>
	({
		cache: 'no-cache',
		credentials: 'same-origin',
		headers: {
			'Content-Type': 'application/json',
		},
		redirect: 'follow',
		referrerPolicy: 'no-referrer',
		method: 'POST',
	} as const);

type Config = {
	releaseChannel: 'dev' | 'beta' | 'stable' | 'nightly';
	environments: Record<string, Record<string, string>>;
	extensions: Record<string, Record<string, any>>;
	minifyExtensionClients: boolean;
	keyMappings: Record<string, string>;
	enableKeyMappings: boolean;
};

export const getConfig = async (): Promise<Config> => {
	const resp = await fetch('/api/rpc/GetConfig', {
		...getDefaultFetchSettings(),
	});
	const value = await resp.json();
	return value;
};

export const updateConfig = async (newValue: string) => {
	const resp = await fetch('/api/rpc/UpdateConfig', {
		...getDefaultFetchSettings(),
		body: newValue,
	});
	const value = await resp.json();
	return value;
};

export const getExtensions = async () => [] as any[];

export const getVersion = async () => ({
	robin: {
		version: '',
		releaseChannel: 'dev' as 'dev' | 'stable' | 'beta' | 'nightly',
	},
});

export const checkForUpdates = async () => ({
	needsUpgrade: false,
	currentVersion: '',
	latestVersion: '',
	channel: 'dev' as 'dev' | 'stable' | 'beta' | 'nightly',
});

export const upgradeRobin = async () => {};

export const setRobinChannel = async () => {};

export const getHeartbeat = async (): Promise<Stream> => {
	return Stream.callStreamRpc('GetHeartbeat', Math.random() + 'adsf');
};

export { Stream };

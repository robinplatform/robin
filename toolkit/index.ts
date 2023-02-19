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

export const getConfig = async () => {
	const resp = await fetch('/api/rpc/GetConfig', {
		...getDefaultFetchSettings(),
	});
	const value = await resp.json();
	return value;
};

export const updateConfig = async () => {};

export const getExtensions = async () => [] as any[];

export const getVersion = async () => ({
	robin: {
		version: '',
		releaseChannel: 'dev' as 'dev' | 'stable' | 'beta' | 'nightly',
	},
});

export const checkForSidekickUpdates = async () => ({
	needsUpgrade: false,
	currentVersion: '',
	latestVersion: '',
	channel: 'dev' as 'dev' | 'stable' | 'beta' | 'nightly',
});

export const upgradeSidekick = async () => {};

export const setSidekickChannel = async () => {};

export const getHeartbeat = async (): Promise<Stream> => {
	return Stream.callStreamRpc('GetHeartbeat', Math.random() + 'adsf');
};

export { Stream };

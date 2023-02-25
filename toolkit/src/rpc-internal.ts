const rpcBaseUrl = `${window.location.protocol}//${window.location.host}`;

/**
 * Creates a wrapper function that can be used to transparently call a remote
 * method on the server. This is intended to be used by the robin compiler to
 * generate client-side RPC methods, but can also be used manually.
 *
 * @param {string} serverFile The absolute path to the server file that contains the method implementation.
 * @param {string} methodName The name of the method to call. This must match the exported function name from the server file.
 */
export function createRpcMethod({
	serverFile,
	methodName,
}: {
	serverFile: string;
	methodName: string;
}) {
	return Object.assign(
		async function rpcMethodWrapper(data: unknown) {
			const url = new URL('/api/apps/v0/rpc', rpcBaseUrl);
			const res = await fetch(url.toString(), {
				method: 'POST',
				body: JSON.stringify({ serverFile, methodName, data }),
				keepalive: true,
			});
			if (!res.ok) {
				// TODO: Attempt to extract error message from response body
				throw new Error(`RPC failed with status ${res.status}`);
			}
			return res.json();
		},
		{
			serverFile,
			methodName,
		},
	);
}

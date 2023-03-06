const rpcBaseUrl = `${window.location.protocol}//${window.location.host}`;

/**
 * Creates a wrapper function that can be used to transparently call a remote
 * method on the server. This is intended to be used by the robin compiler to
 * generate client-side RPC methods, but can also be used manually.
 *
 * @param {string} serverFile The absolute path to the server file that contains the method implementation.
 * @param {string} methodName The name of the method to call. This must match the exported function name from the server file.
 */
export function createRpcMethod(
	appId: string,
	serverFile: string,
	methodName: string,
) {
	return Object.assign(
		async function rpcMethodWrapper(data: unknown) {
			const url = new URL('/api/internal/rpc/RunAppMethod', rpcBaseUrl);
			const res = await fetch(url.toString(), {
				method: 'POST',
				body: JSON.stringify({ appId, serverFile, methodName, data }),
				keepalive: true,
			});

			const resBody = await res.json();
			if (typeof resBody === 'object' && resBody && resBody.type === 'error') {
				throw Object.assign(new Error(resBody.error), {
					...resBody,
					status: res.status,
				});
			}

			if (!res.ok) {
				throw new Error(`${methodName} failed with status ${res.status}`);
			}

			return resBody;
		},
		{
			getQueryKey: (data: unknown) => [serverFile, methodName, data],
		},
	);
}

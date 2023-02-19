import { getConfig, getHeartbeat } from '@robin/toolkit';
import React from 'react';
import { useQuery } from 'react-query';

export default function Home() {
	const { data: config } = useQuery({
		queryKey: ['getConfig'],
		queryFn: getConfig,
	});

	React.useEffect(() => {
		const runner = async () => {
			const stream = await getHeartbeat();
			stream.onmessage = (data) => {
				console.log('heartbeat-message', data);
			};
			stream.onerror = (err) => {
				console.log('error', err);
			};
			stream.start({});
		};
		runner();
	}, []);

	return (
		<div>
			Hello world!
			<pre>
				<code>{JSON.stringify(config, null, '  ')}</code>
			</pre>
		</div>
	);
}

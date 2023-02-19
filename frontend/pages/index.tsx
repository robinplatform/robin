import { getConfig, getHeartbeat } from '@robinplatform/toolkit';
import Head from 'next/head';
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
		<div className={'robin-bg-dark-slate robin-pad full'}>
			<Head>
				<title>Robin</title>
			</Head>

			<div className={'full col robin-rounded robin-bg-slate robin-pad'}>
				Hello world!
				<pre>
					<code>{JSON.stringify(config, null, '  ')}</code>
				</pre>
			</div>
		</div>
	);
}

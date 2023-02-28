import Head from 'next/head';
import React from 'react';

export default function Home() {
	return (
		<div className={'robin-bg-dark-blue robin-pad full'}>
			<Head>
				<title>Robin</title>
			</Head>

			<div className={'full col robin-rounded robin-pad'}>Hello world!</div>
		</div>
	);
}
